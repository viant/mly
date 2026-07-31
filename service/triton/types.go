package triton

import (
	"fmt"
	"reflect"
)

func TritonToGoType(datatype string) reflect.Type {
	switch datatype {
	case "INT64":
		return reflect.TypeOf(int64(0))
	case "INT32":
		return reflect.TypeOf(int32(0))
	case "FP32":
		return reflect.TypeOf(float32(0))
	case "FP64":
		return reflect.TypeOf(float64(0))
	case "BYTES":
		return reflect.TypeOf("")
	default:
		panic(fmt.Sprintf("unsupported Triton datatype: %s", datatype))
	}
}
