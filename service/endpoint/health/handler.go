package health

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/viant/mly/service"
	"github.com/viant/mly/service/config"
)

type HealthHandler struct {
	healths map[string]GetHealth
	mu      *sync.Mutex
}

type GetHealth interface {
	GetHealth() int32
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		mu:      new(sync.Mutex),
		healths: make(map[string]GetHealth),
	}
}

func (h *HealthHandler) RegisterHealthPoint(name string, gh GetHealth) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.healths[name] = gh
}

// implements Hook
func (h *HealthHandler) Hook(model *config.Model, modelSrv *service.Service) {
	h.RegisterHealthPoint(model.ID, modelSrv)
}

// implements http.Handler
func (h *HealthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	healths := make(map[string]int32)
	for name, gh := range h.healths {
		healths[name] = gh.GetHealth()
	}

	JSON, _ := json.Marshal(healths)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}
