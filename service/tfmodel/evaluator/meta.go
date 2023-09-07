package evaluator

import (
	"time"

	"github.com/viant/gmetric"
	"golang.org/x/sync/semaphore"
)

type EvaluatorMeta struct {
	// prevents potentially explosive thread generation due to concurrent requests
	// this should be shared across all Evaluators.
	semaphore *semaphore.Weighted
	// prevents excessive waiting if semaphore is full and no other safeguards in place
	maxEvaluatorWait time.Duration

	semaMetric *gmetric.Operation

	tfMetric *gmetric.Operation
}

func MakeEvaluatorMeta(semaphore *semaphore.Weighted, maxEvaluatorWait time.Duration,
	semaMetric, tfMetric *gmetric.Operation) EvaluatorMeta {

	return EvaluatorMeta{
		semaphore:        semaphore,
		maxEvaluatorWait: maxEvaluatorWait,
		semaMetric:       semaMetric,
		tfMetric:         tfMetric,
	}
}
