package stream

import "github.com/viant/tapper/io"

const propertyKey string = "output"

type KVStrings []string

// implements github.com/viant/tapper/io.Encoder
func (s KVStrings) Encode(m io.Stream) {
	m.PutStrings(propertyKey, []string(s))
}

type KVInts []int

// implements github.com/viant/tapper/io.Encoder
func (s KVInts) Encode(m io.Stream) {
	m.PutInts(propertyKey, []int(s))
}

type KVFloat64s []float64

// implements github.com/viant/tapper/io.Encoder
func (s KVFloat64s) Encode(m io.Stream) {
	m.PutFloats(propertyKey, []float64(s))
}
