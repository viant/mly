package adjust

import "time"

type AdjustConfig struct {
	// Max is the maximum timeout that can be used.
	Max time.Duration

	// Min is the minimum timeout that can be used. Will be [1, Max].
	// If Min >= Max, then adjustment will be turned off.
	Min time.Duration

	// Recalculation is the time.Duration that the Adjust will wait to calculate a new timeout.
	Recalculation time.Duration

	// Target is the the ratio of timeouts within the recalculation duration
	// that will trigger a timeout adjustment.
	// If the ratio of timeouts is greater than the Target, then the timeout
	// duration will be shortened linearly.
	// If the ratio of timeouts is less than the Target, then the timeout
	// duration will be lengthened geometrically.
	// Must be between [0, 1].
	// If Target is 1 then adjustments will be disabled.
	// If Target is 0 then the Min timeout will be used and adjustments disabled.
	// Any value out of bounds will be set to the closest bound.
	Target float64

	// Factor is the ratio of adjustment to the timeout duration.
	// If shortening, it is determined as a ratio of the max time.Duration, and is fixed
	// on Adjust instantiation.
	// If lengthening, it is determined as the ratio of the current timeout plus one.
	// Must be between (0, 1].
	// Use <= 0 for default of 0.05.
	Factor float64

	// MinCount is the minimum number of samples required before making
	// an adjustment.
	// Use 0 for default of 10.
	MinCount uint
}

func (a *AdjustConfig) Init() {
	if a.Min < 1 {
		a.Min = 1
	}

	if a.Recalculation <= 0 {
		a.Recalculation = time.Second
	}

	if a.Target < 0 {
		a.Target = 0
	}

	if a.Target > 1 {
		a.Target = 1
	}

	if a.Factor <= 0 {
		a.Factor = 0.05
	}

	if a.MinCount <= 0 {
		a.MinCount = 10
	}
}
