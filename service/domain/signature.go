package domain

//Signature represents model signature
type Signature struct {
	Method  string
	Inputs  []Input
	Output  Output
	Outputs []Output
}
