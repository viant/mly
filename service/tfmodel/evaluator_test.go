package tfmodel_test

import (
	"github.com/stretchr/testify/assert"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/tfmodel"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBasic(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	t.Logf("Root %s", root)
	modelDest := filepath.Join(root, "example/model/sls_model")

	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	assert.Nil(t, err)

	signature, err := tfmodel.Signature(model)
	assert.Nil(t, err)
	evaluator := tfmodel.NewEvaluator(signature, model.Session)

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}})
	feeds = append(feeds, [][]string{{"c"}})

	result, err := evaluator.Evaluate(feeds)
	assert.Nil(t, err)
	assert.EqualValues(t, []int64{24}, result[0])
}

func TestBasicV2(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	t.Logf("Root %s", root)
	modelDest := filepath.Join(root, "example/model/vec_model")

	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	assert.Nil(t, err)

	signature, err := tfmodel.Signature(model)
	assert.Nil(t, err)
	evaluator := tfmodel.NewEvaluator(signature, model.Session)
	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}, {"b"}})
	feeds = append(feeds, [][]string{{"c"}, {"d"}})

	result, err := evaluator.Evaluate(feeds)
	assert.Nil(t, err)
	assert.EqualValues(t, []int64{42, 30}, result[0])
}
