package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/request"
	"github.com/viant/mly/shared/common"
)

// Handler converts a model prediction HTTP request to its internal calls.
type Handler struct {
	maxDuration time.Duration
	service     *Service
	pool        *buffer.Pool

	poolMetrics *gmetric.Operation
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, httpRequest *http.Request) {
	// use Background() since there are things to be done regardless of if the request is cancceled from the client side.
	ctx := context.Background()
	// TODO: Handle httpRequest.Context() - there are issues since this can be canceled but there should be housekeeping completed.
	ctx, cancel := context.WithTimeout(ctx, h.maxDuration)
	defer cancel()

	isDebug := h.service.config.Debug

	request := h.service.NewRequest()
	response := &Response{Status: common.StatusOK, started: time.Now()}
	if httpRequest.Method == http.MethodGet {
		if err := h.buildRequestFromQuery(httpRequest, request); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		defer httpRequest.Body.Close()

		onDone := h.poolMetrics.Begin(time.Now())
		data, size, err := buffer.Read(h.pool, httpRequest.Body)

		defer h.pool.Put(data)

		func() {
			defer func() { onDone(time.Now()) }()

			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			request.Body = data[:size]
			if isDebug {
				trimmed := strings.Trim(string(request.Body), " \n\r")
				log.Printf("[%v http] input: %s\n", h.service.config.ID, trimmed)
			}

			err = gojay.Unmarshal(data[:size], request)
			if err != nil {
				if isDebug {
					log.Printf("[%v http] unmarshal error: %v\n", h.service.config.ID, err)
				}

				rmsg := fmt.Sprintf("%s (are your input types correct?)", err.Error())
				http.Error(writer, rmsg, http.StatusBadRequest)
				return
			}
		}()
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

func (h *Handler) buildRequestFromQuery(httpRequest *http.Request, request *request.Request) error {
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

func (h *Handler) handleAppRequest(ctx context.Context, writer io.Writer, request *request.Request, response *Response) error {
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
func NewHandler(service *Service, pool *buffer.Pool, maxDuration time.Duration, m *gmetric.Service) *Handler {
	location := reflect.TypeOf(&Handler{}).PkgPath()
	return &Handler{
		service:     service,
		pool:        pool,
		maxDuration: maxDuration,
		poolMetrics: m.OperationCounter(location, service.config.ID+"HttpPool", service.config.ID+" HTTP []byte Pool", time.Microsecond, time.Minute, 2),
	}
}
