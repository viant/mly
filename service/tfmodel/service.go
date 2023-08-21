package tfmodel

import (
	"compress/gzip"
	"context"
	sjson "encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/afs"
	"github.com/viant/afs/option"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/files"
	srvstat "github.com/viant/mly/service/stat"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/stat"
	smetric "github.com/viant/mly/shared/stat/metric"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v2"
)

type Service struct {
	// prevents potentially explosive thread generation due to concurrent requests
	// this should be shared across all Evaluators.
	semaphore *semaphore.Weighted

	// Modifies this object to be used by config endpoints.
	config *config.Model

	inputs     map[string]*domain.Input
	signature  *domain.Signature
	dictionary *common.Dictionary

	fs       afs.Service
	mux      sync.RWMutex
	ReloadOK int32

	evaluator *Evaluator

	metric *gmetric.Operation
}

func (s *Service) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	err := s.semaphore.Acquire(ctx, 1)
	if err != nil {
		return nil, err
	}

	// even if canceled/deadline exceeded, we're going to run the eval
	defer s.semaphore.Release(1)

	startTime := time.Now()
	onDone := s.metric.Begin(startTime)
	onPendingDone := smetric.EnterThenExit(s.metric, startTime, stat.Enter, stat.Exit)
	stats := stat.NewValues()

	defer func() {
		onDone(time.Now(), stats.Values()...)
		onPendingDone()
	}()

	s.mux.RLock()
	tv, err := s.evaluator.Evaluate(params)
	s.mux.RUnlock()

	if err != nil {
		stats.Append(err)
		return nil, err
	}

	return tv, err
}

func (s *Service) Inputs() map[string]*domain.Input {
	return s.inputs
}

func (s *Service) Signature() *domain.Signature {
	return s.signature
}

func (s *Service) Dictionary() *common.Dictionary {
	return s.dictionary
}

// Assumes that after the initial reload, there is no significant changes
// to the inputs and outputs from reloading the model.
// If a model reload results in changes to inputs or outputs, the resulting
// behavior is undefined.
func (s *Service) ReloadIfNeeded(ctx context.Context) error {
	snapshot, err := files.ModifiedSnapshot(ctx, s.fs, s.config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to check changes:%w", err)
	}

	if !s.isModified(snapshot) {
		atomic.StoreInt32(&s.ReloadOK, 1)
		return nil
	}

	model, err := s.loadModel(ctx, err)
	if err != nil {
		return err
	}

	signature, err := Signature(model)
	if err != nil {
		return fmt.Errorf("signature error:%w", err)
	}

	// modifies signature.Inputs[].Vocab
	reconcileIOFromSignature(s.config, signature)

	var dictionary *common.Dictionary

	useDict := s.config.UseDictionary()
	if useDict {
		s.config.DictMeta.Error = ""

		if s.config.DictURL != "" {
			if dictionary, err = s.loadDictionary(ctx, s.config.DictURL); err != nil {
				s.config.DictMeta.Error = err.Error()
				return err
			}
		} else {
			// extract dictionary from the graph, relies on signature being modified by reconcileIOFromSignature
			dictionary, err = Dictionary(model.Session, model.Graph, signature)
			if err != nil {
				s.config.DictMeta.Error = err.Error()
				return fmt.Errorf("dictionary error:%w", err)
			}
		}

		if dictionary != nil {
			var filehash int64
			if useDict && len(dictionary.Layers) == 0 {
				filehash = snapshot.Min.Unix() + snapshot.Max.Unix()
			}

			dictionary.UpdateHash(filehash)
		}
	}

	var modelInputsByName = make(map[string]*domain.Input)
	for i, modelInput := range signature.Inputs {
		modelInputsByName[modelInput.Name] = &signature.Inputs[i]
	}

	// add inputs from the config that aren't in the model
	for _, configInput := range s.config.Inputs {
		configInputName := configInput.Name
		if _, ok := modelInputsByName[configInputName]; ok {
			continue
		}

		input := &domain.Input{
			Name:      configInputName,
			Index:     configInput.Index,
			Auxiliary: configInput.Auxiliary,
		}

		input.Type = configInput.RawType()
		if input.Type == nil {
			input.Type = reflect.TypeOf("")
		}

		modelInputsByName[configInputName] = input
	}

	evaluator := NewEvaluator(signature, model.Session)

	if s.config.OutputType != "" {
		signature.Output.DataType = s.config.OutputType
	}

	// modify all service objects
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.evaluator != nil {
		// this should call Close on the old evaluator
		go s.evaluator.Close()
	}

	s.evaluator = evaluator

	if dictionary != nil {
		s.dictionary = dictionary
		// this updates status as shown in /v1/api/config/
		s.config.DictMeta.Hash = dictionary.Hash
		s.config.DictMeta.Reloaded = time.Now()
	}

	// this updates status as shown in /v1/api/config/
	s.config.Modified = snapshot

	// in theory, these should never materially change, unless the
	// model IO changes, which will result in undefined behavior.
	s.signature = signature
	s.inputs = modelInputsByName

	atomic.StoreInt32(&s.ReloadOK, 1)
	return nil
}

// Attempts to figure out input and output signatures of the model and compares them to
// the configured inputs and outputs.
// Generally, the configured values will override actual values.
// Additionally, any other inputs (auxiliary) will be added.
//
// signature.Inputs[].Vocab may be modified.
// config.Inputs may be modified.
// config.Inputs[].DataType may be modified.
// config.Inputs[].rawType may be modified.
func reconcileIOFromSignature(config *config.Model, signature *domain.Signature) {
	configuredInputsByName := config.FieldByName()

	if len(signature.Inputs) == 0 {
		return
	}

	for modelInputName := range signature.Inputs {
		// use pointer to actual object
		modelInput := &signature.Inputs[modelInputName]

		if modelInput.Type == nil {
			// when would this happen?
			modelInput.Type = reflect.TypeOf("")
		}

		configuredInput, ok := configuredInputsByName[modelInput.Name]
		if !ok {
			configuredInput = &shared.Field{Name: modelInput.Name}
			config.Inputs = append(config.Inputs, configuredInput)
		}

		modelInput.Vocab = !configuredInput.Wildcard && configuredInput.Precision <= 0

		if configuredInput.DataType == "" {
			// If the datatype is not provided in the configuration, overwrite it from the
			// model signature.
			configuredInput.SetRawType(modelInput.Type)
		}

		// remove the configured input as it is "handled"
		delete(configuredInputsByName, configuredInput.Name)
	}

	if len(signature.Outputs) > 0 {
		outputIndex := config.OutputIndex()
		for _, output := range signature.Outputs {
			if _, has := outputIndex[output.Name]; has {
				continue
			}

			field := &shared.Field{Name: output.Name, DataType: output.DataType}
			if field.DataType == "" {
				field.SetRawType(reflect.TypeOf(""))
			}

			config.Outputs = append(config.Outputs, field)
		}
	}

	for k, v := range configuredInputsByName {
		if v.DataType == "" {
			v.SetRawType(reflect.TypeOf(""))
		}

		configuredInputsByName[k].Auxiliary = true
	}
}

func (s *Service) loadModel(ctx context.Context, err error) (*tf.SavedModel, error) {
	options := option.NewSource(&option.NoCache{Source: option.NoCacheBaseURL})

	remoteURL := s.config.URL
	localPath := s.config.Location

	if err := s.fs.Copy(ctx, remoteURL, localPath, options); err != nil {
		return nil, fmt.Errorf("failed to copy model %v, %s, due to %w", remoteURL, localPath, err)
	}

	log.Printf("[%s loadModel] copied %s to %s", s.config.ID, remoteURL, localPath)

	model, err := tf.LoadSavedModel(localPath, s.config.Tags, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %v, %s, due to %w", remoteURL, localPath, err)
	}

	return model, nil
}

func (s *Service) isModified(snapshot *config.Modified) bool {
	if snapshot.Span() > time.Hour || snapshot.Max.IsZero() {
		return false
	}

	s.mux.RLock()
	modified := s.config.Modified
	s.mux.RUnlock()

	return !(modified.Max.Equal(snapshot.Max) && modified.Min.Equal(snapshot.Min))
}

// Deprecated: model metadata should be embedded.
// Loads common.Dictionary from a remote source.
func (s *Service) loadDictionary(ctx context.Context, URL string) (*common.Dictionary, error) {
	var result = &common.Dictionary{}
	rawReader, err := s.fs.OpenURL(ctx, URL)
	if err != nil {
		return nil, err
	}

	defer rawReader.Close()
	var reader io.Reader = rawReader
	if strings.HasSuffix(URL, ".gz") {
		if reader, err = gzip.NewReader(rawReader); err != nil {
			return nil, err
		}
	}

	if strings.Contains(URL, ".yaml") {
		decoder := yaml.NewDecoder(reader)
		return result, decoder.Decode(result)
	}

	decoder := sjson.NewDecoder(reader)
	return result, decoder.Decode(result)
}

func (s *Service) Close() error {
	if s.evaluator == nil {
		return nil
	}

	return s.evaluator.Close()
}

func NewService(cfg *config.Model, fs afs.Service, metrics *gmetric.Service, sema *semaphore.Weighted) *Service {
	location := reflect.TypeOf(&Service{}).PkgPath()

	return &Service{
		semaphore: sema,
		config:    cfg,
		fs:        fs,
		metric:    metrics.MultiOperationCounter(location, cfg.ID+"TFService", cfg.ID+" tensorflow performance", time.Microsecond, time.Minute, 2, srvstat.NewEval()),
	}
}
