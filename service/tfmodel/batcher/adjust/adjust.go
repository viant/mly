package adjust

import "time"

// Adjust is not concurrency safe.
type Adjust struct {
	config *AdjustConfig

	target float64
	factor float64

	// factor * max
	growth time.Duration

	CurrentTimeout time.Duration
	ratio          float64

	timeouts uint64
	total    uint64

	t    *time.Ticker
	stop bool
}

func (a *Adjust) Stats(r map[string]interface{}) {
	r["timeout"] = a.CurrentTimeout
	r["ratio"] = a.ratio

	r["total"] = a.total
	r["timeouts"] = a.timeouts
}

func (a *Adjust) Adjust() {
	for !a.stop {
		select {
		case <-a.t.C:
			a.recalculate()
		}
	}

	a.t.Stop()
}

func (a *Adjust) IncTimeout() {
	a.timeouts++
	a.total++
}

func (a *Adjust) IncOverloaded() {

}

func (a *Adjust) Count() {
	a.total++
}

func (a *Adjust) recalculate() {
	if a.total < uint64(a.config.MinCount) {
		return
	}

	ratio := float64(a.timeouts) / float64(a.total)
	a.ratio = ratio

	newTo := a.CurrentTimeout
	if ratio > a.target {
		newTo = a.CurrentTimeout - a.growth
		if newTo < a.config.Min {
			newTo = a.config.Min
		}

	} else {
		newTo = time.Duration(int64(float64(a.CurrentTimeout) * (1 - a.factor)))
		if newTo > a.config.Max {
			newTo = a.config.Max
		}
	}

	a.CurrentTimeout = newTo

	a.total = 0
	a.timeouts = 0
}

func (a *Adjust) Close() {
	a.stop = true
}

// NewAdjust creates a new Adjust with constraints.
func NewAdjust(ac *AdjustConfig) *Adjust {
	if ac == nil {
		return nil
	}

	max := ac.Max
	growth := time.Duration(ac.Factor * float64(max))
	if growth <= 0 {
		growth = time.Duration(1)
	}

	curTO := max
	target := ac.Target
	if target == 0 {
		curTO = ac.Min
	}

	pt := &Adjust{
		config:         ac,
		growth:         growth,
		CurrentTimeout: curTO,
		t:              time.NewTicker(ac.Recalculation),
		stop:           ac.Min >= max || target == 0 || target == 1,
	}

	go pt.Adjust()

	return pt
}
