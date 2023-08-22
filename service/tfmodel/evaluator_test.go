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
	"github.com/viant/mly/service/domain"
	srvstat "github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/tfmodel/evaluator"
	"golang.org/x/sync/semaphore"
)

func createEvalMeta() evaluator.EvaluatorMeta {
	s := gmetric.New()
	return evaluator.MakeEvaluatorMeta(semaphore.NewWeighted(100),
		s.MultiOperationCounter("test", "test sema", "", time.Microsecond, time.Minute, 2, srvstat.NewEval()),
		s.MultiOperationCounter("test", "test eval", "", time.Microsecond, time.Minute, 2, srvstat.NewEval()))

}

func loadEvaluator(path string, withLoadModel, withSignature func(error)) (*domain.Signature, *evaluator.Service, evaluator.EvaluatorMeta) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	modelDest := filepath.Join(root, path)
	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	withLoadModel(err)
	signature, err := Signature(model)
	withSignature(err)
	met := createEvalMeta()
	return signature, evaluator.NewEvaluator(signature, model.Session, met), met
}

func tLoadEvaluator(t *testing.T, path string) (*domain.Signature, *evaluator.Service, evaluator.EvaluatorMeta) {
	tnil := func(err error) {
		assert.Nil(t, err)
	}

	return loadEvaluator(path, tnil, tnil)
}

func TestBasic(t *testing.T) {
	_, evaluator, _ := tLoadEvaluator(t, "example/model/string_lookups_int_model")

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}})
	feeds = append(feeds, [][]string{{"c"}})

	_, err := evaluator.Evaluate(context.Background(), feeds)
	assert.Nil(t, err)
}

func TestBasicV2(t *testing.T) {
	_, evaluator, _ := tLoadEvaluator(t, "example/model/vectorization_int_model")

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

	_, evaluator, _ := loadEvaluator("example/model/string_lookups_int_model", bnil, bnil)

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
