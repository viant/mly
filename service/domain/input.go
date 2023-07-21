package domain

import (
	"reflect"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

type Input struct {
	Name  string
	Index int

	Placeholder tf.Output // TODO refactor out this usage in service/domain.Signature is different from its usage in service/request.Request

	Vocab     bool // false if embedded vocabulary should be ignored
	Auxiliary bool // true if this input isn't part of the model

	reflect.Type
}
