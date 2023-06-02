package model

import (
	"fmt"

	"github.com/francoispqt/gojay"
	"github.com/viant/bintly"
	"github.com/viant/mly/shared/common"
)

const Namespace string = "slft"

type Segmented struct {
	Class string
}

type Segmenteds []*Segmented

// implements common.Storable
func (s *Segmented) Iterator() common.Iterator {
	return func(p common.Pair) error {
		p("Class", s.Class)
		return nil
	}
}

// implements common.Storable
func (s *Segmenteds) Iterator() common.Iterator {
	return func(p common.Pair) error {
		for i, se := range *s {
			p(fmt.Sprintf("%d-class", i), se.Class)
		}

		return nil
	}
}

// implements common.Storable
func (s *Segmented) Set(iter common.Iterator) error {
	return iter(func(k string, v interface{}) error {
		if k == common.HashBin {
			return nil
		}

		vs, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s expected string, got %T", k, v)
		}

		s.Class = vs
		return nil
	})
}

// implements common.Storable
func (s *Segmenteds) Set(iter common.Iterator) error {
	return iter(func(k string, v interface{}) error {
		e := new(Segmented)

		vs, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s expected string, got %T", k, v)
		}

		e.Class = vs

		t := append(*s, e)
		s = &t
		return nil
	})
}

// implements gojay.UnmarshalJSONObject
func (s *Segmented) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	dec.String(&s.Class)
	return nil
}

// implements gojay.UnmarshalJSONObject
func (s *Segmented) NKeys() int {
	return 0
}

// implements gojay.MarshalJSONObject
func (s *Segmented) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("Class", s.Class)
}

// implements gojay.MarshalJSONObject
func (s *Segmented) IsNil() bool {
	return false
}

// implements bintly.Encoder
func (s *Segmented) EncodeBinary(enc *bintly.Writer) error {
	enc.String(s.Class)
	return nil
}

// implements bintly.Decoder
func (s *Segmented) DecodeBinary(dec *bintly.Reader) error {
	dec.String(&s.Class)
	return nil
}
