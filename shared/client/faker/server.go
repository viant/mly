package faker

import (
	"fmt"
	"net"
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

	Handler *Handler

	*http.Server
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	handler := &Handler{
		baseURL: s.URL,
		fs:      afs.New(),
		debug:   s.Debug,
	}

	mux.Handle("/", handler)
	addr := ":" + strconv.Itoa(s.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	fmt.Printf("starting listen: %v ...\n", addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	go srv.Serve(ln)

	s.Server = srv
	s.Handler = handler

	fmt.Printf("done %v\n", addr)
}

func (s *Server) Stop() {
	if s.Server == nil {
		return
	}

	s.Server.Close()
}
