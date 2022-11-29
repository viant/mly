package common

import (
	"encoding/binary"
	"github.com/viant/tapper/io"
	"hash"
	"hash/fnv"
	"sort"
	"strings"
)

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
		Hash    int
	}

	Layers []Layer
)

func (d *Dictionary) UpdateHash() int {
	d.Hash = 0
	for i := range d.Layers {
		layer := &d.Layers[i]
		aHash := fnv.New64()
		if len(layer.Strings) > 0 {
			sort.Strings(layer.Strings)
			hashStrings(aHash, layer.Strings)
			layer.Hash = int(aHash.Sum64())
		} else if len(layer.Ints) > 0 {
			sort.Ints(layer.Ints)
			hashInts(aHash, layer.Ints)
			layer.Hash = int(aHash.Sum64())
		} else {
			continue
		}
		d.Hash += layer.Hash
	}
	return d.Hash
}

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
		values := normalizeStrings(l.Strings)
		stream.PutStrings("Strings", values)
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
	stream.PutInt("Hash", l.Hash)
}

func normalizeStrings(items []string) []string {
	for i := range items {
		s := items[i]
		s = strings.ReplaceAll(s, "\"", "\\\"")
		s = strings.ReplaceAll(s, "\n", "\\\n")
		items[i] = s
	}
	return items
}

func hashStrings(hash hash.Hash, strings []string) {
	for _, item := range strings {
		hash.Write([]byte(item))
	}
}

func hashInts(hash hash.Hash, ints []int) {
	for _, item := range ints {
		binary.Write(hash, binary.LittleEndian, item)
	}
}
