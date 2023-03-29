package endpoint

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/viant/mly/shared/common"
)

const healthURI = "/v1/api/health"

type healthHandler struct {
	config  *Config
	healths map[string]*int32

	mu *sync.Mutex
}

func NewHealthHandler(config *Config) *healthHandler {
	return &healthHandler{
		config:  config,
		mu:      new(sync.Mutex),
		healths: make(map[string]*int32),
	}
}

func (h *healthHandler) RegisterHealthPoint(name string, isOkPtr *int32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.healths[name] = isOkPtr
}

func (h *healthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !common.IsAuthorized(request, h.config.AllowedSubnet) {
		writer.WriteHeader(http.StatusForbidden)
		return
	}

	JSON, _ := json.Marshal(h.healths)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}
