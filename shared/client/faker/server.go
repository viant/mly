package faker

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/viant/afs"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	Port  int
	URL   string
	Debug bool
	*http.Server
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	handler := &Handler{baseURL: s.URL, fs: afs.New(), debug: s.Debug}
	mux.Handle("/", handler)
	s.Server = &http.Server{
		Addr:    ":" + strconv.Itoa(s.Port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	fmt.Printf("starting listening: %v ...\n", s.Server.Addr)
	s.Server.ListenAndServe()
	fmt.Printf("done %v\n", s.Server.Addr)
}

func (s *Server) Stop() {
	if s.Server == nil {
		return
	}
	s.Server.Close()
}
