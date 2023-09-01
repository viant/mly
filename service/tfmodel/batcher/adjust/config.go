package adjust

import "time"

// AdjustConfig controls exponential timeout backoff in case there is an evaluator
// bottleneck.
type AdjustConfig struct {
	// Base is the initial timeout duration used if the system is considered busy.
	// Use <= 0 for default of 1000ns.
	Base time.Duration

	// Factor is the geometric rate of increase for timeout if the
	// system is considered busy.
	// Use <= 1 for default of 1.5.
	Factor float64
}

func (a *AdjustConfig) Init() {
	if a.Base < 1 {
		a.Base = time.Duration(1000)
	}

	if a.Factor <= 1 {
		a.Factor = 1.5
	}
}
