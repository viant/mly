package adjust

import "time"

type AdjustConfig struct {
	// Increment is multiplied by the number of active evalutors to determine
	// the active timeout.
	Increment time.Duration

	// Max is the maximum Increment multiplier that can be applied as the
	// current timeout.
	// If service/tfmodel/batcher/config.BatcherConfig.MaxEvaluatorConcurrency
	// is set and it is less than Max, then Max will be set to
	// service/tfmodel/batcher/config.BatcherConfig.MaxEvaluatorConcurrency
	Max uint32
}

// Init sets defaults.
func (a *AdjustConfig) Init() {
	if a.Increment == 0 {
		a.Increment = 100
	}
}
