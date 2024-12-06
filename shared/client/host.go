package client

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/viant/mly/shared/circut"
	"github.com/viant/mly/shared/common"
)

var defaultRequestTimeout = 50 * time.Millisecond

// Host represents endpoint host
type Host struct {
	name string
	port int

	// used as both a check to see if a host is down
	// and to pause requests to prevent downstream overload
	RequestTimeout time.Duration

	*circut.Breaker

	// memoization
	prefix string
}

func isSecurePort(port int) bool {
	return port == 443 || port == 1443
}

// IsSecurePort() returns true if secure port
func (h *Host) IsSecurePort() bool {
	return isSecurePort(h.port)
}

// URL returns model eval URL
func (h *Host) evalURL(model string) string {
	return h.prefix + fmt.Sprintf(common.ModelURI, model)
}

// URL returns meta config model eval URL
func (h *Host) metaConfigURL(model string) string {
	return h.prefix + fmt.Sprintf(common.MetaConfigURI, model)
}

// URL returns meta config model eval URL
func (h *Host) metaDictionaryURL(model string) string {
	return h.prefix + fmt.Sprintf(common.MetaDictionaryURI, model)
}

// called by shared/circuit.(*Breaker).resetIfDue()
func (h *Host) Probe() {
	if h.isConnectionUp() {
		h.FlagUp()
		return
	}
}

func (h *Host) isConnectionUp() bool {
	port := h.port
	connection, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", h.name, port), h.RequestTimeout)
	if err != nil {
		return false
	}
	defer connection.Close()
	return true
}

func (h *Host) Init() {
	if h.Breaker == nil {
		h.Breaker = circut.New(h.RequestTimeout, h)
	}

	proto := "http://"
	if h.IsSecurePort() {
		proto = "https://"
	}

	h.prefix = proto + h.name + ":" + strconv.Itoa(h.port)
}

// Name aka domain name
func (h *Host) Name() string {
	return h.name
}

func (h *Host) Port() int {
	return h.port
}

// NewHost returns new host
func NewHost(name string, port int) *Host {
	if port <= 0 {
		port = 80
	}

	return &Host{
		name:           name,
		port:           port,
		RequestTimeout: defaultRequestTimeout,
	}
}

// NewHosts creates hosts
func NewHosts(port int, names []string) []*Host {
	var result = make([]*Host, 0)
	for _, name := range names {
		result = append(result, &Host{name: name, port: port})
	}
	return result
}
