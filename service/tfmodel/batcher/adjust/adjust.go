package adjust

import "time"

// TODO
type Adjust struct {
	config *AdjustConfig

	CurrentTimeout time.Duration
}

func (a *Adjust) Stats(r map[string]interface{}) {
}

func (a *Adjust) Adjust() {
}

func (a *Adjust) IncTimeout() {
}

func (a *Adjust) IncOverloaded() {
}

func (a *Adjust) Count() {
}

func (a *Adjust) recalculate() {
}

func (a *Adjust) Close() {
}

// NewAdjust creates a new Adjust with constraints.
func NewAdjust(ac *AdjustConfig) *Adjust {
	return nil
}
