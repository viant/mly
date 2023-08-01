package datastore

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aerospike/aerospike-client-go"
	"github.com/aerospike/aerospike-client-go/types"
	"github.com/viant/bintly"
	"github.com/viant/gmetric"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore/client"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/scache"
	"github.com/viant/toolbox"
)

type CacheStatus int

const (
	// CacheStatusFoundNoSuchKey we cache the status that we did not find a cache; this no-cache value has a shorter expiry
	CacheStatusFoundNoSuchKey = CacheStatus(iota)
	// CacheStatusNotFound no such key status
	CacheStatusNotFound
	// CacheStatusFound entry found status
	CacheStatusFound
)

//StoreMode represents service mode
type StoreMode int

const (
	//ModeServer server mode
	ModeServer = StoreMode(0)
	//ModeClient client mode
	ModeClient = StoreMode(1)
)

//Service datastore service
type Service struct {
	counter  *gmetric.Operation
	l1Client client.Service
	l2Client client.Service
	useCache bool
	cache    *scache.Cache
	config   *config.Datastore
	mode     StoreMode
}

func (s *Service) Config() *config.Datastore {
	return s.config
}

func (s *Service) debug() bool {
	return s.config.Debug
}

func (s *Service) id() string {
	return s.config.ID
}

func (s *Service) Mode() StoreMode {
	return s.mode
}

func (s *Service) SetMode(mode StoreMode) {
	s.mode = mode
}

func (s *Service) Enabled() bool {
	if s.config == nil {
		return false
	}
	return !s.config.Disabled
}

func (s *Service) Key(key string) *Key {
	return NewKey(s.config, key)
}

// Put puts entry to the datastore
func (s *Service) Put(ctx context.Context, key *Key, value Value, dictHash int) error {
	// Add to local cache first
	if err := s.updateCache(key.AsString(), value, dictHash); err != nil {
		return err
	}

	if s.l1Client == nil || s.mode == ModeClient {
		return nil
	}

	storable := getStorable(value)
	bins, err := storable.Iterator().ToMap()
	if err != nil {
		return err
	}

	if dictHash != 0 {
		bins[common.HashBin] = dictHash
	}

	writeKey, _ := key.Key()

	isDebug := s.debug()
	if isDebug {
		log.Printf("[%s datastore put] l1 %+v bins %+v", s.id(), writeKey, bins)
	}

	wp := key.WritePolicy(0)
	wp.SendKey = true
	if s.l1Client != nil && !s.config.ReadOnly {
		if err = s.l1Client.Put(ctx, wp, writeKey, bins); err != nil {
			return err
		}

		if isDebug {
			log.Printf("[%s datastore put] l1 OK", s.id())
		}
	}

	if s.l2Client != nil && !s.config.L2.ReadOnly {
		k2Key, _ := key.L2.Key()
		err = s.l2Client.Put(ctx, wp, k2Key, bins)

		if isDebug {
			log.Printf("[%s datastore put] l2 err:%v", s.id(), err)
		}
	}
	return err
}

//GetInto gets data into storable or error
func (s *Service) GetInto(ctx context.Context, key *Key, storable Value) (dictHash int, err error) {
	return s.getInto(ctx, key, storable)
}

func (s *Service) getInto(ctx context.Context, key *Key, storable Value) (int, error) {
	onDone := s.counter.Begin(time.Now())
	stats := stat.NewValues()
	defer func() {
		onDone(time.Now(), *stats...)
	}()

	keyString := key.AsString()
	if s.useCache {
		status, dictHash, err := s.readFromCache(keyString, storable, stats)
		if err != nil {
			return 0, err
		}
		switch status {
		case CacheStatusFoundNoSuchKey:
			return 0, types.ErrKeyNotFound
		case CacheStatusFound:
			return dictHash, nil
		}
	}

	if s.mode == ModeServer {
		// in server mode, cache hit rate would be low and expensive, thus skipping it
		return 0, types.ErrKeyNotFound
	}
	if s.l1Client == nil {
		return 0, types.ErrKeyNotFound
	}
	dictHash, err := s.getFromClient(ctx, key, storable, stats)
	if common.IsInvalidNode(err) {
		stats.Append(stat.Down)
		err = nil
		return 0, types.ErrKeyNotFound
	}

	if s.useCache && key != nil {
		if err == nil {
			if storable != nil {
				err = s.updateCache(keyString, storable, dictHash)
			}
		} else if common.IsKeyNotFound(err) {
			if e := s.updateNotFound(keyString); e != nil {
				return 0, types.ErrKeyNotFound
			}
		}
	}
	return dictHash, err
}

func (s *Service) updateNotFound(keyString string) error {
	entry := &Entry{
		Key:      keyString,
		NotFound: true,
		Expiry:   time.Now().Add(s.config.RetryTime()),
	}
	data, err := bintly.Encode(entry)
	if err != nil {
		return err
	}
	return s.cache.Set(keyString, data)
}

func (s *Service) updateCache(keyString string, entryData EntryData, dictHash int) error {
	if entryData == nil {
		return fmt.Errorf("entry was nil")
	}

	if s.cache == nil {
		return nil
	}

	if aMap, ok := entryData.(map[string]interface{}); ok {
		if data, err := json.Marshal(aMap); err == nil {
			entryData = data
		}
	}

	entry := &Entry{
		Hash:   dictHash,
		Key:    keyString,
		Data:   entryData,
		Expiry: time.Now().Add(s.config.TimeToLive()),
	}
	data, err := bintly.Encode(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %v, due to:%w ", keyString, err)
	}
	err = s.cache.Set(keyString, data)
	if err != nil {
		return fmt.Errorf("failed to set cache " + err.Error())
	}

	if s.debug() {
		log.Printf("[%s datastore update] local cache:\"%s\" (%d bytes) set", s.id(), keyString, len(data))
	}

	return nil
}

// reads from local
func (s *Service) readFromCache(keyString string, value Value, stats *stat.Values) (CacheStatus, int, error) {
	data, _ := s.cache.Get(keyString)
	if len(data) == 0 {
		return CacheStatusNotFound, 0, nil
	}

	aMap, useMap := value.(map[string]interface{})
	var rawData = []byte{}
	if useMap {
		value = &rawData
	}

	entry := &Entry{Data: EntryData(value)}
	err := bintly.Decode(data, entry)
	if err != nil {
		return CacheStatusNotFound, 0, fmt.Errorf("failed to unmarshal cache data: %s, err: %w", data, err)
	}

	if useMap {
		if err = json.Unmarshal(rawData, &aMap); err != nil {
			return CacheStatusNotFound, 0, fmt.Errorf("failed to unmarshal cache data: %s, err: %w", data, err)
		}
	}

	if entry.Expiry.Before(time.Now()) {
		stats.Append(stat.CacheExpired)
		s.cache.Delete(keyString)
		return CacheStatusNotFound, 0, nil
	}

	// cached a not found - prevents repeat downstream cache lookups
	if entry.NotFound {
		stats.Append(stat.LocalNoSuchKey)
		return CacheStatusFoundNoSuchKey, 0, nil
	}

	common.SetHash(entry.Data, entry.Hash)

	if keyString != entry.Key {
		if s.debug() {
			log.Printf("[%s datastore readFromCache] key:\"%s\" entry.Key:\"%s\"", s.id(), keyString, entry.Key)
		}

		stats.Append(stat.CacheCollision)
		return CacheStatusNotFound, 0, nil
	}

	stats.Append(stat.LocalHasValue)
	return CacheStatusFound, entry.Hash, nil
}

func (s *Service) getFromClient(ctx context.Context, key *Key, storable Value, stats *stat.Values) (int, error) {
	dictHash, err := s.fromL1Client(ctx, key, storable)
	if common.IsKeyNotFound(err) {
		stats.Append(stat.L1NoSuchKey)
		if s.l2Client == nil || key.L2 == nil {
			return 0, err
		}
		dictHash, err = s.fromL2Client(ctx, key.L2, storable)
		if err != nil {
			if common.IsKeyNotFound(err) {
				stats.Append(stat.L2NoSuchKey)
			} else if common.IsTimeout(err) {
				stats.Append(stat.Timeout)
				stats.Append(err)
			} else {
				stats.Append(err)
			}
			return 0, err
		}
		// TODO don't ignore?
		_ = s.copyTOL1(ctx, storable, key, dictHash, stats)
		return dictHash, nil
	}

	if err != nil {
		if common.IsTimeout(err) {
			stats.Append(stat.Timeout)
		}
		stats.Append(err)
		return dictHash, err
	}

	stats.Append(stat.L1HasValue)
	return dictHash, nil
}

func (s *Service) copyTOL1(ctx context.Context, value Value, key *Key, dictHash int, stats *stat.Values) error {
	storable := getStorable(value)
	bins, err := storable.Iterator().ToMap()
	if err != nil {
		return err
	}
	writeKey, _ := key.Key()
	if dictHash != 0 && len(bins) > 0 {
		bins[common.HashBin] = dictHash
	}
	err = s.l1Client.Put(ctx, key.WritePolicy(0), writeKey, aerospike.BinMap(bins))
	if err != nil {
		stats.Append(err)
		return err
	}
	stats.Append(stat.L1Copy)
	return nil
}

func (s *Service) fromL2Client(ctx context.Context, key *Key, storable Value) (int, error) {
	return s.fromClient(ctx, s.l2Client, key, storable)
}

func (s *Service) fromL1Client(ctx context.Context, key *Key, storable Value) (int, error) {
	return s.fromClient(ctx, s.l1Client, key, storable)
}

func (s *Service) fromClient(ctx context.Context, client client.Service, key *Key, value Value) (int, error) {
	clientKey, err := key.Key()
	if err != nil {
		return 0, fmt.Errorf("failed to create key: %+v, due to %w", key, err)
	}
	record, err := client.Get(ctx, clientKey)
	if err != nil {
		return 0, err
	}
	if record == nil && len(record.Bins) == 0 {
		return 0, nil
	}

	storable := getStorable(value)
	if s.debug() {
		log.Printf("[%s datastore from] storable: %T %+v", s.id(), storable, storable)
		log.Printf("[%s datastore from] bins: %+v", s.id(), record.Bins)
	}

	err = storable.Set(common.MapToIterator(record.Bins))
	if err != nil {
		return 0, fmt.Errorf("failed to map record: %+v, due to %w", key, err)
	}

	dictHash := 0
	if value, ok := record.Bins[common.HashBin]; ok {
		dictHash = toolbox.AsInt(value)
		common.SetHash(storable, dictHash)
	}
	return dictHash, nil
}

// NewWithCache creates a cache optionally
func NewWithCache(config *config.Datastore, l1Client, l2Client client.Service, counter *gmetric.Operation) (*Service, error) {
	srv := &Service{
		config:   config,
		l1Client: l1Client,
		l2Client: l2Client,
		useCache: config.Cache != nil,
		counter:  counter,
	}

	// in server mode, cache hit rate is low, i.e., the chance a key exists not in L1 but in local cache is very low
	// the only time this would happen is when a client makes a request while another client's request is populating L1
	var err error
	if config.Cache != nil {
		srv.cache, err = scache.New(config.Cache)
	}

	return srv, err
}
