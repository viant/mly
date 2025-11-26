package router

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/mly/service/platform"
)

type workRequest struct {
	wg *sync.WaitGroup

	predictor platform.Predictor
	ctx       context.Context
	request   []interface{}

	queuedTime         time.Time
	offset             int
	modelOutputEnabled bool
	routingValueString string

	responseCh chan offsetResults
	errCh      chan error
}

type offsetResults struct {
	offset  int
	results []interface{}
}

func handleWorkRequests(workCh chan *workRequest, observer prometheus.Observer) {
	for request := range workCh {
		if request == nil {
			log.Println("work request is nil, stopping")
			break
		}

		func(request workRequest) {
			observer.Observe(float64(time.Since(request.queuedTime).Microseconds()))

			defer request.wg.Done()
			results, err := request.predictor.Predict(request.ctx, request.request)
			if err != nil {
				request.errCh <- fmt.Errorf("failed to predict for row %d: %w", request.offset, err)
				return
			}

			if request.modelOutputEnabled {
				// TODO fix ordering
				results = append(results, [][]string{{request.routingValueString}})
			}

			request.responseCh <- offsetResults{
				offset:  request.offset,
				results: results,
			}
		}(*request)
	}
}
