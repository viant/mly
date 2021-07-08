package client

import (
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
	"golang.org/x/net/http2"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

//Service represent mly client
type Service struct {
	Config
	sync.RWMutex
	dict        *dictionary
	gmetrics    *gmetric.Service
	datastore   *datastore.Service
	connections sync.Pool
	mux         sync.RWMutex
	messages    Messages
	poolErr     error
	hostIndex   int64
	newStorable func() common.Storable
	dictRefresh int32
}

//NewMessage returns a new message
func (s *Service) NewMessage() *Message {
	message := s.messages.Borrow()
	message.start()
	return message
}

func (s *Service) conn() (*connection, error) {
	result := s.connections.Get()
	if result == nil {
		return nil, s.poolErr
	}
	conn := result.(*connection)
	if conn.lastUsed.IsZero() {
		return conn, nil
	}
	if time.Now().Sub(conn.lastUsed) > requestTimeout {
		_ = conn.Close()
		return s.conn()
	}
	return conn, nil
}

func (s *Service) releaseMessage(input interface{}) {
	releaser, ok := input.(Releaser)
	if ok {
		releaser.Release()
	}
}

func (s *Service) dictionary() *dictionary {
	s.RWMutex.RLock()
	dict := s.dict
	s.RWMutex.RUnlock()
	return dict
}

//Run run model prediction
func (s *Service) Run(ctx context.Context, input interface{}, response *Response) error {
	data, err := NewReader(input)
	defer s.releaseMessage(input)
	if err != nil {
		return err
	}
	cachableKey, ok := input.(Cachable)
	if response.Data == nil && s.newStorable != nil {
		response.Data = s.newStorable()
	}
	var key *datastore.Key
	if ok && s.datastore != nil {
		key = datastore.NewKey(s.datastore.Config, cachableKey.CacheKey())
		if dictHash, err := s.datastore.GetInto(ctx, key, response.Data); err == nil {
			response.Status = common.StatusCached
			response.DictHash = dictHash
			if response.DictHash == 0 || response.DictHash == s.dictionary().hash {
				return nil
			}
		}
	}
	body, err := s.postRequest(data)
	if err != nil {
		return err
	}
	err = gojay.Unmarshal(body, response)
	if err != nil {
		return fmt.Errorf("failed to unmarshal: '%s'; due to %w", body, err)
	}
	if key != nil && response.Status == common.StatusOK {
		s.datastore.Put(ctx, key, response.Data, s.dict.hash)
	}
	s.assertDictHash(response)
	return nil
}

func (s *Service) assertDictHash(response *Response) {
	dict := s.dictionary()
	if dict != nil && response.DictHash != dict.hash {
		if atomic.CompareAndSwapInt32(&s.dictRefresh, 0, 1) {
			go s.refreshMetadata()
		}
	}
}

func (s *Service) postRequest(data []byte) ([]byte, error) {
	var conn *connection
	var err error
	var body []byte
	for i := 0; i < s.MaxRetry; i++ {
		conn, err = s.conn()
		if err != nil {
			continue
		}
		_, err = conn.Write(data)
		if err != nil {
			continue
		}
		body, err = conn.Read()
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	}
	s.connections.Put(conn)
	conn.lastUsed = time.Now()
	return body, nil
}

func (s *Service) getHost() (*Host, error) {
	count := len(s.Hosts)
	switch count {
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

func (s *Service) init(options []Option) error {
	for _, option := range options {
		option.Apply(s)
	}
	if s.Config.MaxRetry == 0 {
		s.Config.MaxRetry = 3
	}

	if err := s.loadModelConfig(); err != nil {
		return err
	}
	if err := s.loadModelDictionary(); err != nil {
		return err
	}
	if s.gmetrics == nil {
		s.gmetrics = gmetric.New()
	}
	if err := s.initDatastore(); err != nil {
		return err
	}
	s.messages = newMessages(s.dictionary)

	host, _ := s.getHost()
	transport := &http2.Transport{
		AllowHTTP: true,
		ConnPool: newPool(host, s.Model),
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}
	httpClient := http.Client{
		Transport: transport,
		Timeout: requestTimeout,
	}

	s.connections.New = func() interface{} {
		host, err := s.getHost()
		if err != nil {
			s.poolErr = err
			return nil
		}
		conn, err := newConnection(host, &httpClient, host.evalURL(s.Model))
		if err != nil {
			s.poolErr = err
		}
		return conn
	}
	return nil
}




func (s *Service) loadModelConfig() error {
	if s.Datastore == nil {
		var err error
		host, err := s.getHost()
		if err != nil {
			return err
		}
		if s.Datastore, err = discoverConfig(host.metaConfigURL(s.Model)); err != nil {
			return err
		}
	}
	s.Config.updateCache()
	return nil
}

func (s *Service) loadModelDictionary() error {
	if s.Datastore == nil {
		return nil
	}

	if len(s.Datastore.KeyFields) == 0 {
		return nil
	}
	host, err := s.getHost()
	if err != nil {
		return err
	}
	URL := host.metaDictionaryURL(s.Model)
	response, err := http.DefaultClient.Get(URL)
	if err != nil {
		return fmt.Errorf("failed to load dictionary: %w", err)
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
	s.dict = newDictionary(dict, s.Datastore.KeyFields)
	s.RWMutex.Unlock()
	s.messages = newMessages(s.dictionary)
	return nil
}

func (s *Service) initDatastore() error {
	if ds := s.Config.Datastore; ds != nil {
		datastores := &sconfig.DatastoreList{
			Connections: ds.Connections,
			Datastores:  []*sconfig.Datastore{&ds.Datastore},
		}
		aMap, err := datastore.NewStores(datastores, s.gmetrics)
		if err != nil {
			return err
		}
		s.datastore = aMap[ds.ID]
		s.datastore.ClientMode = true
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
	conn, err := s.conn()
	if err != nil {
		return err
	}
	return conn.Close()
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

	return aClient, aClient.init(options)
}

func discoverConfig(URL string) (*config.Datastore, error) {
	response, err := http.DefaultClient.Get(URL)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	cfg := &config.Datastore{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %s, %v", data, err)
	}
	return cfg, err
}
