package service

import (
	"bytes"
	"compress/gzip"
	"context"
	sjson "encoding/json"
	"fmt"
	"github.com/google/uuid"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/afs"
	"github.com/viant/afs/url"
	"github.com/viant/gmetric"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/layers"
	"github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
	"github.com/viant/mly/shared/datastore"
	sstat "github.com/viant/mly/shared/stat"
	"github.com/viant/mly/shared/transfer"
	tlog "github.com/viant/tapper/log"
	"github.com/viant/tapper/msg"
	"github.com/viant/tapper/msg/json"
	"github.com/viant/xunsafe"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

//Service represents ml service
type Service struct {
	mux             sync.RWMutex
	fs              afs.Service
	evaluator       *tfmodel.Evaluator
	signature       *domain.Signature
	dictionary      *common.Dictionary
	inputs          map[string]*domain.Input
	useDatastore    bool
	datastore       datastore.Storer
	transformer     domain.Transformer
	newStorable     func() common.Storable
	config          *config.Model
	serviceMetric   *gmetric.Operation
	evaluatorMetric *gmetric.Operation
	closed          int32
	inputProvider   *gtly.Provider
	logger          *tlog.Logger
	msgProvider     *msg.Provider
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
func (s *Service) Dictionary() *common.Dictionary {
	return s.dictionary
}

//Do handles service request
func (s *Service) Do(ctx context.Context, request *Request, response *Response) error {
	err := s.do(ctx, request, response)
	if err != nil {
		response.Error = err.Error()
		response.Status = common.StatusError
	}
	return nil
}

func (s *Service) do(ctx context.Context, request *Request, response *Response) error {
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
		fmt.Printf("Validation error [%v]: %v\n", s.config.ID, err)
	}
	if err != nil {
		stats.Append(err)
		return fmt.Errorf("%w, body: %s", err, request.Body)
	}
	stats.Append(stat.Evaluate)
	tensorValues, err := s.evaluate(ctx, request)
	if err != nil {
		return err
	}

	return s.buildResponse(ctx, request, response, tensorValues)
}

func (s *Service) transformOutput(ctx context.Context, request *Request, output interface{}) (common.Storable, error) {
	inputIndex := s.inputIndex(output)
	anObject := request.input.ObjectAt(s.inputProvider, inputIndex)

	transformed, err := s.transformer(ctx, s.signature, anObject, output)
	if err != nil {
		return nil, fmt.Errorf("failed to transform: %v, %w", s.config.ID, err)
	}
	if s.useDatastore {
		dictHash := s.Dictionary().Hash
		key := s.datastore.Key(request.input.KeyAt(inputIndex))
		go s.datastore.Put(ctx, key, transformed, dictHash)
	}
	return transformed, nil
}

func (s *Service) inputIndex(output interface{}) int {
	inputIndex := 0
	if out, ok := output.(*shared.Output); ok {
		inputIndex = out.InputIndex
	}
	return inputIndex
}

func (s *Service) buildResponse(ctx context.Context, request *Request, response *Response, tensorValues []interface{}) error {
	if s.dictionary != nil {
		response.DictHash = s.dictionary.Hash
	}
	if !request.input.BatchMode() {
		var err error
		response.Data, err = s.transformOutput(ctx, request, tensorValues)
		return err
	}

	output := &shared.Output{Values: tensorValues}
	transformed, err := s.transformOutput(ctx, request, output)
	if err != nil {
		return err
	}

	sliceType := reflect.SliceOf(reflect.TypeOf(transformed))
	sliceValue := reflect.MakeSlice(sliceType, 0, request.input.BatchSize)
	slicePtr := xunsafe.ValuePointer(&sliceValue)
	xSlice := xunsafe.NewSlice(sliceType)
	response.xSlice = xSlice
	response.sliceLen = request.input.BatchSize
	appender := xSlice.Appender(slicePtr)
	appender.Append(transformed)
	for i := 1; i < request.input.BatchSize; i++ {
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

func (s *Service) evaluate(ctx context.Context, request *Request) ([]interface{}, error) {
	startTime := time.Now()
	onDone := s.evaluatorMetric.Begin(startTime)
	stats := sstat.NewValues()
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()
	s.mux.RLock()
	evaluator := s.evaluator
	s.mux.RUnlock()
	result, err := evaluator.Evaluate(request.Feeds)
	if err != nil {
		stats.Append(err)
		return nil, err
	}
	s.logEvaluation(request, result, time.Now().Sub(startTime))
	return result, nil
}

func (s *Service) reloadIfNeeded(ctx context.Context) error {
	snapshot, err := s.modifiedSnapshot(ctx, s.config.URL, &config.Modified{})
	if err != nil {
		return err
	}
	if !s.isModified(snapshot) {
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
	var dictionary *common.Dictionary
	s.reconcileSignatureWithInput(signature)
	if s.config.UseDictionary() {
		if s.config.DictURL != "" {
			if dictionary, err = s.loadDictionary(ctx, s.config.DictURL); err != nil {
				return err
			}
		} else {
			dictionary, err = layers.Dictionary(model.Session, model.Graph, signature)
			if err != nil {
				return err
			}

		}
		s.dictionary = dictionary
	}

	var inputs = make(map[string]*domain.Input)
	for i, input := range signature.Inputs {
		inputs[input.Name] = &signature.Inputs[i]
	}
	for _, input := range s.config.Inputs {
		if _, ok := inputs[input.Name]; ok {
			continue
		}
		fInput := &domain.Input{Name: input.Name, Index: input.Index, Auxiliary: true}
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
	s.dictionary = dictionary
	s.inputs = inputs
	s.config.Modified = snapshot
	return nil
}

//modifiedSnapshot return last modified time of object from the URL
func (s *Service) modifiedSnapshot(ctx context.Context, URL string, resource *config.Modified) (*config.Modified, error) {
	objects, err := s.fs.List(ctx, URL)
	if err != nil {
		return resource, err
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
	if err := s.fs.Copy(ctx, s.config.URL, s.config.Location); err != nil {
		return nil, err
	}
	model, err := tf.LoadSavedModel(s.config.Location, s.config.Tags, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %v, due to %w", s.config.URL, err)
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

//NewRequest creates a new request
func (s *Service) NewRequest() *Request {
	return &Request{inputs: s.inputs, Feeds: make([]interface{}, s.config.KeysLen()), input: &transfer.Input{}}
}

func (s *Service) init(ctx context.Context, cfg *config.Model, datastores map[string]*datastore.Service) error {
	err := s.reloadIfNeeded(ctx)
	if err != nil {
		return err
	}
	s.transformer = getTransformer(cfg.Transformer, s.signature)

	if err = s.initDatastore(cfg, datastores); err != nil {
		return err
	}

	if s.config.Stream != nil {
		ID := s.getStreamID()
		logger, err := tlog.New(s.config.Stream, ID, afs.New())
		if err != nil {
			return err
		}
		s.logger = logger
		s.msgProvider = msg.NewProvider(2048, 32, json.New)
	}

	go s.scheduleModelReload()
	return nil
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
		_ = datastoreConfig.FieldsDescriptor(storable.NewFields(s.signature.Output.Name, cfg.OutputType))
	}
	if s.newStorable == nil {
		s.newStorable = getStorable(datastoreConfig)
	}
	return nil
}

func (s *Service) getStreamID() string {
	ID := ""
	if UUID, err := uuid.NewUUID(); err == nil {
		ID = UUID.String()
	}
	if hostname, err := os.Hostname(); err == nil {
		if host, err := common.GetHostIPv4(hostname); err == nil {
			ID = host
		}
	}
	return ID
}

func (s *Service) scheduleModelReload() {
	for range time.Tick(time.Minute) {
		if err := s.reloadIfNeeded(context.Background()); err != nil {
			fmt.Printf("failed to reload model: %v, due to %v", s.config.ID, err)
		}
		if atomic.LoadInt32(&s.closed) != 0 {
			return
		}
	}
}

func (s *Service) logEvaluation(request *Request, output interface{}, timeTaken time.Duration) {
	if s.logger == nil || len(request.Body) == 0 {
		return
	}
	msg := s.msgProvider.NewMessage()
	defer msg.Free()

	data := request.Body
	begin := bytes.IndexByte(data, '{')
	end := bytes.LastIndexByte(data, '}')
	if end == -1 {
		return
	}
	msg.Put(data[begin+1 : end]) //include original json from request body
	msg.PutByte(',')

	msg.PutInt("eval_duration", int(timeTaken.Microseconds()))
	if bytes.Index(data, []byte("timestamp")) == -1 {
		msg.PutString("timestamp", time.Now().In(time.UTC).Format("2006-01-02 15:04:05.000-07"))
	}

	if s.dictionary != nil {
		msg.PutInt("dict_hash", int(s.dictionary.Hash))
	}
	switch actual := output.(type) {
	case [][]int64:
		if len(actual) > 0 {
			switch len(actual[0]) {
			case 0:
			case 1:
				msg.PutInt(s.signature.Output.Name, int(actual[0][0]))
			default:
				ints := *(*[]int)(unsafe.Pointer(&actual[0]))
				msg.PutInts(s.signature.Output.Name, ints)
			}
		}
	case [][]float32:
		if len(actual) > 0 {
			switch len(actual[0]) {
			case 0:
			case 1:
				msg.PutFloat(s.signature.Output.Name, float64(actual[0][0]))
			default:
				panic("not supported yet")
				//Add support to tapper
				//msg.PutByte(',')
				//msg.PutFloats(s.signature.Output.Name, float64(actual[0][0]))
			}
		}
	}
	if err := s.logger.Log(msg); err != nil {
		fmt.Printf("failed to log model eval: %v %v\n", s.config.ID, err)
	}
}

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

//New creates a service
func New(ctx context.Context, fs afs.Service, cfg *config.Model, metrics *gmetric.Service, datastores map[string]*datastore.Service, options ...Option) (*Service, error) {
	location := reflect.TypeOf(&Service{}).PkgPath()
	if metrics == nil {
		metrics = gmetric.New()
	}
	cfg.Init()
	srv := &Service{
		fs:              fs,
		config:          cfg,
		serviceMetric:   metrics.MultiOperationCounter(location, cfg.ID+"Perf", cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewService()),
		evaluatorMetric: metrics.MultiOperationCounter(location, cfg.ID+"Eval", cfg.ID+" evaluator performance", time.Microsecond, time.Minute, 2, sstat.NewService()),
		useDatastore:    cfg.UseDictionary() && cfg.DataStore != "",
		inputs:          make(map[string]*domain.Input),
	}
	for _, opt := range options {
		opt.Apply(srv)
	}
	var err error
	if err = srv.init(ctx, cfg, datastores); err != nil {
		return nil, err
	}
	if srv.inputProvider, err = newObjectProvider(srv.config.Inputs); err != nil {
		return nil, err
	}
	return srv, err
}

func newObjectProvider(inputs []*shared.Field) (*gtly.Provider, error) {
	var fields = make([]*gtly.Field, len(inputs))
	for i, field := range inputs {
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

func (srv *Service) reconcileSignatureWithInput(signature *domain.Signature) {
	byName := srv.config.FieldByName()
	var signatureInputs = signature.Inputs
	if len(signatureInputs) == 0 {
		return
	}
	for i := range signatureInputs {
		input := &signatureInputs[i]
		if input.Type == nil {
			input.Type = reflect.TypeOf("")
		}
		field, ok := byName[input.Name]
		if !ok {
			field = &shared.Field{Name: input.Name}
			srv.config.Inputs = append(srv.config.Inputs, field)
		}
		input.Wildcard = field.Wildcard
		input.Layer = field.Layer
		if field.DataType == "" {
			field.SetRawType(input.Type)
		}
		delete(byName, field.Name)
	}
	if len(signature.Outputs) > 0 {
		for _, output := range signature.Outputs {

			field := &shared.Field{Name: output.Name, DataType: output.DataType}
			if field.DataType == "" {
				field.SetRawType(reflect.TypeOf(""))
			}
			srv.config.Outputs = append(srv.config.Outputs, field)
		}
	}

	for k, v := range byName {
		if v.DataType == "" {
			v.SetRawType(reflect.TypeOf(""))
		}
		byName[k].Auxiliary = true
	}

}
