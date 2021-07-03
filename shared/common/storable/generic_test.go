package storable

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/shared/common"
	"testing"
)

func TestGeneric_Iterator(t *testing.T) {
	afoo := &foo{A: 1, B: "aer", C: []int{2, 4}}
	g := NewGeneric(afoo)
	aMap := map[string]interface{}{}
	iter := g.Iterator()
	err := iter(func(key string, value interface{}) error {
		aMap[key] = value
		return nil
	})
	assert.Nil(t, err)
	assert.EqualValues(t, map[string]interface{}{
		"A": afoo.A,
		"B": afoo.B,
		"C": afoo.C,
		"D": afoo.D,
	}, aMap)
}

func TestGeneric_Set(t *testing.T) {

	aMap := map[string]interface{}{
		"A": 1,
		"B": "abc",
		"C": []int{2, 4},
		"D": []float64(nil),
	}

	aFoo := &foo{}
	g := NewGeneric(aFoo)
	err := g.Set(func(pair common.Pair) error {
		for k, v := range aMap {
			if err := pair(k, v); err != nil {
				return err
			}
		}
		return nil
	})
	assert.Nil(t, err)

	cloneMap := map[string]interface{}{}
	iter := g.Iterator()
	err = iter(func(key string, value interface{}) error {
		cloneMap[key] = value
		return nil
	})
	assert.Nil(t, err)
	assert.EqualValues(t, aMap, cloneMap)
}

type foo struct {
	A int
	B string
	C []int
	D []float64
}
