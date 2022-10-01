package mock

import (
	"context"
	"github.com/aerospike/aerospike-client-go/types"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore"
	"github.com/viant/xunsafe"
	"sync"
	"unsafe"
)

type Service struct {
	Disabled bool
	Cfg      *config.Datastore
	mode     datastore.StoreMode
	cache    map[string]interface{}
	hash     int
	sync.Mutex
}

func (s *Service) Put(ctx context.Context, key *datastore.Key, value datastore.Value, dictHash int) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.hash = dictHash
	s.cache[key.AsString()] = value
	return nil
}

func (s *Service) Config() *config.Datastore {
	return s.Cfg
}

func (s *Service) GetInto(ctx context.Context, key *datastore.Key, storable datastore.Value) (dictHash int, err error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	val, ok := s.cache[key.AsString()]
	if !ok {
		return 0, types.ErrKeyNotFound
	}
	*(*unsafe.Pointer)(xunsafe.AsPointer(storable)) = *(*unsafe.Pointer)(xunsafe.AsPointer(val))
	return s.hash, nil
}

func (s *Service) Key(key string) *datastore.Key {
	return &datastore.Key{Value: key, Namespace: "test", Set: "test"}
}

func (s *Service) Enabled() bool {
	return !s.Disabled
}

func (s *Service) Mode() datastore.StoreMode {
	return s.mode
}

func (s *Service) SetMode(mode datastore.StoreMode) {
	s.mode = mode
}

func New() *Service {
	return &Service{
		Disabled: false,
		cache:    map[string]interface{}{},
		Mutex:    sync.Mutex{},
		Cfg:      &config.Datastore{Storable: "test"},
	}
}
