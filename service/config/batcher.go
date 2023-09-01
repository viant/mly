package config

import (
	"time"

	"github.com/viant/mly/service/tfmodel/batcher/config"
)

type BatcherConfigFile struct {
	config.BatcherConfig
	BatchWaitMicros int
}

func (b *BatcherConfigFile) Init() {
	if b.BatchWaitMicros > 0 {
		b.BatchWait = time.Microsecond * time.Duration(b.BatchWaitMicros)
	}

	b.BatcherConfig.Init()
}
