package adjust

import (
	"sync"
	"time"
)

type Adjust struct {
	config         *AdjustConfig
	CurrentTimeout time.Duration

	l *sync.Mutex
}

func (a *Adjust) Stats(r map[string]interface{}) {
}

func (a *Adjust) Active(active uint32) {
	a.l.Lock()
	defer a.l.Unlock()

	if active <= a.config.Max {
		a.CurrentTimeout = a.config.Increment * time.Duration(active)
	}
}

// NewAdjust creates a new Adjust with constraints.
func NewAdjust(ac *AdjustConfig) *Adjust {
	return &Adjust{
		config: ac,
		l:      new(sync.Mutex),
	}
}
