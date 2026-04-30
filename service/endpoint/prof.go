package endpoint

import (
	"net/http"
	"runtime/pprof"
	"sync"
)

// Deprecated - all profiling endpoints are on a separate port
const memProfURI = "/v1/api/debug/memprof"

// Deprecated
const cpuProfIndexURI = "/v1/api/debug/pprof/"

// Deprecated
const cpuProfCmdlineURI = "/v1/api/debug/pprof/cmdline"

// Deprecated
const cpuProfProfileURI = "/v1/api/debug/pprof/profile"

// Deprecated
const cpuProfSymbolURI = "/v1/api/debug/pprof/symbol"

// Deprecated
const cpuProfTraceURI = "/v1/api/debug/pprof/trace"

type memProfHandler struct {
	l *sync.Mutex
}

func NewProfHandler() *memProfHandler {
	return &memProfHandler{
		l: new(sync.Mutex),
	}
}

func (h *memProfHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.l.Lock()
	defer h.l.Unlock()

	writer.Header().Set("Content-Disposition", "attachment; filename=memprof.prof")
	pprof.WriteHeapProfile(writer)
}
