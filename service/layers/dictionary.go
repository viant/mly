package layers

import (
	"encoding/binary"
	"fmt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/tfmodel"
	"hash"
	"hash/fnv"
	"sort"
	"unsafe"
)

func Dictionary(signature *domain.Signature, graph *tf.Graph) (*domain.Dictionary, error) {

	var layers []string
	for _, input := range signature.Inputs {
		layers = append(layers, input.Name)
	}
	dictionary, err := discoverDictionary(graph, layers)
	if err != nil {
		return dictionary, err
	}
	return dictionary, nil
}

func discoverDictionary(graph *tf.Graph, layers []string) (*domain.Dictionary, error) {
	var result = &domain.Dictionary{}
	for _, name := range layers {
		aHash := fnv.New64()
		exported, err := tfmodel.Export(graph, name)
		if err != nil {
			return nil, err
		}
		layer := domain.Layer{
			Name: name,
		}
		hashValue := uint64(0)
		switch vals := exported.(type) {
		case []string:
			layer.Strings = make([]string, len(vals))
			copy(layer.Strings, vals)
			sort.Strings(layer.Strings)
			hashStrings(aHash, layer.Strings)
			hashValue = aHash.Sum64()
		case []int64:
			layer.Ints = make([]int, len(vals))
			copy(layer.Ints, *(*[]int)(unsafe.Pointer(&vals)))
			sort.Ints(layer.Ints)
			hashInts(aHash, layer.Ints)
			hashValue = aHash.Sum64()
		default:
			return nil, fmt.Errorf("unsupported data type %T for %v", exported, name)
		}
		result.Layers = append(result.Layers, layer)
		result.Hash += int(hashValue)
	}
	return result, nil
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
