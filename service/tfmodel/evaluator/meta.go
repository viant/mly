package evaluator

import (
	"github.com/viant/gmetric"
	"golang.org/x/sync/semaphore"
)

type EvaluatorMeta struct {
	// prevents potentially explosive thread generation due to concurrent requests
	// this should be shared across all Evaluators.
	semaphore  *semaphore.Weighted
	semaMetric *gmetric.Operation

	tfMetric *gmetric.Operation
}

func MakeEvaluatorMeta(semaphore *semaphore.Weighted, semaMetric, tfMetric *gmetric.Operation) EvaluatorMeta {
	return EvaluatorMeta{
		semaphore:  semaphore,
		semaMetric: semaMetric,
		tfMetric:   tfMetric,
	}
}
