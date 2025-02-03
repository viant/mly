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
)

// Service represents aerospike client Service
type Service struct {
	*aero.Client

	config *datastore.Connection

	// bypassConfiguredTimeout is used to bypass the configured timeout if using WithClientPolicy or WithBasePolicy.
	bypassConfiguredTimeout bool

	// basePolicy can be overridden by WithBasePolicy.
	// Even when overridden, the timeout will be applied UNLESS using WithBypassConfiguredTimeout.
	basePolicy *aero.BasePolicy

	// clientPolicy can be overridden by WithClientPolicy.
	// Even when overridden, the timeout will be applied UNLESS using WithBypassConfiguredTimeout.
	clientPolicy *aero.ClientPolicy

	*circut.Breaker
}

// Get returns record for supplied key and optional bin names.
func (s *Service) Get(ctx context.Context, key *aero.Key, binNames ...string) (record *aero.Record, err error) {
	if !s.IsUp() {
		return nil, common.ErrNodeDown
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to read data from aersopike: panic: %v", r)
		}
	}()
	record, err = s.Client.Get(s.basePolicy, key, binNames...)
	s.checkConnectionError(err)
	return record, err
}

// Put puts a record to Aerospike.
// Context is not supported since the Aerospike library does not support it.
func (s *Service) Put(writePolicy *aero.WritePolicy, key *aero.Key, value aero.BinMap) error {
	if !s.IsUp() {
		return common.ErrNodeDown
	}

	err := s.Client.Put(writePolicy, key, value)
	s.checkConnectionError(err)
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
	return NewWithOptions(config, nil)
}

func NewWithOptions(config *datastore.Connection, options ...Option) (*Service, error) {
	srv := &Service{
		config: config,
	}
	srv.init(options...)
	breaker := circut.New(time.Second, srv)
	srv.Breaker = breaker
	return srv, srv.connect()
}
