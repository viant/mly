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
	CacheSizeMb int
	CacheScope  *CacheScope
	Datastore   *config.Datastore
	MaxRetry    int
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
