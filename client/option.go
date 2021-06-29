package client

import (
	"github.com/viant/gmetric"
)

const (
	//NoCache no cache option
	NoCache = -1 //clients does not read/write to local and remote cache
	//NoLocalCache no local cache
	NoLocalCache = 0 //client does not read/write to local, but it passes key so server may still use hash
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
	c.Config.CacheSizeMb = &o.sizeMB
}

//NewCacheSize returns cache size MB
func NewCacheSize(sizeMB int) Option {
	return &cacheSizeOpt{sizeMB: sizeMB}
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
