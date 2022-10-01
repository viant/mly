package client

import (
	"context"
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
	"path"
	"reflect"
	"testing"
	"time"
)

type TestOutput struct {
	Prediction float32
}

func (t *TestOutput) EncodeBinary(stream *bintly.Writer) error {
	stream.Float32(t.Prediction)
	return nil
}

func (t *TestOutput) DecodeBinary(stream *bintly.Reader) error {
	stream.Float32(&t.Prediction)
	return nil
}

func TestService_Run(t *testing.T) {
	baseURL := toolbox.CallerDirectory(3)

	server := faker.Server{URL: path.Join(baseURL, "testdata"), Port: 8087, Debug: true}
	go server.Start()
	time.Sleep(time.Second)
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

	var testCases = []struct {
		description string
		model       string
		options     []Option
		initMessage func(msg *Message)
		response    func() *Response
		expect      interface{}
	}{
		{
			description: "single prediction",
			model:       "case001",
			options: []Option{
				WithRemoteConfig(&cconfig.Remote{
					Datastore: config.Datastore{
						Cache: &scache.Config{SizeMb: 64, Shards: 10, EntrySize: 1024},
					},
					MetaInput: metaInput,
				}),
				WithCacheScope(CacheScopeLocal),
				WithDictionary(dictionary),
				WithDataStorer(mock.New()),
			},
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
			options: []Option{
				WithRemoteConfig(&cconfig.Remote{
					Datastore: config.Datastore{
						Cache: &scache.Config{SizeMb: 64, Shards: 10, EntrySize: 1024},
					},
					MetaInput: metaInput,
				}),
				WithCacheScope(CacheScopeLocal),
				WithDictionary(dictionary),
				WithDataStorer(mock.New()),
			},
			response: func() *Response {
				predictions := []*TestOutput{}
				return &Response{Data: &predictions}
			},
			initMessage: func(msg *Message) {
				msg.StringsKey("i1", []string{"v1", "v2", "v4"})
				msg.StringsKey("i2", []string{"v10", "v10", "v10"})
			},
			expect: []*TestOutput{
				{
					Prediction: 3.2,
				},
				{
					Prediction: 4.2,
				},
				{
					Prediction: 7.6,
				},
			},
		},
	}

	for _, testCase := range testCases {
		srv, err := New(testCase.model, testHosts(), testCase.options...)

		for i := 0; i < 2; i++ {
			if !assert.Nil(t, err, testCase.description) {
				continue
			}
			response := testCase.response()
			msg := srv.NewMessage()
			testCase.initMessage(msg)

			err = srv.Run(context.Background(), msg, response)
			if !assert.Nil(t, err, testCase.description) {
				continue
			}
			actual := reflect.ValueOf(response.Data).Elem().Interface()
			assert.EqualValues(t, testCase.expect, actual, testCase.description)
			expectStatus := common.StatusOK
			if i == 1 {
				expectStatus = common.StatusCached
			}
			assert.EqualValues(t, expectStatus, response.Status, testCase.description)
			time.Sleep(300 * time.Microsecond)
		}

	}

}

func testHosts() []*Host {
	return []*Host{
		{
			Name: "localhost",
			Port: 8087,
		},
	}
}
