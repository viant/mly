package domain

// Signature represents Tensorflow SavedModel function Signature.
// Contains information required to extract vocabularies, unmarshal requests, and validate request inputs.
// TODO document and address issues if reloaded model IO changes.
type Signature struct {
	Method string

	Inputs  []Input
	Outputs []Output

	Output Output // Deprecated: Use Outputs[0] if there is only 1 output
}
