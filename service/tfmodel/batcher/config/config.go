package config

import "time"

type BatcherConfig struct {
	// MaxBatchSize defines the limit of batch size (input rows) that can be
	// accumulated before the batch will be sent to the model for prediction.
	// If an incoming batch's size is >= MaxBatchSize, then it will be run as
	// its own batch - there's no hard limiting and incoming batch size.
	MaxBatchSize int

	// MaxBatchCounts represent the maximum number of incoming batches to wait
	// for before sending for prediction.
	// If this is set to 1, then service/tfmodel.Service will not start a Batcher.
	MaxBatchCounts int

	// MaxBatchWait indicates maximum wait since the start of the current
	// batch collection.
	// This is not a rolling window.
	MaxBatchWait time.Duration `json:",omitempty" yaml:",omitempty"`

	Verbose *V `json:",omitempty" yaml:",omitempty"`
}

type V struct {
	ID     string
	Output bool
	Input  bool
}
