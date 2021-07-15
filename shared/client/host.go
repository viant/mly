package client

import (
	"fmt"
	"github.com/viant/mly/shared/circut"
	"github.com/viant/mly/shared/common"
	"net"
	"strconv"
	"sync"
	"time"
)

var requestTimeout = 5 * time.Second

//Host represents endpoint host
type Host struct {
	Name         string
	Port         int
	SecurePort   int
	GRPCPort     int
	GRPCPoolSize int
	mux          sync.RWMutex
	*circut.Breaker
	pool *grpcPool
}

//IsSecurePort() returns true if secure port
func (h *Host) IsSecurePort() bool {
	return h.Port%1000 == 443
}

//URL returns model eval URL
func (h *Host) evalURL(model string) string {
	if h.IsSecurePort() {
		return "https://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.ModelURI, model)
	}
	return "http://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.ModelURI, model)
}

//URL returns meta config model eval URL
func (h *Host) metaConfigURL(model string) string {
	if h.IsSecurePort() {
		return "https://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.MetaConfigURI, model)
	}
	return "http://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.MetaConfigURI, model)
}

//URL returns meta config model eval URL
func (h *Host) metaDictionaryURL(model string) string {
	if h.IsSecurePort() {
		return "https://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.MetaDictionaryURI, model)
	}
	return "http://" + h.Name + ":" + strconv.Itoa(h.Port) + fmt.Sprintf(common.MetaDictionaryURI, model)
}

func (h *Host) Probe() {
	if h.isConnectionUp() {
		h.FlagUp()
		return
	}
}

func (h *Host) isConnectionUp() bool {
	connection, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", h.Name, h.Port), requestTimeout)
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
	if h.GRPCPoolSize == 0 {
		h.GRPCPoolSize = 30
	}
}

func (h *Host) gRPCPool() *grpcPool {
	h.mux.RLock()
	pool := h.pool
	h.mux.RUnlock()
	if pool == nil {
		h.mux.Lock()
		if h.pool == nil {
			h.pool = newGrpcPool(h.GRPCPoolSize, fmt.Sprintf("%v:%v", h.Name, h.GRPCPort))
		}
		h.mux.Unlock()
	}
	return h.pool
}

//NewHost returns new host
func NewHost(name string, port, GRPCPort int) *Host {
	if port == 0 {
		port = 80
	}
	return &Host{Name: name, Port: port, GRPCPort: GRPCPort}
}

//NewHosts creates hosts
func NewHosts(port, securePort, grpcPort int, names []string) []*Host {
	var result = make([]*Host, 0)
	for _, name := range names {
		result = append(result, &Host{Name: name, Port: port, GRPCPort: grpcPort})
	}
	return result
}
