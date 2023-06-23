package endpoint

import (
	"net/http"
	"runtime/pprof"
	"sync"
)

const memProfURI = "/v1/api/debug/memprof"

const cpuProfIndexURI = "/v1/api/debug/pprof"

const cpuProfCmdlineURI = "/v1/api/debug/pprof/cmdline"
const cpuProfProfileURI = "/v1/api/debug/pprof/profile"
const cpuProfSymbolURI = "/v1/api/debug/pprof/symbol"
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
