package domain

import (
	"reflect"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

// Input represents model and request input
type Input struct {
	Name  string
	Index int
	reflect.Type
	Placeholder tf.Output
	Wildcard    bool // true if embedded vocabulary should be ignored
	Auxiliary   bool // true if this input isn't part of the model
	Layer       string
}
