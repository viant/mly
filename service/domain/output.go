package domain

import (
	"reflect"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

// Output represents model output
type Output struct {
	// Used in Stream
	Name string

	// Shown in config
	DataType string

	// Only for GBQ tool
	DataTypeKind reflect.Kind

	// Used to extract output from *tf.Operation.
	// Eventually becomes part of tf.Session.Run() parameter fetches ([]tf.Output).
	Index int

	*tf.Operation

	goType reflect.Type
}

func (o *Output) SetType(oType reflect.Type) {
	o.goType = oType
}

func (o *Output) Type() reflect.Type {
	return o.goType
}
