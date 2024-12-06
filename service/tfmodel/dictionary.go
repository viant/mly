package tfmodel

import (
	"fmt"
	"sort"
	"unsafe"

	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/common"
	tf "github.com/wamuir/graft/tensorflow"
)

// Dictionary uses domain.Signature to determine which inputs should have an encoding lookup
// from the Tensorflow graph.
// TODO pull out *domain.Signature, just use a slice of something.
func Dictionary(session *tf.Session, graph *tf.Graph, signature *domain.Signature) (*common.Dictionary, error) {
	var layers []string
	for _, input := range signature.Inputs {
		if !input.Vocab {
			continue
		}

		layers = append(layers, input.Name)
	}

	dictionary, err := DiscoverDictionary(session, graph, layers)
	if err != nil {
		return dictionary, err
	}

	return dictionary, nil
}

// DiscoverDictionary extracts vocabulary from from best guessed layer and operations.
func DiscoverDictionary(session *tf.Session, graph *tf.Graph, layers []string) (*common.Dictionary, error) {
	var result = new(common.Dictionary)
	for _, name := range layers {
		exported, err := Export(session, graph, name)
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

	return result, nil
}
