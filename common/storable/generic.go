package storable

import (
	"fmt"
	"github.com/viant/mly/common"
	"reflect"
)

type generic struct {
	Value interface{}
}

func (s generic) Iterator() common.Iterator {
	v := reflect.ValueOf(s.Value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var aStruct *reflectStruct
	if v.Kind() == reflect.Struct {
		aStruct = _reflect.lookup(v.Type())
	}
	return func(pair common.Pair) error {
		switch v.Kind() {
		case reflect.Struct:
			for _, fieldType := range aStruct.fields {
				field := v.Field(fieldType.index)
				if err := pair(fieldType.name, field.Interface()); err != nil {
					return err
				}
			}
		case reflect.Map:
			switch aMap := s.Value.(type) {
			case map[string]interface{}:
				for k, v := range aMap {
					if err := pair(k, v); err != nil {
						return err
					}
				}
			case map[interface{}]interface{}:
				for k, v := range aMap {
					if err := pair(fmt.Sprintf("%s", k), v); err != nil {
						return err
					}
				}
			}
		default:
			return fmt.Errorf("unsupported type: %T", s.Value)
		}
		return nil
	}
}

func (s *generic) Set(iter common.Iterator) error {
	v := reflect.ValueOf(s.Value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var aStruct *reflectStruct
	if v.Kind() == reflect.Struct {
		aStruct = _reflect.lookup(v.Type())
	}
	return iter(func(key string, value interface{}) error {
		switch v.Kind() {
		case reflect.Struct:
			fieldType, ok := aStruct.byName[key]
			if !ok {
				return fmt.Errorf("unknown field")
			}
			field := v.Field(fieldType.index)
			if value == nil {
				return nil
			}
			rValue := reflect.ValueOf(value)
			if rValue.Kind() == field.Kind() {
				field.Set(reflect.ValueOf(value))
			} else {
				field.Set(rValue.Convert(field.Type()))
			}
			return nil
		case reflect.Map:
			switch aMap := s.Value.(type) {
			case map[string]interface{}:
				aMap[key] = value
			case map[interface{}]interface{}:
				aMap[key] = value
			}
		default:
			return fmt.Errorf("unsupported type: %T", s.Value)
		}
		return nil
	})
}

func NewGeneric(value interface{}) *generic {
	return &generic{Value: value}
}
