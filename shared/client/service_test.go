package client

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/bintly"
	"github.com/viant/mly/shared"
	cconfig "github.com/viant/mly/shared/client/config"
	"github.com/viant/mly/shared/client/faker"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/datastore/mock"
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
