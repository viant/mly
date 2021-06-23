package datastore

import (
	"context"
	"fmt"
	"github.com/aerospike/aerospike-client-go"
	"github.com/aerospike/aerospike-client-go/types"
	"github.com/viant/bintly"
	"github.com/viant/gmetric"
	"github.com/viant/mly/common"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore/client"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/scache"
	"time"
)

type CacheStatus int

const (
	CacheStatusFoundNoSuchKey = CacheStatus(iota)
	CacheStatusNotFound
	CacheStatusFound
)

type Service struct {
	counter  *gmetric.Operation
	l1Client client.Service
	l2Client client.Service
	useCache bool
	cache    *scache.Cache
	Config   *config.Datastore
}

func (s *Service) Put(ctx context.Context, key *Key, storable common.Storable) error {
	//Add to local cache first
	if err := s.updateCache(key, storable); err != nil {
		return nil
	}
	value, err := storable.Iterator().ToMap()
	if err != nil {
		return err
	}
	writeKey, _ := key.Key()
	wp := key.WritePolicy(0)
	wp.SendKey = true
	return s.l1Client.Put(ctx, wp, writeKey, value)
}

func (s *Service) GetInto(ctx context.Context, key *Key, storable common.Storable) (err error) {
	err = s.getInto(ctx, key, storable)
	return err
}

func (s *Service) getInto(ctx context.Context, key *Key, storable common.Storable) error {
	onDone := s.counter.Begin(time.Now())
	stats := stat.NewValues()
	defer func() {
		onDone(time.Now(), *stats...)
	}()
	if s.useCache {
		status, err := s.readFromCache(key, storable, stats)
		if err != nil {
			return err
		}
		switch status {
		case CacheStatusFoundNoSuchKey:
			return types.ErrKeyNotFound
		case CacheStatusFound:
			stats.Append(stat.HasValue)
			return nil
		}
	}
	if s.l1Client == nil {
		return types.ErrKeyNotFound
	}
	err := s.getFromClient(ctx, key, storable, stats)
	if common.IsInvalidNode(err) {
		stats.Append(stat.Down)
	}
	if s.useCache {
		if err == nil {
			err = s.updateCache(key, storable)
		} else if common.IsKeyNotFound(err) {
			if e := s.updateNotFound(key); e != nil {
				return fmt.Errorf("failed to updated cache with not found entry: %e")
			}
		}
	}
	return err
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

func (s *Service) updateCache(key *Key, entryData EntryData) error {
	entry := &Entry{
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
	return nil
}

func (s *Service) readFromCache(key *Key, storable common.Storable, stats *stat.Values) (CacheStatus, error) {
	data, _ := s.cache.Get(key.AsString())
	if len(data) == 0 {
		return CacheStatusNotFound, nil
	}
	entry := &Entry{
		Data: storable.(EntryData),
	}
	err := bintly.Decode(data, entry)
	if err != nil {
		return CacheStatusNotFound, fmt.Errorf("failed to unmarshal cache data: %s, err: %w", data, err)
	}
	if entry.Expiry.Before(time.Now()) {
		stats.Append(stat.CacheExpired)
		s.cache.Delete(key.AsString())
		return CacheStatusNotFound, nil
	}
	if entry.NotFound {
		stats.Append(stat.CacheHit)
		stats.Append(stat.NoSuchKey)
		return CacheStatusFoundNoSuchKey, nil
	}

	if key.AsString() == entry.Key {
		stats.Append(stat.CacheHit)
		return CacheStatusFound, nil
	}
	stats.Append(stat.CacheCollision)
	return CacheStatusNotFound, nil
}

func (s *Service) getFromClient(ctx context.Context, key *Key, storable common.Storable, stats *stat.Values) error {
	err := s.fromL1Client(ctx, key, storable)
	if common.IsKeyNotFound(err) {
		stats.Append(stat.NoSuchKey)
		if s.l2Client == nil || key.L2 == nil {
			return err
		}
		err = s.fromL2Client(ctx, key.L2, storable)
		if err != nil {
			if common.IsKeyNotFound(err) {
				stats.Append(stat.L2NoSuchKey)
			} else if common.IsTimeout(err) {
				stats.Append(stat.Timeout)
				stats.Append(err)
			} else {
				stats.Append(err)
			}
			return err
		}

		s.copyTOL1(ctx, storable, key, stats)
		stats.Append(stat.HasValue)
		return nil
	}
	if err != nil {
		if common.IsTimeout(err) {
			stats.Append(stat.Timeout)
		}
		stats.Append(err)
		return err
	}
	stats.Append(stat.HasValue)
	return nil
}

func (s *Service) copyTOL1(ctx context.Context, storable common.Storable, key *Key, stats *stat.Values) error {
	value, err := storable.Iterator().ToMap()
	if err != nil {
		return err
	}
	writeKey, _ := key.Key()
	err = s.l1Client.Put(ctx, key.WritePolicy(0), writeKey, aerospike.BinMap(value))
	if err != nil {
		stats.Append(err)
		return err
	}
	stats.Append(stat.L1Copy)
	return nil
}

func (s *Service) fromL2Client(ctx context.Context, key *Key, storable common.Storable) error {
	return s.fromClient(ctx, s.l2Client, key, storable)
}

func (s *Service) fromL1Client(ctx context.Context, key *Key, storable common.Storable) error {
	return s.fromClient(ctx, s.l1Client, key, storable)
}

func (s *Service) fromClient(ctx context.Context, client client.Service, key *Key, storable common.Storable) error {
	clientKey, err := key.Key()
	if err != nil {
		return fmt.Errorf( "failed to create key: %+v, due to %w", key, err)
	}
	record, err := client.Get(ctx, clientKey)
	if err != nil {
		return err
	}
	if record == nil && len(record.Bins) == 0 {
		return nil
	}
	err = storable.Set(common.MapToIterator(record.Bins))
	if err != nil {
		return fmt.Errorf( "failed to map record: %+v, due to %w", key, err)
	}
	return nil
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
