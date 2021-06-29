package service

import (
	"context"
	"fmt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/gtly"
	"github.com/viant/mly/common"
	"github.com/viant/mly/common/storable"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/layers"
	"github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/datastore"
	sstat "github.com/viant/mly/shared/stat"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

//Service represents ml service
type Service struct {
	mux             sync.RWMutex
	fs              afs.Service
	evaluator       *tfmodel.Evaluator
	signature       *domain.Signature
	dictionary      *domain.Dictionary
	inputs          map[string]*domain.Input
	useDatastore    bool
	datastore       *datastore.Service
	transformer     domain.Transformer
	newStorable     func() common.Storable
	modified        time.Time
	config          *config.Model
	serviceMetric   *gmetric.Operation
	evaluatorMetric *gmetric.Operation
	closed          int32
	inputProvider   *gtly.Provider
}

//Close closes service
func (s *Service) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return fmt.Errorf("already closed")
	}
	if s.evaluator == nil {
		return nil
	}
	return s.evaluator.Close()
}

//Config returns service config
func (s *Service) Config() *config.Model {
	return s.config
}

//Dictionary returns servie dictionary
func (s *Service) Dictionary() *domain.Dictionary {
	return s.dictionary
}

//Do handles service request
func (s *Service) Do(ctx context.Context, request *Request, response *Response) error {
	err := s.do(ctx, request, response)
	if err != nil {
		response.Error = err.Error()
		response.Status = "error"
	}
	return nil
}

func (s *Service) do(ctx context.Context, request *Request, response *Response) error {
	onDone := s.serviceMetric.Begin(time.Now())
	stats := sstat.NewValues()
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()

	useDatastore := s.useDatastore && request.Key != ""
	var key *datastore.Key
	if useDatastore {
		response.DictHash = s.dictionary.Hash
		key = datastore.NewKey(s.datastore.Config, request.Key)
		sink := s.newStorable()

		err := s.datastore.GetInto(ctx, key, sink)
		if err == nil {
			hasher, isHasher := sink.(common.Hashed)
			if !isHasher {
				response.Data = sink
				return nil
			}
			sinkHash := hasher.Hash()
			if hasHashMatch := sinkHash == s.dictionary.Hash; hasHashMatch || sinkHash == 0 {
				response.Data = sink
				return nil
			}
		}
	}
	stats.Append(stat.EvalKey)
	output, err := s.evaluate(ctx, request.Feeds)
	if err != nil {
		return err
	}
	transformed, err := s.transformer(ctx, s.signature, request.input, output)
	if err != nil {
		return err
	}
	response.Data = transformed
	if useDatastore {
		if hasher, ok := transformed.(common.Hashed); ok {
			hasher.SetHash(s.dictionary.Hash)
		}
		_ = s.datastore.Put(ctx, key, transformed)
	}
	return nil
}

func (s *Service) evaluate(ctx context.Context, params []interface{}) (interface{}, error) {
	onDone := s.evaluatorMetric.Begin(time.Now())
	stats := sstat.NewValues()
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()
	s.mux.RLock()
	evaluator := s.evaluator
	s.mux.RUnlock()
	result, err := evaluator.Evaluate(params)
	if err != nil {
		stats.Append(err)
		return nil, err
	}
	return result, nil
}

func (s *Service) reloadIfNeeded(ctx context.Context) error {
	lastModified, err := s.lastModified(ctx, s.config.URL, s.modified)
	if err != nil {
		return err
	}
	if !s.isModified(lastModified) {
		return nil
	}
	model, err := s.loadModel(ctx, err)
	if err != nil {
		return err
	}
	signature, err := tfmodel.Signature(model)
	if err != nil {
		return err
	}
	var dictionary *domain.Dictionary
	if s.config.UseDictionary() {
		if dictionary, err = layers.Dictionary(model.Session, model.Graph, signature); err != nil {
			return err
		}
	}
	var inputs = make(map[string]*domain.Input)
	for i, input := range signature.Inputs {
		inputs[input.Name] = &signature.Inputs[i]
	}
	evaluator := tfmodel.NewEvaluator(signature, model.Session)
	signature.Output.DataType = s.config.OutputType
	s.mux.Lock()
	defer s.mux.Unlock()
	s.evaluator = evaluator
	s.signature = signature
	s.dictionary = dictionary
	s.inputs = inputs
	s.modified = lastModified
	return nil
}

//lastModified return last modified time of object from the URL
func (s *Service) lastModified(ctx context.Context, URL string, lastModified time.Time) (time.Time, error) {
	objects, err := s.fs.List(ctx, URL)
	if err != nil {
		return lastModified, err
	}
	for i, item := range objects {
		if item.IsDir() && i == 0 {
			continue
		}
		if item.IsDir() {
			lastModified, err = s.lastModified(ctx, item.URL(), lastModified)
			if err != nil {
				return lastModified, err
			}
		}
		if item.ModTime().After(lastModified) {
			lastModified = item.ModTime()
		}
	}
	return lastModified, nil

}

func (s *Service) loadModel(ctx context.Context, err error) (*tf.SavedModel, error) {
	if err := s.fs.Copy(ctx, s.config.URL, s.config.Location); err != nil {
		return nil, err
	}
	model, err := tf.LoadSavedModel(s.config.Location, s.config.Tags, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %v, due to %w", s.config.URL, err)
	}
	return model, nil
}

func (s *Service) isModified(lastModified time.Time) bool {
	s.mux.RLock()
	modified := s.modified
	s.mux.RUnlock()
	return !modified.Equal(lastModified)
}

//NewRequest creates a new request
func (s *Service) NewRequest() *Request {
	return &Request{inputs: s.inputs, Feeds: make([]interface{}, len(s.inputs)), input: s.inputProvider.NewObject()}
}

func (s *Service) init(ctx context.Context, cfg *config.Model, datastores map[string]*datastore.Service) error {
	err := s.reloadIfNeeded(ctx)
	if err != nil {
		return err
	}
	s.transformer = getTransformer(cfg.Transformer, s.signature)
	if s.useDatastore {
		if len(cfg.KeyFields) == 0 {
			for _, input := range s.signature.Inputs {
				cfg.KeyFields = append(cfg.KeyFields, input.Name)
			}
		}
		var ok bool
		if s.datastore, ok = datastores[cfg.DataStore]; !ok {
			return fmt.Errorf("failed to lookup datastore ID: %v", cfg.DataStore)
		}
		datastoreConfig := s.datastore.Config
		if datastoreConfig.Storable == "" && len(datastoreConfig.Fields) == 0 {
			_ = datastoreConfig.FieldsDescriptor(storable.NewFields(s.signature.Output.Name, cfg.OutputType))
		}
		s.newStorable = getStorable(datastoreConfig)
	}
	go s.scheduleModelReload()
	return nil
}

func (s *Service) scheduleModelReload() {
	for range time.Tick(time.Minute) {
		if err := s.reloadIfNeeded(context.Background()); err != nil {
			log.Printf("failed to reload model: %v, due to %v", s.config.ID, err)
		}
		if atomic.LoadInt32(&s.closed) != 0 {
			return
		}
	}
}

//New creates a service
func New(ctx context.Context, fs afs.Service, cfg *config.Model, metrics *gmetric.Service, datastores map[string]*datastore.Service) (*Service, error) {
	location := reflect.TypeOf(&Service{}).PkgPath()
	result := &Service{
		fs:              fs,
		config:          cfg,
		serviceMetric:   metrics.MultiOperationCounter(location, cfg.ID, cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewService()),
		evaluatorMetric: metrics.MultiOperationCounter(location, cfg.ID+"Eval", cfg.ID+" evaluator performance", time.Microsecond, time.Minute, 2, sstat.NewService()),
		useDatastore:    cfg.UseDictionary() && cfg.DataStore != "",
		inputs:          make(map[string]*domain.Input),
		inputProvider:   gtly.NewProvider("input"),
	}
	err := result.init(ctx, cfg, datastores)
	return result, err
}
