package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/shared/pb"
	"io"
	"net/http"
	"time"
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
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *Handler) Evaluate(ctx context.Context, grpcRequest *pb.EvaluateRequest) (*pb.EvaluateResponse, error) {
	request := h.service.NewRequest()
	response := &Response{Status: "ok", started: time.Now()}
	err := gojay.Unmarshal(grpcRequest.Input, request)
	if err == nil {
		writer := new(bytes.Buffer)
		if err = h.handleAppRequest(ctx, writer, request, response); err != nil {
			response.SetError(err)
		}
	}
	output, _ := gojay.Marshal(response)
	return &pb.EvaluateResponse{
		Output: output,
	}, nil
}

func (h *Handler) serveHTTP(writer http.ResponseWriter, httpRequest *http.Request) error {
	ctx, cancel := h.NewContext()
	defer cancel()
	request := h.service.NewRequest()
	response := &Response{Status: "ok", started: time.Now()}
	if httpRequest.Body == nil {
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
		if err := gojay.Unmarshal(data[:size], request); err != nil {
			return err
		}
	}
	return h.handleAppRequest(ctx, writer, request, response)
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
