package domain

import (
	"reflect"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

//Output represents model output
type Output struct {
	Name         string
	DataType     string
	DataTypeKind reflect.Kind
	Index        int
	*tf.Operation
}
