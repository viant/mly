package tfmodel

import (
	"github.com/stretchr/testify/assert"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBasic(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	t.Logf("Root %s", root)
	modelDest := filepath.Join(root, "tools/sls_model")

	model, err := tf.LoadSavedModel(modelDest, []string{"serve"}, nil)
	assert.Nil(t, err)

	signature, err := Signature(model)
	assert.Nil(t, err)

	evaluator := NewEvaluator(signature, model.Session)

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}, {"b"}})
	feeds = append(feeds, [][]string{{"c"}, {"d"}})

	result, err := evaluator.Evaluate(feeds)
	assert.Nil(t, err)

	t.Logf("Results %s", result)
}
