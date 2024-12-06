package tfmodel

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"runtime/trace"

	"github.com/viant/mly/service/clienterr"
	"github.com/viant/mly/service/domain"
	tf "github.com/wamuir/graft/tensorflow"
)

// Evaluator represents evaluator
type Evaluator struct {
	id string

	session *tf.Session
	fetches []tf.Output
	targets []*tf.Operation
	domain.Signature
}

func (e *Evaluator) feeds(feeds []interface{}) (map[tf.Output]*tf.Tensor, error) {
	var result = make(map[tf.Output]*tf.Tensor, len(feeds))
	for _, input := range e.Signature.Inputs {
		tensor, err := tf.NewTensor(feeds[input.Index])
		if err != nil {
			return nil, fmt.Errorf("failed to prepare feed: %v(%v), due to %w", input.Name, feeds[input.Index], err)
		}
		result[input.Placeholder] = tensor
	}
	return result, nil
}

// Evaluate evaluates model
func (e *Evaluator) Evaluate(params []interface{}) ([]interface{}, error) {
	ctx := context.Background()

	if TFSessionPanicDuration > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, TFSessionPanicDuration)
		defer cancel()
	}

	errc := make(chan error)
	done := make(chan struct{})

	var tensorValues []interface{}
	go func() {
		debug.SetPanicOnFault(true)
		defer debug.SetPanicOnFault(false)

		defer func() {
			if r := recover(); r != nil {
				log.Printf("[%s Evaluator.Evaluate] recover()=%v - %+v - TERMINATING PROCESS", e.id, r, params)
				// we are potentially recovering from a segment fault
				// we will exit like we would if there was a segment fault
				os.Exit(139)
			}
		}()

		trr := trace.StartRegion(ctx, "Evaluator.feeds")
		feeds, err := e.feeds(params)
		trr.End()
		if err != nil {
			errc <- clienterr.Wrap(err)
		}

		trr = trace.StartRegion(ctx, "Evaluator.Session")
		output, err := e.session.Run(feeds, e.fetches, e.targets)
		trr.End()

		if err != nil {
			errc <- err
		}

		tensorValues = make([]interface{}, len(output))
		for i := range tensorValues {
			tensorValues[i] = output[i].Value()
		}

		done <- struct{}{}
	}()

	select {
	case err := <-errc:
		return nil, err
	case <-done:
		return tensorValues, nil
	case <-ctx.Done():
		debugStr := fmt.Sprintf("Forced panic invocation, Tensorflow Session took longer than %s", TFSessionPanicDuration)
		log.Printf(debugStr)
		debug.PrintStack()
		os.Exit(5)
		panic(debugStr)
	}

}

// Close closes evaluator
func (e *Evaluator) Close() error {
	return e.session.Close()
}

// NewEvaluator creates new evaluator
func NewEvaluator(id string, signature *domain.Signature, session *tf.Session) *Evaluator {
	fetches := []tf.Output{}
	for _, output := range signature.Outputs {
		fetches = append(fetches, output.Output(output.Index))
	}

	return &Evaluator{
		id: id,

		Signature: *signature,
		session:   session,
		fetches:   fetches,
		targets:   make([]*tf.Operation, 0),
	}
}
