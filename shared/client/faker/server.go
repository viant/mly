package faker

import (
	"fmt"
	"github.com/viant/afs"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
	"strconv"
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
	fmt.Printf("starting listerning: %v ...\n", s.Server.Addr)
	s.Server.ListenAndServe()
}

func (s *Server) Stop() {
	if s.Server == nil {
		return
	}
	s.Server.Close()
}
