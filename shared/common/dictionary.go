package common

import (
	"encoding/binary"
	"hash"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/viant/tapper/io"
)

type (
	// Contains information for bounding cardinality
	Dictionary struct {
		Layers []Layer
		Hash   int // used to check if a model updated
	}

	// Layer represents an encoding layer and its bound cardinality
	Layer struct {
		Name string // should be the name of the input

		Strings   []string
		Ints      []int
		FloatPrec int // acceptable precision 0 means full precision as in practice there should never be a reason to round a float to an integer?

		Hash int // used to check if a model updated
	}
)

// TODO this should use a fixed size integer?
func (d *Dictionary) UpdateHash() int {
	d.Hash = 0
	for i := range d.Layers {
		layer := &d.Layers[i]

		hasher := fnv.New64()
		if len(layer.Strings) > 0 {
			sort.Strings(layer.Strings)
			hashStrings(hasher, layer.Strings)
			layer.Hash = int(hasher.Sum64())
		} else if len(layer.Ints) > 0 {
			sort.Ints(layer.Ints)
			hashInts(hasher, layer.Ints)
			layer.Hash = int(hasher.Sum64())
		} else if layer.FloatPrec > 0 {
			hashInts(hasher, []int{layer.FloatPrec})
			layer.Hash = int(hasher.Sum64())
		} else {
			continue
		}

		d.Hash += layer.Hash
	}
	return d.Hash
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

// implements github.com/viant/tapper/io.Encoder
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
	stream.PutInt("Hash", l.Hash)
}
