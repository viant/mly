package client

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

func TestReconcileData(t *testing.T) {

	messages := NewMessages(func() *Dictionary {
		return NewDictionary(&common.Dictionary{}, []*shared.Field{
			{
				Name:     "initMessage",
				Wildcard: true,
			},
		})
	})

	type Prediction struct {
		Output float32
	}

	var testCases = []struct {
		description string
		Cachable    func() Cachable
		cached      []interface{}
		target      func() interface{}
		expect      interface{}
	}{

		{
			description: "singleKey: nothing cached",
			cached:      []interface{}{},
			Cachable: func() Cachable {
				msg := messages.Borrow()
				msg.StringKey("initMessage", "ab")
				return msg
			},
			target: func() interface{} {
				var result = Prediction{
					Output: 1.13,
				}
				return &result
			},
			expect: Prediction{
				Output: 1.13,
			},
		},
		{
			description: "singleKey: cached",
			cached: []interface{}{
				&Prediction{
					Output: 3.57,
				},
			},
			Cachable: func() Cachable {
				msg := messages.Borrow()
				msg.StringKey("initMessage", "ab")
				msg.FlagCacheHit(0)

				return msg
			},
			target: func() interface{} {
				var result = Prediction{
					Output: 1.13,
				}
				return &result
			},
			expect: Prediction{
				Output: 3.57,
			},
		},

		{
			description: "multiKey: first cached",
			cached: []interface{}{
				&Prediction{
					Output: 1.13,
				},
				&Prediction{
					Output: 2.37,
				},
				&Prediction{
					Output: 3.41,
				},
				nil,
			},

			Cachable: func() Cachable {
				msg := messages.Borrow()
				msg.StringsKey("initMessage", []string{"ab", "cd", "ef", "yz"})
				msg.FlagCacheHit(0)
				msg.FlagCacheHit(1)
				msg.FlagCacheHit(2)
				return msg
			},
			target: func() interface{} {
				var result = []*Prediction{

					{
						Output: 8.42,
					},
				}
				return &result
			},
			expect: []*Prediction{
				{
					Output: 1.13,
				},
				{
					Output: 2.37,
				},
				{
					Output: 3.41,
				},
				{
					Output: 8.42,
				},
			},
		},
		{
			description: "multiKey: all cached",
			cached: []interface{}{
				&Prediction{
					Output: 1.1,
				},
				&Prediction{
					Output: 2.1,
				},
				&Prediction{
					Output: 3.4,
				},
				&Prediction{
					Output: 1.42,
				},
			},

			Cachable: func() Cachable {
				msg := messages.Borrow()
				msg.StringsKey("initMessage", []string{"ab", "cd", "ef", "yz"})
				msg.FlagCacheHit(0)
				msg.FlagCacheHit(1)
				msg.FlagCacheHit(2)
				msg.FlagCacheHit(3)

				return msg
			},
			target: func() interface{} {
				var result = []*Prediction{}
				return &result
			},
			expect: []*Prediction{
				{
					Output: 1.1,
				},
				{
					Output: 2.1,
				},
				{
					Output: 3.4,
				},
				{
					Output: 1.42,
				},
			},
		},

		{
			description: "multiKey: partially cached",
			cached: []interface{}{
				&Prediction{
					Output: 1.11,
				},
				nil,
				&Prediction{
					Output: 3.41,
				},
				nil,
			},

			Cachable: func() Cachable {
				msg := messages.Borrow()
				msg.StringsKey("initMessage", []string{"ab", "cd", "ef", "yz"})
				msg.FlagCacheHit(0)
				msg.FlagCacheHit(2)
				return msg
			},
			target: func() interface{} {
				var result = []*Prediction{
					{
						Output: 2.1,
					},
					{
						Output: 1.42,
					},
				}
				return &result
			},
			expect: []*Prediction{
				{
					Output: 1.11,
				},
				{
					Output: 2.1,
				},
				{
					Output: 3.41,
				},
				{
					Output: 1.42,
				},
			},
		},
		{
			description: "multiKey: all cached",
			cached: []interface{}{
				&Prediction{
					Output: 1.1,
				},
				&Prediction{
					Output: 2.1,
				},
				&Prediction{
					Output: 3.4,
				},
				&Prediction{
					Output: 1.42,
				},
			},

			Cachable: func() Cachable {
				msg := messages.Borrow()
				msg.StringsKey("initMessage", []string{"ab", "cd", "ef", "yz"})
				msg.FlagCacheHit(0)
				msg.FlagCacheHit(1)
				msg.FlagCacheHit(2)
				msg.FlagCacheHit(3)

				return msg
			},
			target: func() interface{} {
				var result = []*Prediction{}
				return &result
			},
			expect: []*Prediction{
				{
					Output: 1.1,
				},
				{
					Output: 2.1,
				},
				{
					Output: 3.4,
				},
				{
					Output: 1.42,
				},
			},
		},
		{
			description: "multiKey: nothing cached",
			cached:      []interface{}{},
			Cachable: func() Cachable {
				msg := messages.Borrow()
				msg.StringsKey("initMessage", []string{"ab", "cd", "ef", "yz"})
				return msg
			},
			target: func() interface{} {
				var result = []*Prediction{
					{
						Output: 1.1,
					},
					{
						Output: 2.1,
					},
					{
						Output: 3.4,
					},
					{
						Output: 1.42,
					},
				}
				return &result
			},
			expect: []*Prediction{
				{
					Output: 1.1,
				},
				{
					Output: 2.1,
				},
				{
					Output: 3.4,
				},
				{
					Output: 1.42,
				},
			},
		},
	}

	for _, testCase := range testCases {
		target := testCase.target()
		err := reconcileData(true, target, testCase.Cachable(), testCase.cached)
		assert.Nil(t, err, testCase.description)
		actual := reflect.ValueOf(target).Elem().Interface()
		assert.EqualValues(t, testCase.expect, actual, testCase.description)
	}

}
