package client

import (
	"github.com/viant/mly/service/domain"
)

const (
	defaultStringValue = "[UKN]"
	defaultIntValue    = 0
	unknownKeyField    = -1
)

//dictionary represents dictionary
type dictionary struct {
	hash     int
	registry map[string]*entry
	//keys to position mapping
	keys map[string]int
}

func (d *dictionary) size() int {
	return len(d.registry)
}

func (d *dictionary) lookupString(key string, value string) (string, int) {
	if d == nil || len(d.registry) == 0 {
		return "", unknownKeyField
	}
	elem, ok := d.registry[key]
	if !ok {
		if index, ok := d.keys[key]; ok {
			return value, index
		}
		return "", unknownKeyField
	}
	if elem.hasString(value) {
		return value, elem.index
	}
	return defaultStringValue, elem.index
}

func (d *dictionary) lookupInt(key string, value int) (int, int) {
	if d == nil || len(d.registry) == 0 {
		return 0, unknownKeyField
	}
	elem, ok := d.registry[key]
	if !ok {
		if index, ok := d.keys[key]; ok {
			return value, index
		}
		return 0, unknownKeyField
	}
	if elem.hasInt(value) {
		return value, elem.index
	}
	return defaultIntValue, elem.index
}

func (d *dictionary) lookupFloat(key string, value float32) (float32, int) {
	if d == nil || len(d.registry) == 0 {
		return 0, unknownKeyField
	}
	elem, ok := d.registry[key]
	if !ok {
		if index, ok := d.keys[key]; ok {
			return value, index
		}
		return 0, unknownKeyField
	}
	if elem.hasFloat32(value) {
		return value, elem.index
	}
	return defaultIntValue, elem.index
}

//NewDictionary creates new dictionary
func newDictionary(dict *domain.Dictionary, keyFields []string) *dictionary {
	var result = &dictionary{
		hash:     dict.Hash,
		keys:     map[string]int{},
		registry: make(map[string]*entry),
	}

	if len(dict.Layers) > 0 {
		for _, layer := range dict.Layers {
			result.registry[layer.Name] = &entry{}

			if len(layer.Ints) > 0 {
				values := make(map[int]bool)
				for _, item := range layer.Ints {
					values[item] = true
				}
				result.registry[layer.Name].ints = values

			} else if len(layer.Floats) > 0 {
				values := make(map[float32]bool)
				for _, item := range layer.Floats {
					values[item] = true
				}
				result.registry[layer.Name].float32s = values

			} else if len(layer.Strings) > 0 {
				values := make(map[string]bool)
				for _, item := range layer.Strings {
					values[item] = true
				}
				result.registry[layer.Name].strings = values
			}
		}
		if len(keyFields) > 0 {
			for i, field := range keyFields {
				result.keys[field] = i
				if item, ok := result.registry[field]; ok {
					item.index = i
				}
			}
		}
	}
	return result
}
