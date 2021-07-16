package client

import (
	"github.com/viant/gmetric"
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

//NewCacheSize returns cache size MB
func NewCacheSize(sizeMB int) Option {
	return &cacheSizeOpt{sizeMB: sizeMB}
}

type cacheScopeOpt struct {
	scope CacheScope
}

//Apply applies settings
func (o *cacheScopeOpt) Apply(c *Service) {
	c.Config.CacheScope = &o.scope
}

//NewCacheScope returns cache scope
func NewCacheScope(scope CacheScope) Option {
	return &cacheScopeOpt{scope: scope}
}

type gmetricsOpt struct {
	gmetrics *gmetric.Service
}

//Apply metrics
func (o *gmetricsOpt) Apply(c *Service) {
	c.gmetrics = o.gmetrics
}

//NewGmetric returns gmetric options
func NewGmetric(gmetrics *gmetric.Service) Option {
	return &gmetricsOpt{gmetrics: gmetrics}
}

type dictHashValidationOpt struct {
	enable bool
}

//Apply metrics
func (o *dictHashValidationOpt) Apply(c *Service) {
	c.Config.DictHashValidation = o.enable
}

//NewDictHashValidation creates a new dict has validation
func NewDictHashValidation(enable bool) Option {
	return &dictHashValidationOpt{enable: enable}
}
