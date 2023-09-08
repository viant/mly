package adjust

import "time"

type AdjustConfig struct {
	Increment time.Duration
	Max       uint32
}

func (a *AdjustConfig) Init() {
	if a.Increment == 0 {
		a.Increment = 100
	}
}
