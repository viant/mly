package storable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/shared/common"
)

var testMap = map[string]interface{}{"A": 1, "B": "abc", "C": []int{2, 4}, "D": []float64(nil)}
var testFoo = &foo{A: 1, B: "abc", C: []int{2, 4}}

func TestGenericIterator(t *testing.T) {
	afoo := testFoo

	// test struct
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

	// test map
	g = NewGeneric(testMap)
	iter = g.Iterator()
	bMap, err := iter.ToMap()
	assert.Nil(t, err)
	assert.EqualValues(t, testMap, bMap)
}

func TestGenericSet(t *testing.T) {
	aMap := testMap

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
	assert.EqualValues(t, aFoo, testFoo)

	cloneMap, err := g.Iterator().ToMap()
	assert.Nil(t, err)
	assert.EqualValues(t, aMap, cloneMap)
}

// TestGenericSetSkipsHashBin verifies that Generic.Set skips the internal HashBin key so that it does not map the dictHash bin into struct fields.
func TestGenericSetSkipsHashBin(t *testing.T) {
	type hashedFoo struct {
		A        int
		DictHash int
	}

	// Refer to shared/datastore/service.go for how common.HashBin is used
	foo := &hashedFoo{A: 0, DictHash: -1}
	data := map[string]interface{}{
		"A":            5,
		common.HashBin: 42,
		"DictHash":     99,
	}
	g := NewGeneric(foo)
	err := g.Set(common.MapToIterator(data))
	assert.Nil(t, err)

	assert.EqualValues(t, 5, foo.A)
	assert.EqualValues(t, 99, foo.DictHash)
}

type foo struct {
	A int
	B string
	C []int
	D []float64
}
