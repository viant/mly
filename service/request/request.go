package request

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/francoispqt/gojay"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/transfer"
)

var exists struct{} = struct{}{}

// Request represents the server-side post-processed information about a request.
// There is no strict struct for request payload since some of the keys of the request are dynamically generated based on the model inputs.
// See shared/client.Message for client-side perspective.
type Request struct {
	Body []byte // usually the POST JSON content

	// Passed through to Evaluator.
	// This is expected to be [numInputs][1][batchSize]T.
	// TODO consider scenario when the second slice does not have length 1.
	Feeds []interface{}

	supplied map[string]struct{} // used to check if the required inputs were provided

	Input *transfer.Input // cache metadata

	// type metadata from service/tfservice.Service.inputs
	// see service/tfmodel.(*Service).reconcileIOFromSignature
	inputs map[string]*domain.Input
}

func NewRequest(keyLen int, inputs map[string]*domain.Input) *Request {
	inputLen := 0
	for _, inp := range inputs {
		if !inp.Auxiliary {
			inputLen++
		}
	}

	return &Request{
		Feeds:    make([]interface{}, inputLen),
		supplied: make(map[string]struct{}, keyLen),
		Input:    &transfer.Input{},
		inputs:   inputs,
	}
}

// Put is used when constructing a request NOT using gojay.
// Deprecated - no longer support this option.
func (r *Request) Put(key string, value string) error {
	if input, ok := r.inputs[key]; ok {
		r.supplied[key] = exists

		switch input.Type.Kind() {
		case reflect.String:
			r.Feeds[input.Index] = [][]string{{value}}
		case reflect.Bool:
			val, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse bool: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = [][]bool{{val}}
		case reflect.Int:
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = [][]int{{int(val)}}
		case reflect.Int32:
			val, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse int32: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = [][]int32{{int32(val)}}
		case reflect.Int64:
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int64: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = [][]int64{{val}}
		case reflect.Float64:
			val, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float64: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = [][]float64{{val}}
		case reflect.Float32:
			val, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return fmt.Errorf("failed to parse float32: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = [][]float32{{float32(val)}}
		default:
			// TODO add more type support
			return fmt.Errorf("unsupported input type: %T", reflect.New(input.Type).Interface())
		}
	}

	return nil
}

// implements gojay.UnmarshalJSONObject
// see service.(*Handler).serveHTTP()
// There is some polymorphism involved, as well as loose-typing.
// Model inputs aren't strictly typed from the perspective of the request.
// The primary polymorphism comes from the support for multiple rows worth of inputs.
// There are 2 optional keys that aren't part of the model input: "batch_size" and "cache_key".
// If the key "batch_size" is provided, then the keys' values should be an array of scalars; otherwise, the keys' values can be a single scalar.
// The key "cache_key" is used for caching, and should be an array of strings if "batch_size" is provided.
func (r *Request) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	if r.Input == nil {
		r.Input = &transfer.Input{}
	}

	if r.supplied == nil {
		r.supplied = make(map[string]struct{})
	}

	switch key {
	case common.BatchSizeKey:
		return dec.Int(&r.Input.BatchSize)
	case common.CacheKey:
		if r.Input.BatchMode() {
			return dec.DecodeArray(&r.Input.Keys)
		}
		var k string
		if err := dec.String(&k); err != nil {
			return err
		}
		if err := r.Input.Keys.Set(k); err != nil {
			return err
		}
	default:
		input, ok := r.inputs[key]
		if !ok {
			return fmt.Errorf("unknown field:%v, available:%v", key, r.inputs)
		}

		r.supplied[key] = exists
		inputValue, err := r.Input.SetAt(input.Index, input.Name, input.Type.Kind())
		if err != nil {
			return err
		}

		if r.Input.BatchMode() {
			if err := dec.DecodeArray(inputValue); err != nil {
				return err
			}
			if !input.Auxiliary {
				r.Feeds[input.Index] = inputValue.Feed(r.Input.BatchSize)
			}
			return nil
		}

		outerr := func() error {
			switch input.Type.Kind() {
			case reflect.String:
				var value string
				if err := dec.String(&value); err != nil {
					return err
				}
				_ = inputValue.Set(value)
				if !input.Auxiliary {
					r.Feeds[input.Index] = [][]string{{value}}
				}
			case reflect.Bool:
				var value bool
				if err := dec.Bool(&value); err != nil {
					return err
				}
				if !input.Auxiliary {
					r.Feeds[input.Index] = [][]bool{{value}}
				}
				_ = inputValue.Set(value)
			case reflect.Int:
				var value int
				if err := dec.Int(&value); err != nil {
					return err
				}
				if !input.Auxiliary {
					r.Feeds[input.Index] = [][]int{{value}}
				}
				_ = inputValue.Set(value)
			case reflect.Int32:
				var value int32
				if err := dec.Int32(&value); err != nil {
					return err
				}
				if !input.Auxiliary {
					r.Feeds[input.Index] = [][]int32{{value}}
				}
				_ = inputValue.Set(value)
			case reflect.Int64:
				var value int64
				if err := dec.Int64(&value); err != nil {
					return err
				}
				if !input.Auxiliary {
					r.Feeds[input.Index] = [][]int64{{value}}
				}
				_ = inputValue.Set(value)
			case reflect.Float64:
				value := float64(0)
				if err := dec.Float64(&value); err != nil {
					return err
				}
				if !input.Auxiliary {
					r.Feeds[input.Index] = [][]float64{{value}}
				}
				_ = inputValue.Set(value)
			case reflect.Float32:
				value := float64(0)
				if err := dec.Float64(&value); err != nil {
					return err
				}
				if !input.Auxiliary {
					r.Feeds[input.Index] = [][]float32{{float32(value)}}
				}
				_ = inputValue.Set(value)
			default:
				return fmt.Errorf("unsupported request input type: %T", reflect.New(input.Type).Interface())
			}

			return nil
		}()

		if outerr != nil {
			return fmt.Errorf("error decoding %s, %v", key, outerr)
		}
	}

	return nil
}

// implements gojay.UnmarshalJSONObject
func (r *Request) NKeys() int {
	return 0
}

// Validate is only used server-side.
func (r *Request) Validate() error {
	missing := make([]string, 0)
	for _, input := range r.inputs {
		_, ok := r.supplied[input.Name]
		if !ok {
			missing = append(missing, input.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("failed to build request due to missing fields: %v", missing)
	}

	return nil
}
