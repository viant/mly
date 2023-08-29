package batcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/mly/service/tfmodel/batcher/adjust"
	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/service/tfmodel/evaluator"
)

// Service sits on top of an Evaluator and collects predictions calls
// and attempts to merge multiple calls into a single call, if they occur
// rapidly enough.
// This *can* be refactored into smaller types; but it's
// really more annoying than convenient.
type Service struct {
	closed bool // prevents new batches from being queued

	wg        sync.WaitGroup // waits for queued batches to start evaluation
	q         chan inputBatch
	bsPool    *sync.Pool
	abPool    *sync.Pool
	evaluator evaluator.Evaluator

	// since dispatcher runs in a single goroutine, we use this
	// to communicate that there are free evaluator "slots"
	waiting    chan struct{}
	free       chan struct{}
	active     uint32
	activeLock *sync.RWMutex

	Adjust *adjust.Adjust

	config.BatcherConfig
	ServiceMeta
}

// ServiceMeta contains things that should live longer than the life of a Service.
type ServiceMeta struct {
	// Measures how long it takes for a request to get queued
	// primarily an issue is the channel is full.
	queueMetric *gmetric.Operation

	// Measures how long each batch lasts before being
	// sent to the evaluator service.
	dispatcherMetric *gmetric.Operation
}

func NewServiceMeta(q, d *gmetric.Operation) ServiceMeta {
	return ServiceMeta{q, d}
}

// inputBatch represents upstream data.
type inputBatch struct {
	closed    bool
	inputData []interface{}
	subBatch
}

// subBatch captures both upstream and downstream members.
type subBatch struct {
	batchSize int // memoization - this should be calculated from InputBatch.inputData

	// TODO check for leaks - these should be straight forward for GC
	channel chan []interface{} // channel for actual model output
	ec      chan error
}

// predictionBatch represents downstream (to service/tfmodel/evaluator.Service) data.
type predictionBatch struct {
	inputData  []interface{}
	subBatches []subBatch
	size       int
}

// Waits for any queued batches to complete then closes underlying resources.
func (b *Service) Close() error {
	// anything that passes this due to sync issues can just continue
	// generally, we rely on upstream to stop sending traffic before
	// calling Close()
	b.closed = true

	// make Dispatcher stop
	b.q <- inputBatch{true, nil, subBatch{}}

	// let Dispatch goroutine finish clearing queue
	b.wg.Wait()

	var err error
	if b.evaluator != nil {
		// the evaluator service will handle itself clearing
		err = b.evaluator.Close()
	}

	return err
}

func (s *Service) isBusy() (busy bool) {
	s.activeLock.RLock()
	defer s.activeLock.RUnlock()
	if s.MaxEvaluatorConcurrency > 0 && s.active >= s.MaxEvaluatorConcurrency {
		busy = true
		s.waiting <- struct{}{}
	}

	return busy
}

// len([]SubBatch) is fixed to MaxBatchCounts.
func (b *Service) getSubBatches() []subBatch {
	return b.bsPool.Get().([]subBatch)
}

// this isn't a really helpful pool?
func (b *Service) getActive() []interface{} {
	return b.abPool.Get().([]interface{})
}

// Evaluate will queue a batch and wait for results.
func (b *Service) Evaluate(ctx context.Context, inputs []interface{}) ([]interface{}, error) {
	sb, err := b.queue(inputs)
	if err != nil {
		return nil, err
	}

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

// queue will queue and provide channels for results.
func (b *Service) queue(inputs []interface{}) (*subBatch, error) {
	onDone := b.queueMetric.Begin(time.Now())
	defer func() { onDone(time.Now()) }()

	if b.closed {
		return nil, fmt.Errorf("closed")
	}

	if b.Verbose != nil && b.Verbose.Input {
		log.Printf("[%s Queue] inputs:%v", b.Verbose.ID, inputs)
	}

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

	if batchSize == 0 {
		return nil, fmt.Errorf("could not determine batch size")
	}

	ch := make(chan []interface{})
	ec := make(chan error)

	sb := &subBatch{
		batchSize: batchSize,
		channel:   ch,
		ec:        ec,
	}

	b.q <- inputBatch{false, inputs, *sb}
	return sb, nil
}

func (s *Service) run(batch predictionBatch) {
	defer s.wg.Done()

	if s.Verbose != nil && s.Verbose.Input {
		log.Printf("[%s run] inputData :%v", s.Verbose.ID, batch.inputData)
	}

	// This is somewhat safe... unless Tensorflow C has a GLOBAL lock somewhere.
	// Eventually we can just have the TF session run.
	ctx := context.TODO()

	s.activeLock.Lock()
	s.active++
	s.activeLock.Unlock()

	results, err := s.evaluator.Evaluate(ctx, batch.inputData)

	s.activeLock.Lock()
	if s.active > 0 {
		s.active--
	}

	select {
	case <-s.waiting:
		s.free <- struct{}{}
	default:
	}
	s.activeLock.Unlock()

	if s.Verbose != nil && s.Verbose.Output {
		log.Printf("[%s run] results:%v", s.Verbose.ID, results)
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

		// len(results) == 1
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
			default:
				inputBatchMeta.ec <- fmt.Errorf("unhandled output type:%v", typedSlice)
			}
		}

		if s.Verbose != nil && s.Verbose.Output {
			log.Printf("[%s run] o:%d batchResult:%v", s.Verbose.ID, o, batchResult)
		}

		inputBatchMeta.channel <- batchResult
		o += inputBatchMeta.batchSize
	}
}

// dispatcher runs and gathers model prediction requests and batches them up.
func (s *Service) dispatcher() {
	s.wg.Add(1)
	defer s.wg.Done()

	var timeout *time.Duration
	if s.Adjust == nil {
		timeout = &s.MinBatchWait
	} else {
		timeout = &s.Adjust.CurrentTimeout
	}

	dspr := &dispatcher{
		Service:    s,
		timeout:    timeout,
		active:     s.getActive(),
		subBatches: s.getSubBatches(),
	}

	var terminate bool
	for !terminate {
		terminate = dspr.dispatch()
	}

	if s.Verbose != nil {
		log.Printf("[%s dispatcher] terminated", s.Verbose.ID)
	}
}

func NewBatcher(evaluator evaluator.Evaluator, inputLen int, batchConfig config.BatcherConfig, serviceMeta ServiceMeta) *Service {
	var adj *adjust.Adjust

	ta := batchConfig.TimeoutAdjustments
	if ta != nil {
		if batchConfig.Verbose != nil {
			log.Printf("[%s NewBatcher] auto-adjust enabled %+v", batchConfig.Verbose.ID, ta)
		}
		adj = adjust.NewAdjust(ta)
	}

	b := &Service{
		q: make(chan inputBatch, batchConfig.MinBatchCounts),
		bsPool: &sync.Pool{
			New: func() interface{} {
				return make([]subBatch, batchConfig.MinBatchCounts)
			},
		},
		abPool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, inputLen)
			},
		},
		evaluator: evaluator,

		waiting:    make(chan struct{}, 1),
		free:       make(chan struct{}, 1),
		activeLock: new(sync.RWMutex),

		Adjust: adj,

		BatcherConfig: batchConfig,
		ServiceMeta:   serviceMeta,
	}

	go b.dispatcher()
	return b
}
