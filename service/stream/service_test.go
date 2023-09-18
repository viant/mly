package stream

import (
	"bytes"
	stdjson "encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/mly/service/domain"
	"github.com/viant/tapper/msg"
	"github.com/viant/tapper/msg/json"
)

// batch:single output:single payload
type sssp struct {
	O1 string `json:"out1"`
}

type ssip struct {
	O1 int `json:"out1"`
}

type ssfp struct {
	O1 float64 `json:"out1"`
}

// batch:multi, output:single or batch:single output:multi string payload
// xx = mixed
type mxsp struct {
	O1 []string `json:"out1"`
}

type mxip struct {
	O1 []int `json:"out1"`
}

type mxfp struct {
	O1 []float64 `json:"out1"`
}

// batch:multi, output:multi string payload
type mmsp struct {
	O1 []struct {
		Ov []string `json:"output"`
	} `json:"out1"`
}

type mmip struct {
	O1 []struct {
		Ov []int `json:"output"`
	} `json:"out1"`
}

type mmfp struct {
	O1 []struct {
		Ov []float64 `json:"output"`
	} `json:"out1"`
}

func TestWriteObject(t *testing.T) {
	p := msg.NewProvider(2048, 32, json.New)
	os := []domain.Output{{Name: "out1"}}

	testCases := []struct {
		name string
		out  []interface{}
		// expected instance
		ei func() interface{}
	}{
		{
			name: "single-single-string",
			out:  []interface{}{[][]string{[]string{"a"}}},
			ei:   func() interface{} { return new(sssp) },
		},
		{
			name: "single-multi-string",
			out:  []interface{}{[][]string{[]string{"a", "b"}}},
			ei:   func() interface{} { return new(mxsp) },
		},
		{
			name: "batch-single-string",
			out:  []interface{}{[][]string{[]string{"a"}, []string{"b"}}},
			ei:   func() interface{} { return new(mxsp) },
		},
		{
			name: "batch-multi-string",
			out:  []interface{}{[][]string{[]string{"a", "b"}, []string{"c", "d"}}},
			ei:   func() interface{} { return new(mmsp) },
		},
		{
			name: "single-single-int",
			out:  []interface{}{[][]int64{[]int64{1}}},
			ei:   func() interface{} { return new(ssip) },
		},
		{
			name: "single-multi-int",
			out:  []interface{}{[][]int64{[]int64{1, 2, 3}}},
			ei:   func() interface{} { return new(mxip) },
		},
		{
			name: "batch-single-int",
			out:  []interface{}{[][]int64{[]int64{1}, []int64{2}}},
			ei:   func() interface{} { return new(mxip) },
		},
		{
			name: "batch-multi-int",
			out:  []interface{}{[][]int64{[]int64{1, 2, 5}, []int64{3, 4, 6}}},
			ei:   func() interface{} { return new(mmip) },
		},
		{
			name: "single-single-float",
			out:  []interface{}{[][]float32{[]float32{1}}},
			ei:   func() interface{} { return new(ssfp) },
		},
		{
			name: "single-multi-float",
			out:  []interface{}{[][]float32{[]float32{1, 2, 3}}},
			ei:   func() interface{} { return new(mxfp) },
		},
		{
			name: "batch-single-float",
			out:  []interface{}{[][]float32{[]float32{1}, []float32{2}}},
			ei:   func() interface{} { return new(mxfp) },
		},
		{
			name: "batch-multi-float",
			out:  []interface{}{[][]float32{[]float32{1, 2, 5}, []float32{3, 4, 6}}},
			ei:   func() interface{} { return new(mmfp) },
		},
	}

	for _, tc := range testCases {
		m := p.NewMessage()
		err := writeObject(m, false, tc.out, os)
		require.Nil(t, err)
		b := new(bytes.Buffer)
		m.WriteTo(b)
		ei := tc.ei()
		err = stdjson.Unmarshal(b.Bytes(), ei)
		require.Nil(t, err)
		m.Free()
	}
}
