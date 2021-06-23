package common

import (
	"fmt"
	"reflect"
	"strings"
)

//DataType return reflect.Type for supplied data type
func DataType(dataType string) (reflect.Type, error) {
	switch strings.ToLower(dataType) {
	case "string":
		return reflect.TypeOf(""), nil
	case "float64":
		return reflect.TypeOf(float64(0)), nil
	case "float32":
		return reflect.TypeOf(float32(0)), nil
	case "int":
		return reflect.TypeOf(int(0)), nil
	case "int64":
		return reflect.TypeOf(int64(0)), nil
	case "int32":
		return reflect.TypeOf(int32(0)), nil
	case "bool":
		return reflect.TypeOf(true), nil
	case "[]int":
		return reflect.TypeOf([]int{}), nil
	case "[]int32":
		return reflect.TypeOf([]int32{}), nil
	case "[]int64":
		return reflect.TypeOf([]int64{}), nil
	case "[]float32":
		return reflect.TypeOf([]float32{}), nil
	case "[]float64":
		return reflect.TypeOf([]float64{}), nil
	default:
		return nil, fmt.Errorf("unsupported data type: %v", dataType)
	}
}
