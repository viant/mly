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
	}

	if b.MaxBatchCounts <= 0 {
		b.MaxBatchCounts = 100
	}

	if b.MaxBatchWaitMicros > 0 {
		b.MaxBatchWait = time.Microsecond * time.Duration(b.MaxBatchWaitMicros)
	}

	if b.MaxBatchWait <= 0 {
		b.MaxBatchWait = time.Millisecond * time.Duration(1)
	}
}
