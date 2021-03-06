package service

import (
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/viant/gtly"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
	"reflect"
	"strconv"
)

//Request represent a request
type Request struct {
	Key       string
	BatchSize int
	Body      []byte
	Feeds     []interface{}
	inputs    map[string]*domain.Input
	supplied  int
	input     *gtly.Object
}

//Put puts data to request
func (r *Request) Put(key string, value string) error {
	//r.Pairs = append(r.Pairs, &Pairs{key , value})
	if input, ok := r.inputs[key]; ok {
		r.supplied++
		switch input.Type.Kind() {
		case reflect.String:
			r.Feeds[input.Index] = value
		case reflect.Bool:
			val, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse bool: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = val
		case reflect.Int:
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = int(val)

		case reflect.Int64:
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int64: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = val
		case reflect.Float64:
			val, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float64: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = val
		case reflect.Float32:
			val, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return fmt.Errorf("failed to parse float32: '%v' for %v, %w", val, key, err)
			}
			r.Feeds[input.Index] = float32(val)
		default:
			//TODO add more type support
			return fmt.Errorf("unsupported input type: %T", reflect.New(input.Type).Interface())
		}
	}
	return nil
}

//UnmarshalJSONObject umarshal
func (r *Request) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case common.BatchSizeKey:
		if err := dec.Int(&r.BatchSize); err != nil {
			return err
		}
	case common.CacheKey:
		if err := dec.String(&r.Key); err != nil {
			return err
		}
	default:
		if input, ok := r.inputs[key]; ok {
			mutator := r.input.Proto().Mutator(key)
			r.supplied++
			switch input.Type.Kind() {
			case reflect.String:
				value := ""
				if err := dec.String(&value); err != nil {
					return err
				}
				mutator.String(r.input, value)
				r.Feeds[input.Index] = [][]string{{value}}
			case reflect.Bool:
				value := false
				if err := dec.Bool(&value); err != nil {
					return err
				}
				r.Feeds[input.Index] = [][]bool{{value}}
				mutator.Bool(r.input, value)
			case reflect.Int:
				value := 0
				if err := dec.Int(&value); err != nil {
					return err
				}
				r.Feeds[input.Index] = [][]int{{value}}
				mutator.Int(r.input, value)
			case reflect.Int64:
				value := 0
				if err := dec.Int(&value); err != nil {
					return err
				}
				r.Feeds[input.Index] = [][]int64{{int64(value)}}
				mutator.Int(r.input, value)
			case reflect.Float64:
				value := float64(0)
				if err := dec.Float64(&value); err != nil {
					return err
				}
				r.Feeds[input.Index] = [][]float64{{value}}
				mutator.SetValue(r.input, float32(value))
			case reflect.Float32:
				value := float64(0)
				if err := dec.Float64(&value); err != nil {
					return err
				}
				r.Feeds[input.Index] = [][]float64{{value}}
				mutator.SetValue(r.input, float32(value))
			default:
				//TODO add more type support
				return fmt.Errorf("unsupported input type: %T", reflect.New(input.Type).Interface())
			}
		}
	}
	return nil
}

//Validate validates if request is valid
func (r *Request) Validate() error {
	if len(r.inputs) != r.supplied {
		missing := make([]string, 0)
		for _, input := range r.inputs {
			if r.Feeds[input.Index] == nil {
				missing = append(missing, input.Name)
			}
		}
		return fmt.Errorf("failed to build request due to missing fields: %v", missing)
	}
	return nil
}

//NKeys returns object keys
func (r *Request) NKeys() int {
	return 2 + len(r.inputs)
}
