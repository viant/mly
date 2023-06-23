package endpoint

import (
	"encoding/json"
	"net/http"
	"sync"
)

const healthURI = "/v1/api/health"

type healthHandler struct {
	healths map[string]*int32
	mu      *sync.Mutex
}

func NewHealthHandler() *healthHandler {
	return &healthHandler{
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
	JSON, _ := json.Marshal(h.healths)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}
