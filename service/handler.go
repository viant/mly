package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/viant/mly/service/buffer"
)

type Handler struct {
	maxDuration time.Duration
	service     *Service
	pool        *buffer.Pool
}

//NewContext creates a new context
func (h *Handler) NewContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	return context.WithTimeout(ctx, h.maxDuration)
}

//ServeHTTP serve HTTP
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if err := h.serveHTTP(writer, request); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) serveHTTP(writer http.ResponseWriter, httpRequest *http.Request) error {
	ctx, cancel := h.NewContext()
	defer cancel()
	request := h.service.NewRequest()
	response := &Response{Status: "ok", started: time.Now()}
	if httpRequest.Method == http.MethodGet {
		if err := h.buildRequestFromQuery(httpRequest, request); err != nil {
			return err
		}
	} else {
		defer httpRequest.Body.Close()
		data, size, err := buffer.Read(h.pool, httpRequest.Body)
		defer h.pool.Put(data)
		if err != nil {
			return err
		}

		request.Body = data[:size]
		if h.service.config.Debug {
			fmt.Printf("[%v] input: %s\n", h.service.config.ID, request.Body)
		}

		err = gojay.Unmarshal(data[:size], request)
		if err != nil {
			return err
		}
	}
	err := h.handleAppRequest(ctx, writer, request, response)
	if h.service.config.Debug {
		data, _ := json.Marshal(response.Data)
		fmt.Printf("[%v] output: %s %T\n", h.service.config.ID, data, response.Data)
	}
	return err
}

func (h *Handler) buildRequestFromQuery(httpRequest *http.Request, request *Request) error {
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

func (h *Handler) handleAppRequest(ctx context.Context, writer io.Writer, request *Request, response *Response) error {
	if err := h.service.Do(ctx, request, response); err != nil {
		response.SetError(err)
	}
	if err := h.writeResponse(writer, response); err != nil {
		return err
	}
	return nil
}

func (h *Handler) writeResponse(writer io.Writer, appResponse *Response) error {
	appResponse.ServiceTimeMcs = int(time.Now().Sub(appResponse.started).Microseconds())
	data, err := gojay.Marshal(appResponse)
	_, err = writer.Write(data)
	return err
}

//NewHandler creates a new HTTP service Handler
func NewHandler(service *Service, pool *buffer.Pool, maxDuration time.Duration) *Handler {
	return &Handler{
		service:     service,
		pool:        pool,
		maxDuration: maxDuration,
	}
}
