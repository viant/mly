package client

import (
	"fmt"
	"github.com/viant/mly/shared/circut"
	"github.com/viant/mly/shared/common"
	"net"
	"strconv"
	"time"
)

var requestTimeout = 5 *time.Second

//Host represents endpoint host
type Host struct {
	Name string
	Port int
	*circut.Breaker

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


func (c *Host) Probe() {
	if c.isConnectionUp() {
		c.FlagUp()
		return
	}
}


func (c *Host) isConnectionUp() bool {
	connection, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", c.Name, c.Port), requestTimeout)
	if err != nil {
		return false
	}
	defer connection.Close()
	return true
}

func (h *Host) Init() {
	if h.Breaker == nil {
		h.Breaker = circut.New(requestTimeout, h)
	}
}



//NewHost returns new host
func NewHost(name string, port int) *Host {
	if port == 0 {
		port = 80
	}
	return &Host{Name: name, Port: port}
}


//NewHosts creates hosts
func NewHosts(port int, names []string) []*Host {
	var result = make([]*Host, 0)
	for _, name := range names {
		result = append(result, &Host{Name: name ,Port: port})
	}
	return result
}
