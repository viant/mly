package config

import (
	"time"

	"github.com/viant/mly/service/tfmodel/batcher/config"
)

type BatcherConfigFile struct {
	config.BatcherConfig
	MaxBatchWaitMicros int
}

func (b *BatcherConfigFile) Init() {
	if b.MaxBatchSize <= 0 {
		b.MaxBatchSize = 100

		// since MaxBatchSize should be >= MaxBatchCounts,
		// if it is set, assume it as an upper bound
		// if MaxBatchCounts is not set, then it's 0,
		// which is < MaxBatchSize, otherwise, it's set
		// and it MaxBatchSize should be = MaxBatchCounts
		if b.MaxBatchSize < b.MaxBatchCounts {
			b.MaxBatchSize = b.MaxBatchCounts
		}
	}

	if b.MaxBatchCounts <= 0 {
		b.MaxBatchCounts = 100

		// since MaxBatchSize should be >= MaxBatchCounts,
		// if it is set, assume it as an upper bound
		// if MaxBatchCounts is not set, then MaxBatchCounts
		// is the desired value
		if b.MaxBatchSize < b.MaxBatchCounts {
			b.MaxBatchCounts = b.MaxBatchSize
		}
	}

	if b.MaxBatchWaitMicros > 0 {
		b.MaxBatchWait = time.Microsecond * time.Duration(b.MaxBatchWaitMicros)
	}

	if b.MaxBatchWait <= 0 {
		b.MaxBatchWait = time.Millisecond * time.Duration(1)
	}

	if b.TimeoutAdjustments != nil {
		b.TimeoutAdjustments.Init()
	}
}
