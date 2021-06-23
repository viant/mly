package service

import (
	"github.com/francoispqt/gojay"
	"time"
)

type Response struct {
	started        time.Time
	Status         string
	Error          string
	ModelHash      int
	Data           gojay.MarshalerJSONObject
	ServiceTimeMcs int
}

func (r *Response) SetError(err error) {
	if err == nil {
		return
	}
	r.Error = err.Error()
	r.Status = "error"
}

func (r *Response) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty("status", r.Status)
	enc.StringKeyOmitEmpty("error", r.Error)
	enc.IntKeyOmitEmpty("modelHash", r.ModelHash)
	enc.IntKey("serviceTimeMcs", r.ServiceTimeMcs)
	if r.Data != nil {
		enc.ObjectKey("data", r.Data)
	}
}

func (r *Response) IsNil() bool {
	return false
}
