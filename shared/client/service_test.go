package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/bintly"
	"github.com/viant/gmetric"
	"github.com/viant/mly/shared"
	cconfig "github.com/viant/mly/shared/client/config"
	"github.com/viant/mly/shared/client/faker"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore/mock"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/scache"
	"github.com/viant/toolbox"
)

type TestOutput struct {
	Prediction float32
}

// caching
func (t *TestOutput) EncodeBinary(stream *bintly.Writer) error {
	stream.Float32(t.Prediction)
	return nil
}

// caching
func (t *TestOutput) DecodeBinary(stream *bintly.Reader) error {
	stream.Float32(&t.Prediction)
	return nil
}

func TestService_Run(t *testing.T) {
	baseURL := toolbox.CallerDirectory(3)

	selectPort := 8087
	server := faker.Server{URL: path.Join(baseURL, "testdata"), Port: selectPort, Debug: true}
	server.Start()

	defer server.Stop()

	var metaInput = shared.MetaInput{
		Inputs: []*shared.Field{
			{
				Name: "i1",
			},
			{
				Name:     "i2",
				Wildcard: true,
			},
		},
	}

	var dictionary = NewDictionary(&common.Dictionary{
		Layers: []common.Layer{
			{
				Name: "i1",
				Strings: []string{
					"v1", "v2",
				},
			},
		},
		Hash: 123,
	}, metaInput.Inputs)

	hosts := []*Host{
		NewHost("localhost", selectPort),
	}

	makeBasicOptions := func() []Option {
		return []Option{
			WithRemoteConfig(&cconfig.Remote{
				Datastore: config.Datastore{
					Cache: &scache.Config{SizeMb: 64, Shards: 10, EntrySize: 1024},
				},
				MetaInput: metaInput,
			}),
			WithCacheScope(CacheScopeLocal),
			WithDictionary(dictionary),
			WithDataStorer(mock.New()),
			WithDebug(true),
		}
	}

	var testCases = []struct {
		description string
		model       string
		options     []Option
		prepHandler func(*faker.Handler, int)
		initMessage func(msg *Message)
		response    func() *Response
		expect      interface{}
		err         bool
		noCache     bool
		contextFn   func(context.Context) (context.Context, func())
	}{
		{
			description: "single prediction",
			model:       "case001",
			options:     makeBasicOptions(),
			response: func() *Response {
				return &Response{Data: &TestOutput{}}
			},
			initMessage: func(msg *Message) {
				msg.StringKey("i1", "v1")
				msg.StringKey("i2", "v10")

			},
			expect: TestOutput{Prediction: 3.2},
		},
		{
			description: "multi prediction",
			model:       "case002",
			options:     makeBasicOptions(),
			response: func() *Response {
				predictions := []*TestOutput{}
				return &Response{Data: &predictions}
			},
			initMessage: func(msg *Message) {
				msg.StringsKey("i1", []string{"v1", "v2", "v4"})
				msg.StringsKey("i2", []string{"v10", "v10", "v10"})
			},
			expect: []*TestOutput{
				{Prediction: 3.2},
				{Prediction: 4.2},
				{Prediction: 7.6},
			},
		},
		{
			description: "404",
			model:       "case003",
			options:     makeBasicOptions(),
			response: func() *Response {
				predictions := []*TestOutput{}
				return &Response{Data: &predictions}
			},
			initMessage: func(msg *Message) {},
			err:         true,
		},
		{
			description: "400",
			prepHandler: func(h *faker.Handler, iter int) {
				h.Then(func(d []byte, w http.ResponseWriter) {
					http.Error(w, "bad request", http.StatusBadRequest)
				})
			},
			options: makeBasicOptions(),
			response: func() *Response {
				predictions := []*TestOutput{}
				return &Response{Data: &predictions}
			},
			initMessage: func(msg *Message) {},
			err:         true,
		},
		{
			description: "500",
			prepHandler: func(h *faker.Handler, iter int) {
				h.Then(func(d []byte, w http.ResponseWriter) {
					http.Error(w, "server error", http.StatusInternalServerError)
				})
			},
			options: makeBasicOptions(),
			response: func() *Response {
				predictions := []*TestOutput{}
				return &Response{Data: &predictions}
			},
			initMessage: func(msg *Message) {},
			err:         true,
		},
	}

	for _, testCase := range testCases {
		srv, err := New(testCase.model, hosts, testCase.options...)
		if !assert.Nil(t, err, testCase.description) {
			return
		}

		for i := 0; i < 2; i++ {
			if testCase.prepHandler != nil {
				testCase.prepHandler(server.Handler, i)
			}

			func() {
				caseDesc := fmt.Sprintf("%s model:%s %d", testCase.description, testCase.model, i)

				msg := srv.NewMessage()
				testCase.initMessage(msg)

				msgs := msg.Strings()
				fmt.Printf("Message:%v\n", msgs)

				ctx := context.Background()
				if testCase.contextFn != nil {
					var dfn func()
					ctx, dfn = testCase.contextFn(ctx)
					defer dfn()
				}

				response := testCase.response()
				err = srv.Run(ctx, msg, response)
				if testCase.err {
					assert.NotNil(t, err, caseDesc)
					return
				}

				if !assert.Nil(t, err, caseDesc) {
					return
				}

				fmt.Printf("response.Data:%+V\n", response.Data)

				// unwrap pointer
				actual := reflect.ValueOf(response.Data).Elem().Interface()
				assert.EqualValues(t, testCase.expect, actual, caseDesc)

				expectStatus := common.StatusOK
				if !testCase.noCache && i == 1 {
					expectStatus = common.StatusCached
				}
				assert.EqualValues(t, expectStatus, response.Status, fmt.Sprintf("%s - status", caseDesc))
			}()
		}
	}
}

// TestService_Run_ShedIncrementsBreakerShedMetric verifies that when the
// host's circuit breaker is in the down state at request time, Run()
// increments the new ClientHTTP_shed marker on the http counter (and does
// NOT increment _down, which is reserved for the trip event itself).
//
// Before this fix, shed requests were conflated into the generic _error
// counter, leaving operators unable to distinguish "request rejected
// pre-flight by the breaker" from "request reached httpPost and failed
// there." See shared/client/service.go postRequest.
func TestService_Run_ShedIncrementsBreakerShedMetric(t *testing.T) {
	baseURL := toolbox.CallerDirectory(3)

	selectPort := 8089
	server := faker.Server{URL: path.Join(baseURL, "testdata"), Port: selectPort, Debug: true}
	server.Start()
	defer server.Stop()

	metaInput := shared.MetaInput{
		Inputs: []*shared.Field{
			{Name: "i1"},
			{Name: "i2", Wildcard: true},
		},
	}
	dictionary := NewDictionary(&common.Dictionary{
		Layers: []common.Layer{{Name: "i1", Strings: []string{"v1", "v2"}}},
		Hash:   123,
	}, metaInput.Inputs)
	hosts := []*Host{NewHost("localhost", selectPort)}

	gmetrics := gmetric.New()
	const modelID = "shed_metric_case"
	options := []Option{
		WithGmetrics(gmetrics),
		WithRemoteConfig(&cconfig.Remote{
			Datastore: config.Datastore{
				Cache: &scache.Config{SizeMb: 64, Shards: 10, EntrySize: 1024},
			},
			MetaInput: metaInput,
		}),
		WithCacheScope(CacheScopeLocal),
		WithDictionary(dictionary),
		WithDataStorer(mock.New()),
		WithDebug(true),
	}
	srv, err := New(modelID, hosts, options...)
	require.NoError(t, err)

	// Force the host's breaker into the down state so getHost() will
	// return ErrNodeDown without ever calling httpPost.
	hosts[0].FlagDown()
	require.False(t, hosts[0].IsUp(), "host must be flagged down for the shed path")

	msg := srv.NewMessage()
	msg.StringKey("i1", "v1")
	msg.StringKey("i2", "v10")

	response := &Response{Data: &TestOutput{}}
	err = srv.Run(context.Background(), msg, response)

	require.Error(t, err, "shed request must surface as a non-nil err")
	assert.True(t, errors.Is(err, common.ErrNodeDown), "shed err must wrap ErrNodeDown, got %v", err)

	// Inspect the cumulative counter values for <model>ClientHTTP. The
	// new Shed marker must increment by 1; the existing Down marker must
	// stay at 0 (no FlagDown was called by this request -- the breaker
	// was already down before getHost was called).
	shedCount := gmetrics.LookupOperationCumulativeMetric(modelID+"ClientHTTP", stat.Shed)
	downCount := gmetrics.LookupOperationCumulativeMetric(modelID+"ClientHTTP", stat.Down)
	errorCount := gmetrics.LookupOperationCumulativeMetric(modelID+"ClientHTTP", stat.ErrorKey)

	assert.EqualValues(t, 1, shedCount, "ClientHTTP_shed must increment on shed")
	assert.EqualValues(t, 0, downCount, "ClientHTTP_down must NOT increment on shed (only on trip)")
	// _error still increments because Run()'s AppendError fires for the
	// non-context ErrNodeDown -- this is the historical behavior that
	// the new _shed marker disambiguates without changing.
	assert.EqualValues(t, 1, errorCount, "ClientHTTP_error continues to increment as before")
}

// TestService_Run_ParsesErrorBody verifies that when the server returns
// a non-2xx response with a JSON-encoded Response body (the v0.20.0+
// error-response contract), Run() does a best-effort unmarshal of the
// body so the caller's response struct has Status="error" and Error
// populated -- in addition to receiving a non-nil err return value.
//
// Backward-compatibility: when the server returns a plain-text body
// (older mly versions, or any non-JSON body), the unmarshal silently
// fails and the response struct stays untouched. The non-nil err
// return remains the source-of-truth signal in either case.
func TestService_Run_ParsesErrorBody(t *testing.T) {
	baseURL := toolbox.CallerDirectory(3)

	selectPort := 8088
	server := faker.Server{URL: path.Join(baseURL, "testdata"), Port: selectPort, Debug: true}
	server.Start()
	defer server.Stop()

	metaInput := shared.MetaInput{
		Inputs: []*shared.Field{
			{Name: "i1"},
			{Name: "i2", Wildcard: true},
		},
	}
	dictionary := NewDictionary(&common.Dictionary{
		Layers: []common.Layer{{Name: "i1", Strings: []string{"v1", "v2"}}},
		Hash:   123,
	}, metaInput.Inputs)
	hosts := []*Host{NewHost("localhost", selectPort)}
	options := []Option{
		WithRemoteConfig(&cconfig.Remote{
			Datastore: config.Datastore{
				Cache: &scache.Config{SizeMb: 64, Shards: 10, EntrySize: 1024},
			},
			MetaInput: metaInput,
		}),
		WithCacheScope(CacheScopeLocal),
		WithDictionary(dictionary),
		WithDataStorer(mock.New()),
		WithDebug(true),
	}

	cases := []struct {
		description     string
		bodyContentType string
		body            string
		statusCode      int
		expectErrorMsg  string // non-empty if response.Error should be populated
		expectStatus    string // non-empty if response.Status should be populated
	}{
		{
			description:     "v0.20.0 server: 400 JSON error body populates response.Error",
			bodyContentType: "application/json",
			body:            `{"status":"error","error":"invalid input shape","serviceTimeMcs":150}`,
			statusCode:      http.StatusBadRequest,
			expectErrorMsg:  "invalid input shape",
			expectStatus:    common.StatusError,
		},
		{
			description:     "v0.20.0 server: 500 JSON error body populates response.Error",
			bodyContentType: "application/json",
			body:            `{"status":"error","error":"upstream blew up","serviceTimeMcs":2200}`,
			statusCode:      http.StatusInternalServerError,
			expectErrorMsg:  "upstream blew up",
			expectStatus:    common.StatusError,
		},
		{
			description:     "older server: 400 plain-text body leaves response.Error empty",
			bodyContentType: "text/plain",
			body:            "bad request\n",
			statusCode:      http.StatusBadRequest,
			expectErrorMsg:  "",
			expectStatus:    "", // gojay.Unmarshal silently fails on non-JSON; struct untouched
		},
		{
			description:     "older server: 500 plain-text body leaves response.Error empty",
			bodyContentType: "text/plain",
			body:            "server error\n",
			statusCode:      http.StatusInternalServerError,
			expectErrorMsg:  "",
			expectStatus:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			body := tc.body
			contentType := tc.bodyContentType
			statusCode := tc.statusCode
			server.Handler.Then(func(d []byte, w http.ResponseWriter) {
				w.Header().Set("Content-Type", contentType)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
				w.WriteHeader(statusCode)
				_, _ = w.Write([]byte(body))
			})

			srv, err := New("error_body_case", hosts, options...)
			require.NoError(t, err)

			msg := srv.NewMessage()
			msg.StringKey("i1", "v1")
			msg.StringKey("i2", "v10")

			response := &Response{Data: &TestOutput{}}
			err = srv.Run(context.Background(), msg, response)

			assert.Error(t, err, "non-2xx must always surface as a non-nil err return")
			assert.Equal(t, tc.expectErrorMsg, response.Error,
				"response.Error population (best-effort JSON unmarshal of error body)")
			assert.Equal(t, tc.expectStatus, response.Status,
				"response.Status population (best-effort JSON unmarshal of error body)")
		})
	}
}
