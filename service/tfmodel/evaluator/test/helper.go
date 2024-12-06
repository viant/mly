package test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/tfmodel/evaluator"
	"github.com/viant/mly/service/tfmodel/signature"
	tfstat "github.com/viant/mly/service/tfmodel/stat"
	tf "github.com/wamuir/graft/tensorflow"
	"golang.org/x/sync/semaphore"
)

func createEvalMeta() evaluator.EvaluatorMeta {
	s := gmetric.New()
	return evaluator.MakeEvaluatorMeta(
		"test",
		semaphore.NewWeighted(100),
		time.Duration(1*time.Second),
		s.MultiOperationCounter("test", "test sema", "", time.Microsecond, time.Minute, 2, tfstat.NewSema()),
		s.MultiOperationCounter("test", "test eval", "", time.Microsecond, time.Minute, 2, tfstat.NewTfs()),
	)
}

func LoadEvaluator(path string, withLoadModel, withSignature func(error)) (*domain.Signature, *evaluator.Service, evaluator.EvaluatorMeta) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../../../..")
	modelDest := filepath.Join(root, path)
	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	withLoadModel(err)
	signature, err := signature.Signature(model)
	withSignature(err)
	met := createEvalMeta()
	return signature, evaluator.NewEvaluator(signature, model.Session, met), met
}

func TLoadEvaluator(t *testing.T, path string) (*domain.Signature, *evaluator.Service, evaluator.EvaluatorMeta) {
	tnil := func(err error) {
		assert.Nil(t, err)
	}

	return LoadEvaluator(path, tnil, tnil)
}
