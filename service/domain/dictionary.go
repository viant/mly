package domain

type (
	Dictionary struct {
		Layers []Layer
		Hash   int
	}

	Layer struct {
		Name    string
		Strings []string
		Ints    []int
		Floats  []float32
	}
)
