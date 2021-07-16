package client

import (
	"github.com/viant/mly/shared/client/config"
)

//Config represents a client config
type Config struct {
	Hosts       []*Host
	Model       string
	CacheSizeMb int
	CacheScope  *CacheScope
	Datastore   *config.Datastore
	MaxRetry    int
	DictHashValidation bool
}

//CacheSize returns cache size
func (c *Config) CacheSize() int {
	if c.CacheSizeMb == 0 {
		return 0
	}
	return 1024 * 1024 * (c.CacheSizeMb)
}

func (c *Config) updateCache() {
	if c.CacheSizeMb > 0 {
		if c.Datastore != nil && c.Datastore.Cache != nil {
			c.Datastore.Cache.SizeMb = c.CacheSizeMb
		}
	}

	scope := c.CacheScope
	if scope == nil {
		return
	}
	if !scope.IsL2() && c.Datastore != nil {
		c.Datastore.Datastore.L2 = nil
	}
	if !scope.IsL1() && c.Datastore != nil {
		c.Datastore.Datastore.Connection = ""
		c.Datastore.Connections = nil
	}
	if !scope.IsLocal() && c.Datastore != nil {
		c.Datastore = nil
	}
}
