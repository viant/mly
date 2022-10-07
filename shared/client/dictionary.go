package client

import (
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
	"log"
)

const (
	defaultStringValue = "[UNK]"
	defaultIntValue    = 0
	unknownKeyField    = -1
)

//Dictionary represents Dictionary
type Dictionary struct {
	hash     int
	registry map[string]*entry
	inputs   map[string]*shared.Field
}

func (d *Dictionary) KeysLen() int {
	return len(d.inputs)
}

func (d *Dictionary) inputSize() int {
	return len(d.inputs)
}

func (d *Dictionary) size() int {
	return len(d.registry)
}

func (d *Dictionary) lookupString(key string, value string) (string, int) {
	if d == nil {
		return "", unknownKeyField
	}
	input, ok := d.inputs[key]
	if !ok {
		return "", unknownKeyField
	}
	if input.Wildcard {
		return value, input.Index
	}
	if len(d.registry) == 0 {
		return "", unknownKeyField
	}
	elem, ok := d.registry[key]
	if !ok {
		return "", unknownKeyField
	}
	if elem == nil {
		log.Printf("dict element was nil for %v: %v\n", key, value)
		return defaultStringValue, unknownKeyField
	}
	if elem.hasString(value) {
		return value, input.Index
	}
	return defaultStringValue, input.Index
}

func (d *Dictionary) lookupInt(key string, value int) (int, int) {
	if d == nil {
		return 0, unknownKeyField
	}
	input, ok := d.inputs[key]
	if !ok {
		return 0, unknownKeyField
	}
	if input.Wildcard {
		return value, input.Index
	}

	if len(d.registry) == 0 {
		return 0, unknownKeyField
	}
	elem, ok := d.registry[key]
	if !ok {
		return 0, unknownKeyField
	}

	if elem == nil {
		log.Printf("dict element was nil for %v: %v\n", key, value)
		return defaultIntValue, unknownKeyField
	}
	if elem.hasInt(value) {
		return value, input.Index
	}
	return defaultIntValue, input.Index
}

func (d *Dictionary) lookupFloat(key string, value float32) (float32, int) {
	if d == nil {
		return 0, unknownKeyField
	}
	input, ok := d.inputs[key]
	if !ok {
		return 0, unknownKeyField
	}
	if input.Wildcard {
		return value, input.Index
	}

	if len(d.registry) == 0 {
		return 0, unknownKeyField
	}
	elem, ok := d.registry[key]
	if !ok {
		return 0, unknownKeyField
	}

	if elem == nil {
		log.Printf("dict element was nil for %v: %v\n", key, value)
		return 0, unknownKeyField
	}
	if elem.hasFloat32(value) {
		return value, input.Index
	}
	return 0, input.Index
}

//NewDictionary creates new Dictionary
func NewDictionary(dict *common.Dictionary, inputs []*shared.Field) *Dictionary {
	var result = &Dictionary{
		inputs:   map[string]*shared.Field{},
		hash:     dict.Hash,
		registry: make(map[string]*entry),
	}
	for i, input := range inputs {
		result.inputs[input.Name] = inputs[i]
	}
	result.init(dict)
	return result
}

func (d *Dictionary) init(dict *common.Dictionary) {
	if dict == nil || len(dict.Layers) == 0 {
		return
	}
	for _, layer := range dict.Layers {
		d.registry[layer.Name] = &entry{}
		if len(layer.Ints) > 0 {
			values := make(map[int]bool)
			for _, item := range layer.Ints {
				values[item] = true
			}
			d.registry[layer.Name].ints = values

		} else if len(layer.Floats) > 0 {
			values := make(map[float32]bool)
			for _, item := range layer.Floats {
				values[item] = true
			}
			d.registry[layer.Name].float32s = values

		} else if len(layer.Strings) > 0 {
			values := make(map[string]bool)
			for _, item := range layer.Strings {
				values[item] = true
			}
			d.registry[layer.Name].strings = values
		}
	}
}
