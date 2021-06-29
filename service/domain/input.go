package domain

import (
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"reflect"
)

//Input represents model input
type Input struct {
	Name  string
	Index int
	reflect.Type
	Placeholder tf.Output
}
