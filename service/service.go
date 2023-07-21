package service

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
	"github.com/viant/afs/url"
	"github.com/viant/gmetric"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/layers"
	"github.com/viant/mly/service/request"
	"github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/stream"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/service/transform"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
	"github.com/viant/mly/shared/datastore"
	"github.com/viant/mly/shared/semaph"
	sstat "github.com/viant/mly/shared/stat"
	"github.com/viant/xunsafe"
	"gopkg.in/yaml.v3"
)

type Service struct {
	config *config.Model
	closed int32

	// TODO how does this interact with Service.inputs
	inputProvider *gtly.Provider

	// reload
	ReloadOK int32
	fs       afs.Service
	mux      sync.RWMutex

	// model
	sema      *semaph.Semaph // prevents potentially explosive thread generation due to concurrent requests
	evaluator *tfmodel.Evaluator

	signature  *domain.Signature
	dictionary *common.Dictionary
	inputs     map[string]*domain.Input

	// caching
	useDatastore bool
	datastore    datastore.Storer

	// outputs
	transformer domain.Transformer
	newStorable func() common.Storable

	// metrics
	serviceMetric   *gmetric.Operation
	evaluatorMetric *gmetric.Operation

	// logging
	stream *stream.Service
}

func (s *Service) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return fmt.Errorf("already closed")
	}

	if s.evaluator == nil {
		return nil
	}

	return s.evaluator.Close()
}

func (s *Service) Config() *config.Model {
	return s.config
}

func (s *Service) Dictionary() *common.Dictionary {
	return s.dictionary
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
	onDone := s.serviceMetric.Begin(startTime)
	stats := sstat.NewValues()
	onDoneDecrement := s.incrementPending(startTime)
	defer func() {
		onDone(time.Now(), stats.Values()...)
		onDoneDecrement()
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

	if err != nil {
		stats.Append(err)
		log.Printf("[%v do] limiter error:(%+v) request:(%+v)", s.config.ID, err, request)
		return err
	}

	stats.Append(stat.Evaluate)
	tensorValues, err := s.evaluate(ctx, request)
	if err != nil {
		stats.Append(err)
		log.Printf("[%v do] eval error:(%+v) request:(%+v)", s.config.ID, err, request)
		return err
	}

	err = ctx.Err()
	if err != nil {
		log.Printf("[%v do] server context Err():%v", s.config.ID, err)
	}

	return s.buildResponse(ctx, request, response, tensorValues)
}

func (s *Service) transformOutput(ctx context.Context, request *request.Request, output interface{}) (common.Storable, error) {
	inputIndex := inputIndex(output)
	inputObject := request.Input.ObjectAt(s.inputProvider, inputIndex)

	transformed, err := s.transformer(ctx, s.signature, inputObject, output)
	if err != nil {
		return nil, fmt.Errorf("failed to transform: %v, %w", s.config.ID, err)
	}

	if s.useDatastore {
		dictHash := s.Dictionary().Hash
		cacheKey := request.Input.KeyAt(inputIndex)

		isDebug := s.config.Debug
		key := s.datastore.Key(cacheKey)

		go func() {
			err := s.datastore.Put(ctx, key, transformed, dictHash)
			if err != nil {
				log.Printf("[%s trout] put error:%v", s.config.ID, err)
			}

			if isDebug {
				log.Printf("[%s trout] put:\"%s\" ok", s.config.ID, cacheKey)
			}
		}()
	}

	return transformed, nil
}

func inputIndex(output interface{}) int {
	inputIndex := 0
	if out, ok := output.(*shared.Output); ok {
		inputIndex = out.InputIndex
	}
	return inputIndex
}

func (s *Service) buildResponse(ctx context.Context, request *request.Request, response *Response, tensorValues []interface{}) error {
	if s.dictionary != nil {
		response.DictHash = s.dictionary.Hash
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

func (s *Service) incrementPending(startTime time.Time) func() {
	s.serviceMetric.IncrementValue(stat.Pending)
	recentCounter := s.serviceMetric.Recent[s.serviceMetric.Index(startTime)]
	recentCounter.IncrementValue(stat.Pending)

	return func() {
		s.serviceMetric.DecrementValue(stat.Pending)
		recentCounter := s.serviceMetric.Recent[s.serviceMetric.Index(startTime)]
		recentCounter.DecrementValue(stat.Pending)
	}
}

func (s *Service) evaluate(ctx context.Context, request *request.Request) ([]interface{}, error) {
	err := s.sema.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	onDone := s.evaluatorMetric.Begin(startTime)
	stats := sstat.NewValues()
	defer func() {
		s.sema.Release()
		onDone(time.Now(), stats.Values()...)
	}()

	s.mux.RLock()
	evaluator := s.evaluator
	s.mux.RUnlock()

	result, err := evaluator.Evaluate(request.Feeds)
	if err != nil {
		// this branch is logged by the caller
		stats.Append(err)
		return nil, err
	}

	if s.config.Debug {
		log.Printf("[%s eval] %+v", s.config.ID, result)
	}

	// TODO doesn't need to block this thread
	if s.stream != nil {
		s.stream.Log(request, result, time.Now().Sub(startTime))
	}

	return result, nil
}

func (s *Service) reloadIfNeeded(ctx context.Context) error {
	snapshot, err := s.modifiedSnapshot(ctx, s.config.URL, &config.Modified{})
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

	signature, err := tfmodel.Signature(model)
	if err != nil {
		return fmt.Errorf("signature error:%w", err)
	}

	var dictionary *common.Dictionary
	reconcileIOFromSignature(s.config, signature)
	useDict := s.config.UseDictionary()
	if useDict {
		s.config.DictMeta.Error = ""
		if s.config.DictURL != "" {
			if dictionary, err = s.loadDictionary(ctx, s.config.DictURL); err != nil {
				s.config.DictMeta.Error = err.Error()
				return err
			}
		} else {
			// extract dictionary from the graph
			dictionary, err = layers.Dictionary(model.Session, model.Graph, signature)
			if err != nil {
				s.config.DictMeta.Error = err.Error()
				return fmt.Errorf("dictionary error:%w", err)
			}
		}
	}

	var inputs = make(map[string]*domain.Input)
	for i, input := range signature.Inputs {
		inputs[input.Name] = &signature.Inputs[i]
	}

	if dictionary != nil {
		var layerIdx = make(map[string]*common.Layer)
		for _, dl := range dictionary.Layers {
			layerIdx[dl.Name] = &dl
		}
	}

	// add inputs from the config that aren't in the model
	for _, input := range s.config.Inputs {
		if _, ok := inputs[input.Name]; ok {
			continue
		}

		fInput := &domain.Input{
			Name:      input.Name,
			Index:     input.Index,
			Auxiliary: true,
		}

		fInput.Type = input.RawType()
		if fInput.Type == nil {
			fInput.Type = reflect.TypeOf("")
		}

		inputs[input.Name] = fInput
	}

	evaluator := tfmodel.NewEvaluator(signature, model.Session)
	if s.config.OutputType != "" {
		signature.Output.DataType = s.config.OutputType
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	s.evaluator = evaluator
	s.signature = signature

	if dictionary != nil {
		dictionary.UpdateHash()

		s.dictionary = dictionary
		s.config.DictMeta.Hash = dictionary.Hash
		s.config.DictMeta.Reloaded = time.Now()
	}

	s.inputs = inputs
	s.config.Modified = snapshot

	atomic.StoreInt32(&s.ReloadOK, 1)

	return nil
}

// modifiedSnapshot checks and updates modified times based on the object in URL
func (s *Service) modifiedSnapshot(ctx context.Context, URL string, resource *config.Modified) (*config.Modified, error) {
	objects, err := s.fs.List(ctx, URL)
	if err != nil {
		return resource, fmt.Errorf("failed to list URL:%s; error:%w", URL, err)
	}

	if extURL := url.SchemeExtensionURL(URL); extURL != "" {
		object, err := s.fs.Object(ctx, extURL)
		if err != nil {
			return nil, err
		}
		resource.Max = object.ModTime()
		resource.Min = object.ModTime()
		return resource, nil
	}

	for i, item := range objects {
		if item.IsDir() && i == 0 {
			continue
		}
		if item.IsDir() {
			resource, err = s.modifiedSnapshot(ctx, item.URL(), resource)
			if err != nil {
				return resource, err
			}
			continue
		}
		if resource.Max.IsZero() {
			resource.Max = item.ModTime()
		}
		if resource.Min.IsZero() {
			resource.Min = item.ModTime()
		}

		if item.ModTime().After(resource.Max) {
			resource.Max = item.ModTime()
		}
		if item.ModTime().Before(resource.Min) {
			resource.Min = item.ModTime()
		}
	}
	return resource, nil
}

func (s *Service) loadModel(ctx context.Context, err error) (*tf.SavedModel, error) {
	options := option.NewSource(&option.NoCache{Source: option.NoCacheBaseURL})
	if err := s.fs.Copy(ctx, s.config.URL, s.config.Location, options); err != nil {
		return nil, fmt.Errorf("failed to copy model %v, %s, due to %w", s.config.URL, s.config.Location, err)
	}

	model, err := tf.LoadSavedModel(s.config.Location, s.config.Tags, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %v, %s, due to %w", s.config.URL, s.config.Location, err)
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

// NewRequest should be used for Do()
func (s *Service) NewRequest() *request.Request {
	return request.NewRequest(s.config.KeysLen(), s.inputs)
}

func (s *Service) initDatastore(cfg *config.Model, datastores map[string]*datastore.Service) error {
	if !s.useDatastore {
		return nil
	}

	if s.signature == nil {
		return fmt.Errorf("signature was emtpy")
	}

	if len(cfg.KeyFields) == 0 {
		for _, input := range s.signature.Inputs {
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
		_ = datastoreConfig.FieldsDescriptor(storable.NewFields(s.signature.Output.Name, cfg.OutputType))
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

		// TODO extend backwards compatibility testing
		err := s.reloadIfNeeded(ctx)
		if err != nil {
			log.Printf("[%s reload] failed to reload model:%v", s.config.ID, err)
			atomic.StoreInt32(&s.ReloadOK, 0)
		}

		if atomic.LoadInt32(&s.closed) != 0 {
			log.Printf("[%s reload] stopping reload loop", s.config.ID)
			// we are shutting down
			return
		}
	}
}

// Deprecated and unused. Loads common.Dictionary from a remote source.
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

// New creates a service
func New(ctx context.Context, fs afs.Service, cfg *config.Model, metrics *gmetric.Service, sema *semaph.Semaph, datastores map[string]*datastore.Service, options ...Option) (*Service, error) {
	if metrics == nil {
		metrics = gmetric.New()
	}

	location := reflect.TypeOf(&Service{}).PkgPath()

	cfg.Init()

	srv := &Service{
		fs:              fs,
		config:          cfg,
		serviceMetric:   metrics.MultiOperationCounter(location, cfg.ID+"Perf", cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewProvider()),
		evaluatorMetric: metrics.MultiOperationCounter(location, cfg.ID+"Eval", cfg.ID+" evaluator performance", time.Microsecond, time.Minute, 2, sstat.NewService()),
		useDatastore:    cfg.UseDictionary() && cfg.DataStore != "",
		inputs:          make(map[string]*domain.Input),
		sema:            sema,
	}

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
			srv.stream, err = stream.NewService(cfg.ID, cfg.Stream, fs, func() *common.Dictionary {
				return srv.dictionary
			}, func() []domain.Output {
				return srv.signature.Outputs
			})
		}

		if err != nil {
			return err
		}

		if srv.inputProvider, err = srv.newObjectProvider(); err != nil {
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

func (s *Service) newObjectProvider() (*gtly.Provider, error) {
	verbose := s.config.Debug
	inputs := s.config.Inputs

	var fields = make([]*gtly.Field, len(inputs))
	for i, field := range inputs {
		if verbose {
			log.Printf("[%s obj] %s %v", s.config.ID, field.Name, field.RawType())
		}

		fields[i] = &gtly.Field{
			Name: field.Name,
			Type: field.RawType(),
		}
	}
	provider, err := gtly.NewProvider("input", fields...)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// Attempts to figure out input and output signatures of the model and compares them to
// the configured inputs and outputs.
// Generally, the configured values will override actual values.
// Additionally, any other inputs (auxiliary) will be added.
// config will be modified to match the signature from the model with the overrides.
func reconcileIOFromSignature(config *config.Model, signature *domain.Signature) {
	byName := config.FieldByName()

	if len(signature.Inputs) == 0 {
		return
	}

	for ii, _ := range signature.Inputs {
		input := &signature.Inputs[ii]

		if input.Type == nil {
			input.Type = reflect.TypeOf("")
		}

		fieldCfg, ok := byName[input.Name]
		if !ok {
			fieldCfg = &shared.Field{Name: input.Name}
			config.Inputs = append(config.Inputs, fieldCfg)
		}

		input.Vocab = !fieldCfg.Wildcard && fieldCfg.Precision <= 0

		if fieldCfg.DataType == "" {
			fieldCfg.SetRawType(input.Type)
		}

		delete(byName, fieldCfg.Name)
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

	for k, v := range byName {
		if v.DataType == "" {
			v.SetRawType(reflect.TypeOf(""))
		}

		byName[k].Auxiliary = true
	}
}
