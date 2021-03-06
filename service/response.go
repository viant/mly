package service

import (
	"encoding/json"
	"github.com/francoispqt/gojay"
	"github.com/viant/mly/shared/common"
	"log"
	"time"
)

//Response represents service response
type Response struct {
	started        time.Time
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
		if marshaler, ok := r.Data.(gojay.MarshalerJSONObject); ok {
			enc.ObjectKey("data", marshaler)
		} else {
			if data, err := json.Marshal(r.Data); err == nil {
				embeded := gojay.EmbeddedJSON(data)
				if err = enc.EncodeEmbeddedJSON(&embeded);err != nil {
					log.Printf("faild to encode: %s %v", data,  err)
				}
			}
		}
	}
}

//IsNil returns true if nil (gojay json API)
func (r *Response) IsNil() bool {
	return false
}
