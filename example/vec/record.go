package vec

import (
	"github.com/francoispqt/gojay"
	"github.com/viant/bintly"
	"github.com/viant/mly/shared/common"
	"github.com/viant/toolbox"
)

type Records []*Record

func (r *Records) UnmarshalJSONArray(dec *gojay.Decoder) error {
	record := &Record{}
	if err := dec.Object(record); err != nil {
		return err
	}
	*r = append(*r, record)
	return nil
}

type Record struct {
	Value int64
	Sa    string
}

//EncodeBinary bintly encoder (for in local process memory cache)
func (r *Record) EncodeBinary(enc *bintly.Writer) error {
	enc.Int64(r.Value)
	enc.String(r.Sa)
	return nil
}

//DecodeBinary bintly decoder (for in local process memory cache)
func (r *Record) DecodeBinary(dec *bintly.Reader) error {
	dec.Int64(&r.Value)
	dec.String(&r.Sa)
	return nil
}

//MarshalJSONObject fastest JSON marshaler for REST over HTTP/2.0
func (r *Record) MarshalJSONObject(enc *gojay.Encoder) {
	enc.IntKey("sum", int(r.Value))
	enc.StringKey("sa", r.Sa)
}

func (r *Record) IsNil() bool {
	return r == nil
}

//UnmarshalJSONObject fastest JSON unmarshaler for REST over HTTP/2.0
func (r *Record) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "sum":
		value := int(0)
		err := dec.Int(&value)
		r.Value = int64(value)
		return err
	case "sa":
		return dec.String(&r.Sa)
	}
	return nil
}
func (r *Record) NKeys() int {
	return 0
}

//Iterator returns SumOutput iterator to store in datastore (aerospike)
func (r *Record) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		if err := pair("value", r.Value); err != nil {
			return err
		}
		if err := pair("sa", r.Sa); err != nil {
			return err
		}
		return nil
	}
}

//Set set SumOutput from datastore (aerospike) iterator
func (r *Record) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		switch key {
		case "value":
			r.Value = int64(toolbox.AsInt(value))
		case "sa":
			r.Sa = toolbox.AsString(value)
		}
		return nil
	})
}
