package service

import (
	"context"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/posener/h2conn"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/shared/common"
	"io"
	"net/http"
	"time"
)

type handler struct {
	maxDuration time.Duration
	service     *Service
	pool        *buffer.Pool
}

//NewContext creates a new context
func (h *handler) NewContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	return context.WithTimeout(ctx, h.maxDuration)
}

//ServeHTTP serve HTTP
func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == "PUT" {
		if err := h.serveHTTP2(writer, request); err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	if err := h.serveHTTP(writer, request); err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *handler) serveHTTP2(writer http.ResponseWriter, httpRequest *http.Request) error {
	conn, err := h2conn.Accept(writer, httpRequest)
	if err != nil {
		return err
	}
	defer conn.Close()
	data := h.pool.Get()
	defer h.pool.Put(data)
	for {
		if err = h.handleHTTP2(conn, data); err != nil {
			break
		}
	}
	return err
}

func (h *handler) handleHTTP2(conn io.ReadWriter, data []byte) error {
	response := &Response{Status: common.StatusOK, started: time.Now()} //this is bit
	size, err := conn.Read(data)
	if err != nil {
		return err
	}
	ctx, cancel := h.NewContext()
	defer cancel()
	request := h.service.NewRequest()
	request.Body = data[:size]
	if err = gojay.Unmarshal(data[:size], request); err != nil {
		return err
	}
	return h.handleAppRequest(ctx, conn, request, response)
}

func (h *handler) serveHTTP(writer http.ResponseWriter, httpRequest *http.Request) error {
	ctx, cancel := h.NewContext()
	defer cancel()
	request := h.service.NewRequest()
	response := &Response{Status: "ok", started: time.Now()}
	var err error
	if httpRequest.Body == nil {
		err = h.buildRequestFromQuery(httpRequest, request)
	} else {
		data, size, err := buffer.Read(h.pool, httpRequest.Body)
		defer h.pool.Put(data)
		if err != nil {
			return err
		}
		request.Body = data[:size]
		if err := gojay.Unmarshal(data[:size], request); err != nil {
			return err
		}
	}
	if err == nil {
		err = request.Validate()
	}
	if err != nil {
		response.setError(err)
		return h.writeResponse(writer, response)
	}
	return h.handleAppRequest(ctx, writer, request, response)
}

func (h *handler) buildRequestFromQuery(httpRequest *http.Request, request *Request) error {
	err := httpRequest.ParseForm()
	if err != nil {
		return fmt.Errorf("failed to parse get request: %w", err)
	}
	values := httpRequest.Form
	for k := range values {
		if err := request.Put(k, values.Get(k)); err != nil {
			return err
		}
	}
	return nil
}

func (h *handler) handleAppRequest(ctx context.Context, writer io.Writer, request *Request, response *Response) error {
	err := request.Validate()
	if err != nil {
		response.setError(err)
		if err = h.writeResponse(writer, response); err != nil {
			return err
		}
		return nil
	}
	if err = h.service.Do(ctx, request, response); err != nil {
		response.setError(err)
	}
	if err = h.writeResponse(writer, response); err != nil {
		return err
	}
	return nil
}

func (h *handler) writeResponse(writer io.Writer, appResponse *Response) error {
	appResponse.ServiceTimeMcs = int(time.Now().Sub(appResponse.started).Microseconds())
	data, err := gojay.Marshal(appResponse)
	_, err = writer.Write(data)
	return err
}

//NewHandler creates a new HTTP service handler
func NewHandler(service *Service, pool *buffer.Pool, maxDuration time.Duration) http.Handler {
	return &handler{
		service:     service,
		pool:        pool,
		maxDuration: maxDuration,
	}
}
