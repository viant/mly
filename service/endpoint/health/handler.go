package health

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/viant/mly/service"
	"github.com/viant/mly/service/config"
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

// implements Hook
func (h *HealthHandler) Hook(model *config.Model, modelSrv *service.Service) {
	h.RegisterHealthPoint(model.ID, &modelSrv.ReloadOK)
}

// implements http.Handler
func (h *HealthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	JSON, _ := json.Marshal(h.healths)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}
