package evaluator

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/shared/stat"
	"github.com/viant/mly/shared/stat/metric"
	tf "github.com/wamuir/graft/tensorflow"
)

// Represents a TF session for running predictions.
type Service struct {
	EvaluatorMeta

	wg sync.WaitGroup

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
func (s *Service) Evaluate(ctx context.Context, params []interface{}) ([]interface{}, error) {
	s.wg.Add(1)
	defer s.wg.Done()

	var cancel func()
	if s.maxEvaluatorWait > 0 {
		ctx, cancel = context.WithTimeout(ctx, s.maxEvaluatorWait)
	}

	release, err := s.acquire(ctx)
	cancel()
	if err != nil {
		return nil, err
	}
	defer release()

	onDone := s.tfMetric.Begin(time.Now())
	stats := stat.NewValues()
	defer func() {
		onDone(time.Now(), stats.Values()...)
	}()

	feeds, err := s.feeds(params)
	if err != nil {
		return nil, clienterr.Wrap(err)
	}

	onExit := metric.EnterThenExit(s.tfMetric, time.Now(), stat.Enter, stat.Exit)
	defer onExit()

	debug.SetPanicOnFault(true)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[%s Evaluate] recover()=%v - %+v - EXIT with code 139", s.id, r, params)
			// we are potentially recovering from a segment fault
			// so we will exit like there was a seg fault.
			os.Exit(139)
		}
	}()
	output, err := s.session.Run(feeds, s.fetches, s.targets)
	debug.SetPanicOnFault(false)

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

// Close will block until all goroutines waiting on the semaphore are released,
// and complete the Tensorflow session run, or their context errors.
// Will then close the session.
func (s *Service) Close() error {
	s.wg.Wait()
	return s.session.Close()
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
