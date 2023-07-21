package domain

// Signature represents Tensorflow SavedModel function Signature
type Signature struct {
	Method  string
	Inputs  []Input
	Output  Output // Deprecated: Use Outputs[0] if there is only 1 output
	Outputs []Output
}
