package storable

import (
	"github.com/francoispqt/gojay"
	"github.com/viant/bintly"
	"github.com/viant/mly/common"
	"reflect"
)

type Slice struct {
	hash   int `json:"-"`
	keys   []string
	values []interface{}
	types  []reflect.Type
}

func (s *Slice) SetHash(hash int) {
	s.hash = hash
}

func (s *Slice) Hash() int {
	return s.hash
}

func (s *Slice) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		s.keys = append(s.keys, key)
		s.values = append(s.values, value)
		return nil
	})
}

func (s *Slice) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		for i, key := range s.keys {
			if err := pair(key, s.values[i]); err != nil {
				return err
			}
		}
		return nil
	}
}

//EncodeBinary bintly encoder
func (s *Slice) EncodeBinary(stream *bintly.Writer) error {
	stream.Int(s.hash)
	for i := range s.types {
		if err := stream.Any(s.values[i]);err != nil {
			return err
		}
	}
	return nil
}

//DecodeBinary bintly decoder
func  (s *Slice) DecodeBinary(stream *bintly.Reader) error {
	stream.Int(&s.hash)
	s.values = make([]interface{}, len(s.types))
	for i, aType := range s.types {
		value := reflect.New(aType)
		if err := stream.Any(value.Interface());err != nil {
			return err
		}
		s.values[i] = value.Elem().Interface()
	}
	return nil
}


//MarshalJSONObject implement MarshalerJSONObject
func (s *Slice) MarshalJSONObject(enc *gojay.Encoder) {
	for i, key := range s.keys {
		switch value := s.values[i].(type) {
		case string:
			enc.StringKey(key, value)
		case int:
			enc.IntKey(key, value)
		case int64:
			enc.IntKey(key, int(value))
		case float32:
			enc.Float32Key(key, value)
		case float64:
			enc.Float64Key(key, value)
		case []string:
			enc.AddSliceStringKey(key, value)
		case []int:
			enc.AddSliceIntKey(key, value)
		case []int32:
			enc.ArrayKey(key, gojay.EncodeArrayFunc(func(enc *gojay.Encoder) {
				for _, i := range value {
					enc.Int(int(i))
				}
			}))

		case []int64:
			enc.ArrayKey(key, gojay.EncodeArrayFunc(func(enc *gojay.Encoder) {
				for _, i := range value {
					enc.Int(int(i))
				}
			}))
		}
	}
}


//IsNil returns nil
func (s *Slice) IsNil() bool {
	return s == nil
}

//New return new storable
func New(types []reflect.Type) *Slice {
	return &Slice{types: types}
}

