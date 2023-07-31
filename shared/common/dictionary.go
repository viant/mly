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

		Strings []string
		Ints    []int

		Hash int // memoization; used to check if a model updated
	}
)

// TODO this should use a fixed size integer?
// UpdateHash will memoize dictionary hashing.
// Since wildcard fields don't provide an actual dictionary, we use the modification time information to generate a hash based on the file, passed in as fsHash.
func (d *Dictionary) UpdateHash(fsHash int64) int {
	d.Hash = int(fsHash)

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
		}

		// precision changes will result in a change in hash as the hash's goal
		// is to capture changes that would occur where the cache key is the same
		// but the value is supposed to be different.
		// a precision change would result in a different key.

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
