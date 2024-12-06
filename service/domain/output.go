package domain

import (
	"reflect"

	tf "github.com/wamuir/graft/tensorflow"
)

// Output represents model output
type Output struct {
	Name         string
	DataType     string
	DataTypeKind reflect.Kind
	Index        int
	*tf.Operation
}
