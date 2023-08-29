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
	if b.MaxBatchWaitMicros > 0 {
		b.MinBatchWait = time.Microsecond * time.Duration(b.MaxBatchWaitMicros)
	}

	b.BatcherConfig.Init()
}
