package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/viant/mly/service/tfmodel/batcher/adjust"
)

var sysMaxQueuedBatches int

func init() {
	mqbs := os.Getenv("DISPATCHER_MAX_QUEUED_BATCHES")
	envMQB, err := strconv.Atoi(mqbs)
	if err != nil {
		log.Printf("could not Atoi %s use default 16384", mqbs)
		envMQB = 16384
	}

	sysMaxQueuedBatches = envMQB

}

type BatcherConfig struct {
	// MaxBatchSize is a hard limit to the number of elements in a batch
	// that we attempt to send to the model for prediction.
	// Currently will indicate a minimum batch size, i.e. a batch will be forced
	// if the current batch size is >= MaxBatchSize.
	// TODO make this an exact constraint.
	// If this is <= 0, then the maximum size of a batch will be unbounded.
	// This constraint will always force a batch to be queued.
	MaxBatchSize int `json:",omitempty" yaml:",omitempty"`

	// BatchWait indicates the maximum amount of time to wait to collect
	// requests to make a batch.
	// This is not a rolling window.
	// Set to <= 0 to effectively turn off batching.
	// This constraint may sometimes cause a batch to be queued.
	// If MaxEvaluatorConcurrency is NOT set then the batch will be queued
	// (and run) as soon MaxBatchSize is hit (if set) or BatchWait elapsed.
	// If MaxEvaluatorConcurrency is set and the system is busy, then
	// BatchWait will be ignored until either MaxBatchSize is hit, or an
	// Evaluator is free while the batch continues to grow in size.
	BatchWait time.Duration `json:",omitempty" yaml:",omitempty"`

	// MaxEvaluatorConcurrency determines the maximum number of evaluators to
	// have running.
	// Set to 0 to have it unbounded.
	// If set to > 0, but MaxBatchSize is not set, then requests will queue into
	// a single "mega-batch," and MaxQueuedBatches is ignored.
	// If set to > 0, and MaxBatchSize is set, then batches will queue until
	// MaxQueuedBatches is hit.
	MaxEvaluatorConcurrency uint32 `json:",omitempty" yaml:",omitempty"`

	// MaxQueuedBatches works IFF both MaxEvaluatorConcurrency and
	// MaxBatchSize are set.
	// Requests will be rejected if the number of batches queued to be evaluated
	// exceeds MaxQueuedBatches.
	// In the case that there are too many queued batches, the server will start to reject
	// requests with HTTP code 429.
	MaxQueuedBatches int `json:",omitempty" yaml:",omitempty"`

	TimeoutAdjustments *adjust.AdjustConfig `json:",omitempty" yaml:",omitempty"`

	Verbose *V `json:",omitempty" yaml:",omitempty"`
}

type V struct {
	ID     string
	Output bool
	Input  bool
}

func (b *BatcherConfig) Init() {
	if b.MaxBatchSize < 0 {
		b.MaxBatchSize = 0
	}

	if b.BatchWait < 0 {
		b.BatchWait = 0
	}

	if b.MaxEvaluatorConcurrency < 0 {
		b.MaxEvaluatorConcurrency = 0
	}

	if b.MaxQueuedBatches <= 0 {
		b.MaxQueuedBatches = sysMaxQueuedBatches
	}

	if b.TimeoutAdjustments != nil {
		b.TimeoutAdjustments.Init()
	}
}
