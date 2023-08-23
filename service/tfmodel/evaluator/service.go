package evaluator

import (
	"context"
	"fmt"
	"time"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/mly/shared/stat/metric"
)

// Represents a TF session for running predictions.
type Service struct {
	EvaluatorMeta

	session *tf.Session

	fetches []tf.Output
	targets []*tf.Operation

	signature domain.Signature
}

func (e *Service) feeds(feeds []interface{}) (map[tf.Output]*tf.Tensor, error) {
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

func (e *Service) acquire(ctx context.Context) (func(), error) {
	now := time.Now()
	onDone := e.semaMetric.Begin(now)
	stats := stat.NewValues()
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()

	onExit := metric.EnterThenExit(e.semaMetric, now, stat.Enter, stat.Exit)
	defer onExit()

	err := e.semaphore.Acquire(ctx, 1)
	if err != nil {
		stats.AppendError(err)
		return nil, err
	}

	return func() { e.semaphore.Release(1) }, nil
}

// Evaluate runs the primary model prediction via Cgo Tensorflow.
// params is expected to be [inputs][1][batch]T - see service/request.Request.Feeds and related methods.
func (e *Service) Evaluate(ctx context.Context, params []interface{}) ([]interface{}, error) {
	release, err := e.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	onDone := e.tfMetric.Begin(time.Now())
	stats := stat.NewValues()
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()

	feeds, err := e.feeds(params)
	if err != nil {
		return nil, clienterr.Wrap(err)
	}

	onExit := metric.EnterThenExit(e.tfMetric, time.Now(), stat.Enter, stat.Exit)
	defer onExit()

	output, err := e.session.Run(feeds, e.fetches, e.targets)
	if err != nil {
		stats.Append(err)
		return nil, err
	}

	var tensorValues = make([]interface{}, len(output))
	for i := range tensorValues {
		tensorValues[i] = output[i].Value()
	}

	return tensorValues, nil
}

// Close closes the Tensorflow session.
func (e *Service) Close() error {
	return e.session.Close()
}

func NewEvaluator(signature *domain.Signature, session *tf.Session, meta EvaluatorMeta) *Service {
	fetches := []tf.Output{}
	for _, output := range signature.Outputs {
		fetches = append(fetches, output.Output(output.Index))
	}

	evaluator := &Service{
		EvaluatorMeta: meta,

		signature: *signature,
		session:   session,
		fetches:   fetches,
		targets:   make([]*tf.Operation, 0),
	}

	return evaluator
}
