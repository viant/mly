package test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	_, evaluator, _ := TLoadEvaluator(t, "example/model/string_lookups_int_model")

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}})
	feeds = append(feeds, [][]string{{"c"}})

	_, err := evaluator.Evaluate(context.Background(), feeds)
	assert.Nil(t, err)
}

func TestBasicV2(t *testing.T) {
	_, evaluator, _ := TLoadEvaluator(t, "example/model/vectorization_int_model")

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a b c"}, {"b d d"}})
	feeds = append(feeds, [][]string{{"c"}, {"d"}})

	_, err := evaluator.Evaluate(context.Background(), feeds)
	assert.Nil(t, err)
}

func BenchmarkEvaluatorParallel(b *testing.B) {
	bnil := func(err error) {
		if err != nil {
			b.Error(err)
		}
	}

	_, evaluator, _ := LoadEvaluator("example/model/string_lookups_int_model", bnil, bnil)

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}, {"b"}})
	feeds = append(feeds, [][]string{{"c"}, {"d"}})

	evaluator.Evaluate(context.Background(), feeds)

	wg := new(sync.WaitGroup)

	var errors int32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg.Add(1)

			go func() {
				_, err := evaluator.Evaluate(context.Background(), feeds)
				if err != nil {
					atomic.AddInt32(&errors, 1)
				}

				wg.Done()
			}()
		}
		wg.Wait()
	})

	if errors > 0 {
		b.Error("had errors")
	}

	evaluator.Close()
}
