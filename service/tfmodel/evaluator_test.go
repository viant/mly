package tfmodel

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/gmetric"
	srvstat "github.com/viant/mly/service/stat"
	"golang.org/x/sync/semaphore"
)

func createEvalMeta() EvaluatorMeta {
	s := gmetric.New()
	return EvaluatorMeta{
		semaphore:  semaphore.NewWeighted(100),
		semaMetric: s.MultiOperationCounter("test", "test sema", "", time.Microsecond, time.Minute, 2, srvstat.NewEval()),
		tfMetric:   s.MultiOperationCounter("test", "test eval", "", time.Microsecond, time.Minute, 2, srvstat.NewEval()),
	}
}

func TestBasic(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	t.Logf("Root %s", root)
	modelDest := filepath.Join(root, "example/model/string_lookups_int_model")

	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	assert.Nil(t, err)

	signature, err := Signature(model)
	assert.Nil(t, err)
	evaluator := NewEvaluator(signature, model.Session, createEvalMeta())

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}})
	feeds = append(feeds, [][]string{{"c"}})

	_, err = evaluator.Evaluate(context.Background(), feeds)
	assert.Nil(t, err)
}

func TestBasicV2(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	t.Logf("Root %s", root)
	modelDest := filepath.Join(root, "example/model/vectorization_int_model")

	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	assert.Nil(t, err)

	signature, err := Signature(model)
	assert.Nil(t, err)
	evaluator := NewEvaluator(signature, model.Session, createEvalMeta())
	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a b c"}, {"b d d"}})
	feeds = append(feeds, [][]string{{"c"}, {"d"}})

	_, err = evaluator.Evaluate(context.Background(), feeds)
	assert.Nil(t, err)
}

func TestKeyedOut(t *testing.T) {
}

func BenchmarkEvaluatorParallel(b *testing.B) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	modelDest := filepath.Join(root, "example/model/string_lookups_int_model")

	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	if err != nil {
		b.Error(err)
	}

	signature, err := Signature(model)
	if err != nil {
		b.Error(err)
	}

	met := createEvalMeta()
	evaluator := NewEvaluator(signature, model.Session, met)

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
