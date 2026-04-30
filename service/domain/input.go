package domain

import (
	"reflect"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

type Input struct {
	Name string

	// Position of Tensor in model input.
	// Overwritten when the TF signature is parsed.
	Index int

	// TODO refactor out this usage in service/domain.Signature is different from its usage in service/request.Request
	Placeholder tf.Output

	// Vocab is false if embedded vocabulary should be ignored
	// This is used in evaluators that support a deeper graph traversal (TensorFlow).
	Vocab bool

	// Auxiliary is true if this input isn't part of the model
	Auxiliary bool

	Type reflect.Type
}
