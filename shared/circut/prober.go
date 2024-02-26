package circut

// Prober represents abstraction to check if service is up.
// Types implementing this interface should also have access to the Breaker
// that will call Prober.Probe().
type Prober interface {
	// Probe checks if service is up, if so it should call FlagUp
	Probe()
}
