package datastore

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aerospike/aerospike-client-go"
	"github.com/aerospike/aerospike-client-go/types"
	"github.com/viant/bintly"
	"github.com/viant/gmetric"
	"github.com/viant/mly/common"
	"github.com/viant/mly/log"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore/client"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/scache"
	"github.com/viant/toolbox"
	"time"
)

type CacheStatus int

const (
	//CacheStatusFoundNoSuchKey cacheable no such key status
	CacheStatusFoundNoSuchKey = CacheStatus(iota)
	//CacheStatusNotFound no such key status
	CacheStatusNotFound
	//CacheStatusFound entry found status
	CacheStatusFound
)

//Service datastore service
type Service struct {
	counter    *gmetric.Operation
	l1Client   client.Service
	l2Client   client.Service
	useCache   bool
	cache      *scache.Cache
	Config     *config.Datastore
	ClientMode bool
}

//Put puts entry to the datastore
func (s *Service) Put(ctx context.Context, key *Key, value Value, dictHash int) error {
	//Add to local cache first
	if err := s.updateCache(key, value, dictHash); err != nil {
		return err
	}
	if s.l1Client == nil || s.ClientMode {
		return nil
	}
	storable := getStorable(value)
	bins, err := storable.Iterator().ToMap()
	if err != nil {
		return err
	}
	if dictHash > 0 {
		bins[common.HashBin] = dictHash
	}
	writeKey, _ := key.Key()
	wp := key.WritePolicy(0)
	wp.SendKey = true
	log.Debug("put %v -> %v\n", key.AsString(), bins)
	if err = s.l1Client.Put(ctx, wp, writeKey, bins); err == nil && s.l2Client != nil {
		k2Key, _ := key.L2.Key()
		err = s.l2Client.Put(ctx, wp, k2Key, bins)
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
	if s.useCache {
		status, dictHash, err := s.readFromCache(key, storable, stats)
		if err != nil {
			return 0, err
		}
		switch status {
		case CacheStatusFoundNoSuchKey:
			return 0, types.ErrKeyNotFound
		case CacheStatusFound:
			stats.Append(stat.HasValue)
			return dictHash, nil
		}
	}
	if s.l1Client == nil {
		return 0, types.ErrKeyNotFound
	}
	dictHash, err := s.getFromClient(ctx, key, storable, stats)
	if common.IsInvalidNode(err) {
		stats.Append(stat.Down)
	}

	if s.useCache && key != nil {
		if err == nil {
			err = s.updateCache(key, storable, dictHash)
		} else if common.IsKeyNotFound(err) {
			if e := s.updateNotFound(key); e != nil {
				return 0, fmt.Errorf("failed to updated cache with not found entry: %e")
			}
		}
	}
	return dictHash, err
}

func (s *Service) updateNotFound(key *Key) error {
	entry := &Entry{
		Key:      key.AsString(),
		NotFound: true,
		Expiry:   time.Now().Add(s.Config.RetryTime()),
	}
	data, err := bintly.Encode(entry)
	if err != nil {
		return err
	}
	return s.cache.Set(key.AsString(), data)
}

func (s *Service) updateCache(key *Key, entryData EntryData, dictHash int) error {
	if aMap, ok := entryData.(map[string]interface{}); ok {
		if data, err := json.Marshal(aMap); err == nil {
			entryData = data
		}
	}
	entry := &Entry{
		Hash:   dictHash,
		Key:    key.AsString(),
		Data:   entryData,
		Expiry: time.Now().Add(s.Config.TimeToLive()),
	}
	data, err := bintly.Encode(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %v, due to:%w ", key.AsString(), err)
	}
	err = s.cache.Set(key.AsString(), data)
	if err != nil {
		return fmt.Errorf("failed to set cache " + err.Error())
	}
	log.Debug("updated local cache: %v %T(%+v)\n", key.AsString(), entry.Data, entry.Data)

	return nil
}

func (s *Service) readFromCache(key *Key, value Value, stats *stat.Values) (CacheStatus, int, error) {
	data, _ := s.cache.Get(key.AsString())
	if len(data) == 0 {
		return CacheStatusNotFound, 0, nil
	}
	aMap, useMap := value.(map[string]interface{})
	var rawData = []byte{}
	if useMap {
		value = &rawData
	}
	entry := &Entry{
		Data: EntryData(value),
	}
	err := bintly.Decode(data, entry)
	if err != nil {
		return CacheStatusNotFound, 0, fmt.Errorf("failed to unmarshal cache data: %s, err: %w", data, err)
	}
	log.Debug("found in local cache: %v %T(%+v)\n", key.AsString(), value, value)
	if useMap {
		if err = json.Unmarshal(rawData, &aMap); err != nil {
			return CacheStatusNotFound, 0, fmt.Errorf("failed to unmarshal cache data: %s, err: %w", data, err)
		}
	}

	if entry.Expiry.Before(time.Now()) {
		stats.Append(stat.CacheExpired)
		s.cache.Delete(key.AsString())
		return CacheStatusNotFound, 0, nil
	}
	if entry.NotFound {
		stats.Append(stat.CacheHit)
		stats.Append(stat.NoSuchKey)
		return CacheStatusFoundNoSuchKey, 0, nil
	}

	common.SetHash(entry.Data, entry.Hash)

	if key.AsString() == entry.Key {
		stats.Append(stat.CacheHit)
		return CacheStatusFound, entry.Hash, nil
	}
	stats.Append(stat.CacheCollision)
	return CacheStatusNotFound, 0, nil
}

func (s *Service) getFromClient(ctx context.Context, key *Key, storable Value, stats *stat.Values) (int, error) {
	dictHash, err := s.fromL1Client(ctx, key, storable)
	if common.IsKeyNotFound(err) {
		stats.Append(stat.NoSuchKey)
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
		_ = s.copyTOL1(ctx, storable, key, dictHash, stats)
		stats.Append(stat.HasValue)
		return dictHash, nil
	}
	if err != nil {
		if common.IsTimeout(err) {
			stats.Append(stat.Timeout)
		}
		stats.Append(err)
		return dictHash, err
	}
	stats.Append(stat.HasValue)
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
	log.Debug("fetching %v -> %v, %v\n", clientKey.String(), record, err)
	if err != nil {
		return 0, err
	}
	if record == nil && len(record.Bins) == 0 {
		return 0, nil
	}

	storable := getStorable(value)
	log.Debug("fetched %v -> %v\n", key.AsString(), record.Bins)
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

//New returns a new Service
func New(l1Client, l2Client client.Service, counter *gmetric.Operation) *Service {
	srv := &Service{l1Client: l1Client, l2Client: l2Client, counter: counter}
	return srv
}

//NewWithCache creates Service with cache
func NewWithCache(config *config.Datastore, l1Client, l2Client client.Service, counter *gmetric.Operation) (*Service, error) {
	if config.Cache == nil {
		return New(l1Client, l2Client, counter), nil
	}
	srv := &Service{
		Config:   config,
		l1Client: l1Client,
		l2Client: l2Client,
		useCache: true,
		counter:  counter,
	}
	var err error
	srv.cache, err = scache.New(config.Cache)
	return srv, err
}
