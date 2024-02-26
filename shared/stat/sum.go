package stat

import "sync/atomic"

// TODO consider https://github.com/beorn7/perks
type Sum struct {
	Total uint64 `json:",omitempty"`
}

type Count uint64

// Implements counter.CustomCounter so that it can be used by counter.CustomCounter
func (c Count) Aggregate(value interface{}) {}

func (s *Sum) Aggregate(value interface{}) {
	c, ok := value.(Count)
	if !ok {
		return
	}

	atomic.AddUint64(&s.Total, uint64(c))
}
