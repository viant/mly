package config

import (
	"time"

	"github.com/viant/mly/service/tfmodel/batcher/adjust"
)

type BatcherConfig struct {
	// MinBatchSize defines the lower bound of batch size (input rows) that can be
	// accumulated before we attempt to send the batch to the model for prediction.
	// The batch size may exceed this number if MaxEvaluatorconcurrency prevents
	// an immediate prediction.
	MinBatchSize int

	// MinBatchCounts represent the minimum number of incoming batches to wait
	// for before attempting to send for prediction.
	// If this is set to 1, then service/tfmodel.Service will not start a Batcher.
	// if this is unset or set to 0, then it will use a default of 100.
	// The batch count may exceed this number if MaxEvaluatorconcurrency prevents
	// an immediate prediction.
	MinBatchCounts int

	// MinBatchWait indicates minimum wait since the start of the current
	// batch collection.
	// This is not a rolling window.
	// The wait time may exceed this number if MaxEvaluatorconcurrency prevents
	// an immediate prediction.
	MinBatchWait time.Duration `json:",omitempty" yaml:",omitempty"`

	// MaxEvaluatorConcurrency determines the maximum number of predictions to
	// have running.
	// If the number of max evaluators is too high, then batching will continue
	// even if MinBatchSize or MinBatchCounts have been passed or MinBatchWait
	// has been waited.
	// Set to 0 to have it unbounded.
	MaxEvaluatorConcurrency uint32 `json:",omitempty" yaml:",omitempty"`

	TimeoutAdjustments *adjust.AdjustConfig `json:",omitempty" yaml:",omitempty"`

	Verbose *V `json:",omitempty" yaml:",omitempty"`
}

type V struct {
	ID     string
	Output bool
	Input  bool
}

func (b *BatcherConfig) Init() {
	if b.MinBatchSize <= 0 {
		b.MinBatchSize = 100

		// since MaxBatchSize should be >= MaxBatchCounts,
		// if it is set, assume it as an upper bound
		// if MaxBatchCounts is not set, then it's 0,
		// which is < MaxBatchSize, otherwise, it's set
		// and it MaxBatchSize should be = MaxBatchCounts
		if b.MinBatchSize < b.MinBatchCounts {
			b.MinBatchSize = b.MinBatchCounts
		}
	}

	if b.MinBatchCounts <= 0 {
		b.MinBatchCounts = 100

		// since MaxBatchSize should be >= MaxBatchCounts,
		// if it is set, assume it as an upper bound
		// if MaxBatchCounts is not set, then MaxBatchCounts
		// is the desired value
		if b.MinBatchSize < b.MinBatchCounts {
			b.MinBatchCounts = b.MinBatchSize
		}
	}

	if b.MinBatchWait <= 0 {
		b.MinBatchWait = time.Millisecond * time.Duration(1)
	}

	if b.TimeoutAdjustments != nil {
		b.TimeoutAdjustments.Init()
	}
}
