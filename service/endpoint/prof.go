package endpoint

import (
	"net/http"
	"runtime/pprof"
	"sync"

	"github.com/viant/mly/shared/common"
)

const memProfURI = "/v1/api/debug/memprof"

type profHandler struct {
	config *Config
	l      *sync.Mutex
}

func NewProfHandler(config *Config) *profHandler {
	return &profHandler{
		config: config,
		l:      new(sync.Mutex),
	}
}

func (h *profHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !common.IsAuthorized(request, h.config.AllowedSubnet) {
		writer.WriteHeader(http.StatusForbidden)
		return
	}

	h.l.Lock()
	defer h.l.Unlock()

	writer.Header().Set("Content-Disposition", "attachment; filename=memprof.prof")
	pprof.WriteHeapProfile(writer)
	return
}
