package client

import (
	"encoding/json"
	"fmt"
	"github.com/francoispqt/gojay"
)

//NewReader creates a new data reader
func NewReader(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("data was nil")
	}
	switch val := data.(type) {
	case *Message:
		if !val.isValid() {
			return nil, fmt.Errorf("invalid message: has been already sent before")
		}
		if err := val.end(); err != nil {
			return nil, fmt.Errorf("failed create message reader: %v", err)
		}
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
