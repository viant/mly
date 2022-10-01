package common

import "github.com/viant/tapper/io"

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

	Layers []Layer
)

func (l *Layers) Encoders() []io.Encoder {
	var layers = make([]io.Encoder, len(*l))
	for i := range *l {
		layers[i] = &(*l)[i]
	}
	return layers
}

func (l *Layer) Encode(stream io.Stream) {
	stream.PutByte('\n')
	stream.PutString("Name", l.Name)
	if len(l.Strings) > 0 {
		stream.PutStrings("Strings", l.Strings)
	}
	if len(l.Ints) > 0 {
		stream.PutInts("Ints", l.Ints)
	}
	if len(l.Floats) > 0 {
		var floats = make([]float64, len(l.Floats))
		for i, f := range l.Floats {
			floats[i] = float64(f)
		}
		stream.PutFloats("Floats", floats)
	}

}
