package request

import (
	"reflect"
	"testing"

	"github.com/francoispqt/gojay"
	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/service/domain"
)

func TestDecode(t *testing.T) {
	modelInputs := []*domain.Input{
		&domain.Input{Name: "a1", Type: reflect.TypeOf(string(""))},
		&domain.Input{Name: "a2", Type: reflect.TypeOf(string(""))},
		&domain.Input{
			Name:      "a3",
			Auxiliary: true,
			Type:      reflect.TypeOf(string("")),
		},
	}

	inputs := make(map[string]*domain.Input, len(modelInputs))

	for i, modelInput := range modelInputs {
		modelInput.Index = i
		inputs[modelInput.Name] = modelInput
	}

	numInputs := len(modelInputs)

	testCases := []struct {
		desc            string
		requestEnc      string
		isValid         bool
		additionalCheck func(*Request, *testing.T)
	}{
		{
			desc: "simple",
			requestEnc: `{
	"batch_size": 1,
	"a2": ["a2_0"],
	"a1": ["a1_0"], 
	"a3": ["a3_0"],
	"cache_key": ["ck1"],
}`,
			isValid: true,
		},
		{
			desc: "string_issue",
			requestEnc: `{
	"a1": "a1_0",
	"a2": "a2_0",
	"a3": "a3_0",
	"cache_key": "ck0"
}`,
			isValid: true,
		},
		{
			desc: "invalid",
			requestEnc: `{
	"batch_size": 1,
	"a1": ["a1_0"], 
	"a3": ["a3_0"],
	"cache_key": ["ck1"],
}`,
			isValid: false,
		},
		{
			desc: "duplicate_aux",
			requestEnc: `{
	"batch_size": 1,
	"a1": ["a1_0"], 
	"a2": ["a1_0"], 
	"a3": ["a3_0"], 
	"a3": ["a3_1"],
	"cache_key": ["ck1"],
}`,
			isValid: true,
		},
		{
			desc: "duplicate_input",
			requestEnc: `{
	"batch_size": 1,
	"a1": ["a1_0"], 
	"a2": ["a2_0"], 
	"a2": ["a2_1"], 
	"a3": ["a3_0"],
	"cache_key": ["ck1"],
}`,
			isValid: true,
			additionalCheck: func(r *Request, t *testing.T) {
				a2Idx := inputs["a2"].Index
				v, ok := r.Feeds[a2Idx].([][]string)
				assert.True(t, ok)
				assert.Equal(t, v[0][0], "a2_1")
			},
		},
	}

	for _, tc := range testCases {
		r := &Request{
			inputs: inputs,
			Feeds:  make([]interface{}, numInputs, numInputs),
		}

		err := gojay.Unmarshal([]byte(tc.requestEnc), r)

		assert.Nil(t, err)
		err = r.Validate()
		if tc.isValid {
			var msg string
			if err != nil {
				msg = err.Error()
			}
			assert.Nil(t, err, tc.desc, msg)
		} else {
			var msg string
			if err != nil {
				msg = err.Error()
			}
			assert.NotNil(t, err, tc.desc, msg)
		}

		if tc.additionalCheck != nil {
			tc.additionalCheck(r, t)
		}
	}
}
