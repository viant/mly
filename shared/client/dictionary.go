package client

import (
	"log"

	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

type fieldOffset int

const (
	// oov = out of vocabulary
	// TODO - technically the OOV value can be overwritten - [UNK] may be a valid value
	oovString = "[UNK]"
	oovInt    = 0

	defaultPrec = 10

	unknownKeyField = fieldOffset(-1)
)

// Dictionary helps identify any out-of-vocabulary input values for reducing the cache space, as well as an explicit cache-invalidation strategy via hash.
// See shared/common.Dictionary
type Dictionary struct {
	hash int

	// registry key is the input name
	registry map[string]*entry

	// inputs is an index, key is the input name
	inputs map[string]*shared.Field
}

func (d *Dictionary) KeysLen() int {
	return len(d.inputs)
}

func (d *Dictionary) inputSize() int {
	return len(d.inputs)
}

// TODO refactor, this has a singular use case
func (d *Dictionary) Fields() map[string]*shared.Field {
	return d.inputs
}

func (d *Dictionary) getInput(n string) *shared.Field {
	if d == nil {
		return nil
	}

	input, ok := d.inputs[n]
	if !ok {
		return nil
	}

	return input
}

func (d *Dictionary) getEntry(n string) *entry {
	if d == nil {
		return nil
	}

	if len(d.registry) == 0 {
		return nil
	}

	elem, ok := d.registry[n]
	if !ok {
		return nil
	}

	if elem == nil {
		// generally speaking, if d.registry has data, it should have data for ALL columns
		// TODO this shouldn't print, it should tick some counter
		log.Printf("registry entry was nil for %v", n)
	}

	return elem
}

// lookupString returns the mapped key, or unknownKeyField, meaning no mapping exists
func (d *Dictionary) lookupString(key string, value string) (string, fieldOffset) {
	input := d.getInput(key)
	if input == nil {
		return "", unknownKeyField
	}

	ii := fieldOffset(input.Index)

	if input.Wildcard {
		return value, ii
	}

	entr := d.getEntry(key)
	if entr == nil {
		return "", unknownKeyField
	}

	if entr.hasString(value) {
		return value, ii
	}

	return oovString, ii
}

// lookupInt returns the mapped key, or unknownKeyField, meaning no mapping exists
func (d *Dictionary) lookupInt(key string, value int) (int, fieldOffset) {
	input := d.getInput(key)
	if input == nil {
		return 0, unknownKeyField
	}

	ii := fieldOffset(input.Index)

	if input.Wildcard {
		return value, ii
	}

	entr := d.getEntry(key)
	if entr == nil {
		return 0, unknownKeyField
	}

	if entr.hasInt(value) {
		return value, ii
	}

	return oovInt, ii
}

// reduceFloat returns a lower-precision float key, or unknownKeyField, meaning no reduction exists
func (d *Dictionary) reduceFloat(key string, value float32) (float32, int, fieldOffset) {
	input := d.getInput(key)
	if input == nil {
		return value, defaultPrec, unknownKeyField
	}

	ii := fieldOffset(input.Index)

	if input.Wildcard {
		// this isn't really a valid case
		return value, defaultPrec, ii
	}

	entr := d.getEntry(key)
	if entr == nil {
		return value, defaultPrec, unknownKeyField
	}

	usePrec := defaultPrec
	if entr.prec > 0 {
		usePrec = int(entr.prec)
	}

	return entr.reduceFloat32(value), usePrec, ii
}

// NewDictionary creates new Dictionary
func NewDictionary(dict *common.Dictionary, inputs []*shared.Field) *Dictionary {
	// index by name
	inputIdx := make(map[string]*shared.Field)

	for i, input := range inputs {
		inputIdx[input.Name] = inputs[i]
	}

	var result = &Dictionary{
		inputs:   inputIdx,
		hash:     dict.Hash,
		registry: make(map[string]*entry),
	}

	if dict == nil {
		return result
	}

	layerIdx := make(map[string]*common.Layer)

	for _, layer := range dict.Layers {
		e := new(entry)

		if len(layer.Ints) > 0 {
			values := make(map[int]bool)
			for _, item := range layer.Ints {
				values[item] = true
			}
			e.ints = values
		} else if len(layer.Strings) > 0 {
			values := make(map[string]bool)
			for _, item := range layer.Strings {
				values[item] = true
			}
			e.strings = values
		}

		ln := layer.Name

		result.registry[ln] = e
	}

	for _, input := range inputs {
		iName := input.Name

		if _, ok := layerIdx[iName]; ok {
			continue
		}

		if input.Precision > 0 {
			result.registry[iName] = FloatEntry(input.Precision)
		}
	}

	return result
}
