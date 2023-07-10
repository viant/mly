package health

import (
	"encoding/json"
	"net/http"
	"sync"
)

type HealthHandler struct {
	healths map[string]*int32
	mu      *sync.Mutex
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		mu:      new(sync.Mutex),
		healths: make(map[string]*int32),
	}
}

func (h *HealthHandler) RegisterHealthPoint(name string, isOkPtr *int32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.healths[name] = isOkPtr
}

func (h *HealthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	JSON, _ := json.Marshal(h.healths)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}
