package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/viant/gmetric"
	"github.com/viant/mly/shared/client/config"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
	sconfig "github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/xunsafe"
	"golang.org/x/net/http2"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

//Service represent mly client
type Service struct {
	Config
	sync.RWMutex
	dict        *Dictionary
	gmetrics    *gmetric.Service
	counter     *gmetric.Operation
	datastore   datastore.Storer
	mux         sync.RWMutex
	messages    Messages
	poolErr     error
	hostIndex   int64
	newStorable func() common.Storable
	dictRefresh int32
	httpClient  http.Client
}

//NewMessage returns a new message
func (s *Service) NewMessage() *Message {
	message := s.messages.Borrow()
	message.start()
	return message
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

//Run run model prediction
func (s *Service) Run(ctx context.Context, input interface{}, response *Response) error {
	onDone := s.counter.Begin(time.Now())
	stats := stat.NewValues()
	defer func() {
		onDone(time.Now(), *stats...)
	}()

	cachable, isCachable := input.(Cachable)

	batchSize := cachable.BatchSize()
	if response.Data == nil {
		return fmt.Errorf("response data was empty - aborting request")
	}
	var err error
	var cachedCount int
	var cached []interface{}
	if isCachable {
		cached = make([]interface{}, batchSize)
		dataType := response.DataItemType()
		if batchSize > 0 {
			if cachedCount, err = s.readFromCacheInBatch(ctx, batchSize, dataType, cachable, response, cached); err != nil {
				if !common.IsTransientError(err) {
					log.Printf("cache error: %v", err)
				}
			}
		} else {
			key := cachable.CacheKey()
			has, dictHash, err := s.readFromCache(ctx, key, response.Data)
			if err != nil && !common.IsTransientError(err) {
				log.Printf("cache error: %v", err)
			}
			if has {
				response.Status = common.StatusCached
				response.DictHash = dictHash
				return nil
			}
		}
	}

	if cachedCount == batchSize && batchSize > 0 {
		response.Status = common.StatusCached
		return s.handleResponse(ctx, response.Data, cached, cachable)
	}

	data, err := NewReader(input)
	if err != nil {
		return err
	}
	defer s.releaseMessage(input)
	stats.Append(stat.NoSuchKey)
	body, err := s.postRequest(ctx, data)
	if err != nil {
		stats.Append(err)
		return err
	}
	err = gojay.Unmarshal(body, response)
	if err != nil {
		return fmt.Errorf("failed to unmarshal: '%s'; due to %w", body, err)
	}
	if response.Status != common.StatusOK {
		return nil
	}

	if err = s.handleResponse(ctx, response.Data, cached, cachable); err != nil {
		return fmt.Errorf("failed to handle resp: %w", err)
	}
	go s.updatedCacheInBackground(ctx, response.Data, cachable, s.dict.hash)
	s.assertDictHash(response)
	return nil
}

func (s *Service) readFromCacheInBatch(ctx context.Context, batchSize int, dataType reflect.Type, cachable Cachable, response *Response, cached []interface{}) (int, error) {
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(batchSize)
	var err error
	mux := sync.Mutex{}
	var cachedCount = 0
	for i := 0; i < batchSize; i++ {
		go func(index int) {
			defer waitGroup.Done()
			cacheEntry := reflect.New(dataType.Elem()).Interface()
			key := cachable.CacheKeyAt(index)
			has, dictHash, e := s.readFromCache(ctx, key, cacheEntry)
			mux.Lock()
			defer mux.Unlock()
			if e != nil {
				err = e
			}
			if has {
				response.DictHash = dictHash
				cachable.FlagCacheHit(index)
				cached[index] = cacheEntry
				cachedCount++
			}
		}(i)
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
		return false, 0, fmt.Errorf("invalida response data type: expeted ptr but had: %T", target)
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

func (s *Service) init(options []Option) error {
	for _, option := range options {
		option.Apply(s)
	}
	if s.gmetrics == nil {
		s.gmetrics = gmetric.New()
	}
	location := reflect.TypeOf(Service{}).PkgPath()
	s.counter = s.gmetrics.MultiOperationCounter(location, s.Model+"Client", s.Model+" client performance", time.Microsecond, time.Minute, 2, stat.NewStore())
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
	if host.IsSecurePort() {
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
	if !host.IsSecurePort() {
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
	host, err := s.getHost()
	if err != nil {
		return err
	}
	URL := host.metaDictionaryURL(s.Model)

	httpClient := s.getHTTPClient(host)
	response, err := httpClient.Get(URL)
	if err != nil {
		return fmt.Errorf("failed to load Dictionary: %w", err)
	}
	if response.Body == nil {
		return fmt.Errorf("unable to load dictioanry body was empty")
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	dict := &common.Dictionary{}
	if err = json.Unmarshal(data, dict); err != nil {
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
	ds := s.Config.Datastore
	if ds.Datastore.ID == "" {
		return nil
	}
	if ds == nil {
		return nil
	}
	var stores = map[string]*datastore.Service{}
	var err error
	datastores := &sconfig.DatastoreList{
		Datastores:  []*sconfig.Datastore{&ds.Datastore},
		Connections: ds.Connections,
	}
	if stores, err = datastore.NewStores(datastores, s.gmetrics); err != nil {
		return err
	}
	s.datastore = stores[ds.ID]
	if len(ds.Fields) > 0 {
		if err := ds.FieldsDescriptor(ds.Fields); err != nil {
			return err
		}
		s.newStorable = func() common.Storable {
			return storable.New(ds.Fields)
		}
	}
	return nil
}

//Close closes the service
func (s *Service) Close() error {
	s.httpClient.CloseIdleConnections()
	return nil
}

func (s *Service) refreshMetadata() {
	defer atomic.StoreInt32(&s.dictRefresh, 0)
	if err := s.loadModelDictionary(); err != nil {
		log.Printf("failed to refresh meta data: %v", err)
	}
}

//New creates new mly client
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
	err := ReconcileData(target, cachable, cached)
	return err
}

func (s *Service) assertDictHash(response *Response) {
	dict := s.dictionary()
	if dict != nil && response.DictHash != dict.hash {
		if atomic.CompareAndSwapInt32(&s.dictRefresh, 0, 1) {
			go s.refreshMetadata()
		}
	}
}

func (s *Service) postRequest(ctx context.Context, data []byte) ([]byte, error) {
	host, err := s.getHost()
	if err != nil {
		return nil, err
	}
	var output []byte
	output, err = s.httpPost(ctx, data, host)
	if common.IsConnectionError(err) {
		host.FlagDown()
	}
	return output, err
}

func (s *Service) httpPost(ctx context.Context, data []byte, host *Host) ([]byte, error) {
	var postErr error
	for i := 0; i < s.MaxRetry; i++ {
		postErr = nil
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, host.evalURL(s.Model), io.NopCloser(bytes.NewReader(data)))
		if err != nil {
			return nil, err
		}
		respone, err := s.httpClient.Do(request)
		if err != nil {
			postErr = err
			continue
		}

		if respone.Body != nil {
			data, err := io.ReadAll(respone.Body)
			_ = respone.Body.Close()
			if respone.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("invalid response: %v, %s", respone.StatusCode, data)
			}
			if err != nil {
				postErr = err
				continue
			}
			return data, nil
		}

	}
	return nil, postErr
}

func (s *Service) getHost() (*Host, error) {
	count := len(s.Hosts)
	switch count {
	case 0:

	case 1:
		candidate := s.Hosts[0]
		if !candidate.IsUp() {
			return nil, fmt.Errorf("%v:%v %w", candidate.Name, candidate.Port, common.ErrNodeDown)
		}
		return candidate, nil
	default:
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
	return nil, fmt.Errorf("%v:%v %w", s.Hosts[0].Name, s.Hosts[0].Port, common.ErrNodeDown)
}

func (s *Service) updatedCacheInBackground(ctx context.Context, target interface{}, cachable Cachable, hash int) {
	if s.datastore == nil {
		return
	}
	targetType := reflect.TypeOf(target).Elem()
	switch targetType.Kind() {
	case reflect.Struct:
		key := cachable.CacheKey()
		storeKey := s.datastore.Key(cachable.CacheKey())
		if key == "" {
			fmt.Printf("key was empty for %T,%v", target, storeKey.Set)
			return
		}
		if err := s.datastore.Put(ctx, storeKey, target, s.dict.hash); err != nil {
			log.Printf("failed to write to cache: %v", err)
		}
		return
	}
	batchSize := cachable.BatchSize()
	var offsets = make([]int, batchSize) //index offsets to recncile local cache hits
	offset := 0
	for i := 0; i < batchSize; i++ {
		offsets[offset] = i
		if !cachable.CacheHit(i) {
			offset++
		}
	}

	xSlice := xunsafe.NewSlice(targetType)
	dataPtr := xunsafe.AsPointer(target)
	for index := 0; index < batchSize; index++ {
		value := xSlice.ValuePointerAt(dataPtr, index)
		cacheableIndex := offsets[index]
		if cachable.CacheHit(cacheableIndex) {
			continue
		}
		key := s.datastore.Key(cachable.CacheKeyAt(cacheableIndex))
		s.datastore.Put(ctx, key, value, hash)
	}
}
