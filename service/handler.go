package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/buffer"
	"github.com/viant/mly/service/clienterr"
	serrs "github.com/viant/mly/service/errors"
	"github.com/viant/mly/service/request"
	sstat "github.com/viant/mly/service/stat"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/stat"
)

// responseMarshalError signals that gojay.Marshal of the Response struct
// failed during writeResponse. The HTTP response is NOT yet committed,
// so ServeHTTP can still emit an explicit 5xx with a meaningful body.
// Surfaced as a typed error so it can be routed to its own metric bucket
// (sstat.ResponseMarshalError) and distinguished from upstream errors.
type responseMarshalError struct{ err error }

func (e *responseMarshalError) Error() string { return e.err.Error() }
func (e *responseMarshalError) Unwrap() error { return e.err }

// responseCommittedError signals that the HTTP response status line and
// headers have already been flushed to the client when the wrapped error
// occurred. The caller MUST NOT attempt to send a different status code:
// net/http will drop the second WriteHeader and emit a "superfluous
// response.WriteHeader call" warning, while the client still observes the
// original (200) status. Surfaced so ServeHTTP can log + exit instead of
// trying to overwrite the status line, and so the failure can be routed
// to its own metric bucket (sstat.ResponseCommittedError).
type responseCommittedError struct{ err error }

func (e *responseCommittedError) Error() string { return e.err.Error() }
func (e *responseCommittedError) Unwrap() error { return e.err }

// Handler converts a model prediction HTTP request to its internal calls.
type Handler struct {
	maxDuration time.Duration
	service     *Service
	pool        *buffer.Pool

	overheadMetrics    *gmetric.Operation
	httpContextMetrics *gmetric.Operation

	// indicates the last time a request came in
	lastRequest time.Time
	lrLock      *sync.Mutex
	lrObserver  prometheus.Observer
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, httpRequest *http.Request) {
	// use Background() since there are things to be done regardless of if the request is cancceled from the client side.
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, h.maxDuration)
	defer cancel()

	h.trackIdle()

	handlerOnDone := h.httpContextMetrics.Begin(time.Now())
	hStats := stat.NewValues()
	defer func() { handlerOnDone(time.Now(), hStats.Values()...) }()

	isDebug := h.service.config.Debug

	// TODO this context isn't guaranteed to leak - the model can change in the
	// middle of a request
	var request *request.Request

	response := &Response{Status: common.StatusOK, started: time.Now()}
	if httpRequest.Method == http.MethodGet {
		request = h.service.NewRequest()
		if err := h.buildRequestFromQuery(httpRequest, request); err != nil {
			h.writeError(writer, response, hStats, http.StatusBadRequest, err)
			return
		}
	} else {
		defer httpRequest.Body.Close()

		onDone := h.overheadMetrics.Begin(time.Now())
		stats := stat.NewValues()
		data, size, err := buffer.Read(h.pool, httpRequest.Body)
		defer h.pool.Put(data)
		err = func() error {
			defer func() { onDone(time.Now(), stats.Values()...) }()

			if err != nil {
				stats.Append(sstat.ReadError{Error: err})
				if isDebug {
					log.Printf("[%v http] read error: %v\n", h.service.config.ID, err)
				}

				code := http.StatusInternalServerError
				if errors.Is(err, buffer.ErrBufferTooSmall) {
					code = http.StatusRequestEntityTooLarge
				}

				h.writeError(writer, response, hStats, code, err)
				return err
			}

			request = h.service.NewRequest()
			// MODIFICATION: data will be truncated, but if this caused a problem, then this function should have returned an error already.
			request.Body = data[:size]
			if isDebug {
				trimmed := strings.Trim(string(request.Body), " \n\r")
				log.Printf("[%v http] input: %s\n", h.service.config.ID, trimmed)
			}

			err = gojay.Unmarshal(data[:size], request)
			if err != nil {
				werr := fmt.Errorf("unmarshal error: %w data: %s", err, string(data[:size]))
				stats.Append(sstat.UnmarshalError{Error: werr})

				if isDebug {
					log.Printf("[%v http] unmarshal error: %v\n", h.service.config.ID, err)
				}

				displayErr := fmt.Errorf("%s (are your input types correct?)", err.Error())
				h.writeError(writer, response, hStats, http.StatusBadRequest, displayErr)
				return err
			}

			return nil
		}()

		if err != nil {
			return
		}
	}

	if request == nil {
		// This isn't a particularly helpful message.
		// Currently, the only case this handles is if the request is too large.
		h.writeError(writer, response, hStats, http.StatusBadRequest, errors.New("no request"))
		return
	}

	err := h.service.Do(ctx, request, response)
	if err != nil {
		response.SetError(err)
	} else {
		err = h.writeResponse(writer, response, http.StatusOK)
	}

	if isDebug {
		data, merr := json.Marshal(response.Data)

		if merr == nil {
			log.Printf("[%v http] output:%s", h.service.config.ID, data)
		} else {
			log.Printf("[%v http] marshal error:%v data:%s", h.service.config.ID, merr, response.Data)
		}
	}

	reqCtx := httpRequest.Context()
	if reqCtx != nil && reqCtx.Err() != nil {
		hStats.AppendError(reqCtx.Err())
	}

	if err != nil {
		// If the response was already committed (status + headers flushed),
		// the wire status code is fixed at 200 and cannot be changed. Calling
		// writeError here would log "superfluous WriteHeader" and silently
		// drop the new status — the client still sees 200 + truncated body.
		// Log unconditionally so this defect is visible in production, and
		// emit a dedicated metric so it can be alerted independently of
		// the generic ErrorKey bucket.
		var committed *responseCommittedError
		if errors.As(err, &committed) {
			hStats.Append(sstat.ResponseCommittedError{Error: err})
			log.Printf("[%v http] response committed but write failed: %v", h.service.config.ID, err)
			return
		}

		// Marshal failure: response NOT committed; we will emit an explicit
		// 5xx below. Track it in its own metric bucket so the operator can
		// distinguish "we never sent anything" from "we sent something we
		// shouldn't have". writeError clears response.Data before retrying,
		// so the second marshal cannot fail for the same reason.
		var marshal *responseMarshalError
		if errors.As(err, &marshal) {
			hStats.Append(sstat.ResponseMarshalError{Error: err})
		}

		var status int
		if _, ok := err.(*clienterr.ClientError); ok {
			status = http.StatusBadRequest
		} else if errors.Is(err, serrs.OverloadedError) {
			status = http.StatusTooManyRequests
		} else {
			status = http.StatusInternalServerError
		}

		if isDebug {
			log.Printf("[%v http] status:%d error:%v", h.service.config.ID, status, err)
		}

		h.writeError(writer, response, hStats, status, err)
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

// writeResponse marshals appResponse and emits it with explicit-commit
// semantics:
//
//   - Marshal first; on failure return a typed responseMarshalError -- the
//     response is NOT yet committed and the caller can still set a different
//     status (typically a 5xx).
//   - Set Content-Length explicitly so a truncated body is detectable on
//     the client side as io.ErrUnexpectedEOF (without it, the client cannot
//     distinguish "done" from "connection broke mid-body" on a 200 OK).
//   - Call WriteHeader(status) explicitly so the status line is committed
//     in a known order, not as a side effect of the first Write.
//   - On Write failure return responseCommittedError so the caller knows
//     the status code can no longer be changed.
//
// status is typically http.StatusOK for success responses; the writeError
// helper passes the appropriate 4xx/5xx for error responses so the wire
// shape is uniform across success and failure paths.
//
// This addresses the silent "200 OK + empty body" failure mode where a
// canceled connection caused the implicit auto-200 from Write to flush
// headers while the body bytes were lost.
func (h *Handler) writeResponse(writer http.ResponseWriter, appResponse *Response, status int) error {
	appResponse.ServiceTimeMcs = int(time.Since(appResponse.started).Microseconds())

	data, err := gojay.Marshal(appResponse)
	if err != nil {
		return &responseMarshalError{err: fmt.Errorf("marshal response: %w", err)}
	}

	if h.service.config.Debug {
		log.Printf("[%v write] output:%s", h.service.config.ID, data)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	writer.WriteHeader(status)

	if _, err := writer.Write(data); err != nil {
		return &responseCommittedError{err: fmt.Errorf("write response body: %w", err)}
	}

	return nil
}

// writeError emits an error response with the given HTTP status code as
// a JSON-encoded Response object (status="error", populated error
// message). It is the error-path counterpart to writeResponse and shares
// the same explicit-commit contract so clients always see Content-Length
// and a parseable JSON body regardless of success or failure.
//
// Side-effects on the response struct:
//
//   - response.SetError(err) populates response.Error and sets
//     response.Status = "error".
//   - response.Data is cleared. This guarantees the marshal will succeed
//     regardless of the prior state of Data, which matters when the
//     original failure was itself a marshal error on a populated Data
//     value.
//
// On a post-commit write failure (responseCommittedError) the status is
// already on the wire; we only log + emit the dedicated metric.
//
// On a marshal failure of the (cleared) error response (essentially
// impossible -- the struct now contains only string + int fields), we
// fall back to http.Error so the client at least receives a status code.
func (h *Handler) writeError(writer http.ResponseWriter, response *Response, hStats *stat.Values, status int, err error) {
	response.SetError(err)
	response.Data = nil

	werr := h.writeResponse(writer, response, status)
	if werr == nil {
		return
	}

	var committed *responseCommittedError
	if errors.As(werr, &committed) {
		hStats.Append(sstat.ResponseCommittedError{Error: werr})
		log.Printf("[%v http] error response committed but write failed: %v (original error: %v)", h.service.config.ID, werr, err)
		return
	}

	var marshal *responseMarshalError
	if errors.As(werr, &marshal) {
		hStats.Append(sstat.ResponseMarshalError{Error: werr})
	}
	log.Printf("[%v http] failed to write error response: %v (original error: %v)", h.service.config.ID, werr, err)
	http.Error(writer, err.Error(), status)
}

func (h *Handler) trackIdle() {
	now := time.Now()
	h.lrLock.Lock()
	lr := h.lastRequest
	h.lastRequest = now
	h.lrLock.Unlock()
	elapsed := now.Sub(lr)
	h.lrObserver.Observe(float64(elapsed))
}

// NewHandler creates a new HTTP service Handler
func NewHandler(service *Service, pool *buffer.Pool, maxDuration time.Duration,
	m *gmetric.Service, lrOV prometheus.ObserverVec) *Handler {

	location := reflect.TypeOf(Handler{}).PkgPath()
	modelID := service.config.ID
	return &Handler{
		service:     service,
		pool:        pool,
		maxDuration: maxDuration,

		lastRequest: time.Now(),
		lrLock:      new(sync.Mutex),
		lrObserver:  lrOV.With(prometheus.Labels{"model": modelID}),

		overheadMetrics:    m.MultiOperationCounter(location, modelID+"HTTPOverhead", modelID+" server HTTP startup overhead", time.Microsecond, time.Minute, 2, sstat.NewHttp()),
		httpContextMetrics: m.MultiOperationCounter(location, modelID+"HTTPHandler", modelID+" server HTTP handler", time.Microsecond, time.Minute, 2, sstat.NewHandler()),
	}
}
