package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/layers"
	"github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
	"github.com/viant/mly/shared/datastore"
	sstat "github.com/viant/mly/shared/stat"
	tlog "github.com/viant/tapper/log"
	"github.com/viant/tapper/msg"
	"os"
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
	dictionary      *common.Dictionary
	inputs          map[string]*domain.Input
	useDatastore    bool
	datastore       *datastore.Service
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
	if err != nil {
		stats.Append(err)
		return fmt.Errorf("%w, body: %s", err, request.Body)
	}
	useDatastore := s.useDatastore && request.Key != ""
	var key *datastore.Key
	if useDatastore {
		dictHash := s.Dictionary().Hash
		response.DictHash = dictHash
		key = datastore.NewKey(s.datastore.Config, request.Key)
		sink := s.newStorable()
		entryDictHash, err := s.datastore.GetInto(ctx, key, sink)
		if err == nil {
			isConsistent := entryDictHash == 0 || entryDictHash == dictHash
			if isConsistent {
				response.Data = sink
				return nil
			}
		}
	}

	stats.Append(stat.Evaluate)
	output, err := s.evaluate(ctx, request)
	if err != nil {
		return err
	}
	transformed, err := s.transformer(ctx, s.signature, request.input, output)
	if err != nil {
		return err
	}
	response.Data = transformed
	if useDatastore {
		dictHash := s.Dictionary().Hash
		go s.datastore.Put(ctx, key, transformed, dictHash)
	}
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

func (s *Service) evaluate(ctx context.Context, request *Request) (interface{}, error) {
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
	if s.config.UseDictionary() {
		s.updateWildcardInput(signature)
		if dictionary, err = layers.Dictionary(model.Session, model.Graph, signature); err != nil {
			return err
		}
	}

	var inputs = make(map[string]*domain.Input)
	for i, input := range signature.Inputs {
		inputs[input.Name] = &signature.Inputs[i]
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

func (s *Service) updateWildcardInput(signature *domain.Signature) {
	if len(s.config.WildcardFields) == 0 {
		return
	}
	var index = map[string]bool{}
	for i := range s.config.WildcardFields {
		index[s.config.WildcardFields[i]] = true
	}
	for i := range signature.Inputs {
		input := &signature.Inputs[i]
		input.WildCard = index[input.Name]
	}
}

//modifiedSnapshot return last modified time of object from the URL
func (s *Service) modifiedSnapshot(ctx context.Context, URL string, resource *config.Modified) (*config.Modified, error) {
	objects, err := s.fs.List(ctx, URL)
	if err != nil {
		return resource, err
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
	return &Request{inputs: s.inputs, Feeds: make([]interface{}, len(s.inputs)), input: s.inputProvider.NewObject()}
}

func (s *Service) init(ctx context.Context, cfg *config.Model, datastores map[string]*datastore.Service) error {
	err := s.reloadIfNeeded(ctx)
	if err != nil {
		return err
	}
	s.transformer = getTransformer(cfg.Transformer, s.signature)
	if s.useDatastore {
		if s.signature == nil {
			return fmt.Errorf("signature was emtpy")
		}
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

	if s.config.Stream != nil {
		ID := s.getStreamID()
		logger, err := tlog.New(s.config.Stream, ID, afs.New())
		if err != nil {
			return err
		}
		s.logger = logger
		s.msgProvider = msg.NewProvider(2048, 32)
	}

	go s.scheduleModelReload()
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
	case [][]float32:
		if len(actual) > 0 {
			switch len(actual[0]) {
			case 0:
			case 1:
				msg.PutFloat(s.signature.Output.Name, float64(actual[0][0]))
			default:
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

//New creates a service
func New(ctx context.Context, fs afs.Service, cfg *config.Model, metrics *gmetric.Service, datastores map[string]*datastore.Service) (*Service, error) {
	location := reflect.TypeOf(&Service{}).PkgPath()
	srv := &Service{
		fs:              fs,
		config:          cfg,
		serviceMetric:   metrics.MultiOperationCounter(location, cfg.ID+"Perf", cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewService()),
		evaluatorMetric: metrics.MultiOperationCounter(location, cfg.ID+"Eval", cfg.ID+" evaluator performance", time.Microsecond, time.Minute, 2, sstat.NewService()),
		useDatastore:    cfg.UseDictionary() && cfg.DataStore != "",
		inputs:          make(map[string]*domain.Input),
	}
	err := srv.init(ctx, cfg, datastores)
	if err != nil {
		return nil, err
	}
	var fields = make([]*gtly.Field, len(srv.signature.Inputs))

	for i := range srv.signature.Inputs {
		input := &srv.signature.Inputs[i]
		inputType := input.Type
		if input.WildCard {
			inputType = reflect.TypeOf("")
		}
		fields[input.Index] = &gtly.Field{
			Name: input.Name,
			Type: inputType,
		}
	}
	provider, err := gtly.NewProvider("input", fields...)
	if err != nil {
		return nil, err
	}
	srv.inputProvider = provider
	return srv, err
}
