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
	"github.com/viant/mly/service/tfmodel/batcher"
	batchconfig "github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/service/tfmodel/evaluator"
	"github.com/viant/mly/service/tfmodel/signature"
	tfstat "github.com/viant/mly/service/tfmodel/stat"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v2"
)

// Service is responsible for being the entrypoint for all Tensorflow
// model runs.
// It manages loading and reloading the model files, as well as providing
// metadata based off the model and configuration.
type Service struct {
	// Modifies this object to be used by config endpoints.
	config *config.Model

	evaluatorMeta *evaluator.EvaluatorMeta
	batcherConfig *batchconfig.BatcherConfig
	batcherMetric *gmetric.Operation

	// transitory evaluator
	batcher   *batcher.Service
	evaluator *evaluator.Service

	mux sync.RWMutex
	wg  *sync.WaitGroup // prevents calling to a closed Evaluator

	inputs     map[string]*domain.Input
	signature  *domain.Signature
	dictionary *common.Dictionary

	fs afs.Service

	// Should point to service.Service.ReloadOK
	ReloadOK *int32
}

func (s *Service) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	s.mux.RLock()
	// Maybe use interface?
	var batcher *batcher.Service
	var evaluator *evaluator.Service
	if s.batcher != nil {
		batcher = s.batcher
	} else {
		evaluator = s.evaluator
	}
	wg := s.wg

	wg.Add(1)
	s.mux.RUnlock()

	var tv []interface{}
	var err error
	if batcher != nil {
		tv, err = batcher.Evaluate(ctx, params)
	} else {
		tv, err = evaluator.Evaluate(ctx, params)
	}

	wg.Done()

	return tv, err
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
		atomic.StoreInt32(s.ReloadOK, 1)
		return nil
	}

	model, err := s.loadModel(ctx, err)
	if err != nil {
		return err
	}

	signature, err := signature.Signature(model)
	if err != nil {
		return fmt.Errorf("signature error:%w", err)
	}

	// modifies signature.Inputs[].Vocab for Dictionary()
	reconcileIOFromSignature(s.config, signature)

	var dictionary *common.Dictionary

	useDict := s.config.UseDictionary()
	if useDict {
		s.config.DictMeta.Error = ""

		if s.config.DictURL != "" {
			// Deprecated branch
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

	if s.config.OutputType != "" {
		signature.Output.DataType = s.config.OutputType
	}

	newEvaluator := evaluator.NewEvaluator(signature, model.Session, *s.evaluatorMeta)

	var newBatchSrv *batcher.Service
	if s.config.Batch.MaxBatchCounts > 1 {
		newBatchSrv = batcher.NewBatcher(newEvaluator, len(signature.Inputs), (*s.config.Batch).BatcherConfig)
	}

	// modify all service objects

	s.mux.Lock()

	oldBatcher := s.batcher
	s.batcher = newBatchSrv

	oldEvaluator := s.evaluator
	s.evaluator = newEvaluator

	oldWg := s.wg
	s.wg = new(sync.WaitGroup)

	s.mux.Unlock()
	// from this point, nothing should pick up the oldEvaluator or oldBatcher, but there
	// may still be goroutines that have yet to call oldEvaluator.Evaluate()
	// but they should have Add()-ed to the oldWg.
	go func() {
		if oldWg != nil {
			oldWg.Wait()
		}

		if oldBatcher != nil {
			oldBatcher.Close()
		}

		if oldEvaluator != nil {
			oldEvaluator.Close()
		}
	}()

	if dictionary != nil {
		s.dictionary = dictionary

		// updates status as shown in /v1/api/config/
		s.config.DictMeta.Hash = dictionary.Hash
		s.config.DictMeta.Reloaded = time.Now()
	}

	// updates status as shown in /v1/api/config/
	s.config.Modified = snapshot

	// in theory, these should never materially change, unless the
	// model IO changes, which will result in undefined behavior.
	s.signature = signature
	s.inputs = modelInputsByName

	atomic.StoreInt32(s.ReloadOK, 1)
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
// config.Inputs[].Auxiliary may be modified.
// config.Outputs may be modified
func reconcileIOFromSignature(config *config.Model, signature *domain.Signature) {
	configuredInputsByName := config.FieldByName()

	if len(signature.Inputs) == 0 {
		return
	}

	// go through inputs from the model
	for modelInputName := range signature.Inputs {
		// use pointer to modify object
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

	// if another reloadModelIfNeeded() is running, wait until it is completed...
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

func (s *Service) Inputs() map[string]*domain.Input {
	return s.inputs
}

func (s *Service) Signature() *domain.Signature {
	return s.signature
}

func (s *Service) Dictionary() *common.Dictionary {
	return s.dictionary
}

func (s *Service) Close() error {
	if s.evaluator == nil {
		return nil
	}

	return s.evaluator.Close()
}

// NewService creates an unprepared Service.
// This service isn't ready until RelodIfNeeded() is called.
func NewService(cfg *config.Model, fs afs.Service, metrics *gmetric.Service, sema *semaphore.Weighted) *Service {
	location := reflect.TypeOf(&Service{}).PkgPath()

	id := cfg.ID

	semaMetric := metrics.MultiOperationCounter(location, id+"Semaphore", id+" Tensorflow semaphore", time.Microsecond, time.Minute, 2, tfstat.NewSema())
	tfMetric := metrics.MultiOperationCounter(location, id+"TFService", id+" Tensorflow performance", time.Microsecond, time.Minute, 2, tfstat.NewTfs())
	meta := evaluator.MakeEvaluatorMeta(sema, semaMetric, tfMetric)

	return &Service{
		evaluatorMeta: &meta,
		config:        cfg,
		fs:            fs,
	}
}
