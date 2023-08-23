package batcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/service/tfmodel/evaluator"
	"github.com/viant/mly/shared/stat"
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
	evaluator *evaluator.Service

	config.BatcherConfig
	ServiceMeta
}

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

func (b *Service) run(batch predictionBatch) {
	defer b.wg.Done()

	if b.Verbose != nil && b.Verbose.Input {
		log.Printf("[%s run] inputData :%v", b.Verbose.ID, batch.inputData)
	}

	// This is somewhat safe... unless Tensorflow C has a GLOBAL lock somewhere.
	// Eventually we can just have the TF session run.
	ctx := context.TODO()
	results, err := b.evaluator.Evaluate(ctx, batch.inputData)

	if b.Verbose != nil && b.Verbose.Output {
		log.Printf("[%s run] results:%v", b.Verbose.ID, results)
	}

	defer b.bsPool.Put(batch.subBatches)
	defer func() {
		for i := range batch.inputData {
			batch.inputData[i] = nil
		}

		b.abPool.Put(batch.inputData)
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

		if b.Verbose != nil && b.Verbose.Output {
			log.Printf("[%s run] o:%d batchResult:%v", b.Verbose.ID, o, batchResult)
		}

		inputBatchMeta.channel <- batchResult
		o += inputBatchMeta.batchSize
	}
}

// dispatcher runs and gathers model prediction requests and batches them up.
func (s *Service) dispatcher() {
	s.wg.Add(1)
	defer s.wg.Done()

	active := s.getActive()
	subBatches := s.getSubBatches()
	curBatches := 0
	curBatchRows := 0

	var onDone counter.OnDone
	var stats *stat.Values

	submit := func() {
		s.wg.Add(1)
		onDone(time.Now(), stats.Values()...)
		go s.run(predictionBatch{active, subBatches, curBatches})

		curBatches = 0
		curBatchRows = 0
		active = s.getActive()
		subBatches = s.getSubBatches()

	}

	maxBatchWait := s.MaxBatchWait
	var deadline *time.Timer

	for {
		if deadline == nil {
			// empty batch queue
			select {
			case batch := <-s.q:
				if batch.closed {
					return
				}

				onDone = s.dispatcherMetric.Begin(time.Now())
				stats = stat.NewValues()

				curBatchRows = batch.batchSize
				subBatches[curBatches] = batch.subBatch
				curBatches = 1

				for o, inSlice := range batch.inputData {
					active[o] = inSlice
				}

				if curBatchRows >= s.MaxBatchSize || s.MaxBatchCounts == 1 {
					if s.Verbose != nil {
						log.Printf("[%s Dispatcher] go curBatched:%d", s.Verbose.ID, curBatchRows)
					}

					// in theory, if b.MaxBatchCounts == 1, an upstream would
					// skip the batching
					stats.Append(mkbs(FullElements, curBatchRows, curBatches))
					submit()

					// deadline will be nil since there's no batching
				} else {
					if s.Verbose != nil {
						log.Printf("[%s Dispatcher] wait curBatched:%d", s.Verbose.ID, curBatchRows)
					}

					deadline = time.NewTimer(maxBatchWait)
				}
			}
		} else {
			select {
			case batch := <-s.q:
				if batch.closed {
					if curBatches > 0 {
						stats.Append(mkbs(Closing, curBatchRows, curBatches))
						s.wg.Add(1)
						go s.run(predictionBatch{active, subBatches, curBatches})
						onDone(time.Now(), stats.Values()...)
					}

					return
				}

				batchSize := batch.batchSize
				if curBatchRows+batchSize > s.MaxBatchSize {
					if s.Verbose != nil {
						log.Printf("[%s Dispatcher] go expectedBatchSize:%d", s.Verbose.ID, curBatchRows+batchSize)
					}

					stats.Append(mkbs(FullElements, curBatchRows, curBatches))
					submit()

					onDone = s.dispatcherMetric.Begin(time.Now())
					stats = stat.NewValues()

					// relying on GC
					deadline = time.NewTimer(maxBatchWait)
				}

				// TODO this may result in the batch having inconsistent state
				var err error
				for o, inSlice := range batch.inputData {
					if active[o] == nil {
						active[o] = inSlice
						continue
					}

					// TODO maybe use xSlice
					switch slicedActive := active[o].(type) {
					case [][]int32:
						switch aat := inSlice.(type) {
						case [][]int32:
							for _, untyped := range aat {
								slicedActive = append(slicedActive, untyped)
							}
						default:
							err = fmt.Errorf("inner expected int32 got %v", aat)
							break
						}
						active[o] = slicedActive
					case [][]int64:
						switch aat := inSlice.(type) {
						case [][]int64:
							for _, untyped := range aat {
								slicedActive = append(slicedActive, untyped)
							}
						default:
							err = fmt.Errorf("inner expected int64 got %v", aat)
							break
						}
						active[o] = slicedActive
					case [][]float32:
						switch aat := inSlice.(type) {
						case [][]float32:
							for _, untyped := range aat {
								slicedActive = append(slicedActive, untyped)
							}
						default:
							err = fmt.Errorf("inner expected float32 got %v", aat)
							break
						}
						active[o] = slicedActive
					case [][]float64:
						switch aat := inSlice.(type) {
						case [][]float64:
							for _, untyped := range aat {
								slicedActive = append(slicedActive, untyped)
							}
						default:
							err = fmt.Errorf("inner expected float64 got %v", aat)
							break
						}
						active[o] = slicedActive
					case [][]string:
						switch aat := inSlice.(type) {
						case [][]string:
							for _, untyped := range aat {
								slicedActive = append(slicedActive, untyped)
							}
						default:
							err = fmt.Errorf("inner expected string got %v", aat)
							break
						}
						active[o] = slicedActive
					default:
						err = fmt.Errorf("unexpected initial value: %v", slicedActive)
						break
					}
				}

				if err != nil {
					// TODO record
					batch.subBatch.ec <- err
					continue
				}

				curBatchRows += batchSize
				subBatches[curBatches] = batch.subBatch
				curBatches++

				// curBatchRows should always be < s.MaxBatchSize
				if curBatchRows >= s.MaxBatchSize || curBatches >= s.MaxBatchCounts {
					if s.Verbose != nil {
						log.Printf("[%s Dispatcher] go curBatched:%d curBatches:%d", s.Verbose.ID, curBatchRows, curBatches)
					}

					stats.Append(mkbs(MaxBatches, curBatchRows, curBatches))
					submit()

					onDone = nil
					stats = nil
					deadline = nil
				}
			case <-deadline.C:
				// curBatches should always be > 0
				if curBatches > 0 {
					if s.Verbose != nil {
						log.Printf("[%s Dispatcher] go timeout curBatches:%d curBatchRows:%d", s.Verbose.ID, curBatches, curBatchRows)
					}

					stats.Append(mkbs(stat.Timeout, curBatchRows, curBatches))
					submit()
				}

				onDone = nil
				stats = nil
				deadline = nil
			}
		}

		if s.closed {
			return
		}
	}
}

func NewBatcher(evaluator *evaluator.Service, inputLen int, batchConfig config.BatcherConfig, serviceMeta ServiceMeta) *Service {
	b := &Service{
		BatcherConfig: batchConfig,
		q:             make(chan inputBatch, batchConfig.MaxBatchCounts),
		evaluator:     evaluator,

		bsPool: &sync.Pool{
			New: func() interface{} {
				return make([]subBatch, batchConfig.MaxBatchCounts)
			},
		},

		abPool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, inputLen)
			},
		},
		ServiceMeta: serviceMeta,
	}

	go b.dispatcher()
	return b
}
