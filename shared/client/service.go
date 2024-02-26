package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/viant/gmetric"
	"github.com/viant/mly/shared/client/config"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
	sconfig "github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/mly/shared/stat/metric"
	"github.com/viant/mly/shared/tracker"
	"github.com/viant/mly/shared/tracker/mg"
	"github.com/viant/xunsafe"
	"golang.org/x/net/http2"
)

//Service represent mly client
type Service struct {
	Config
	hostIndex   int64
	httpClient  http.Client
	newStorable func() common.Storable

	messages Messages
	poolErr  error

	sync.RWMutex
	dict               *Dictionary
	dictRefreshPending int32
	datastore          datastore.Storer

	// container for Datastore gmetric objects
	gmetrics *gmetric.Service

	counter        *gmetric.Operation
	httpCounter    *gmetric.Operation
	httpCliCounter *gmetric.Operation
	dictCounter    *gmetric.Operation

	ErrorHistory tracker.Tracker
}

// NewMessage returns a new message
func (s *Service) NewMessage() *Message {
	message := s.messages.Borrow()
	message.start()
	return message
}

// Run will fetch the model prediction and populate response.Data with the result.
// input can vary in types, but if it is an instance of Cachable, then the configured
// caching system will be used.
func (s *Service) Run(ctx context.Context, input interface{}, response *Response) error {
	onDone := s.counter.Begin(time.Now())
	stats := stat.NewValues()
	defer func() {
		onDone(time.Now(), *stats...)
		s.releaseMessage(input)
	}()

	if ctx.Err() != nil {
		stats.Append(stat.EarlyCtxError)
	}

	if response.Data == nil {
		return fmt.Errorf("response data was empty - aborting request")
	}

	cachable, isCachable := input.(Cachable)
	var err error
	var cachedCount int
	var cached []interface{}

	batchSize := 0
	if isCachable {
		batchSize = cachable.BatchSize()

		cachedCount, err = s.loadFromCache(ctx, &cached, batchSize, response, cachable)
		if err != nil {
			stats.AppendError(err)
			if ctx.Err() == nil && s.ErrorHistory != nil {
				go s.ErrorHistory.AddBytes([]byte(err.Error()))
			}

			return err
		}
	}

	isDebug := s.Config.Debug

	var modelName string
	if isDebug {
		modelName = s.Config.Model
		s.reportBatch(cachedCount, cached)
	}

	if (batchSize > 0 && cachedCount == batchSize) || (batchSize == 0 && cachedCount > 0) {
		response.Status = common.StatusCached
		return s.handleResponse(ctx, response.Data, cached, cachable)
	}

	data, err := Marshal(input, modelName)
	if err != nil {
		stats.AppendError(err)
		return err
	}

	stats.Append(stat.NoSuchKey)
	if isDebug {
		log.Printf("[%s] request: %s", modelName, strings.Trim(string(data), " \n"))
	}

	body, err := func() ([]byte, error) {
		httpOnDone := s.httpCounter.Begin(time.Now())
		httpStats := stat.NewValues()

		od := metric.EnterThenExit(s.httpCounter, time.Now(), stat.Enter, stat.Exit)

		defer func() {
			httpOnDone(time.Now(), httpStats.Values()...)
			od()
		}()

		body, err := s.postRequest(ctx, data, httpStats)
		if isDebug {
			log.Printf("[%s] response.Body:%s", modelName, body)
			log.Printf("[%s] error:%s", modelName, err)
		}

		if err != nil {
			httpStats.AppendError(err)
		}

		return body, err
	}()

	if err != nil {
		stats.AppendError(err)
		if ctx.Err() == nil && s.ErrorHistory != nil {
			go s.ErrorHistory.AddBytes([]byte(err.Error()))
		}

		return err
	}

	err = gojay.Unmarshal(body, response)
	if err != nil {
		stats.AppendError(err)
		return fmt.Errorf("failed to unmarshal: '%s'; due to %w", body, err)
	}

	if isDebug {
		log.Printf("[%v] response.Data: %s, %v", modelName, response.Data, err)
	}

	if response.Status != common.StatusOK {
		return nil
	}

	if err = s.handleResponse(ctx, response.Data, cached, cachable); err != nil {
		stats.AppendError(err)
		return fmt.Errorf("failed to handle resp: %w", err)
	}

	s.updatedCache(ctx, response.Data, cachable, s.dict.hash)
	s.assertDictHash(response)

	return nil
}

func (s *Service) loadFromCache(ctx context.Context, cached *[]interface{}, batchSize int, response *Response, cachable Cachable) (int, error) {
	*cached = make([]interface{}, batchSize)
	dataType, err := response.DataItemType()
	if err != nil {
		return 0, err
	}

	if batchSize > 0 {
		cachedCount, err := s.readFromCacheInBatch(ctx, batchSize, dataType, cachable, response, *cached)
		if err != nil && !common.IsTransientError(err) {
			log.Printf("cache error: %v", err)
		}

		return cachedCount, nil
	}

	key := cachable.CacheKey()
	has, dictHash, err := s.readFromCache(ctx, key, response.Data)
	if err != nil && !common.IsTransientError(err) {
		log.Printf("cache error: %v", err)
	}
	cachedCount := 0
	if has {
		cachedCount = 1
		response.Status = common.StatusCached
		response.DictHash = dictHash
	}
	return cachedCount, nil
}

func (s *Service) readFromCacheInBatch(ctx context.Context, batchSize int, dataType reflect.Type, cachable Cachable, response *Response, cached []interface{}) (int, error) {
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(batchSize)
	var err error
	mux := sync.Mutex{}
	var cachedCount = 0
	for k := 0; k < batchSize; k++ {
		go func(index int) {
			defer waitGroup.Done()
			key := cachable.CacheKeyAt(index)
			cacheEntry := reflect.New(dataType.Elem()).Interface()
			has, dictHash, e := s.readFromCache(ctx, key, cacheEntry)
			mux.Lock()
			defer mux.Unlock()
			if e != nil {
				err = e
				return
			}
			if has {
				response.DictHash = dictHash
				cachable.FlagCacheHit(index)
				cached[index] = cacheEntry
				cachedCount++
			}
		}(k)
	}
	waitGroup.Wait()
	return cachedCount, err
}

func (s *Service) readFromCache(ctx context.Context, key string, target interface{}) (bool, int, error) {
	if s.datastore == nil || !s.datastore.Enabled() {
		return false, 0, nil
	}

	dataType := reflect.TypeOf(target)
	if dataType.Kind() != reflect.Ptr {
		return false, 0, fmt.Errorf("invalid response data type: expeted ptr but had: %T", target)
	}

	storeKey := s.datastore.Key(key)
	dictHash, err := s.datastore.GetInto(ctx, storeKey, target)
	if err == nil {
		if (!s.Config.DictHashValidation) || dictHash == 0 || dictHash == s.dictionary().hash {
			return true, dictHash, nil
		}
	}

	return false, 0, err
}

func (s *Service) releaseMessage(input interface{}) {
	releaser, ok := input.(Releaser)
	if ok {
		releaser.Release()
	}
}

func (s *Service) dictionary() *Dictionary {
	s.RWMutex.RLock()
	dict := s.dict
	s.RWMutex.RUnlock()
	return dict
}

func (s *Service) init(options []Option) error {
	for _, option := range options {
		option.Apply(s)
	}

	if s.gmetrics == nil {
		s.gmetrics = gmetric.New()
	}

	location := reflect.TypeOf(Service{}).PkgPath()
	s.counter = s.gmetrics.MultiOperationCounter(location, s.Model+"Client", s.Model+" client performance", time.Microsecond, time.Minute, 2, stat.NewClient())
	s.httpCounter = s.gmetrics.MultiOperationCounter(location, s.Model+"ClientHTTP", s.Model+" client HTTP overall performance", time.Microsecond, time.Minute, 2, stat.NewHttp())
	s.httpCliCounter = s.gmetrics.MultiOperationCounter(location, s.Model+"ClientHTTPCli", s.Model+" client HTTP client performance", time.Microsecond, time.Minute, 2, stat.NewCtxErrOnly())
	s.dictCounter = s.gmetrics.MultiOperationCounter(location, s.Model+"ClientDict", s.Model+" client dictionary performance", time.Microsecond, time.Minute, 1, stat.ErrorOnly())

	if s.ErrorHistory == nil {
		s.ErrorHistory = mg.NewK(20)
	}

	if s.Config.MaxRetry == 0 {
		s.Config.MaxRetry = 3
	}

	err := s.initHTTPClient()
	if err != nil {
		return err
	}

	if s.Config.Datastore == nil {
		if err := s.loadModelConfig(); err != nil {
			return err
		}
	}
	if s.dict == nil {
		if err := s.loadModelDictionary(); err != nil {
			return err
		}
	}
	if ds := s.Config.Datastore; ds != nil {
		ds.Init()
		if err = ds.Validate(); err != nil {
			return err
		}
	}

	if s.datastore == nil {
		err := s.initDatastore()
		return err
	}
	s.messages = NewMessages(s.dictionary)
	return nil
}

func (s *Service) initHTTPClient() error {
	host, _ := s.getHost()
	var tslConfig *tls.Config
	if host != nil && host.IsSecurePort() {
		cert, err := getCertPool()
		if err != nil {
			return fmt.Errorf("failed to create certificate: %v", err)
		}

		tslConfig = &tls.Config{
			RootCAs: cert,
		}
	}

	http2Transport := &http2.Transport{
		TLSClientConfig: tslConfig,
	}

	if host == nil || !host.IsSecurePort() {
		http2Transport.AllowHTTP = true
		http2Transport.DialTLS = func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		}
	}

	s.httpClient.Transport = http2Transport
	return nil
}

func (s *Service) loadModelConfig() error {
	var err error
	host, err := s.getHost()
	if err != nil {
		return err
	}
	if s.Config.Datastore, err = s.discoverConfig(host, host.metaConfigURL(s.Model)); err != nil {
		return err
	}
	s.Config.updateCache()
	return nil
}

func (s *Service) loadModelDictionary() error {
	stats := stat.NewValues()

	onDone := s.dictCounter.Begin(time.Now())
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()

	host, err := s.getHost()
	if err != nil {
		stats.Append(err)
		return err
	}
	URL := host.metaDictionaryURL(s.Model)

	httpClient := s.getHTTPClient(host)
	response, err := httpClient.Get(URL)
	if err != nil {
		// no context errors supported
		stats.Append(err)
		return fmt.Errorf("failed to load Dictionary: %w", err)
	}

	if response.Body == nil {
		err = fmt.Errorf("unable to load dictioanry body was empty")
		stats.Append(err)
		return err
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		stats.Append(err)
		return fmt.Errorf("failed to read body: %w", err)
	}

	dict := &common.Dictionary{}
	if err = json.Unmarshal(data, dict); err != nil {
		stats.Append(err)
		return fmt.Errorf("failed to unmarshal dict: %w", err)
	}

	s.RWMutex.Lock()
	s.dict = NewDictionary(dict, s.Datastore.Inputs)
	s.RWMutex.Unlock()

	s.messages = NewMessages(s.dictionary)
	return nil
}

func (s *Service) getHTTPClient(host *Host) *http.Client {
	httpClient := http.DefaultClient
	if host.IsSecurePort() {
		httpClient = &s.httpClient
	}
	return httpClient
}

func (s *Service) initDatastore() error {
	remoteCfg := s.Config.Datastore
	if remoteCfg == nil {
		return nil
	}

	if remoteCfg.Datastore.ID == "" {
		return nil
	}

	var stores = map[string]*datastore.Service{}
	var err error
	datastores := &sconfig.DatastoreList{
		Datastores:  []*sconfig.Datastore{&remoteCfg.Datastore},
		Connections: remoteCfg.Connections,
	}

	if stores, err = datastore.NewStores(datastores, s.gmetrics); err != nil {
		return err
	}

	s.datastore = stores[remoteCfg.ID]
	if len(remoteCfg.Fields) > 0 {
		if err := remoteCfg.FieldsDescriptor(remoteCfg.Fields); err != nil {
			return err
		}

		s.newStorable = func() common.Storable {
			return storable.New(remoteCfg.Fields)
		}
	}

	if s.datastore != nil {
		s.datastore.SetMode(datastore.ModeClient)
	}

	return nil
}

func (s *Service) Close() error {
	s.httpClient.CloseIdleConnections()
	if s.ErrorHistory != nil {
		s.ErrorHistory.Close()
	}

	return nil
}

// New creates new client.
func New(model string, hosts []*Host, options ...Option) (*Service, error) {
	for i := range hosts {
		hosts[i].Init()
	}

	aClient := &Service{
		Config: Config{
			Model: model,
			Hosts: hosts,
		},
	}
	err := aClient.init(options)
	return aClient, err
}

func (s *Service) discoverConfig(host *Host, URL string) (*config.Remote, error) {
	httpClient := s.getHTTPClient(host)

	response, err := httpClient.Get(URL)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	cfg := &config.Remote{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse load %v, config:   %s, %v", URL, data, err)
	}

	if s.Config.Debug {
		prefix := fmt.Sprintf("[%s] config.Remote.", s.Config.Model)

		for i, c := range cfg.Connections {
			log.Printf("%sConnections[%d]:%+v", prefix, i, c)
		}

		ds := cfg.Datastore
		if ds.Reference != nil {
			log.Printf("%sDatastore:[%s].Cache: %+v", prefix, ds.ID, ds.Cache)
			log.Printf("%sDatastore:[%s].L1: %+v", prefix, ds.ID, ds.Connection)

			if ds.L2 != nil {
				log.Printf("%sDatastore:[%s].L2: %+v", prefix, cfg.Datastore.ID, cfg.Datastore.L2)
			}
		}

		for i, mi := range cfg.MetaInput.Inputs {
			log.Printf("%sMetaInput.Inputs[%d]:%+v", prefix, i, *mi)
		}

		log.Printf("%sMetaInput.Auxiliary:%v", prefix, cfg.MetaInput.Auxiliary)
		log.Printf("%sMetaInput.KeyFields:%v", prefix, cfg.MetaInput.KeyFields)

		for i, mo := range cfg.MetaInput.Outputs {
			log.Printf("%sMetaInput.Outputs[%d]:%+v", prefix, i, *mo)
		}
	}

	return cfg, err
}

func (s *Service) handleResponse(ctx context.Context, target interface{}, cached []interface{}, cachable Cachable) error {
	if cachable == nil {
		return nil
	}
	targetType := reflect.TypeOf(target).Elem()
	switch targetType.Kind() {
	case reflect.Struct:
		return nil
	case reflect.Slice:
	default:
		return fmt.Errorf("invalid response type expected []*T, but had: %T", target)
	}

	var debugPrefix string
	if s.Config.Debug {
		debugPrefix = fmt.Sprintf("[%s]", s.Config.Model)
	}
	err := reconcileData(debugPrefix, target, cachable, cached)
	return err
}

func (s *Service) assertDictHash(response *Response) {
	dict := s.dictionary()
	if dict != nil && response.DictHash != dict.hash {
		if atomic.CompareAndSwapInt32(&s.dictRefreshPending, 0, 1) {
			go s.refreshMetadata()
		}
	}
}

func (s *Service) refreshMetadata() {
	defer atomic.StoreInt32(&s.dictRefreshPending, 0)
	if err := s.loadModelDictionary(); err != nil {
		log.Printf("failed to refresh meta data: %v", err)
	}
}

func (s *Service) postRequest(ctx context.Context, data []byte, mvt *stat.Values) ([]byte, error) {
	// TODO per-host counters
	host, err := s.getHost()
	if err != nil {
		return nil, err
	}

	var output []byte

	output, err = s.httpPost(ctx, data, host)
	if common.IsConnectionError(err) {
		if s.Config.Debug {
			log.Printf("[%s postRequest] connection error:%s", s.Config.Model, err)
		}
		mvt.Append(stat.Down)
		host.FlagDown()
	}

	return output, err
}

func (s *Service) httpPost(ctx context.Context, data []byte, host *Host) ([]byte, error) {
	evalUrl := host.evalURL(s.Model)
	var terminate bool
	var postErr error
	for i := 0; i < s.MaxRetry; i++ {
		data, err := func() ([]byte, error) {
			onDone := s.httpCliCounter.Begin(time.Now())
			stats := stat.NewValues()

			defer func() {
				onDone(time.Now(), stats.Values()...)
			}()

			request, err := http.NewRequestWithContext(ctx, http.MethodPost, evalUrl, bytes.NewReader(data))
			if err != nil {
				stats.AppendError(err)
				return nil, err
			}

			response, err := s.httpClient.Do(request)
			if s.Config.Debug {
				log.Printf("http try:%d err:%s", i, err)
			}

			if err != nil {
				stats.AppendError(err)
				return nil, err
			}

			var data []byte
			if response.Body != nil {
				data, err = io.ReadAll(response.Body)
				_ = response.Body.Close()
			}

			if response.StatusCode != http.StatusOK {
				// as long as this func is run synchronously,
				// this is safe
				terminate = true
				return nil, fmt.Errorf("HTTP Code:%d, Body:\"%s\" (read nil:%v error:%v)",
					response.StatusCode, string(data), response.Body == nil, err)
			}

			return data, nil
		}()

		if err != nil {
			postErr = err
		}

		if terminate || ctx.Err() != nil {
			// stop trying if deadline exceeded or canceled
			break
		}

		if data != nil {
			return data, nil
		}
	}

	return nil, postErr
}

func (s *Service) getHost() (*Host, error) {
	count := len(s.Hosts)
	switch count {
	case 0:
		return nil, fmt.Errorf("no hosts configured")
	case 1:
		candidate := s.Hosts[0]
		if !candidate.IsUp() {
			return nil, fmt.Errorf("%v:%v %w", candidate.Name(), candidate.Port(), common.ErrNodeDown)
		}
		return candidate, nil
	default:
		// TODO introduce a fallback mode
		index := atomic.AddInt64(&s.hostIndex, 1) % int64(count)
		candidate := s.Hosts[index]
		if candidate.IsUp() {
			return candidate, nil
		}

		for i := 0; i < len(s.Hosts); i++ {
			if s.Hosts[i].IsUp() {
				return s.Hosts[i], nil
			}
		}
	}

	addrs := make([]string, count, count)
	for i, h := range s.Hosts {
		addrs[i] = h.Name() + ":" + strconv.Itoa(h.Port())
	}

	hostsDesc := strings.Join(addrs, ",")

	return nil, fmt.Errorf("no working hosts:%s %w", hostsDesc, common.ErrNodeDown)
}

func (s *Service) updatedCache(ctx context.Context, target interface{}, cachable Cachable, hash int) {
	if s.datastore == nil || !s.datastore.Enabled() {
		return
	}
	targetType := reflect.TypeOf(target).Elem()
	switch targetType.Kind() {
	case reflect.Struct:
		s.updateSingleEntry(ctx, target, cachable)
		return
	case reflect.Slice:
	default:
		log.Printf("unsupported target type: %T", target)
	}

	batchSize := cachable.BatchSize()
	offsets := mapNonCachedPositions(batchSize, cachable)

	xSlice := xunsafe.NewSlice(targetType)
	dataPtr := xunsafe.AsPointer(target)
	xSliceLen := xSlice.Len(dataPtr) //response data is a slice, iterate vi slice to update response
	if xSliceLen > batchSize {
		xSliceLen = batchSize
	}
	for index := 0; index < xSliceLen; index++ {
		value := xSlice.ValuePointerAt(dataPtr, index)
		if value == nil { //no actual value was returned from mly service
			continue
		}
		cacheableIndex := offsets[index]
		if cachable.CacheHit(cacheableIndex) {
			return
		}
		key := s.datastore.Key(cachable.CacheKeyAt(cacheableIndex))
		s.datastore.Put(ctx, key, value, hash)
	}
}

//mapNonCachedPositions maps non cachable original slice element position  into actial request slice positions
/*
	client.data:[v1, v2, v3, v4]
	assuming v1 and v3 are found in local cache
	only v2, v3 needs to send to mly server, and once we receive response we need to map index 0, 1, into original item positions: 1, 3
*/
func mapNonCachedPositions(batchSize int, cachable Cachable) []int {
	var offsets = make([]int, batchSize) //index offsets to recncile local cache hits
	offset := 0
	for i := 0; i < batchSize; i++ {
		offsets[offset] = i
		if !cachable.CacheHit(i) {
			offset++
		}
	}
	return offsets
}

func (s *Service) updateSingleEntry(ctx context.Context, target interface{}, cachable Cachable) {
	key := cachable.CacheKey()
	storeKey := s.datastore.Key(cachable.CacheKey())
	if key == "" {
		return
	}
	if err := s.datastore.Put(ctx, storeKey, target, s.dict.hash); err != nil {
		log.Printf("[%s] failed to write to cache: %v", s.Model, err)
	}
	return
}

func (s *Service) reportBatch(count int, cached []interface{}) {
	log.Printf("[%s] batchSize: %v, found in cache: %v", s.Model, len(cached), count)
	if count == 0 {
		return
	}
	for i, v := range cached {
		if v == nil {
			continue
		}
		log.Printf("[%s] cached[%v] %+v", s.Model, i, v)
	}
}
