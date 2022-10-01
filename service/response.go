package service

import (
	"encoding/json"
	"github.com/francoispqt/gojay"
	"github.com/viant/mly/shared/common"
	"github.com/viant/xunsafe"
	"log"
	"time"
	"unsafe"
)

//Response represents service response
type Response struct {
	started        time.Time
	xSlice         *xunsafe.Slice
	sliceLen       int
	Status         string
	Error          string
	DictHash       int
	Data           interface{}
	ServiceTimeMcs int
}

//SetError sets errors
func (r *Response) SetError(err error) {
	if err == nil {
		return
	}
	r.Error = err.Error()
	r.Status = common.StatusError
}

//MarshalJSONObject marshal response
func (r *Response) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty("status", r.Status)
	enc.StringKeyOmitEmpty("error", r.Error)
	enc.IntKeyOmitEmpty("dictHash", r.DictHash)
	enc.IntKey("serviceTimeMcs", r.ServiceTimeMcs)
	if r.Data != nil {
		if r.xSlice != nil {
			aMarshaler := &sliceMarshaler{len: r.sliceLen, xSlice: r.xSlice, ptr: xunsafe.AsPointer(r.Data)}
			enc.ArrayKey("data", aMarshaler)
			return
		}
		if marshaler, ok := r.Data.(gojay.MarshalerJSONObject); ok {
			enc.ObjectKey("data", marshaler)
		} else {
			if data, err := json.Marshal(r.Data); err == nil {
				embeded := gojay.EmbeddedJSON(data)
				if err = enc.EncodeEmbeddedJSON(&embeded); err != nil {
					log.Printf("faild to encode: %s %v", data, err)
				}
			}
		}
	}
}

//IsNil returns true if nil (gojay json API)
func (r *Response) IsNil() bool {
	return false
}

type sliceMarshaler struct {
	len    int
	xSlice *xunsafe.Slice
	ptr    unsafe.Pointer
}

// MarshalerJSONArray is the interface to implement
// for a slice or an array to be encoded
func (m *sliceMarshaler) MarshalJSONArray(enc *gojay.Encoder) {
	for i := 0; i < m.len; i++ {
		value := m.xSlice.ValuePointerAt(m.ptr, i)
		marshaler, ok := value.(gojay.MarshalerJSONObject)
		if ok {
			enc.Object(marshaler)
		} else {
			if data, err := json.Marshal(value); err == nil {
				embeded := gojay.EmbeddedJSON(data)
				if err = enc.EncodeEmbeddedJSON(&embeded); err != nil {
					log.Printf("faild to encode: %s %v", data, err)
				}
			}
		}
	}
}

func (m *sliceMarshaler) IsNil() bool {
	return false
}
