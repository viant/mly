package client

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

type fakeLayer struct {
	name    string
	strings []string
	ints    []int
	fp      uint
	wc      bool
	typen   string
}

func makeMessages(fls []fakeLayer) Messages {
	l := make([]common.Layer, len(fls))
	inputs := make([]*shared.Field, len(fls))
	for i, fl := range fls {
		cl := common.Layer{
			Name: fl.name,
		}

		if len(fl.strings) > 0 {
			cl.Strings = fl.strings
		}

		if len(fl.ints) > 0 {
			cl.Ints = fl.ints
		}

		l[i] = cl

		sf := &shared.Field{
			Name:     fl.name,
			Index:    i,
			Wildcard: fl.wc,
			DataType: fl.typen,
		}

		if fl.fp > 0 {
			sf.Precision = int(fl.fp)
		}

		inputs[i] = sf
	}

	commonDict := &common.Dictionary{
		Layers: l,
		Hash:   1,
	}

	makeDict := func() *Dictionary {
		dict := NewDictionary(commonDict, inputs)
		return dict
	}

	return NewMessages(makeDict)
}

func TestMessage_FloatKey(t *testing.T) {
	msgs := makeMessages([]fakeLayer{
		{
			name:  "ft",
			fp:    3,
			typen: "float32",
		},
		{
			name:  "s",
			typen: "string",
			wc:    true,
		},
	})

	msg := msgs.Borrow()
	msg.FloatKey("ft", 1.23456)
	msg.StringKey("s", "a")

	var key string
	key = msg.CacheKey()
	assert.Equal(t, "1.235/a", key)
}

func TestMessage_FloatsKey(t *testing.T) {
	msgs := makeMessages([]fakeLayer{
		{
			name:  "ft",
			fp:    3,
			typen: "float32",
		},
		{
			name:  "s",
			typen: "string",
		},
	})

	msg := msgs.Borrow()
	msg.SetBatchSize(2)
	msg.FloatsKey("ft", []float32{1.23456, 5.6789})
	msg.StringsKey("s", []string{"a", "b"})

	msg.end()
	bytes := msg.Bytes()
	fmt.Printf("%s\n", bytes)

	var key string
	key = msg.CacheKeyAt(0)
	assert.Equal(t, "1.235/[UNK]", key, "key 0")

	key = msg.CacheKeyAt(1)
	assert.Equal(t, "5.679/[UNK]", key, "key 1")
}

func TestMessage(t *testing.T) {
	msgs := makeMessages([]fakeLayer{
		{
			name:  "copied",
			typen: "string",
		},
		{
			name:    "multi",
			strings: []string{"a", "b"},
			typen:   "string",
		},
	})
	msg := msgs.Borrow()

	msg.SetBatchSize(2)

	msg.StringsKey("copied", []string{"1"})
	msg.StringsKey("multi", []string{"a", "b"})

	msg.end()
	bytes := msg.Bytes()
	fmt.Printf("%s\n", bytes)

	var key string
	key = msg.CacheKeyAt(0)
	assert.Equal(t, "[UNK]/a", key)

	key = msg.CacheKeyAt(1)
	assert.Equal(t, "[UNK]/b", key)
}

// multi copied payload
type mcpb struct {
	Multi     []string `json:"multi"`
	Copied    []string `json:"copied"`
	BatchSize int      `json:"batch_size"`
	CacheKeys []string `json:"cache_key"`
}

func TestMessageBug(t *testing.T) {
	msgs := makeMessages([]fakeLayer{
		{
			name:    "multi",
			strings: []string{"a", "b"},
			typen:   "string",
		},
		{
			name:  "copied",
			typen: "string",
		},
	})

	m := msgs.Borrow()
	m.start()

	m.SetBatchSize(3)

	m.StringsKey("multi", []string{"a", "b", "c"})
	m.StringsKey("copied", []string{"1"})

	m.FlagCacheHit(0)

	err := m.end()
	assert.Nil(t, err)

	bytes := m.Bytes()

	pl := new(mcpb)
	json.Unmarshal(bytes, pl)

	fmt.Printf("%+v\n", pl)
	assert.Equal(t, []string{"1"}, pl.Copied)

	assert.Equal(t, []string{"b", "c"}, pl.Multi)
}
