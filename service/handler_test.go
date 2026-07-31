package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/mly/service/config"
	sstat "github.com/viant/mly/service/stat"
	"github.com/viant/mly/shared/stat"
)

// writeFailingResponseWriter wraps httptest.ResponseRecorder so that
// Write returns a configurable error after a configurable number of
// bytes. Used to simulate a broken-pipe condition where the client has
// already closed the connection (e.g. the bidder's 40 ms client timeout
// fired while MLY was mid-response). httptest.ResponseRecorder by itself
// never errors on Write.
//
// failAfter == 0 → the very first Write call errors after writing zero
// body bytes (the headers have still been committed at that point by
// writer.WriteHeader, which is the precise condition that produced the
// observed 200-OK-with-empty-body wire trace).
type writeFailingResponseWriter struct {
	*httptest.ResponseRecorder
	failAfter int // bytes written successfully before Write starts erroring
	written   int // total successful body bytes so far
	failErr   error
}

func newWriteFailingResponseWriter(failAfter int) *writeFailingResponseWriter {
	return &writeFailingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		failAfter:        failAfter,
		failErr:          errors.New("simulated broken pipe"),
	}
}

func (w *writeFailingResponseWriter) Write(p []byte) (int, error) {
	remaining := w.failAfter - w.written
	if remaining <= 0 {
		return 0, w.failErr
	}
	if len(p) <= remaining {
		n, err := w.ResponseRecorder.Write(p)
		w.written += n
		return n, err
	}
	n, _ := w.ResponseRecorder.Write(p[:remaining])
	w.written += n
	return n, w.failErr
}

// newTestHandler constructs a Handler with the minimum scaffolding
// required to exercise writeResponse. The metric Operations are left
// nil — writeResponse does not touch them, only ServeHTTP does. Tests
// that exercise ServeHTTP need a different fixture (not provided here
// because Service.Do depends on a fully wired tfmodel.Service which is
// out of scope for a unit test).
func newTestHandler(modelID string, debug bool) *Handler {
	return &Handler{
		service: &Service{
			config: &config.Model{ID: modelID, Debug: debug},
		},
	}
}

// TestWriteResponse_Success verifies the happy path: a fully populated
// Response is marshaled, headers are set explicitly (Content-Type,
// Content-Length), the status is committed at 200, and the body bytes
// match the marshaled JSON. This locks in the explicit-commit contract
// that lets clients detect truncation via Content-Length mismatch.
func TestWriteResponse_Success(t *testing.T) {
	h := newTestHandler("test", false)
	resp := &Response{
		Status:   "ok",
		DictHash: 42,
		started:  time.Now().Add(-time.Millisecond),
	}

	rec := httptest.NewRecorder()
	err := h.writeResponse(rec, resp, http.StatusOK)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"),
		"Content-Type must be set explicitly")

	cl, atoiErr := strconv.Atoi(rec.Header().Get("Content-Length"))
	require.NoError(t, atoiErr, "Content-Length must be a parseable integer")
	assert.Equal(t, rec.Body.Len(), cl,
		"Content-Length must match actual body length so clients can detect truncation")

	body := rec.Body.String()
	assert.Contains(t, body, `"status":"ok"`)
	assert.Contains(t, body, `"dictHash":42`)
	assert.Contains(t, body, `"serviceTimeMcs":`,
		"serviceTimeMcs must be present so the bidder can record mly_eval_duration_us")
}

// TestWriteResponse_WriteFailureReturnsCommittedError simulates the
// failure mode that drives the bidder's invalid_json class on the wire:
// the body Write fails (broken pipe) AFTER WriteHeader has already
// committed the 200 status line. The post-condition is that:
//   - writeResponse returns *responseCommittedError so the caller knows
//     the status code can no longer be changed,
//   - the status code on the wire is the originally-committed 200 (NOT
//     the 500 we would otherwise want to send),
//   - Content-Length was set, so a downstream client correctly checking
//     it would observe an early-EOF / unexpected-EOF condition rather
//     than silently treating the empty body as a valid response.
func TestWriteResponse_WriteFailureReturnsCommittedError(t *testing.T) {
	h := newTestHandler("test", false)
	resp := &Response{Status: "ok", started: time.Now()}

	rec := newWriteFailingResponseWriter(0)
	err := h.writeResponse(rec, resp, http.StatusOK)

	require.Error(t, err)

	var committed *responseCommittedError
	require.True(t, errors.As(err, &committed),
		"expected *responseCommittedError, got %T: %v", err, err)
	assert.ErrorIs(t, err, rec.failErr,
		"wrapped error chain must reach the underlying broken-pipe error")

	assert.Equal(t, http.StatusOK, rec.Code,
		"status was committed before Write failed; explicit-commit contract")
	assert.NotEmpty(t, rec.Header().Get("Content-Length"),
		"Content-Length must be set BEFORE Write so client can detect truncation")
	assert.Equal(t, 0, rec.Body.Len(),
		"no body bytes should have been written on the failAfter=0 case")
}

// TestWriteResponse_PartialWriteReturnsCommittedError covers the
// truncated-body case: the headers + status flush, then a few body
// bytes succeed, then the connection breaks. The committed-error type
// must still surface so ServeHTTP's error branch knows not to call
// http.Error (which would emit "superfluous WriteHeader" log noise).
func TestWriteResponse_PartialWriteReturnsCommittedError(t *testing.T) {
	h := newTestHandler("test", false)
	resp := &Response{Status: "ok", started: time.Now()}

	rec := newWriteFailingResponseWriter(5)
	err := h.writeResponse(rec, resp, http.StatusOK)

	require.Error(t, err)
	var committed *responseCommittedError
	require.True(t, errors.As(err, &committed),
		"partial write must also yield *responseCommittedError")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 5, rec.Body.Len(),
		"exactly 5 body bytes should have been flushed before failure")
}

// TestWriteResponse_HasContentLengthMatchingBody locks in the invariant
// that Content-Length declared in the header equals the bytes the
// handler intends to write. Without this, a client cannot distinguish
// "done" from "connection broke mid-body" on a 200 OK response — which
// is the root mechanism that allowed the bidder-side io.ReadAll swallow
// (shared/client/service.go) to silently produce empty-body
// invalid_json events.
func TestWriteResponse_HasContentLengthMatchingBody(t *testing.T) {
	h := newTestHandler("test", false)

	cases := []struct {
		name string
		resp *Response
	}{
		{"empty", &Response{started: time.Now()}},
		{"with-status", &Response{Status: "ok", started: time.Now()}},
		{"with-error", &Response{Status: "error", Error: "something failed", started: time.Now()}},
		{"with-dict-hash", &Response{Status: "ok", DictHash: 0xdeadbeef, started: time.Now()}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			require.NoError(t, h.writeResponse(rec, tc.resp, http.StatusOK))

			cl, atoiErr := strconv.Atoi(rec.Header().Get("Content-Length"))
			require.NoError(t, atoiErr)
			assert.Equal(t, rec.Body.Len(), cl,
				"declared Content-Length must equal actual body bytes")
		})
	}
}

// TestResponseCommittedError_Unwrap verifies that the typed error
// participates correctly in errors.Is / errors.As chains. ServeHTTP
// relies on errors.As to detect committed-error from arbitrarily-deep
// wrappings.
func TestResponseCommittedError_Unwrap(t *testing.T) {
	inner := errors.New("underlying broken pipe")
	wrapped := &responseCommittedError{err: inner}

	var target *responseCommittedError
	assert.True(t, errors.As(wrapped, &target))
	assert.True(t, errors.Is(wrapped, inner),
		"errors.Is must traverse Unwrap to the underlying cause")
}

// TestResponseMarshalError_Unwrap is the symmetric assertion for the
// marshal-failure sentinel, used by ServeHTTP to route marshal failures
// to their own metric bucket while still emitting an HTTP 5xx response.
func TestResponseMarshalError_Unwrap(t *testing.T) {
	inner := errors.New("malformed Response struct")
	wrapped := &responseMarshalError{err: inner}

	var target *responseMarshalError
	assert.True(t, errors.As(wrapped, &target))
	assert.True(t, errors.Is(wrapped, inner))
}

// TestWriteResponse_HonorsStatusParam verifies that writeResponse
// commits the supplied status code rather than always 200. This is the
// foundation for the unified error-response wire format: writeError
// uses writeResponse with 4xx/5xx so success and error responses share
// shape (Content-Type, Content-Length, JSON body) and only differ in
// status line + populated fields.
func TestWriteResponse_HonorsStatusParam(t *testing.T) {
	h := newTestHandler("test", false)
	cases := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusRequestEntityTooLarge,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	}
	for _, status := range cases {
		t.Run(http.StatusText(status), func(t *testing.T) {
			resp := &Response{Status: "ok", started: time.Now()}
			rec := httptest.NewRecorder()
			require.NoError(t, h.writeResponse(rec, resp, status))
			assert.Equal(t, status, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			assert.NotEmpty(t, rec.Header().Get("Content-Length"))
		})
	}
}

// TestWriteError_EmitsJSONErrorWithStatus locks in the wire shape
// promised to clients on the error path: 4xx/5xx + JSON Response body
// with status="error", populated error message, and serviceTimeMcs.
// This is what makes a defensive consumer-side check
// (e.g. mediator's `if response.Error != "" { ... }`) actually fire on
// real predict-time errors instead of silently no-op'ing.
func TestWriteError_EmitsJSONErrorWithStatus(t *testing.T) {
	h := newTestHandler("test", false)
	resp := &Response{Status: "ok", started: time.Now(), Data: "leftover-data"}
	rec := httptest.NewRecorder()
	hStats := stat.NewValues()

	h.writeError(rec, resp, hStats, http.StatusInternalServerError, errors.New("upstream blew up"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code,
		"error status must reach the wire")
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	cl, atoiErr := strconv.Atoi(rec.Header().Get("Content-Length"))
	require.NoError(t, atoiErr)
	assert.Equal(t, rec.Body.Len(), cl)

	body := rec.Body.String()
	assert.Contains(t, body, `"status":"error"`,
		"writeError must populate response.Status as error")
	assert.Contains(t, body, `"error":"upstream blew up"`,
		"writeError must populate response.Error from the supplied error")
	assert.NotContains(t, body, "leftover-data",
		"writeError must clear response.Data so the original Data does not leak into the error body")
}

// TestWriteError_FallsBackToHTTPErrorOnCommittedFailure verifies that
// when the error response's body Write fails after status commit, the
// fallback path appends a metric and returns without panic. The status
// is already on the wire so http.Error inside the fallback is a no-op,
// but the metric attribution and log line are what matter for
// diagnosing the cliff scenario where both the success and error
// responses fail to flush.
func TestWriteError_FallsBackToHTTPErrorOnCommittedFailure(t *testing.T) {
	h := newTestHandler("test", false)
	resp := &Response{Status: "ok", started: time.Now()}
	rec := newWriteFailingResponseWriter(0)
	hStats := stat.NewValues()

	h.writeError(rec, resp, hStats, http.StatusInternalServerError, errors.New("upstream blew up"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code,
		"status was committed before the body write failed")
	assert.NotEmpty(t, hStats.Values(),
		"hStats must record the post-commit failure for metric attribution")

	var sawCommitted bool
	for _, v := range hStats.Values() {
		if _, ok := v.(sstat.ResponseCommittedError); ok {
			sawCommitted = true
			break
		}
	}
	assert.True(t, sawCommitted,
		"hStats must include a sstat.ResponseCommittedError marker")
}
