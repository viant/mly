package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/service/clienterr"
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
func (h *Handler) ServeHTTP(writer http.ResponseWriter, httpRequest *http.Request) {
	ctx, cancel := h.NewContext()
	defer cancel()

	isDebug := h.service.config.Debug

	request := h.service.NewRequest()
	response := &Response{Status: "ok", started: time.Now()}
	if httpRequest.Method == http.MethodGet {
		if err := h.buildRequestFromQuery(httpRequest, request); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		defer httpRequest.Body.Close()
		data, size, err := buffer.Read(h.pool, httpRequest.Body)
		defer h.pool.Put(data)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		request.Body = data[:size]
		if isDebug {
			log.Printf("[%v http] input: %s\n", h.service.config.ID, strings.Trim(string(request.Body), " \n\r"))
		}

		err = gojay.Unmarshal(data[:size], request)
		if err != nil {
			if isDebug {
				log.Printf("[%v http] unmarshal error: %v\n", h.service.config.ID, err)
			}

			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
	}

	err := h.handleAppRequest(ctx, writer, request, response)
	if isDebug {
		data, merr := json.Marshal(response.Data)

		if merr == nil {
			log.Printf("[%v http] output:%s", h.service.config.ID, data)
		} else {
			log.Printf("[%v http] marshal error:%v data:%s", h.service.config.ID, merr, response.Data)
		}

	}

	if err != nil {
		var status int
		if _, ok := err.(*clienterr.ClientError); ok {
			status = http.StatusBadRequest
		} else {
			status = http.StatusInternalServerError
		}

		if isDebug {
			log.Printf("[%v http] status:%d error:%v", h.service.config.ID, status, err)
		}

		http.Error(writer, err.Error(), status)
	}
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
		return err
	}

	if err := h.writeResponse(writer, response); err != nil {
		return err
	}

	return nil
}

func (h *Handler) writeResponse(writer io.Writer, appResponse *Response) error {
	appResponse.ServiceTimeMcs = int(time.Now().Sub(appResponse.started).Microseconds())
	data, err := gojay.Marshal(appResponse)
	if h.service.config.Debug {
		log.Printf("[%v write] output:%s", h.service.config.ID, data)
	}
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
