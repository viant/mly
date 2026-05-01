package client

import (
	"time"

	"github.com/viant/gmetric"
	cconfig "github.com/viant/mly/shared/client/config"
	"github.com/viant/mly/shared/datastore"
	dscli "github.com/viant/mly/shared/datastore/client"
)

// Option is a pattern to apply a client option.
type Option interface {
	// Apply applies settings
	Apply(c *Service)
}

type cacheSizeOpt struct {
	sizeMB int
}

func (o *cacheSizeOpt) Apply(c *Service) {
	c.Config.CacheSizeMb = o.sizeMB
}

// WithCacheSize overrides the cache size provided by the server.
func WithCacheSize(sizeMB int) Option {
	return &cacheSizeOpt{sizeMB: sizeMB}
}

type gmetricsOpt struct {
	gmetrics *gmetric.Service
}

func (o *gmetricsOpt) Apply(c *Service) {
	c.gmetrics = o.gmetrics
}

// WithGmetrics binds the *gmetric.Service to the client.
func WithGmetrics(gmetrics *gmetric.Service) Option {
	return &gmetricsOpt{gmetrics: gmetrics}
}

type dictHashValidationOpt struct {
	enable bool
}

func (o *dictHashValidationOpt) Apply(c *Service) {
	c.Config.DictHashValidation = o.enable
}

// WithHashValidation overrides DictHashValidation.
func WithHashValidation(enable bool) Option {
	return &dictHashValidationOpt{enable: enable}
}

type withDebug struct {
	enable bool
}

func (o *withDebug) Apply(c *Service) {
	c.Config.Debug = o.enable
}

// WithDebug sets debugging.
func WithDebug(enable bool) Option {
	return &withDebug{enable: enable}
}

type cacheScopeOption struct {
	scope CacheScope
}

func (o *cacheScopeOption) Apply(c *Service) {
	c.Config.CacheScope = &o.scope
}

// WithCacheScope creates cache scope option
func WithCacheScope(scope CacheScope) Option {
	return &cacheScopeOption{scope: scope}
}

type clientRemoteOption struct {
	config *cconfig.Remote
}

func (o *clientRemoteOption) Apply(c *Service) {
	c.Config.Datastore = o.config
	c.Config.Datastore.Init()
}

// WithRemoteConfig will provide a hard-coded configuration instead of sending a request
// to the mly server to fetch the configuration.
func WithRemoteConfig(config *cconfig.Remote) Option {
	return &clientRemoteOption{config: config}
}

type dictionaryOption struct {
	dictionary *Dictionary
}

func (o *dictionaryOption) Apply(c *Service) {
	c.dict = o.dictionary
}

// WithDictionary overwrites the initial dictionary.
func WithDictionary(dictionary *Dictionary) Option {
	return &dictionaryOption{dictionary: dictionary}
}

type storerOption struct {
	storer datastore.Storer
}

func (o *storerOption) Apply(c *Service) {
	c.datastore = o.storer
}

// WithDataStorer will provide a coded instane of datastore instead of
// determining it based off configuration values.
func WithDataStorer(storer datastore.Storer) Option {
	return &storerOption{storer: storer}
}

type connectionSharingOption struct {
	connections map[string]*dscli.Service
}

func (o *connectionSharingOption) Apply(c *Service) {
	c.connections = o.connections
}

// WithConnectionSharing enables Aerospike connection sharing.
// Note that connections are shared via ID, not by Hostnames.
func WithConnectionSharing(connections map[string]*dscli.Service) Option {
	return &connectionSharingOption{connections: connections}
}

type clientOptionsOption struct {
	clientOptions []dscli.Option
}

func (o *clientOptionsOption) Apply(c *Service) {
	c.clientOptions = o.clientOptions
}

// WithClientOptions adds shared/datastore/client.Option options.
func WithClientOptions(clientOptions ...dscli.Option) Option {
	return &clientOptionsOption{clientOptions: clientOptions}
}

type latencyBreakerOpt struct {
	latest, rolling, window time.Duration
	k                       int
	fraction                float64
}

func (o *latencyBreakerOpt) Apply(c *Service) {
	c.Config.LatencyBreakerLatestThreshold = o.latest
	c.Config.LatencyBreakerRollingThreshold = o.rolling
	c.Config.LatencyBreakerRollingWindow = o.window
	c.Config.LatencyBreakerKConsecutive = o.k
	c.Config.LatencyBreakerPassThroughFraction = o.fraction
}

// WithLatencyBreaker enables the latency-aware breaker on each host
// constructed for this Service. Defaults are applied during Service
// init: pass-through fraction 0.01, rolling window 1s, KConsecutive 3.
//
// Setting both latest and rolling to zero leaves the breaker disabled
// (backward compatible). Both thresholds are taken as raw durations;
// the caller is responsible for sizing them appropriately for the
// model's traffic profile and the caller's request timeout.
func WithLatencyBreaker(latest, rolling, window time.Duration, k int, fraction float64) Option {
	return &latencyBreakerOpt{latest: latest, rolling: rolling, window: window, k: k, fraction: fraction}
}
