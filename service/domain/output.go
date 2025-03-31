package domain

import (
	"reflect"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

// Output represents model output
type Output struct {
	Name string

	// Primarily shown in config
	DataType string

	// DataTypeKind is used only for GBQ tool
	DataTypeKind reflect.Kind
	Index        int
	*tf.Operation

	goType reflect.Type
}

func (o *Output) SetType(oType reflect.Type) {
	o.goType = oType
}

func (o *Output) Type() reflect.Type {
	return o.goType
}
