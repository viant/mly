package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/gmetric"
	"github.com/viant/mly/shared/stat"
)

func TestHTTPPost_ReturnsCompletedBodyWhenContextIsCanceled(t *testing.T) {
	cases := []struct {
		name string
		body []byte
	}{
		{name: "json", body: []byte(`{"scores":[1]}`)},
		{name: "empty", body: []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			rt := &scriptedRoundTripper{fn: func(int) (*http.Response, error) {
				return responseWith(http.StatusOK, tc.body), nil
			}}
			svc := newHTTPPostService(rt, 3)

			got, err := svc.httpPost(ctx, []byte("req"), httpPostHost())

			require.NoError(t, err)
			require.Equal(t, tc.body, got)
			require.Equal(t, int32(1), rt.hits.Load())
		})
	}
}

func TestHTTPPost_Non200KeepsStatusErrorWhenContextIsCanceled(t *testing.T) {
	payload := []byte(`{"status":"error","error":"bad"}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rt := &scriptedRoundTripper{fn: func(int) (*http.Response, error) {
		return responseWith(http.StatusBadRequest, payload), nil
	}}
	svc := newHTTPPostService(rt, 3)

	got, err := svc.httpPost(ctx, []byte("req"), httpPostHost())

	require.Equal(t, payload, got)
	require.ErrorContains(t, err, "HTTP Code:400")
	require.NotErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(1), rt.hits.Load())
}

func TestHTTPPost_RetriesWhileContextIsActive(t *testing.T) {
	payload := []byte(`{"scores":[2]}`)
	rt := &scriptedRoundTripper{fn: func(hit int) (*http.Response, error) {
		if hit == 1 {
			return nil, errors.New("temporary")
		}
		return responseWith(http.StatusOK, payload), nil
	}}
	svc := newHTTPPostService(rt, 3)

	got, err := svc.httpPost(context.Background(), []byte("req"), httpPostHost())

	require.NoError(t, err)
	require.Equal(t, payload, got)
	require.Equal(t, int32(2), rt.hits.Load())
}

func TestHTTPPost_FailedAttemptStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rt := &scriptedRoundTripper{fn: func(int) (*http.Response, error) {
		return nil, errors.New("temporary")
	}}
	svc := newHTTPPostService(rt, 3)

	got, err := svc.httpPost(ctx, []byte("req"), httpPostHost())

	require.Nil(t, got)
	require.ErrorContains(t, err, "temporary")
	require.NotErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(1), rt.hits.Load())
}

func TestHTTPPost_PartialReadStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rt := &scriptedRoundTripper{fn: func(int) (*http.Response, error) {
		resp := responseWith(http.StatusOK, nil)
		resp.Body = errorReadCloser{err: io.ErrUnexpectedEOF}
		return resp, nil
	}}
	svc := newHTTPPostService(rt, 3)

	got, err := svc.httpPost(ctx, []byte("req"), httpPostHost())

	require.Nil(t, got)
	require.ErrorContains(t, err, "partial body read")
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.Equal(t, int32(1), rt.hits.Load())
}

func TestHTTPPost_PartialReadRetriesWhileContextIsActive(t *testing.T) {
	payload := []byte(`{"scores":[3]}`)
	rt := &scriptedRoundTripper{fn: func(hit int) (*http.Response, error) {
		if hit == 1 {
			resp := responseWith(http.StatusOK, nil)
			resp.Body = errorReadCloser{err: io.ErrUnexpectedEOF}
			return resp, nil
		}
		return responseWith(http.StatusOK, payload), nil
	}}
	svc := newHTTPPostService(rt, 3)

	got, err := svc.httpPost(context.Background(), []byte("req"), httpPostHost())

	require.NoError(t, err)
	require.Equal(t, payload, got)
	require.Equal(t, int32(2), rt.hits.Load())
}

type scriptedRoundTripper struct {
	hits atomic.Int32
	fn   func(hit int) (*http.Response, error)
}

func (r *scriptedRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return r.fn(int(r.hits.Add(1)))
}

type errorReadCloser struct {
	err error
}

func (e errorReadCloser) Read([]byte) (int, error) { return 0, e.err }
func (errorReadCloser) Close() error               { return nil }

func responseWith(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

func newHTTPPostService(rt http.RoundTripper, maxRetry int) *Service {
	metrics := gmetric.New()
	return &Service{
		Config:     Config{Model: "widget", MaxRetry: maxRetry},
		httpClient: http.Client{Transport: rt},
		httpCliCounter: metrics.MultiOperationCounter(
			reflect.TypeOf(Service{}).PkgPath(),
			"widgetClientHTTPCli",
			"httpPost test",
			time.Microsecond,
			time.Minute,
			2,
			stat.NewCtxErrOnly(),
		),
	}
}

func httpPostHost() *Host {
	return &Host{prefix: "http://example.test:80"}
}
