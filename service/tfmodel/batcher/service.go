package batcher

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/service/errors"
	"github.com/viant/mly/service/evaluator"
	"github.com/viant/mly/service/tfmodel/batcher/adjust"
	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/shared/stat"
)

// Service sits on top of an Evaluator and collects predictions calls
// and attempts to merge multiple calls into a single call, if they occur
// rapidly enough.
// This *can* be refactored into smaller types; but it's
// really more annoying than convenient.
type Service struct {
	closed bool            // prevents new batches from being queued
	wg     *sync.WaitGroup // waits for everything to finish

	// shedding is true if MaxQueuedBatches AND MaxEvaluatorConcurrency are hit
	shedding     bool
	sheddingLock *sync.RWMutex

	// batcher (dispatcher) reads from here and pushes to batchQ
	inputQ   chan inputBatch
	blockInQ chan blockQDebug

	// indicates the dispatcher is waiting for the batch queue to be available
	bqWaiting chan struct{}
	// indicates that the batch queue became available
	bqFree chan struct{}

	// batch queue reads from here and spawn evaluator goroutines
	batchQ chan predictionBatch

	// indicates that the batch queue is waiting for a free evaluator
	waiting chan struct{}
	// batch queue listens to this to continue queueing
	free chan struct{}

	// number of running evaluators
	active     uint32
	activeLock *sync.RWMutex
	evaluator  evaluator.Evaluator
	bsPool     *sync.Pool
	abPool     *sync.Pool

	// for Stats()
	queueService *queueService
	dispatcher   *dispatcher

	*adjust.Adjust
	config.BatcherConfig
	ServiceMeta
}

// ServiceMeta contains things that should live longer than the life of a Service.
type ServiceMeta struct {
	queueMetric *gmetric.Operation

	// Measures how long each batch lasts before being
	// sent to the evaluator service.
	dispatcherMetric *gmetric.Operation

	dispatcherLoop *gmetric.Operation

	blockQDelay *gmetric.Operation
	inputQDelay *gmetric.Operation
}

func NewServiceMeta(m *stat.GMeter, id string) ServiceMeta {
	batcherLocation := reflect.TypeOf(ServiceMeta{}).PkgPath()
	return ServiceMeta{
		queueMetric:      m.Op(batcherLocation, id+"BatcherQueue", "batch enters dispatcher queue"),
		dispatcherMetric: m.MOp(batcherLocation, id+"Dispatcher", "batch exists in dispatcher", NewDispatcherP()),
		dispatcherLoop:   m.Op(batcherLocation, id+"DispatcherLoop", "dispatch loop"),
		blockQDelay:      m.Op(batcherLocation, id+"BSBQWait", "wait until dispatcher accepts input (verbose)"),
		inputQDelay:      m.Op(batcherLocation, id+"BSIQWait", "wait until dispatcher receives input (verbose)"),
	}
}

// inputBatch represents upstream data.
type inputBatch struct {
	inputData []interface{}
	subBatch
	created time.Time
}

func (ib inputBatch) closed() bool {
	return ib.subBatch.batchSize < 0
}

// subBatch captures both upstream and downstream members.
type subBatch struct {
	batchSize int // memoization - this should be calculated from InputBatch.inputData

	channel chan []interface{} // channel for actual model output
	ec      chan error
}

// predictionBatch represents downstream (to service/tfmodel/evaluator.Service) data.
type predictionBatch struct {
	inputData  []interface{}
	subBatches []subBatch
	size       int
}

func (s *Service) Stats(r map[string]interface{}) {
	r["activeEvaluators"] = s.active
	r["batchQueueLen"] = len(s.batchQ)

	d := s.dispatcher
	if d != nil {
		r["timeout"] = d.timeout
	}

	r["shedding"] = s.shedding
}

// Waits for any queued batches to complete then closes underlying resources.
func (s *Service) Close() error {
	// anything that passes this due to sync issues can just continue
	// generally, we rely on upstream to stop sending traffic before
	// calling Close()
	s.closed = true

	s.Verbose.Debug("Close", "queue close inputQ")
	s.inputQ <- inputBatch{nil, subBatch{-1, nil, nil}, time.Now()}

	s.Verbose.Debug("Close", "done close inputQ, wait WaitGroup")

	// let Dispatch goroutine finish clearing queue
	s.wg.Wait()
	s.Verbose.Debug("Close", "done WaitGroup, close batchQ")

	s.batchQ <- predictionBatch{nil, nil, -1}
	s.Verbose.Debug("Close", "done close batchQ")

	var err error
	if s.evaluator != nil {
		// the evaluator service will handle itself clearing
		err = s.evaluator.Close()
	}

	return err
}

// Evaluate will queue a batch and wait for results.
// This is safe to run in any number of goroutines.
//
// Evaluate will send the input to be (further) batched to a dispatcher, via the inputQ.
// See dispatcher.dispatch(), which will then call Service.queueBatch(), which
// pushes to batchQ.
// dispatcher also determines if waiting to send to batchQ is appropriate.
// See queueService.dispatch(), which reads from batchQ which will then finally run
// evaluator.Evaluator.Evaluate() when appropriate and return the response
// via a dedicated channel for the goroutine that called Service.Evaluate.
func (s *Service) Evaluate(ctx context.Context, inputs []interface{}) ([]interface{}, error) {
	s.Verbose.Debug("Evaluate", "start")

	sb, err := s.queue(ctx, inputs)
	if err != nil {
		return nil, err
	}

	s.Verbose.Debug("Evaluate", "queued")

	defer func() {
		s.Verbose.Debug("Evaluate", "done")
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v := <-sb.channel:
		return v, nil
	case err := <-sb.ec:
		return nil, err
	}

	return nil, fmt.Errorf("unhandled select")
}

func (s *Service) setShedding(shedding bool) {
	s.sheddingLock.Lock()
	defer s.sheddingLock.Unlock()
	s.shedding = shedding
}

func (s *Service) setNotShedding() {
	s.sheddingLock.RLock()
	if s.shedding {
		s.Verbose.Debug("setNotShedding", "no longer shedding")
		s.sheddingLock.RUnlock()
		s.setShedding(false)
	} else {
		s.sheddingLock.RUnlock()
	}
}

// technically this should wait until a dispatch cycle runs
func (s *Service) checkShedding() error {
	s.sheddingLock.RLock()
	defer s.sheddingLock.RUnlock()
	if s.shedding {
		return errors.OverloadedError
	}

	return nil
}

func determineBatchSize(inputs []interface{}) (int, error) {
	var batchSize int
	for _, iSlice := range inputs {
		switch typedSlice := iSlice.(type) {
		case [][]int32:
			batchSize = len(typedSlice)
		case [][]int64:
			batchSize = len(typedSlice)
		case [][]float32:
			batchSize = len(typedSlice)
		case [][]float64:
			batchSize = len(typedSlice)
		case [][]string:
			batchSize = len(typedSlice)
		default:
			continue
		}

		break
	}

	var err error
	if batchSize == 0 {
		err = fmt.Errorf("could not determine batch size")
	}

	return batchSize, err
}

// queue for requests i.e. input batch
func (s *Service) queue(ctx context.Context, inputs []interface{}) (*subBatch, error) {
	if s.closed {
		return nil, fmt.Errorf("closed")
	}

	err := s.checkShedding()
	if err != nil {
		return nil, err
	}

	s.Verbose.DebugFn("Queue", func() string { return fmt.Sprintf("inputs:%v", inputs) }, s.Verbose.InputEnabled())

	batchSize, err := determineBatchSize(inputs)
	if err != nil {
		return nil, err
	}

	ch := make(chan []interface{}, 1)
	ec := make(chan error, 1)

	sb := &subBatch{
		batchSize: batchSize,
		channel:   ch,
		ec:        ec,
	}

	// See dispatcher.dispatch() for the consumer side.
	s.inputQ <- inputBatch{inputs, *sb, time.Now()}
	// The reason this synchronization mechanism exists is because the
	// dispatcher may sometimes have to contend with a deadline (for
	// batching timeout).
	// If that wasn't the case, we could inline and effectively have
	// synchronous batch dispatching.
	onDone := s.queueMetric.Begin(time.Now())
	defer func() { onDone(time.Now()) }()

	select {
	case bqd := <-s.blockInQ:
		if s.Verbose != nil {
			s.blockQDelay.Begin(bqd.start)(time.Now())
		}
	case <-ctx.Done():
		go func() {
			// in case there was a context error, the dispatcher will
			// be waiting for us to complete, so we take from the channel
			// to unblock the dispatcher.
			bqd := <-s.blockInQ
			if s.Verbose != nil {
				s.blockQDelay.Begin(bqd.start)(time.Now())
			}
		}()

		return nil, ctx.Err()
	}

	return sb, nil
}

func (s *Service) queueBatch(batch predictionBatch) (shedding bool) {
	select {
	case s.batchQ <- batch:
		s.Verbose.Debug("queueBatch", "queued batch")
	default:
		s.setShedding(true)
		s.Verbose.DebugFn("queueBatch", func() string { return fmt.Sprintf("full batchQ len:%d (+1)", len(s.batchQ)) }, nil)

		go func() {
			// I suspect this is not completing due
			s.batchQ <- batch
			s.Verbose.Debug("queueBatch", "+1 done")
		}()

		return true
	}

	return false
}

func (s *Service) newQueueService() *queueService {
	qsrv := &queueService{
		run:            s.run,
		setNotShedding: s.setNotShedding,

		active:     &s.active,
		activeLock: s.activeLock,

		batchQ:  s.batchQ,
		waiting: s.waiting,
		free:    s.free,

		Adjust:        s.Adjust,
		BatcherConfig: s.BatcherConfig,
	}

	s.queueService = qsrv

	return qsrv
}

func (s *Service) startBatchQueueing() {
	qsrv := s.newQueueService()

	run := true
	for run {
		run = qsrv.dispatch()
	}

	s.Verbose.Debug("startBatchQueueing", "terminating")
}

func (s *Service) run(batch predictionBatch) {
	// this should have been incremented by dispatcher.submit()
	defer s.wg.Done()

	debugging := s.Verbose != nil
	if debugging && s.Verbose.Input {
		log.Printf("[%s run] inputData :%v", s.Verbose.ID, batch.inputData)
	}

	ctx := context.TODO()
	results, err := s.evaluator.Evaluate(ctx, batch.inputData)

	if debugging {
		log.Printf("[%s run] locking...", s.Verbose.ID)
	}
	s.activeLock.Lock()
	if debugging {
		log.Printf("[%s run] locked", s.Verbose.ID)
	}

	if s.active > 0 {
		// Should have been incremented in queueService.dispatch().
		s.active--
	}

	if s.Adjust != nil {
		s.Adjust.Active(s.active)
	}

	select {
	case <-s.waiting:
		// This means that queueService may or may not be waiting for
		// the Evaluator to finish.

		// See queueService.dispatch() for consumer.
		// This is going to block until the queueService.dispatch()
		// runs again... does that make sense to do?
		if debugging {
			log.Printf("[%s run] free <- struct{}{} start", s.Verbose.ID)
		}
		s.free <- struct{}{}
		if debugging {
			log.Printf("[%s run] free <- struct{}{} done", s.Verbose.ID)
		}
	default:
		if debugging {
			log.Printf("[%s run] default", s.Verbose.ID)
		}
	}
	s.activeLock.Unlock()
	if debugging {
		log.Printf("[%s run] unlocked", s.Verbose.ID)
		if s.Verbose.Output {
			log.Printf("[%s run] results:%v", s.Verbose.ID, results)
		}
	}

	defer s.bsPool.Put(batch.subBatches)
	defer func() {
		for i := range batch.inputData {
			batch.inputData[i] = nil
		}

		s.abPool.Put(batch.inputData)
	}()

	// result shape is [0][input][outputs]
	if err != nil {
		for _, mp := range batch.subBatches {
			mp.ec <- err
		}

		return
	}

	o := 0
	for i := 0; i < batch.size; i++ {
		inputBatchMeta := batch.subBatches[i]
		// input batch size
		r := o + inputBatchMeta.batchSize
		batchResult := make([]interface{}, len(results))

		// usually len(results) == 1
		// TODO investigate other cases
		for resOffset, resSlice := range results {
			switch typedSlice := resSlice.(type) {
			case [][]int32:
				batchResult[resOffset] = typedSlice[o:r]
			case [][]int64:
				batchResult[resOffset] = typedSlice[o:r]
			case [][]float32:
				batchResult[resOffset] = typedSlice[o:r]
			case [][]float64:
				batchResult[resOffset] = typedSlice[o:r]
			case [][]string:
				batchResult[resOffset] = typedSlice[o:r]
			case []int32:
				batchResult[resOffset] = typedSlice[o:r]
			case []int64:
				batchResult[resOffset] = typedSlice[o:r]
			case []float32:
				batchResult[resOffset] = typedSlice[o:r]
			case []float64:
				batchResult[resOffset] = typedSlice[o:r]
			case []string:
				batchResult[resOffset] = typedSlice[o:r]
			default:
				inputBatchMeta.ec <- fmt.Errorf("batching unhandled output type:%V", typedSlice)
			}
		}

		if debugging && s.Verbose.Output {
			log.Printf("[%s run] o:%d batchResult:%v", s.Verbose.ID, o, batchResult)
		}

		inputBatchMeta.channel <- batchResult
		o += inputBatchMeta.batchSize
	}

}

// startDispatcher runs and gathers model prediction requests and batches them up.
// this should run in a goroutine
func (s *Service) startDispatcher() {
	s.wg.Add(1)
	defer s.wg.Done()

	dspr := s.newDispatcher()

	var terminate bool
	for !terminate {
		terminate = dspr.dispatch()
	}

	s.Verbose.Debug("dispatcher", "terminated")
}

func (s *Service) newDispatcher() *dispatcher {
	var timeout *time.Duration
	if s.Adjust == nil {
		timeout = &s.BatchWait
	} else {
		timeout = &s.Adjust.CurrentTimeout
	}

	dspr := &dispatcher{
		timeout: timeout,

		wg:     s.wg,
		inputQ: s.inputQ,
		blockQ: s.blockInQ,

		batchQ: s.batchQ,

		abPool: s.abPool,
		bsPool: s.bsPool,

		queueBatch: s.queueBatch,

		Adjust:        s.Adjust,
		BatcherConfig: s.BatcherConfig,
		ServiceMeta:   s.ServiceMeta,
	}

	dspr.resetSlices()

	s.dispatcher = dspr

	return dspr
}

func (s *Service) Start() {
	go s.startDispatcher()
	go s.startBatchQueueing()
}

func NewBatcher(evaluator evaluator.Evaluator, inputLen int, batchConfig config.BatcherConfig, serviceMeta ServiceMeta) *Service {
	var adj *adjust.Adjust
	ta := batchConfig.TimeoutAdjustments
	if ta != nil {
		batchConfig.Verbose.DebugFn("NewBatcher", func() string { return fmt.Sprintf("auto-adjust enabled %+v", ta) }, nil)
		adj = adjust.NewAdjust(ta)
	}

	inputQSize := batchConfig.MaxBatchSize
	if inputQSize < 0 {
		inputQSize = 0
	}

	bsPool := &sync.Pool{
		New: func() interface{} {
			return make([]subBatch, 100)
		},
	}

	abPool := &sync.Pool{
		New: func() interface{} {
			return make([]interface{}, inputLen)
		},
	}

	bqSize := batchConfig.MaxQueuedBatches
	if bqSize > 0 {
		// this is bqSize - 1 because we queue before checking it is full
		// so if the queue fails because it is full then we add 1 batch
		// via a goroutine to make it a total of bqSize waiting batches.
		bqSize = bqSize - 1
	}

	s := &Service{
		wg: new(sync.WaitGroup),

		sheddingLock: new(sync.RWMutex),

		inputQ:   make(chan inputBatch, inputQSize),
		blockInQ: make(chan blockQDebug, 0),

		bqWaiting: make(chan struct{}, 1),
		bqFree:    make(chan struct{}, 0),

		batchQ: make(chan predictionBatch, bqSize),

		waiting: make(chan struct{}, 1),
		free:    make(chan struct{}, 0),

		activeLock: new(sync.RWMutex),
		evaluator:  evaluator,
		bsPool:     bsPool,
		abPool:     abPool,

		Adjust:        adj,
		BatcherConfig: batchConfig,
		ServiceMeta:   serviceMeta,
	}

	return s
}
