package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

func TestMessage(t *testing.T) {
	commonDict := &common.Dictionary{
		Layers: []common.Layer{
			common.Layer{
				Name:    "copied",
				Strings: []string{},
			},
			common.Layer{
				Name:    "multi",
				Strings: []string{"a", "b"},
			},
		},
		Hash: 1,
	}

	inputs := []*shared.Field{
		&shared.Field{
			Name:     "copied",
			Index:    0,
			DataType: "string",
			//Wildcard: true,
		},
		&shared.Field{
			Name:     "multi",
			Index:    1,
			DataType: "string",
		},
	}

	makeDict := func() *Dictionary {
		dict := NewDictionary(commonDict, inputs)
		return dict
	}

	msgs := NewMessages(makeDict)
	msg := msgs.Borrow()

	msg.SetBatchSize(2)

	msg.StringsKey("multi", []string{"a", "b"})
	msg.StringsKey("copied", []string{"1"})

	var key string
	key = msg.CacheKeyAt(0)
	assert.Equal(t, "[UNK]/a", key)

	key = msg.CacheKeyAt(1)
	assert.Equal(t, "[UNK]/b", key)
}
