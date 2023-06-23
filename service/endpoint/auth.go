package endpoint

import (
	"net/http"

	"github.com/viant/mly/shared/common"
)

type AuthHandler struct {
	*Config
	handler http.Handler
}

func NewAuthHandler(config *Config, handler http.Handler) *AuthHandler {
	h := new(AuthHandler)
	h.Config = config
	h.handler = handler
	return h
}

func (h *AuthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !common.IsAuthorized(request, h.Config.AllowedSubnet) {
		writer.WriteHeader(http.StatusForbidden)
		return
	}

	h.handler.ServeHTTP(writer, request)
}

type AuthMux struct {
	mux    *http.ServeMux
	config *Config
}

func NewAuthMux(mux *http.ServeMux, config *Config) *AuthMux {
	am := new(AuthMux)
	am.mux = mux
	am.config = config
	return am
}

func (m *AuthMux) Handle(path string, handler http.Handler) {
	a := NewAuthHandler(m.config, handler)
	m.mux.Handle(path, a)
}

type w struct {
	hf func(w http.ResponseWriter, r *http.Request)
}

func (w *w) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	w.hf(wr, r)
}

func (m *AuthMux) HandleFunc(path string, hf func(w http.ResponseWriter, r *http.Request)) {
	m.Handle(path, &w{hf})
}
