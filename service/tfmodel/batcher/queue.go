package batcher

import (
	"log"
	"sync"

	"github.com/viant/mly/service/tfmodel/batcher/config"
)

type queueService struct {
	run            func(predictionBatch)
	setNotShedding func()

	busy       bool
	active     *uint32
	activeLock *sync.RWMutex

	batchQ chan predictionBatch
	// indicates that the batch queue is waiting for a free evaluator
	waiting chan struct{}
	// batch queue listens to this to continue queueing
	free chan struct{}

	config.BatcherConfig
}

func (s *queueService) debug(m string) {
	if s.BatcherConfig.Verbose != nil {
		log.Printf("[%s queue] %s", s.BatcherConfig.Verbose.ID, m)
	}
}

// See Service.queueBatch() and dispatcher.dispatch() for producer.
func (s *queueService) dispatch() bool {
	if s.busy {
		s.debug("wait for free evaluator")
		// See Service.run() for producer.
		<-s.free
		s.debug("done waiting for free evaluator")
	}

	select {
	case pb := <-s.batchQ:
		if pb.size < 0 {
			return false
		}

		s.activeLock.Lock()
		s.setNotShedding()

		// Gets decremented in Service.run().
		*s.active++
		if *s.active >= s.BatcherConfig.MaxEvaluatorConcurrency {
			s.busy = true
			// Indicate that queueService.dispatch() may block
			// on queueService.free.
			s.debug("waiting <- struct{}{}")
			s.waiting <- struct{}{}
		}

		go func(pb predictionBatch) {
			s.debug("run...")
			s.run(pb)
			s.debug("run ok")
		}(pb)

		s.activeLock.Unlock()
	}

	return true
}
