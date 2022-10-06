package service_test

import (
	"context"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs"
	"github.com/viant/bintly"
	"github.com/viant/gtly"
	"github.com/viant/mly/service"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/domain/transformer"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/datastore"
	"github.com/viant/mly/shared/datastore/mock"
	"github.com/viant/toolbox"
	"path/filepath"
	"testing"
)

const (
	testTransformer001 = "testTransformer001"
	testTransformer002 = "testTransformer002"
)

func TestService_Do(t *testing.T) {
	transformer.Register(testTransformer001, case001Transformer)
	transformer.Register(testTransformer002, case002Transformer)

	baseURL := toolbox.CallerDirectory(3)

	var trueValue = true
	var testCases = []struct {
		description string
		config      *config.Model
		options     []service.Option
		newResponse func() *service.Response
		requestData string
		expect      interface{}
	}{
		{
			description: "single mode",
			options: []service.Option{
				service.WithDataStorer(mock.New()),
			},

			config: &config.Model{
				ID:         "1",
				UseDict:    &trueValue,
				URL:        filepath.Join(baseURL, "../example/model/sls_model"),
				OutputType: "int64",

				Transformer: testTransformer001,
				DataStore:   "local",
				MetaInput: shared.MetaInput{
					Inputs: []*shared.Field{
						{
							Name:     "sl",
							DataType: "string",
						},
						{
							Name:     "sa",
							DataType: "string",
							Wildcard: true,
						},
					},
				},
			},
			newResponse: func() *service.Response {
				response := &service.Response{
					Data: &SumOutput{},
				}
				return response
			},
			requestData: `{
  "cache_key":"a/z",
  "sl":"a",
  "sa":"z"
}`,
			expect: &SumOutput{Value: 6, Sl: "a"},
		},
		{
			description: "multi mode",
			options: []service.Option{
				service.WithDataStorer(mock.New()),
			},

			config: &config.Model{
				ID:         "1",
				UseDict:    &trueValue,
				URL:        filepath.Join(baseURL, "../example/model/vec_model"),
				OutputType: "int64",

				Transformer: testTransformer002,
				DataStore:   "local",
				MetaInput: shared.MetaInput{
					Inputs: []*shared.Field{
						{
							Name:     "sl",
							DataType: "string",
							Wildcard: true,
						},
						{
							Name:     "tv",
							DataType: "string",
							Wildcard: true,
						},
					},
				},
			},
			newResponse: func() *service.Response {
				data := []*SumOutput{}
				response := &service.Response{
					Data: data,
				}
				return response
			},
			requestData: `{
  "batch_size":2,
  "cache_key":["a/z","b/z"],
  "sl":["a","b"],
  "tv":["z", "z"]
}`,
			expect: []*SumOutput{
				{Value: 14, Sl: "a"},
				{Value: 12, Sl: "b"},
			},
		},
		{
			description: "multi mode - compressed",
			options: []service.Option{
				service.WithDataStorer(mock.New()),
			},

			config: &config.Model{
				ID:         "1",
				UseDict:    &trueValue,
				URL:        filepath.Join(baseURL, "../example/model/vec_model"),
				OutputType: "int64",

				Transformer: testTransformer002,
				DataStore:   "local",
				MetaInput: shared.MetaInput{
					Inputs: []*shared.Field{
						{
							Name:     "sl",
							DataType: "string",
							Wildcard: true,
						},
						{
							Name:     "tv",
							DataType: "string",
							Wildcard: true,
						},
					},
				},
			},
			newResponse: func() *service.Response {
				data := []*SumOutput{}
				response := &service.Response{
					Data: data,
				}
				return response
			},
			requestData: `{
  "batch_size":2,
  "cache_key":["a/z","b/z"],
  "sl":["a","b"],
  "tv":["z"]
}`,
			expect: []*SumOutput{
				{Value: 14, Sl: "a"},
				{Value: 12, Sl: "b"},
			},
		},
	}

	fs := afs.New()
	for _, testCase := range testCases {
		fmt.Printf("loading: %v\n", testCase.config.URL)
		testCase.config.Init()
		srv, err := service.New(context.TODO(), fs, testCase.config, nil, map[string]*datastore.Service{}, testCase.options...)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		request := srv.NewRequest()
		err = gojay.Unmarshal([]byte(testCase.requestData), request)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		response := testCase.newResponse()
		err = srv.Do(context.Background(), request, response)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}
		assert.EqualValues(t, testCase.expect, response.Data, testCase.description)
	}

}

type SumOutput struct {
	Value int64
	Sl    string
}

func (p *SumOutput) EncodeBinary(enc *bintly.Writer) error {
	enc.Int64(p.Value)
	enc.String(p.Sl)
	return nil
}

func (p *SumOutput) DecodeBinary(dec *bintly.Reader) error {
	dec.Int64(&p.Value)
	dec.String(&p.Sl)
	return nil
}

func (s *SumOutput) MarshalJSONObject(enc *gojay.Encoder) {
	enc.IntKey("value", int(s.Value))
	enc.StringKey("sl", s.Sl)

}

func (s *SumOutput) IsNil() bool {
	return s == nil
}

//
func (p *SumOutput) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		if err := pair("value", p.Value); err != nil {
			return err
		}
		if err := pair("sl", p.Sl); err != nil {
			return err
		}
		return nil
	}
}

func (p *SumOutput) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		switch key {
		case "value":
			p.Value = int64(toolbox.AsInt(value))
		case "sl":
			p.Sl = toolbox.AsString(value)
		}
		return nil
	})
}

func case001Transformer(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &SumOutput{}
	result.Sl = toolbox.AsString(input.Value("sl"))
	switch actual := output.(type) {
	case []int64:
		result.Value = actual[0]
	case []interface{}:
		switch raw := actual[0].(type) {
		case []int64:
			result.Value = raw[0]
		}
	case *shared.Output:
		switch val := actual.Values[0].(type) {
		case []int64:
			result.Value = val[actual.InputIndex]
		}
	default:
		return nil, fmt.Errorf("unsupproted type: %T", actual)
	}
	return result, nil
}

func case002Transformer(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &SumOutput{}
	result.Sl = toolbox.AsString(input.Value("sa"))
	switch actual := output.(type) {
	case []int64:
		result.Value = actual[0]
	case *shared.Output:
		switch val := actual.Values[0].(type) {
		case []int64:
			result.Value = val[actual.InputIndex]
		}
	default:
		return nil, fmt.Errorf("unsupproted type: %T", actual)
	}
	return result, nil
}
