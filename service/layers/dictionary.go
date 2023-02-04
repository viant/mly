package layers

import (
	"encoding/binary"
	"fmt"
	"hash"
	"sort"
	"unsafe"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/common"
)

// Dictionary generates dictionary from TF model
func Dictionary(session *tf.Session, graph *tf.Graph, signature *domain.Signature) (*common.Dictionary, error) {
	var layers []string
	for _, input := range signature.Inputs {
		if input.Wildcard {
			continue
		}

		layer := input.Name
		if input.Layer != "" {
			layer = input.Layer
		}
		layers = append(layers, layer)
	}
	dictionary, err := DiscoverDictionary(session, graph, layers)
	if err != nil {
		return dictionary, err
	}
	return dictionary, nil
}

// DiscoverDictionary extracts vocabulary from layers
func DiscoverDictionary(session *tf.Session, graph *tf.Graph, layers []string) (*common.Dictionary, error) {
	var result = &common.Dictionary{}
	for _, name := range layers {
		exported, err := tfmodel.Export(session, graph, name)
		if err != nil {
			return nil, err
		}
		layer := common.Layer{
			Name: name,
		}
		switch vals := exported.(type) {
		case []string:
			layer.Strings = make([]string, len(vals))
			copy(layer.Strings, vals)
			sort.Strings(layer.Strings)
		case []int64:
			layer.Ints = make([]int, len(vals))
			copy(layer.Ints, *(*[]int)(unsafe.Pointer(&vals)))
			sort.Ints(layer.Ints)
		default:
			return nil, fmt.Errorf("unsupported data type %T for %v", exported, name)
		}
		result.Layers = append(result.Layers, layer)
	}
	result.UpdateHash()
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
