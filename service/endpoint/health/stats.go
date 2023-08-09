package health

import (
	"encoding/json"
	"net/http"
)

// Deprecated. Used for maintaining non-model specific statistics
type StatsHandler struct {
}

type statDisplay struct {
	SemaphoreTokens int
}

func (h *StatsHandler) stats() statDisplay {
	return statDisplay{}
}

func NewStatsHandler() *StatsHandler {
	return &StatsHandler{}
}

// implements http.Handler
func (h *StatsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	JSON, _ := json.Marshal(h.stats())
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(JSON)
}
