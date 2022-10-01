package client

import (
	"github.com/viant/gmetric"
	cconfig "github.com/viant/mly/shared/client/config"
	"github.com/viant/mly/shared/datastore"
)

//Option client option
type Option interface {
	Apply(c *Service)
}

type cacheSizeOpt struct {
	sizeMB int
}

//Apply applies settings
func (o *cacheSizeOpt) Apply(c *Service) {
	c.Config.CacheSizeMb = o.sizeMB
}

//WithCacheSize returns cache size MB
func WithCacheSize(sizeMB int) Option {
	return &cacheSizeOpt{sizeMB: sizeMB}
}

type gmetricsOpt struct {
	gmetrics *gmetric.Service
}

//Apply metrics
func (o *gmetricsOpt) Apply(c *Service) {
	c.gmetrics = o.gmetrics
}

//WithGmetrics returns gmetric options
func WithGmetrics(gmetrics *gmetric.Service) Option {
	return &gmetricsOpt{gmetrics: gmetrics}
}

type dictHashValidationOpt struct {
	enable bool
}

//Apply dict validation
func (o *dictHashValidationOpt) Apply(c *Service) {
	c.Config.DictHashValidation = o.enable
}

//WithHashValidation creates a new dict has validation
func WithHashValidation(enable bool) Option {
	return &dictHashValidationOpt{enable: enable}
}

type cacheScopeOption struct {
	scope CacheScope
}

//Apply metrics
func (o *cacheScopeOption) Apply(c *Service) {
	c.Config.CacheScope = &o.scope
}

//WithCacheScope creates cache scope option
func WithCacheScope(scope CacheScope) Option {
	return &cacheScopeOption{scope: scope}
}

type clientRemoteOption struct {
	config *cconfig.Remote
}

//Apply metrics
func (o *clientRemoteOption) Apply(c *Service) {
	c.Config.Datastore = o.config
	c.Config.Datastore.Init()
}

//WithCacheScope creates cache scope option
func WithRemoteConfig(config *cconfig.Remote) Option {
	return &clientRemoteOption{config: config}
}

type dictionaryOption struct {
	dictionary *Dictionary
}

//Apply metrics
func (o *dictionaryOption) Apply(c *Service) {
	c.dict = o.dictionary
}

//WithDictionary creates dictionary option
func WithDictionary(dictionary *Dictionary) Option {
	return &dictionaryOption{dictionary: dictionary}
}

type storerOption struct {
	storer datastore.Storer
}

//Apply metrics
func (o *storerOption) Apply(c *Service) {
	c.datastore = o.storer
}

//WithDataStorer creates dictionary option
func WithDataStorer(storer datastore.Storer) Option {
	return &storerOption{storer: storer}
}
