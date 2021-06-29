package domain

import (
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

type Output struct {
	Name     string
	DataType string
	*tf.Operation
}
