package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	serrs "github.com/viant/mly/service/errors"
	"github.com/viant/mly/service/gtlyop"
	"github.com/viant/mly/service/request"
	"github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/stream"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/service/transform"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
	"github.com/viant/mly/shared/datastore"
	sstat "github.com/viant/mly/shared/stat"
	"github.com/viant/xunsafe"
)

// Service serves as the entrypoint for using the ML model.
// It is responsible for caching, the ML model provides some metadata
// related to caching.
type Service struct {
	config *config.Model
	closed int32

	// TODO document how does this interacts with Service.inputs
	inputProvider *gtly.Provider

	// reload TODO finish refactor
	ReloadOK int32

	// tensorflow evaluator factory & instance
	tfService *tfmodel.Service

	// caching
	useDatastore bool
	datastore    datastore.Storer

	// outputs
	transformer domain.Transformer
	newStorable func() common.Storable

	// serviceMetric measures validate + model + transformer
	serviceMetric *gmetric.Operation
	reloadMetric  *gmetric.Operation

	// logging
	stream *stream.Service
}

func (s *Service) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return fmt.Errorf("already closed")
	}

	return s.tfService.Close()
}

func (s *Service) Config() *config.Model {
	return s.config
}

func (s *Service) Signature() *domain.Signature {
	return s.tfService.Signature()
}

func (s *Service) Dictionary() *common.Dictionary {
	return s.tfService.Dictionary()
}

func (s *Service) Stats() map[string]interface{} {
	st := make(map[string]interface{})
	s.tfService.Stats(st)
	return st
}

func (s *Service) Do(ctx context.Context, request *request.Request, response *Response) error {
	err := s.do(ctx, request, response)
	if err != nil {
		response.Error = err.Error()
		response.Status = common.StatusError
		return err
	}

	return nil
}

func (s *Service) do(ctx context.Context, request *request.Request, response *Response) error {
	startTime := time.Now()
	onDone := s.serviceMetric.Operation.Begin(startTime)
	stats := sstat.NewValues()
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()

	err := request.Validate()
	if s.config.Debug && err != nil {
		log.Printf("[%v do] validation error: %v\n", s.config.ID, err)
	}

	if err != nil {
		// only captures missing fields
		stats.Append(stat.Invalid)
		return clienterr.Wrap(fmt.Errorf("%w, body: %s", err, request.Body))
	}

	tensorValues, err := s.evaluate(ctx, request)
	if err != nil {
		isOverloaded := errors.Is(err, serrs.OverloadedError)
		if isOverloaded {
			stats.Append(stat.Overloaded)
		} else {
			stats.AppendError(err)
		}

		if !isOverloaded && ctx.Err() == nil {
			log.Printf("[%v do] eval error:(%+v) request:(%+v)", s.config.ID, err, request)
		}

		return err
	}

	stats.Append(stat.Evaluate)
	return s.buildResponse(ctx, request, response, tensorValues)
}

func (s *Service) evaluate(ctx context.Context, request *request.Request) ([]interface{}, error) {
	startTime := time.Now()
	result, err := s.tfService.Predict(ctx, request.Feeds)
	if err != nil {
		return nil, err
	}

	if s.config.Debug {
		log.Printf("[%s eval] %+v", s.config.ID, result)
	}

	if s.stream != nil {
		s.stream.Log(request.Body, result, time.Now().Sub(startTime))
	}

	return result, nil
}

func (s *Service) buildResponse(ctx context.Context, request *request.Request, response *Response, tensorValues []interface{}) error {
	dictionary := s.Dictionary()
	if dictionary != nil {
		response.DictHash = dictionary.Hash
	}

	// TODO change with understanding that batched / multi-request always operates on the first dimension
	if !request.Input.BatchMode() {
		var err error
		// single input
		response.Data, err = s.transformOutput(ctx, request, tensorValues)
		return err
	}

	output := &shared.Output{Values: tensorValues}
	// index 0 call to get data type
	transformed, err := s.transformOutput(ctx, request, output)
	if err != nil {
		return err
	}

	sliceType := reflect.SliceOf(reflect.TypeOf(transformed))
	// xSlice = make([]`sliceType`, request.Input.BatchSize)
	sliceValue := reflect.MakeSlice(sliceType, 0, request.Input.BatchSize)
	slicePtr := xunsafe.ValuePointer(&sliceValue)
	xSlice := xunsafe.NewSlice(sliceType)

	response.xSlice = xSlice
	response.sliceLen = request.Input.BatchSize

	// xSlice = append(xSlice, transformed)
	appender := xSlice.Appender(slicePtr)
	appender.Append(transformed)

	// index 1 - end calls
	for i := 1; i < request.Input.BatchSize; i++ {
		output.InputIndex = i
		if transformed, err = s.transformOutput(ctx, request, output); err != nil {
			return err
		}
		appender.Append(transformed)
	}

	response.Data = sliceValue.Interface()
	return nil
}

func (s *Service) transformOutput(ctx context.Context, request *request.Request, output interface{}) (common.Storable, error) {
	inputIndex := 0
	if out, ok := output.(*shared.Output); ok {
		inputIndex = out.InputIndex
	}

	inputObject := request.Input.ObjectAt(s.inputProvider, inputIndex)

	signature := s.Signature()
	transformed, err := s.transformer(ctx, signature, inputObject, output)
	if err != nil {
		return nil, fmt.Errorf("failed to transform: %v, %w", s.config.ID, err)
	}

	if s.useDatastore {
		cacheKey := request.Input.KeyAt(inputIndex)
		key := s.datastore.Key(cacheKey)

		dictHash := s.Dictionary().Hash

		go func() {
			err := s.datastore.Put(ctx, key, transformed, dictHash)
			if err != nil {
				log.Printf("[%s trout] put error:%v", s.config.ID, err)
			}

			if s.config.Debug {
				log.Printf("[%s trout] put:\"%s\" dictHash:%d ok", s.config.ID, cacheKey, dictHash)
			}
		}()
	}

	return transformed, nil
}

// New creates a service
func New(ctx context.Context, cfg *config.Model, tfsrv *tfmodel.Service, fs afs.Service, metrics *gmetric.Service, datastores map[string]*datastore.Service, options ...Option) (*Service, error) {
	if metrics == nil {
		metrics = gmetric.New()
	}

	location := reflect.TypeOf(Service{}).PkgPath()

	cfg.Init()

	srv := &Service{
		config:        cfg,
		tfService:     tfsrv,
		useDatastore:  cfg.UseDictionary() && cfg.DataStore != "",
		serviceMetric: metrics.MultiOperationCounter(location, cfg.ID+"Perf", cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewProvider()),
		reloadMetric:  metrics.MultiOperationCounter(location, cfg.ID+"Reload", cfg.ID+" reloading", time.Microsecond, time.Minute, 1, sstat.NewCtxErrOnly()),
	}

	tfsrv.ReloadOK = &srv.ReloadOK

	for _, opt := range options {
		opt.Apply(srv)
	}

	err := func() error {
		err := srv.reloadIfNeeded(ctx)
		if err != nil {
			return err
		}

		srv.transformer, err = transform.Get(cfg.Transformer)
		if err != nil {
			return err
		}

		if err = srv.initDatastore(cfg, datastores); err != nil {
			return err
		}

		if cfg.Stream != nil {
			srv.stream, err = stream.NewService(cfg.ID, cfg.Stream, fs, srv.Dictionary, func() []domain.Output {
				return srv.Signature().Outputs
			})
		}

		if err != nil {
			return err
		}

		if srv.inputProvider, err = gtlyop.NewObjectProvider(cfg); err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		return nil, err
	}

	go srv.scheduleModelReload()

	return srv, err
}

func (s *Service) reloadIfNeeded(ctx context.Context) error {
	return s.tfService.ReloadIfNeeded(ctx)
}

// NewRequest should be used for Do()
func (s *Service) NewRequest() *request.Request {
	numKeyInputs := s.config.KeysLen()
	// This may change mid-request, but that only matters
	// under exceptional circumstances.
	inputs := s.tfService.Inputs()
	return request.NewRequest(numKeyInputs, inputs)
}

func (s *Service) initDatastore(cfg *config.Model, datastores map[string]*datastore.Service) error {
	if !s.useDatastore {
		return nil
	}

	signature := s.Signature()
	if signature == nil {
		return fmt.Errorf("signature was emtpy")
	}

	if len(cfg.KeyFields) == 0 {
		for _, input := range signature.Inputs {
			cfg.KeyFields = append(cfg.KeyFields, input.Name)
		}
	}

	if s.datastore == nil {
		var ok bool
		if s.datastore, ok = datastores[cfg.DataStore]; !ok {
			return fmt.Errorf("failed to lookup datastore ID: %v", cfg.DataStore)
		}
	}

	datastoreConfig := s.datastore.Config()
	if datastoreConfig.Storable == "" && len(datastoreConfig.Fields) == 0 {
		// TODO check for multi-output models
		fields := storable.NewFields(signature.Output.Name, cfg.OutputType)
		_ = datastoreConfig.FieldsDescriptor(fields)
	}

	if s.newStorable == nil {
		s.newStorable = getStorable(datastoreConfig)
	}

	return nil
}

func (s *Service) scheduleModelReload() {
	for range time.Tick(time.Minute) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		onDone := s.reloadMetric.Begin(time.Now())
		stats := sstat.NewValues()
		defer func() {
			onDone(time.Now(), stats.Values()...)
		}()

		err := s.reloadIfNeeded(ctx)
		if err != nil {
			stats.AppendError(err)
			log.Printf("[%s reload] failed to reload model:%v", s.config.ID, err)
			atomic.StoreInt32(&s.ReloadOK, 0)
		}

		if atomic.LoadInt32(&s.closed) != 0 {
			log.Printf("[%s reload] shutting down, stopping reload loop", s.config.ID)
			return
		}
	}
}
