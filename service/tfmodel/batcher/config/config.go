package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/viant/mly/service/tfmodel/batcher/adjust"
)

// SysMaxQueuedBatches will be used as the channel size for
// unbounded batch queues.
// Can be overridden by environment variable DISPATCHER_MAX_QUEUED_BATCHES.
var SysMaxQueuedBatches int

func init() {
	mqbs := os.Getenv("DISPATCHER_MAX_QUEUED_BATCHES")

	envMQB := 16384
	var err error
	if mqbs != "" {
		envMQB, err = strconv.Atoi(mqbs)
	}

	if err != nil {
		log.Printf("could not Atoi %s use default 16384", mqbs)
		envMQB = 16384
	}

	SysMaxQueuedBatches = envMQB
}

// BatcherConfig provides a fixed set of parameters for the batching system.
// By default, if no properties are explicitly declared, then a only
// the BatchWait will be set to 500 microseconds.
type BatcherConfig struct {
	// MaxBatchSize is a hard limit to the number of elements in a batch
	// that we attempt to send to the model for prediction.
	// Currently will indicate a minimum batch size, i.e. a batch will be forced
	// if the current batch size is >= MaxBatchSize.
	//
	// TODO make this an exact constraint.
	//
	// If this is <= 0, then the maximum size of a batch will be unbounded.
	// This constraint will always force a batch to be queued.
	MaxBatchSize int `json:",omitempty" yaml:",omitempty"`

	// BatchWait indicates the maximum amount of time to wait to collect
	// requests to make a batch.
	// This is not a rolling window.
	//
	// Set to < 0 to turn off batching.
	// If this is 0, then a default of 500 * time.Microsecond will be used.
	// This constraint may sometimes cause a batch to be queued.
	// If there TimeoutAdjustments are provided, then this value
	// will be ignored.
	//
	// If MaxEvaluatorConcurrency is NOT set then the batch will be queued
	// (and run) as soon MaxBatchSize is hit (if set) or BatchWait elapsed.
	BatchWait time.Duration `json:",omitempty" yaml:",omitempty"`

	// MaxEvaluatorConcurrency determines the maximum number of evaluators to
	// have running.
	//
	// Set to 0 to have it unbounded - it will use SysMaxQueuedBatches
	// as channel size.
	// If set to > 0, but MaxBatchSize is not set, then requests will queue into
	// a single "mega-batch," and MaxQueuedBatches is ignored.
	// If set to > 0, and MaxBatchSize is set, then batches will queue until
	// MaxQueuedBatches is hit.
	MaxEvaluatorConcurrency int `json:",omitempty" yaml:",omitempty"`

	// MaxQueuedBatches works IFF both MaxEvaluatorConcurrency and
	// MaxBatchSize are set.
	//
	// Requests will be rejected if the number of batches queued to be evaluated
	// exceeds MaxQueuedBatches.
	//
	// In the case that there are too many queued batches, the server will start to reject
	// requests with HTTP code 429.
	MaxQueuedBatches int `json:",omitempty" yaml:",omitempty"`

	TimeoutAdjustments *adjust.AdjustConfig `json:",omitempty" yaml:",omitempty"`

	// Verbose enables a lot of debugging messages.
	Verbose *V `json:",omitempty" yaml:",omitempty"`
}

type V struct {
	ID     string
	Output bool
	Input  bool
}

func (v *V) Debug(tag string, msg string) {
	if v == nil {
		return
	}

	log.Printf("[%s %s] %s", v.ID, tag, msg)
}

func (v *V) DebugFn(tag string, mfn func() string, second func() bool) {
	if v == nil {
		return
	}

	if second == nil || second() {
		log.Printf("[%s %s] %s", v.ID, tag, mfn())
	}
}

func (v *V) InputEnabled() func() bool {
	if v == nil {
		return nil
	}

	return func() bool { return v.Input }
}

func (b *BatcherConfig) Init() {
	if b.MaxBatchSize < 0 {
		b.MaxBatchSize = 0
	}

	if b.BatchWait < 0 {
		b.BatchWait = 0
	} else if b.BatchWait == 0 {
		b.BatchWait = 500 * time.Microsecond
	}

	if b.MaxEvaluatorConcurrency < 0 {
		b.MaxEvaluatorConcurrency = 0
	}

	if b.MaxQueuedBatches <= 0 {
		b.MaxQueuedBatches = SysMaxQueuedBatches
	}

	ta := b.TimeoutAdjustments
	if ta != nil {
		ta.Init()

		if b.MaxEvaluatorConcurrency > 0 && ta.Max > uint32(b.MaxEvaluatorConcurrency) {
			ta.Max = uint32(b.MaxEvaluatorConcurrency)
		}
	}
}
