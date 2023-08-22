package tfmodel

import (
	"context"
	"fmt"
	"time"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/stat"

	"golang.org/x/sync/semaphore"
)

// Represents a TF session for running predictions.
type Evaluator struct {
	EvaluatorMeta

	session *tf.Session

	fetches []tf.Output
	targets []*tf.Operation

	signature domain.Signature
}

type EvaluatorMeta struct {
	// prevents potentially explosive thread generation due to concurrent requests
	// this should be shared across all Evaluators.
	semaphore  *semaphore.Weighted
	semaMetric *gmetric.Operation

	tfMetric *gmetric.Operation
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

func (e *Evaluator) acquire(ctx context.Context) (func(), error) {
	onDone := e.semaMetric.Begin(time.Now())
	stats := stat.NewValues()
	defer onDone(time.Now())

	err := e.semaphore.Acquire(ctx, 1)
	if err != nil {
		stats.AppendError(err)
		return nil, err
	}

	return func() { e.semaphore.Release(1) }, nil
}

// Evaluate runs the primary model prediction via Cgo Tensorflow.
// params is expected to be [inputs][1][batch]T - see service/request.Request.Feeds and related methods.
func (e *Evaluator) Evaluate(ctx context.Context, params []interface{}) ([]interface{}, error) {
	release, err := e.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	onDone := e.tfMetric.Begin(time.Now())
	defer func() {
		onDone(time.Now())
	}()

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

func NewEvaluator(signature *domain.Signature, session *tf.Session, meta EvaluatorMeta) *Evaluator {
	fetches := []tf.Output{}
	for _, output := range signature.Outputs {
		fetches = append(fetches, output.Output(output.Index))
	}

	evaluator := &Evaluator{
		EvaluatorMeta: meta,

		signature: *signature,
		session:   session,
		fetches:   fetches,
		targets:   make([]*tf.Operation, 0),
	}

	return evaluator
}
