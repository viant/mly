package circut

//Prober represents abstraction to check if service is up
type Prober interface {
	//Probe checks if service is up, if so it should call FlagUp
	Probe()
}
