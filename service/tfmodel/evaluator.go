package tfmodel

import (
	"fmt"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/domain"
)

type Evaluator struct {
	session *tf.Session

	fetches   []tf.Output
	targets   []*tf.Operation
	signature domain.Signature
}

func (e *Evaluator) feeds(feeds []interface{}) (map[tf.Output]*tf.Tensor, error) {
	var result = make(map[tf.Output]*tf.Tensor, len(feeds))
	for _, input := range e.signature.Inputs {
		tensor, err := tf.NewTensor(feeds[input.Index])
		if err != nil {
			return nil, fmt.Errorf("failed to prepare feed: %v(%v), due to %w", input.Name, feeds[input.Index], err)
		}
		result[input.Placeholder] = tensor
	}
	return result, nil
}

// Evaluate runs the primary model prediction via Cgo Tensorflow.
func (e *Evaluator) Evaluate(params []interface{}) ([]interface{}, error) {
	feeds, err := e.feeds(params)
	if err != nil {
		return nil, clienterr.Wrap(err)
	}

	output, err := e.session.Run(feeds, e.fetches, e.targets)
	if err != nil {
		return nil, err
	}
	var tensorValues = make([]interface{}, len(output))
	for i := range tensorValues {
		tensorValues[i] = output[i].Value()
	}

	return tensorValues, nil
}

// Close closes the Tensorflow session.
func (e *Evaluator) Close() error {
	return e.session.Close()
}

func NewEvaluator(signature *domain.Signature, session *tf.Session) *Evaluator {
	fetches := []tf.Output{}
	for _, output := range signature.Outputs {
		fetches = append(fetches, output.Output(output.Index))
	}

	return &Evaluator{
		signature: *signature,
		session:   session,
		fetches:   fetches,
		targets:   make([]*tf.Operation, 0),
	}
}
