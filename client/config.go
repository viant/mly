package client

import (
	"fmt"
	"github.com/viant/mly/client/config"
	"github.com/viant/mly/common"
	"strconv"
)

//Config represents a client config
type Config struct {
	Hosts     []*Host
	Model     string
	CacheSize int
	Datastore *config.Datastore
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



func NewHost(name string, port int) *Host {
	return &Host{Name: name, Port: port}
}
