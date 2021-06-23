package client

import (
	"encoding/json"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/viant/mly/client/msgbuf"
)

func NewReader(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("data was nil")
	}
	switch val := data.(type) {
	case *msgbuf.Message:
		return val.Bytes(), nil
	case gojay.MarshalerJSONObject:
		data, err := gojay.Marshal(val)
		if err != nil {
			return nil, err
		}
		return data, nil
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}
