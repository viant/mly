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
	"github.com/viant/mly/service/platform"
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
	"golang.org/x/sync/semaphore"
)

// Service serves as the entrypoint for using the ML model.
// It is responsible for caching, the ML model provides some metadata
// related to caching.
type Service struct {
	config *config.Model
	closed int32

	maxEvaluatorWait time.Duration

	// continueOnRecover if false, will re-panic on recover
	continueOnRecover bool

	// TODO how does this interact with Service.inputs
	inputProvider *gtly.Provider

	// reload TODO finish refactor
	ReloadOK int32

	// Platform router for multi-platform support
	platformRouter platform.PlatformRouter

	// tensorflow evaluator factory & instance (kept for backward compatibility)
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

	if s.platformRouter != nil {
		return s.platformRouter.Close()
	}

	// Fallback to TensorFlow service for backward compatibility
	if s.tfService != nil {
		return s.tfService.Close()
	}

	return nil
}

func (s *Service) Config() *config.Model {
	return s.config
}

func (s *Service) Signature() *domain.Signature {
	if s.platformRouter != nil {
		if sig := s.platformRouter.Signature(); sig != nil {
			if domainSig, ok := sig.(*domain.Signature); ok {
				return domainSig
			}
		}
	}

	// Fallback to TensorFlow service for backward compatibility
	if s.tfService != nil {
		return s.tfService.Signature()
	}

	return nil
}

func (s *Service) Dictionary() *common.Dictionary {
	if s.platformRouter != nil {
		return s.platformRouter.Dictionary()
	}

	// Fallback to TensorFlow service for backward compatibility
	if s.tfService != nil {
		return s.tfService.Dictionary()
	}

	return nil
}

func (s *Service) Stats() map[string]interface{} {
	st := make(map[string]interface{})

	if s.platformRouter != nil {
		s.platformRouter.Stats(st)
	} else if s.tfService != nil {
		// Fallback to TensorFlow service for backward compatibility
		s.tfService.Stats(st)
	}

	return st
}

func (s *Service) Do(ctx context.Context, request *request.Request, response *Response) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[%s do] panic:%v feeds:%+v", s.config.ID, r, request.Feeds)
			if !s.continueOnRecover {
				panic(r)
			}

			err = fmt.Errorf("panic model:%s, recovered:%v", s.config.ID, r)
		}
	}()

	err = s.do(ctx, request, response)
	if err != nil {
		response.Error = err.Error()
		response.Status = common.StatusError
		return err
	}

	return nil
}

// do is the main entrypoint for using the service.
// First run validation (just checks if all required fields are present).
// Then run evaluation (which can be immediate or time-batched).
// Then (in buildResponse) runs the transformer (if any).
// Then (in buildResponse in transformOutput) stores the result in the datastore if configured.
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
			log.Printf("[%v do] eval error:(%+v) request.Feeds:(%+v)", s.config.ID, err, request.Feeds)
		}

		// we waited or there was an issue with evaluation; in either case
		// the prediction never finished so there is nothing left to clean up
		return err
	}

	stats.Append(stat.Evaluate)
	return s.buildResponse(ctx, request, response, tensorValues)
}

func (s *Service) evaluate(ctx context.Context, request *request.Request) ([]interface{}, error) {
	startTime := time.Now()

	var result []interface{}
	var err error

	if s.platformRouter != nil {
		result, err = s.platformRouter.Predict(ctx, request.Feeds)
	} else if s.tfService != nil {
		// Fallback to TensorFlow service for backward compatibility
		result, err = s.tfService.Predict(ctx, request.Feeds)
	} else {
		return nil, fmt.Errorf("no evaluator configured for model %s", s.config.ID)
	}

	if err != nil {
		return nil, err
	}

	if s.config.Debug {
		log.Printf("[%s eval] %+v", s.config.ID, result)
	}

	if s.stream != nil {
		s.stream.Log(request.Body, result, time.Since(startTime))
	}

	return result, nil
}

// buildResponse handles single and batched requests for calling transformOutput().
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

// transformOutput runs the transformer (if any) and stores the result in the datastore if configured.
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

		var dictHash int
		if dict := s.Dictionary(); dict != nil {
			dictHash = dict.Hash
		} else {
			dictHash = 0
		}

		go func() {
			err := s.datastore.Put(ctx, key, transformed, dictHash)
			if err != nil {
				log.Printf("[%s trout] put error: \"%s\" %v", s.config.ID, cacheKey, err)
			}

			if s.config.Debug {
				log.Printf("[%s trout] put: \"%s\" dictHash: %d ok", s.config.ID, cacheKey, dictHash)
			}
		}()
	}

	return transformed, nil
}

// New creates a service with TensorFlow service (legacy)
func New(ctx context.Context,
	cfg *config.Model, tfsrv *tfmodel.Service, fs afs.Service, metrics *gmetric.Service, datastores map[string]*datastore.Service,
	options ...Option) (*Service, error) {

	if metrics == nil {
		metrics = gmetric.New()
	}

	location := reflect.TypeOf(Service{}).PkgPath()

	cfg.Init(nil)

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

	err := srv.initializeService(ctx, cfg, fs, metrics, datastores)
	if err != nil {
		return nil, err
	}

	go srv.scheduleModelReload()

	return srv, err
}

func (s *Service) initializeService(ctx context.Context, cfg *config.Model, fs afs.Service, metrics *gmetric.Service, datastores map[string]*datastore.Service) error {
	err := s.reloadIfNeeded(ctx)
	if err != nil {
		return err
	}

	s.transformer, err = transform.Get(cfg.Transformer)
	if err != nil {
		return err
	}

	if err = s.initDatastore(cfg, datastores); err != nil {
		return err
	}

	if cfg.Stream != nil {
		s.stream, err = stream.NewService(cfg.ID, cfg.Stream, fs, s.Dictionary, func() []domain.Output {
			if sig := s.Signature(); sig != nil {
				return sig.Outputs
			}
			return nil
		}, metrics)
	}

	if err != nil {
		return err
	}

	if s.inputProvider, err = gtlyop.NewObjectProvider(cfg); err != nil {
		return err
	}

	return nil
}

// NewWithPlatform creates a service with platform router support
func NewWithPlatform(ctx context.Context,
	cfg *config.Model, fs afs.Service, metrics *gmetric.Service, datastores map[string]*datastore.Service,
	sema *semaphore.Weighted, maxEvaluatorWait time.Duration, options ...Option) (*Service, error) {

	if metrics == nil {
		metrics = gmetric.New()
	}

	location := reflect.TypeOf(Service{}).PkgPath()

	cfg.Init(nil)

	// Create platform router
	router, err := platform.CreateRouter(cfg, fs, metrics, sema, maxEvaluatorWait)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform router for model %s: %w", cfg.ID, err)
	}

	srv := &Service{
		config:         cfg,
		platformRouter: router,
		tfService:      nil, // Will be set for TensorFlow platform for backward compatibility
		useDatastore:   cfg.UseDictionary() && cfg.DataStore != "",
		serviceMetric:  metrics.MultiOperationCounter(location, cfg.ID+"Perf", cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewProvider()),
	}

	if cfg.GetPlatform() != "triton" {
		srv.reloadMetric = metrics.MultiOperationCounter(location, cfg.ID+"Reload", cfg.ID+" reloading", time.Microsecond, time.Minute, 1, sstat.NewCtxErrOnly())
	}

	// For backward compatibility, still expose tfService for TensorFlow models
	if cfg.GetPlatform() == "tensorflow" {
		if tfWrapper, ok := router.Evaluator.(*platform.TensorFlowEvaluator); ok {
			srv.tfService = tfWrapper.TfService
			srv.tfService.ReloadOK = &srv.ReloadOK
		}
	}

	for _, opt := range options {
		opt.Apply(srv)
	}

	err = srv.initializeService(ctx, cfg, fs, metrics, datastores)
	if err != nil {
		return nil, err
	}

	if cfg.GetPlatform() != "triton" {
		go srv.scheduleModelReload()
	}

	return srv, err
}

func (s *Service) reloadIfNeeded(ctx context.Context) error {
	if s.tfService != nil {
		return s.tfService.ReloadIfNeeded(ctx)
	}

	// For platform router, we need to handle reload through the evaluator
	if s.platformRouter != nil {
		if router, ok := s.platformRouter.(*platform.Router); ok {
			if tfEvaluator, ok := router.Evaluator.(*platform.TensorFlowEvaluator); ok {
				return tfEvaluator.TfService.ReloadIfNeeded(ctx)
			}
		}
	}

	return nil
}

// NewRequest should be used for Do()
func (s *Service) NewRequest() *request.Request {
	numKeyInputs := s.config.KeysLen()
	// This may change mid-request, but that only matters
	// under exceptional circumstances.

	var inputs map[string]*domain.Input

	if s.tfService != nil {
		inputs = s.tfService.Inputs()
	} else if s.platformRouter != nil {
		// For platform router, get inputs from the evaluator
		if router, ok := s.platformRouter.(*platform.Router); ok {
			evaluatorInputs := router.Evaluator.Inputs()
			// Convert from map[string]interface{} to map[string]*domain.Input
			inputs = make(map[string]*domain.Input)
			for k, v := range evaluatorInputs {
				if domainInput, ok := v.(*domain.Input); ok {
					inputs[k] = domainInput
				}
			}
		}
	}

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
		fields := []*storable.Field{}
		for _, output := range signature.Outputs {
			f := &storable.Field{Name: output.Name, DataType: output.DataType}
			fields = append(fields, f)
		}
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
