package batcher

import (
	"fmt"
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

	runN uint64 // tracker for debugging purposes
}

func (s *queueService) debug(m string) {
	if s.BatcherConfig.Verbose != nil {
		log.Printf("[%s queue %d] %s active:%d", s.BatcherConfig.Verbose.ID, s.runN, m, *s.active)
	}
}

func (s *queueService) debugf(f func() string) {
	if s.BatcherConfig.Verbose != nil {
		log.Printf("[%s queue %d] %s active:%d", s.BatcherConfig.Verbose.ID, s.runN, f(), *s.active)
	}
}

// See Service.queueBatch() and dispatcher.dispatch() for producer.
func (s *queueService) dispatch() bool {
	s.runN++

	if s.busy {
		s.debug("<-free evaluator wait")
		// See Service.run() for producer.
		<-s.free
		s.debug("<-free evaluator wait done")

		s.busy = false
	}

	select {
	case pb := <-s.batchQ:
		if pb.size < 0 {
			return false
		}

		s.debug("locking...")
		s.activeLock.Lock()
		s.debug("locked")
		s.setNotShedding()

		// Gets decremented in Service.run().
		*s.active = *s.active + 1
		if s.BatcherConfig.MaxEvaluatorConcurrency > 0 && *s.active >= s.BatcherConfig.MaxEvaluatorConcurrency {
			s.busy = true
			// Indicate that queueService.dispatch() may block
			// on queueService.free.
			s.debug("waiting <- struct{}{}")
			s.waiting <- struct{}{}
		}

		s.debug("go run")
		go func(pb predictionBatch, i uint64) {
			s.debugf(func() string { return fmt.Sprintf("run %d...", i) })

			s.run(pb)

			s.debugf(func() string { return fmt.Sprintf("run %d ok", i) })
		}(pb, s.runN)

		s.activeLock.Unlock()
		s.debug("unlocked")
	}

	return true
}
