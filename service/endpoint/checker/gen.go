package checker

import (
	"fmt"

	"github.com/francoispqt/gojay"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

type genType struct {
	D map[string]interface{}
	s map[string]*shared.Field
}

type genTypes []*genType

func Generated(outputs []*shared.Field, batch int) func() common.Storable {
	return func() common.Storable {
		mapped := make(map[string]*shared.Field, len(outputs))
		for _, oField := range outputs {
			mapped[oField.Name] = oField
		}

		if batch > 0 {
			dgts := make([]*genType, batch)
			gts := genTypes(dgts)
			for i, _ := range gts {
				gt := new(genType)
				gt.s = mapped
				gts[i] = gt
			}

			return &gts
		} else {
			gt := new(genType)
			gt.s = mapped
			return gt
		}
	}
}

// implements gojay.MarshalerJSONObject
func (g *genType) MarshalJSONObject(enc *gojay.Encoder) {
	// lazy here and do nothing
}

// implements gojay.MarshalerJSONObject
func (g *genType) IsNil() bool {
	return g == nil
}

// implements gojay.UnmarshalerJSONObject
func (g *genType) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	if g.D == nil {
		g.D = make(map[string]interface{}, len(g.s))
	}

	of, ok := g.s[key]
	if !ok {
		return fmt.Errorf("no such field %s", key)
	}

	var err error
	switch of.DataType {
	case "int":
		var i int
		err = dec.Int(&i)
		g.D[key] = i
	case "int32":
		var i int32
		err = dec.Int32(&i)
		g.D[key] = i
	case "int64":
		var i int64
		err = dec.Int64(&i)
		g.D[key] = i
	case "float32":
		var f float32
		err = dec.Float32(&f)
		g.D[key] = f
	case "float64":
		var f float64
		err = dec.Float64(&f)
		g.D[key] = f
	case "string":
		var s string
		err = dec.String(&s)
		g.D[key] = s
	default:
		return fmt.Errorf("unknown type")
	}
	return err
}

// implements gojay.UnmarshalerJSONObject
func (g *genType) NKeys() int {
	return 0
}

// implements shared/common.Storable
func (g *genType) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		return nil
	}
}

// implements shared/common.Storable
func (g *genType) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		if g.D == nil {
			g.D = make(map[string]interface{})
		}

		g.D[key] = value
		return nil
	})
}

// implements gojay.UnmarshalerJSONArray
func (g *genTypes) UnmarshalJSONArray(dec *gojay.Decoder) error {
	return dec.DecodeObject((*g)[dec.Index()])
}

// implements shared/common.Storable
func (g *genTypes) Iterator() common.Iterator {
	return func(pair common.Pair) error {
		return nil
	}
}

// implements shared/common.Storable
func (g *genTypes) Set(iter common.Iterator) error {
	return iter(func(key string, value interface{}) error {
		return nil
	})
}
