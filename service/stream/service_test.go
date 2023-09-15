package stream

import (
	"bytes"
	ejson "encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/service/domain"
	"github.com/viant/tapper/msg"
	"github.com/viant/tapper/msg/json"
)

type siP struct {
	O1 string `json:"out1"`
}

type muP struct {
	O1 []string `json:"out1"`
}

type mbP struct {
	O1 []struct {
		Ov []string `json:"output"`
	} `json:"out1"`
}

func TestWriteObject(t *testing.T) {
	p := msg.NewProvider(2048, 32, json.New)
	os := []domain.Output{{Name: "out1"}}

	testCases := []struct {
		name   string
		out    []interface{}
		verify func([]byte)
	}{
		{
			name: "single-single-dim",
			out: []interface{}{[][]string{
				[]string{"a"},
			}},
			verify: func(b []byte) {
				p := new(siP)
				err := ejson.Unmarshal(b, &p)
				assert.Nil(t, err)
			},
		},
		{
			name: "single-multi-dim",
			out: []interface{}{[][]string{
				[]string{"a", "b"},
			}},
			verify: func(b []byte) {
				p := new(muP)
				err := ejson.Unmarshal(b, &p)
				assert.Nil(t, err)
			},
		},
		{
			name: "batch-single-dim",
			out: []interface{}{[][]string{
				[]string{"a"},
				[]string{"b"},
			}},
			verify: func(b []byte) {
				p := new(muP)
				err := ejson.Unmarshal(b, &p)
				assert.Nil(t, err)
			},
		},
		{
			name: "batch-multi-dim",
			out: []interface{}{[][]string{
				[]string{"a", "b"},
				[]string{"c", "d"},
			}},
			verify: func(b []byte) {
				mbp := new(mbP)
				err := ejson.Unmarshal(b, &mbp)
				assert.Nil(t, err)
			},
		},
	}

	for _, tc := range testCases {
		m := p.NewMessage()
		writeObject(m, false, tc.out, os)
		b := new(bytes.Buffer)
		m.WriteTo(b)
		tc.verify(b.Bytes())
		m.Free()
	}
}
