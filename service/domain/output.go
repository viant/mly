package domain

import (
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

//Output represents model output
type Output struct {
	Name     string
	DataType string
	Index    int
	*tf.Operation
}
