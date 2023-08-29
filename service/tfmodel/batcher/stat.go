package batcher

import (
	"sync/atomic"

	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	Closing      = "closing"
	MaxBatches   = "maxBatches"
	FullElements = "fullElements"
	Waiting      = "waiting"

	elements = "elements"
)

type batchStat struct {
	key string

	Elements uint64
	Batches  uint64
}

func mkbs(key string, e, b int) *batchStat {
	return &batchStat{key, uint64(e), uint64(b)}
}

func (s *batchStat) Aggregate(value interface{}) {
	var other batchStat
	switch maybe := value.(type) {
	case *batchStat:
		other = *maybe
	default:
		return
	}

	atomic.AddUint64(&s.Elements, other.Elements)
	atomic.AddUint64(&s.Batches, other.Batches)
}

type dispatcherStats struct{}

func (d dispatcherStats) Keys() []string {
	return []string{
		stat.Timeout,
		Closing,
		MaxBatches,
		FullElements,
		Waiting,
	}
}

func (d dispatcherStats) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	switch v := value.(type) {
	case string:
		switch v {
		case stat.Timeout:
			return 0
		case Closing:
			return 1
		case MaxBatches:
			return 2
		case FullElements:
			return 3
		case Waiting:
			return 4
		}
	case *batchStat:
		switch v.key {
		case stat.Timeout:
			return 0
		case Closing:
			return 1
		case MaxBatches:
			return 2
		case FullElements:
			return 3
		case Waiting:
			return 4
		}
	}

	return -1
}

func (d dispatcherStats) NewCounter() counter.CustomCounter {
	return new(batchStat)
}

func NewDispatcherP() counter.Provider {
	return dispatcherStats{}
}
