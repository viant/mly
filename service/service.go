package service

import (
	"context"
	"fmt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/common"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/layers"
	"github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/tfmodel"
	sconfig "github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore"
	sstat "github.com/viant/mly/shared/stat"
	"reflect"
	"sync"
	"time"
)

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
}

func (s *Service) Config() *config.Model {
	return s.config
}

func (s *Service) Dictionary() *domain.Dictionary {
	return s.dictionary
}

func (s *Service) Do(ctx context.Context, request *Request, response *Response) error {
	err := s.do(ctx, request, response)
	if err != nil {
		response.Error = err.Error()
		response.Status = "error"
	}
	return nil
}

func (s *Service) do(ctx context.Context, request *Request, response *Response) error {
	useDatastore := s.useDatastore && request.Key != ""
	var key *datastore.Key
	if useDatastore {
		response.ModelHash = s.dictionary.Hash
		key = datastore.NewKey(s.datastore.Config, request.Key)
		sink := s.newStorable()
		if err := s.datastore.GetInto(ctx, key, sink); err == nil && sink.Hash() == s.dictionary.Hash {
			response.Data = sink
			return nil
		}
	}
	output, err := s.evaluate(ctx, request.Feeds)
	if err != nil {
		return err
	}
	transformed, err := s.transformer(ctx, s.signature, output)
	if err != nil {
		return err
	}
	response.Data = transformed
	if useDatastore {
		_ = s.datastore.Put(ctx, key, transformed)
	}
	return nil
}

func (s *Service) evaluate(ctx context.Context, params []interface{}) (interface{}, error) {
	onDone := s.evaluatorMetric.Begin(time.Now())
	stats := sstat.NewValues()
	defer onDone(time.Now(), stats.Values()...)
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

func (s *Service) ReloadIfNeeded(ctx context.Context) error {
	object, err := s.fs.Object(ctx, s.config.URL)
	if err != nil {
		return err
	}
	if !s.isModified(object.ModTime()) {
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
		if dictionary, err = layers.Dictionary(signature, model.Graph); err != nil {
			return err
		}
	}
	var inputs = make(map[string]*domain.Input)
	for i, input := range signature.Inputs {
		inputs[input.Name] = &signature.Inputs[i]
	}
	evaluator := tfmodel.NewEvaluator(signature, model.Session)
	s.mux.Lock()
	defer s.mux.Unlock()
	s.modified = object.ModTime()
	s.evaluator = evaluator
	s.signature = signature
	s.dictionary = dictionary
	s.inputs = inputs
	return nil
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

func (s *Service) NewRequest() *Request {
	return &Request{inputs: s.inputs, Feeds: make([]interface{}, len(s.inputs))}
}

func New(ctx context.Context, fs afs.Service, cfg *config.Model, metrics *gmetric.Service, datastores map[string]*datastore.Service) (*Service, error) {
	location := reflect.TypeOf(&Service{}).PkgPath()
	result := &Service{
		fs:              fs,
		config:          cfg,
		serviceMetric:   metrics.MultiOperationCounter(location, cfg.ID, cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewService()),
		evaluatorMetric: metrics.MultiOperationCounter(location, cfg.ID+"Eval", cfg.ID+" evaluator performance", time.Microsecond, time.Minute, 2, sstat.NewService()),
		useDatastore:    cfg.UseDictionary() && cfg.DataStore != "",
		inputs:          make(map[string]*domain.Input),
	}
	err := result.ReloadIfNeeded(ctx)
	if err != nil {
		return nil, err
	}
	result.transformer = getTransformer(cfg.Transformer, result.signature)
	if result.useDatastore {
		if len(cfg.KeyFields) == 0 {
			for _, input := range result.signature.Inputs {
				cfg.KeyFields = append(cfg.KeyFields, input.Name)
			}
		}
		var ok bool
		if result.datastore, ok = datastores[cfg.DataStore]; !ok {
			return nil, fmt.Errorf("failed to lookup datastore ID: %v", cfg.DataStore)
		}
		datastoreConfig := result.datastore.Config
		if datastoreConfig.Storable == "" && len(datastoreConfig.Fields) == 0 {
			_ = datastoreConfig.FieldsDescriptor(sconfig.NewFields(result.signature.Output.Name, cfg.OutputType))
		}
		result.newStorable = getStorable(datastoreConfig)
	}
	return result, nil
}
