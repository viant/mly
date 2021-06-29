package storable

import (
	"reflect"
	"strings"
	"sync"
)

var _reflect = &reflectCache{cache: make(map[string]*reflectStruct)}

type reflectCache struct {
	cache map[string]*reflectStruct
	mux   sync.RWMutex
}

func (c *reflectCache) lookup(aType reflect.Type) *reflectStruct {
	c.mux.RLock()
	result, ok := c.cache[aType.Name()]
	c.mux.RUnlock()
	if ok {
		return result
	}
	result = newReflectStruct(aType)
	c.mux.Lock()
	c.cache[aType.Name()] = result
	c.mux.Unlock()
	return result
}

type reflectStruct struct {
	byName map[string]*reflectField
	fields []*reflectField
}

type reflectField struct {
	index int
	name  string
	reflect.StructField
}

func newReflectStruct(aType reflect.Type) *reflectStruct {
	result := &reflectStruct{byName: make(map[string]*reflectField)}
	for i := 0; i < aType.NumField(); i++ {
		fieldType := aType.Field(i)
		aField := &reflectField{index: i, name: fieldType.Name}
		if JSON, ok := fieldType.Tag.Lookup("json"); ok {
			if name := strings.Split(JSON, ",")[0]; name != "" {
				aField.name = name
			}
		}
		result.fields = append(result.fields, aField)
		result.byName[fieldType.Name] = aField
		result.byName[aField.name] = aField
	}
	return result
}
