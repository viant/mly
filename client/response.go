package client

import (
	"encoding/json"
	"github.com/francoispqt/gojay"
	"time"
)

//Response represents a response
type Response struct {
	Status      string        `json:"status"`
	Error       string        `json:"error,omitempty"`
	ServiceTime time.Duration `json:"serviceTime"`
	DictHash    int           `json:"dictHash"`
	Data        interface{}   `json:"data"`
}

//UnmarshalJSONObject unmsrhal JSON (gojay API)
func (r *Response) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "status":
		if err := dec.String(&r.Status); err != nil {
			return err
		}
	case "error":
		if err := dec.String(&r.Error); err != nil {
			return err
		}
	case "dictHash":
		if err := dec.Int(&r.DictHash); err != nil {
			return err
		}
	case "serviceTimeMcs":
		serviceTime := 0
		if err := dec.Int(&serviceTime); err != nil {
			return err
		}
		r.ServiceTime = time.Duration(serviceTime) * time.Microsecond
	case "data":
		isEmpty := r.Data == nil

		var embedded = gojay.EmbeddedJSON{}
		if !isEmpty {
			if unmarshaler, ok := r.Data.(gojay.UnmarshalerJSONObject); ok {
				return dec.Object(unmarshaler)
			}
		}

		if err := dec.EmbeddedJSON(&embedded); err != nil {
			return err
		}
		if isEmpty {
			var aMap = make(map[string]interface{})
			if err := json.Unmarshal(embedded, &aMap); err != nil {
				return err
			}
			r.Data = aMap
			return nil
		} else if aMap, ok := r.Data.(map[string]interface{}); ok {
			if err := json.Unmarshal(embedded, &aMap); err != nil {
				return err
			}
			r.Data = aMap
			return nil
		}
		if err := json.Unmarshal(embedded, r.Data); err != nil {
			return err
		}
	}
	return nil
}

//NKeys returns object keys JSON (gojay API)
func (r *Response) NKeys() int {
	return 0
}

//NewResponse creates a new response
func NewResponse(data interface{}) *Response {
	return &Response{
		Data: data,
	}
}
