package client

import (
	"encoding/json"
	"fmt"

	"github.com/francoispqt/gojay"
)

func Marshal(data interface{}, id string) ([]byte, error) {
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

		if id != "" {
			fmt.Printf("[%s Marshal] Message\n", id)
		}
		return val.Bytes(), nil
	case gojay.MarshalerJSONObject:
		data, err := gojay.Marshal(val)
		if err != nil {
			return nil, err
		}
		if id != "" {
			fmt.Printf("[%s Marshal] gojay\n", id)
		}
		return data, nil
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}
		if id != "" {
			fmt.Printf("[%s json] json\n", id)
		}
		return data, nil
	}
}
