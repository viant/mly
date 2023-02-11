package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"testing"

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
	tconf "github.com/viant/tapper/config"
	"github.com/viant/toolbox"
)

const (
	floatTName = "floatTransformer"
	intTName   = "intTransformer"
)

func TestService_Do(t *testing.T) {
	transformer.Register(floatTName, floatTransformer)
	transformer.Register(intTName, intTransformer)

	baseURL := toolbox.CallerDirectory(3)

	var tapperSamplePct float64
	tapperSamplePct = 100

	var trueValue = true

	var testCases = []struct {
		description string
		config      *config.Model
		options     []service.Option
		newResponse func() *service.Response
		makeRequest func() interface{}
		testLog     func(b []byte, e ExampleLog)
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
				URL:        filepath.Join(baseURL, "../example/model/string_lookups_float_model"),
				OutputType: "float32",

				Transformer: floatTName,
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
				Stream: &tconf.Stream{
					FlushMod:  2,
					URL:       "/tmp/test_float.json",
					SamplePct: &tapperSamplePct,
				},
			},
			newResponse: func() *service.Response {
				response := &service.Response{
					Data: &IntOutput{},
				}
				return response
			},
			makeRequest: func() interface{} {
				return stringLookupSingleRequest{
					Cache:  "a/z",
					Lookup: "a",
					Alt:    "z",
				}
			},
			expect: &FloatOutput{Value: 1.4285715, Sl: "a"},
			testLog: func(b []byte, e ExampleLog) {
				assert.NotNil(t, e.Expand.Number, string(b))
				floatValue, err := e.Expand.Number.Float64()
				assert.Nil(t, err)
				assert.Equal(t, float64(14285715), math.Round(floatValue*10000000), string(b))
			},
		},
		{
			description: "multi mode",
			options: []service.Option{
				service.WithDataStorer(mock.New()),
			},

			config: &config.Model{
				ID:         "1",
				UseDict:    &trueValue,
				URL:        filepath.Join(baseURL, "../example/model/vectorization_int_model"),
				OutputType: "int64",

				Transformer: intTName,
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
				data := []*IntOutput{}
				response := &service.Response{
					Data: data,
				}
				return response
			},
			makeRequest: func() interface{} {
				return vectorizationMultiRequest{
					BatchSize: 2,
					Caches:    []string{"a/z", "b/z"},
					Lookups:   []string{"a", "b"},
					Vecs:      []string{"z", "z"},
				}
			},
			expect: []*IntOutput{
				{Value: 10, Sl: "a"},
				{Value: 8, Sl: "b"},
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
				URL:        filepath.Join(baseURL, "../example/model/vectorization_int_model"),
				OutputType: "int64",

				Transformer: intTName,
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
				Stream: &tconf.Stream{
					FlushMod:  2,
					URL:       "/tmp/test.json",
					SamplePct: &tapperSamplePct,
				},
			},
			newResponse: func() *service.Response {
				data := []*IntOutput{}
				response := &service.Response{
					Data: data,
				}
				return response
			},
			makeRequest: func() interface{} {
				return vectorizationMultiRequest{
					BatchSize: 2,
					Caches:    []string{"a/z", "b/z"},
					Lookups:   []string{"a", "b"},
					Vecs:      []string{"z"},
				}
			},
			expect: []*IntOutput{
				{Value: 10, Sl: "a"},
				{Value: 8, Sl: "b"},
			},
			testLog: func(b []byte, e ExampleLog) {
				ns := e.Expand.Numbers
				assert.NotNil(t, ns, string(b))
				intVals := make([]int, 0, len(ns))
				for _, jn := range ns {
					intVal, err := jn.Int64()
					assert.Nil(t, err)

					intVals = append(intVals, int(intVal))
				}
				assert.Equal(t, []int{10, 8}, intVals, string(b), ns)
			},
		},
	}

	fs := afs.New()
	for _, testCase := range testCases {
		body, err := json.Marshal(testCase.makeRequest())
		if !assert.Nil(t, err, testCase.description) {
			continue
		}

		fmt.Println(string(body))

		fmt.Printf("loading: %v\n", testCase.config.URL)
		testCase.config.Init()
		srv, err := service.New(context.TODO(), fs, testCase.config, nil, map[string]*datastore.Service{}, testCase.options...)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}

		request := srv.NewRequest()
		err = gojay.Unmarshal(body, request)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}

		request.Body = body

		response := testCase.newResponse()
		err = srv.Do(context.Background(), request, response)
		if !assert.Nil(t, err, testCase.description) {
			continue
		}

		if !assert.Empty(t, response.Error, testCase.description) {
			continue
		}

		assert.EqualValues(t, testCase.expect, response.Data, testCase.description)

		if testCase.config.Stream != nil {
			rc, err := fs.OpenURL(context.Background(), testCase.config.Stream.URL)
			if !assert.Nil(t, err, testCase.config.Stream.URL) {
				continue
			}

			jsonBytes := func() []byte {
				defer rc.Close()
				buf := new(bytes.Buffer)
				n, err := io.Copy(buf, rc)

				fmt.Println("Read", n)

				if !assert.Nil(t, err) {
					return []byte{}
				}

				return buf.Bytes()
			}()

			parsedOutput := ExampleLog{}
			err = json.Unmarshal(jsonBytes, &parsedOutput)
			assert.Nil(t, err)

			if testCase.testLog != nil {
				testCase.testLog(jsonBytes, parsedOutput)
			}

			fs.Delete(context.Background(), testCase.config.Stream.URL)
		}
	}
}

type ExampleLog struct {
	ReduceSum []int           `json:"reduce_sum"`
	DenseOut  []float64       `json:"dense_2"`
	Expand    MultiTypeOutput `json:"expand"`
}

type MultiTypeOutput struct {
	Numbers []json.Number
	Number  json.Number
}

func (o *MultiTypeOutput) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		return json.Unmarshal(data, &o.Numbers)
	}
	return json.Unmarshal(data, &o.Number)
}

type stringLookupSingleRequest struct {
	Cache  string `json:"cache_key"`
	Lookup string `json:"sl"`
	Alt    string `json:"sa"`
}

type vectorizationMultiRequest struct {
	BatchSize int      `json:"batch_size"`
	Caches    []string `json:"cache_key"`
	Lookups   []string `json:"sl"`
	Vecs      []string `json:"tv"`
}

type IntOutput struct {
	Value int64
	Sl    string
}

func (p *IntOutput) EncodeBinary(enc *bintly.Writer) error {
	enc.Int64(p.Value)
	enc.String(p.Sl)
	return nil
}

func (p *IntOutput) DecodeBinary(dec *bintly.Reader) error {
	dec.Int64(&p.Value)
	dec.String(&p.Sl)
	return nil
}

func (s *IntOutput) MarshalJSONObject(enc *gojay.Encoder) {
	enc.IntKey("value", int(s.Value))
	enc.StringKey("sl", s.Sl)
}

func (s *IntOutput) IsNil() bool {
	return s == nil
}

func (p *IntOutput) Iterator() common.Iterator {
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

func (p *IntOutput) Set(iter common.Iterator) error {
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

type FloatOutput struct {
	Value float32
	Sl    string
}

func (p *FloatOutput) EncodeBinary(enc *bintly.Writer) error {
	enc.Float32(p.Value)
	enc.String(p.Sl)
	return nil
}

func (p *FloatOutput) DecodeBinary(dec *bintly.Reader) error {
	dec.Float32(&p.Value)
	dec.String(&p.Sl)
	return nil
}

func (s *FloatOutput) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Float32Key("value", s.Value)
	enc.StringKey("sl", s.Sl)
}

func (s *FloatOutput) IsNil() bool {
	return s == nil
}

func (p *FloatOutput) Iterator() common.Iterator {
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

func (p *FloatOutput) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		switch key {
		case "value":
			p.Value = float32(toolbox.AsFloat(value))
		case "sl":
			p.Sl = toolbox.AsString(value)
		}
		return nil
	})
}

func floatTransformer(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &FloatOutput{}
	result.Sl = toolbox.AsString(input.Value("sl"))
	switch actual := output.(type) {
	case []interface{}:
		return floatTransformer(ctx, signature, input, actual[0])
	case [][]float32:
		result.Value = actual[0][0]
	case *shared.Output:
		switch val := actual.Values[0].(type) {
		case [][]float32:
			result.Value = val[actual.InputIndex][0]
		}
	default:
		return nil, fmt.Errorf("unsupported type: %T", actual)
	}
	return result, nil
}

func intTransformer(ctx context.Context, signature *domain.Signature, input *gtly.Object, output interface{}) (common.Storable, error) {
	result := &IntOutput{}
	result.Sl = toolbox.AsString(input.Value("sa"))
	switch actual := output.(type) {
	case []interface{}:
		return intTransformer(ctx, signature, input, actual[0])
	case [][]int64:
		result.Value = actual[0][0]
	case *shared.Output:
		switch val := actual.Values[0].(type) {
		case [][]int64:
			result.Value = val[actual.InputIndex][0]
		}
	default:
		return nil, fmt.Errorf("unsupported type: %T", actual)
	}
	return result, nil
}
