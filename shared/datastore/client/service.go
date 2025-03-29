package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/viant/mly/shared/circut"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/config/datastore"
	"golang.org/x/sync/singleflight"
)

const (
	// DefaultPutSingleflightTimeout is based off default value for BasePolicy.SocketTimeout
	// Since retries are not recommended for Put operations, we will use this simple value
	DefaultPutSingleflightTimeout = time.Second * 30

	// This might be too short for a pure default configuration.
	// It looks like it should be something like SocketTimeout + SleepBetweenRetries * cumulative product from 1 to MaxRetries of SleepMultiplier...
	// But in our common use case, 30 seconds is very long, and the Aerospike Client should timeout first.
	DefaultGetSingleflightTimeout = time.Second * 30
)

// Service represents aerospike client Service
type Service struct {
	Client Aero

	config *datastore.Connection

	// bypassConfiguredTimeout is used to bypass the configured timeout if using WithClientPolicy or WithBasePolicy.
	bypassConfiguredTimeout bool

	// basePolicy can be overridden by WithBasePolicy.
	// basePolicy is only used for Get operations.
	// Even when overridden, the timeout will be applied UNLESS using WithBypassConfiguredTimeout.
	basePolicy *aero.BasePolicy

	// clientPolicy can be overridden by WithClientPolicy.
	// Even when overridden, the timeout will be applied UNLESS using WithBypassConfiguredTimeout.
	clientPolicy *aero.ClientPolicy

	// group is used to dedupe concurrent puts
	group *singleflight.Group

	*circut.Breaker
}

// Get returns record for supplied key and optional bin names.
func (s *Service) Get(ctx context.Context, key *aero.Key, binNames ...string) (record *aero.Record, err error) {
	if !s.IsUp() {
		return nil, common.ErrNodeDown
	}

	defer func() {
		if r := recover(); r != nil {
			connection := s.config.ID
			err = fmt.Errorf("get aerospike[%s]: panic: %v", connection, r)
		}
	}()

	record, err = s.Client.Get(s.basePolicy, key, binNames...)
	s.checkConnectionError(err)
	return record, err
}

// Put puts a record to Aerospike.
// Context is not supported since the Aerospike library does not support it.
func (s *Service) Put(writePolicy *aero.WritePolicy, key *aero.Key, value aero.BinMap) (err error) {
	if !s.IsUp() {
		return common.ErrNodeDown
	}

	if writePolicy == nil {
		writePolicy = aero.NewWritePolicy(0, 0)
	}

	keyStr := keyString(key)

	defer func() {
		if r := recover(); r != nil {
			connection := s.config.ID
			err = fmt.Errorf("put aerospike[%s] key: %s panic: %v", connection, keyStr, r)
		}
	}()

	var timeout time.Duration
	if writePolicy.TotalTimeout > 0 {
		timeout = time.Millisecond * writePolicy.TotalTimeout
	} else {
		timeout = DefaultPutSingleflightTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := s.group.DoChan(keyStr, func() (interface{}, error) {
		err := s.Client.Put(writePolicy, key, value)
		s.checkConnectionError(err)
		return nil, err
	})

	select {
	case <-ctx.Done():
		err = fmt.Errorf("put aerospike[%s] key: %s singleflight: %w", s.config.ID, keyStr, ctx.Err())
	case res := <-ch:
		if res.Err != nil {
			err = fmt.Errorf("put aerospike[%s] key: %s shared: %v error: %w", s.config.ID, keyStr, res.Shared, res.Err)
		}
	}

	return err
}

func (s *Service) Probe() {
	if err := s.connect(); err == nil {
		s.FlagUp()
	}
}

func (s *Service) checkConnectionError(err error) {
	if err == nil {
		return
	}
	if common.IsInvalidNode(err) {
		s.FlagDown()
	}
}

func (s *Service) connect() error {
	hosts := s.hosts()
	if len(hosts) == 0 {
		return fmt.Errorf("hostname was empty")
	}
	client, err := aero.NewClientWithPolicyAndHost(s.clientPolicy, hosts...)
	if err != nil {
		return err
	}
	s.Client = client
	return err
}

func (s *Service) hosts() []*aero.Host {
	var hosts = make([]*aero.Host, 0)
	for _, name := range strings.Split(s.config.Hostnames, ",") {
		hosts = append(hosts, &aero.Host{Name: name, Port: s.config.Port})
	}
	return hosts
}

func (s *Service) init(options ...Option) {
	for _, option := range options {
		option(s)
	}

	clientPolicy := s.clientPolicy
	if clientPolicy == nil {
		clientPolicy = aero.NewClientPolicy()
		s.clientPolicy = clientPolicy
	}

	basePolicy := s.basePolicy
	if basePolicy == nil {
		basePolicy = aero.NewPolicy()
		s.basePolicy = basePolicy
	}

	if !s.bypassConfiguredTimeout {
		timeout := s.config.Timeout
		if timeout.Connection > 0 {
			clientPolicy.Timeout = timeout.DurationUnit() * time.Duration(timeout.Connection)
		}

		if timeout.Socket > 0 {
			basePolicy.SocketTimeout = timeout.DurationUnit() * time.Duration(timeout.Socket)
		}

		if timeout.Total > 0 {
			basePolicy.TotalTimeout = timeout.DurationUnit() * time.Duration(timeout.Total)
		}
	}
}

// New creates a new Aerospike service
func New(config *datastore.Connection) (*Service, error) {
	return NewWithOptions(config)
}

func NewWithOptions(config *datastore.Connection, options ...Option) (*Service, error) {
	srv := &Service{
		config: config,
		group:  new(singleflight.Group),
	}

	srv.init(options...)
	breaker := circut.New(time.Second, srv)
	srv.Breaker = breaker
	return srv, srv.connect()
}
