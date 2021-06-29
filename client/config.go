package client

import (
	"fmt"
	"github.com/viant/mly/client/config"
	"github.com/viant/mly/common"
	"strconv"
)

//Config represents a client config
type Config struct {
	Hosts       []*Host
	Model       string
	CacheSizeMb *int
	Datastore   *config.Datastore
	MaxRetry    int
}

//CacheSize returns cache size
func (c *Config) CacheSize() int {
	if c.CacheSizeMb == nil {
		return 0
	}
	return 1024 * 1024 * (*c.CacheSizeMb)
}

func (c *Config) updateCache() {
	if c.CacheSizeMb == nil {
		if c.Datastore != nil && c.Datastore.Cache != nil {
			c.CacheSizeMb = &c.Datastore.Cache.SizeMb
		}
		return
	}
	if *c.CacheSizeMb == NoCache {
		c.Datastore = nil
		return
	}
	if c.Datastore != nil {
		if cache := c.Datastore.Cache; cache != nil {
			cache.SizeMb = *c.CacheSizeMb
		}
	}
}

//Host represents endpoint host
type Host struct {
	Name string
	Port int
}

//URL returns model eval URL
func (h *Host) evalURL(model string) string {
	return "http://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.ModelURI, model)
}

//URL returns meta config model eval URL
func (h *Host) metaConfigURL(model string) string {
	return "http://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.MetaConfigURI, model)
}

//URL returns meta config model eval URL
func (h *Host) metaDictionaryURL(model string) string {
	return "http://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.MetaDictionaryURI, model)
}

//NewHost returns new host
func NewHost(name string, port int) *Host {
	return &Host{Name: name, Port: port}
}
