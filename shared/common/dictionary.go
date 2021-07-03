package common

type (
	//Dictionary represents model dictionary
	Dictionary struct {
		Layers []Layer
		Hash   int
	}

	//Layer represents model layer
	Layer struct {
		Name    string
		Strings []string
		Ints    []int
		Floats  []float32
	}
)
