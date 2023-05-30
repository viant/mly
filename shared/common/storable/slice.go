package storable

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/francoispqt/gojay"
	"github.com/viant/bintly"
	"github.com/viant/mly/shared/common"
	"github.com/viant/toolbox"
)

//Slice represents slice registry
type Slice struct {
	hash      int
	batchSize int
	Values    []interface{}
	Fields    []*Field
}

//SetHash sets hash
func (s *Slice) SetHash(hash int) {
	s.hash = hash
}

//Hash returns hash
func (s *Slice) Hash() int {
	return s.hash
}

//SetHash sets hash
func (s *Slice) SetBatchSize(size int) {
	s.batchSize = size
}

//Hash returns hash
func (s *Slice) BatchSize() int {
	return s.batchSize
}

//Set sets value
func (s *Slice) Set(iter common.Iterator) error {
	s.Values = make([]interface{}, len(s.Fields))
	err := iter(func(key string, value interface{}) error {
		for i, field := range s.Fields {
			valueType := reflect.ValueOf(value)
			if field.Name == key {
				if field.dataType == nil {
					field.dataType = valueType.Type()
				}
				if field.Type().Kind() == valueType.Kind() {
					s.Values[i] = value
				} else {
					s.Values[i] = valueType.Convert(field.Type()).Interface()
				}
				break
			}
		}
		return nil
	})
	return err
}

//Iterator return storable iterator
func (s *Slice) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		for i, field := range s.Fields {

			if err := pair(field.Name, s.Values[i]); err != nil {
				return err
			}
		}
		return nil
	}
}

//EncodeBinary bintly encoder
func (s *Slice) EncodeBinary(stream *bintly.Writer) error {
	stream.Int(s.hash)
	for i := range s.Fields {
		if err := stream.Any(s.Values[i]); err != nil {
			return err
		}
	}
	return nil
}

//DecodeBinary bintly decoder
func (s *Slice) DecodeBinary(stream *bintly.Reader) error {
	stream.Int(&s.hash)
	s.Values = make([]interface{}, len(s.Fields))
	for i, field := range s.Fields {
		value := reflect.New(field.Type())
		if err := stream.Any(value.Interface()); err != nil {
			return err
		}
		s.Values[i] = value.Elem().Interface()
	}
	return nil
}

//MarshalJSONObject implement MarshalerJSONObject
func (s *Slice) MarshalJSONObject(enc *gojay.Encoder) {
	for i, field := range s.Fields {
		key := field.Name
		switch value := s.Values[i].(type) {
		case string:
			enc.StringKey(key, value)
		case int:
			enc.IntKey(key, value)
		case int32:
			enc.Int32Key(key, value)
		case int64:
			enc.Int64Key(key, value)
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

//UnmarshalJSONObject unmarshal json with gojay parser
func (s *Slice) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	s.Values = make([]interface{}, len(s.Fields))
	for i, field := range s.Fields {
		if field.Name == key {
			switch field.Type().Kind() {
			case reflect.Float32:
				value := float32(0)
				if err := dec.Float32(&value); err != nil {
					return err
				}
				s.Values[i] = value
			case reflect.Float64:
				value := float64(0)
				if err := dec.Float64(&value); err != nil {
					return err
				}
				s.Values[i] = value
			case reflect.Int:
				value := 0
				if err := dec.Int(&value); err != nil {
					return err
				}
				s.Values[i] = value
			case reflect.Int64:
				value := 0
				if err := dec.Int(&value); err != nil {
					return err
				}
				s.Values[i] = int64(value)
			case reflect.String:
				value := ""
				if err := dec.String(&value); err != nil {
					return err
				}
				s.Values[i] = value
			default:
				return fmt.Errorf("not yey unuspported type: %v", field.Type().Name())
			}
			break
		}
	}
	return nil
}

//NKeys returns object key count
func (s *Slice) NKeys() int {
	return 0
}

//MarshalJSON default json marshaler
func (s *Slice) MarshalJSON() ([]byte, error) {
	builder := strings.Builder{}
	builder.WriteByte('{')
	for i := range s.Fields {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteByte('"')
		builder.WriteString(s.Fields[i].Name)
		isText := s.Fields[i].Type().Kind() == reflect.String
		if isText {
			builder.WriteString(`":"`)
		} else {
			builder.WriteString(`":`)
		}
		builder.WriteString(toolbox.AsString(s.Values[i]))
		if isText {
			builder.WriteString(`"`)
		}
	}
	builder.WriteByte('}')
	return []byte(builder.String()), nil
}

//New return new storable
func New(fields []*Field) *Slice {
	if len(fields) == 0 {
		panic("field were empty\n")
	}
	return &Slice{Fields: fields}
}
