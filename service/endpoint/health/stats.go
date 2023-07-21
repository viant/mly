package health

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/viant/mly/service"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/shared/semaph"
)

// Used for maintaing non-model specific statistics
type StatsHandler struct {
	mu *sync.Mutex

	sema *semaph.Semaph
}

type stats struct {
	SemaphoreTokens int32
}

func (h *StatsHandler) stats() stats {
	return stats{
		SemaphoreTokens: h.sema.R(),
	}
}

func NewStatsHandler() *StatsHandler {
	return &StatsHandler{
		mu: new(sync.Mutex),
	}
}

// implements Hook
func (h *StatsHandler) Hook(c *config.Model, s *service.Service) {
	h.sema = s.Sema()
}

// implements http.Handler
func (h *StatsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	JSON, _ := json.Marshal(h.stats())
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}
