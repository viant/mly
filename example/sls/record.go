package sls

import (
	"github.com/francoispqt/gojay"
	"github.com/viant/bintly"
	"github.com/viant/mly/shared/common"
	"github.com/viant/toolbox"
)

type Record struct {
	Value int64
	Sl    string
	X     string //auxiliary inptut //could be used to drive transformer logic
}


//EncodeBinary bintly encoder (for in local process memory cache)
func (r *Record) EncodeBinary(enc *bintly.Writer) error {
	enc.Int64(r.Value)
	enc.String(r.Sl)
	enc.String(r.X)
	return nil
}

//DecodeBinary bintly decoder (for in local process memory cache)
func (r *Record) DecodeBinary(dec *bintly.Reader) error {
	dec.Int64(&r.Value)
	dec.String(&r.Sl)
	dec.String(&r.X)
	return nil
}

//MarshalJSONObject fastest JSON marshaler for REST over HTTP/2.0
func (r *Record) MarshalJSONObject(enc *gojay.Encoder) {
	enc.IntKey("sum", int(r.Value))
	enc.StringKey("sl", r.Sl)
	enc.StringKey("x", r.X)
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
	case "sl":
		return dec.String(&r.Sl)
	case "x":
		return dec.String(&r.X)
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
		if err := pair("sl", r.Sl); err != nil {
			return err
		}
		if err := pair("x", r.X); err != nil {
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
		case "sl":
			r.Sl = toolbox.AsString(value)
		case "x":
			r.X = toolbox.AsString(value)
		}
		return nil
	})
}
