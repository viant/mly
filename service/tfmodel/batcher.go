package tfmodel

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Batcher sits on top of an Evaluator and collects predictions calls
// and attempts to merge multiple calls into a single call, if they occur
// often enough.
type Batcher struct {
	closed    bool
	closeChan chan bool

	evaluator *Evaluator
	inputLen  int

	bsPool *sync.Pool
	abPool *sync.Pool

	q chan InputBatch

	BatcherConfig
}

type BatcherConfig struct {
	// MaxBatchSize defines the limit of batch size (input rows) that can be
	// accumulated before the batch will be sent to the model for prediction.
	// If an incoming batch's size is >= MaxBatchSize, then it will be run as
	// its own batch - there's no hard limiting and incoming batch size.
	MaxBatchSize int

	// MaxBatchCounts represent the maximum number of incoming batches to wait
	// for before sending for prediction.
	MaxBatchCounts int

	// MaxBatchWait indicates maximum wait since the start of the current
	// batch collection.
	// This is not a rolling window.
	MaxBatchWait time.Duration

	Verbose *V
}

type V struct {
	ID     string
	Output bool
}

// InputBatch represents upstream data.
type InputBatch struct {
	inputData []interface{}
	SubBatch
}

// SubBatch captures both upstream and downstream members.
type SubBatch struct {
	batchSize int // memoization - this should be calculated from InputBatch.inputData

	// TODO check for leaks - these should be straight forward for GC
	channel chan []interface{} // channel for actual model output
	ec      chan error
}

// predictionBatch represents downstream data.
type predictionBatch struct {
	inputData  []interface{}
	subBatches []SubBatch
	size       int
}

func (b *Batcher) Close() error {
	b.closeChan <- true
	b.closed = true
	return nil
}

// len([]SubBatch) is fixed to MaxBatchCounts.
func (b *Batcher) getSubBatches() []SubBatch {
	return b.bsPool.Get().([]SubBatch)
}

// this isn't a really helpful pool?
func (b *Batcher) getActive() []interface{} {
	return b.abPool.Get().([]interface{})
}

func (b *Batcher) Evaluate(ctx context.Context, inputs []interface{}) ([]interface{}, error) {
	sb, err := b.Queue(inputs)
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

func (b *Batcher) Queue(inputs []interface{}) (*SubBatch, error) {
	if b.closed {
		return nil, fmt.Errorf("closed")
	}

	if b.Verbose != nil {
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

	sb := &SubBatch{
		batchSize: batchSize,
		channel:   ch,
		ec:        ec,
	}

	b.q <- InputBatch{inputs, *sb}

	return sb, nil
}

func (b *Batcher) run(batch predictionBatch) {
	if b.Verbose != nil {
		log.Printf("[%s run] inputData :%v", b.Verbose.ID, batch.inputData)
	}

	ctx := context.TODO()
	results, err := b.evaluator.Evaluate(ctx, batch.inputData)
	if b.Verbose != nil {
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

// Dispatcher runs and gathers model prediction requests and batches them up.
func (b *Batcher) Dispatcher() {
	active := b.getActive()
	subBatches := b.getSubBatches()

	curBatches := 0
	curBatched := 0

	maxBatchWait := b.MaxBatchWait
	var deadline *time.Timer

	for {
		if deadline == nil {
			// empty batch queue
			select {
			case <-b.closeChan:
				// do nothing
			case batch := <-b.q:
				curBatched = batch.batchSize
				subBatches[curBatches] = batch.SubBatch
				curBatches = 1

				for o, inSlice := range batch.inputData {
					active[o] = inSlice
				}

				if curBatched >= b.MaxBatchSize || b.MaxBatchCounts == 1 {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] go curBatched:%d", b.Verbose.ID, curBatched)
					}

					go b.run(predictionBatch{active, subBatches, curBatches})

					curBatches = 0
					curBatched = 0

					active = b.getActive()
					subBatches = b.getSubBatches()
				} else {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] wait curBatched:%d", b.Verbose.ID, curBatched)
					}

					deadline = time.NewTimer(maxBatchWait)
				}
			}
		} else {
			select {
			case <-b.closeChan:
				// nothing
			case batch := <-b.q:
				batchSize := batch.batchSize
				if curBatched+batchSize > b.MaxBatchSize {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] go expectedBatchSize:%d", b.Verbose.ID, curBatched+batchSize)
					}

					// run current batch, use new batch
					go b.run(predictionBatch{active, subBatches, curBatches})

					curBatches = 0
					curBatched = 0

					subBatches = b.getSubBatches()
					active = b.getActive()

					// relying on GC
					deadline = time.NewTimer(maxBatchWait)
				}

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
					batch.SubBatch.ec <- err
					continue
				}

				curBatched += batchSize
				subBatches[curBatches] = batch.SubBatch
				curBatches++

				if curBatched >= b.MaxBatchSize || curBatches >= b.MaxBatchCounts {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] go curBatched:%d curBatches:%d", b.Verbose.ID, curBatched, curBatches)
					}

					go b.run(predictionBatch{active, subBatches, curBatches})

					curBatches = 0
					curBatched = 0

					subBatches = b.getSubBatches()
					active = b.getActive()

					deadline = nil
				}
			case <-deadline.C:
				// curBatches should always be > 0
				if curBatches > 0 {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] go timeout curBatches:%d", b.Verbose.ID, curBatches)
					}

					go b.run(predictionBatch{active, subBatches, curBatches})

					curBatches = 0
					curBatched = 0

					active = b.getActive()
					subBatches = b.getSubBatches()
				}

				deadline = nil
			}
		}

		if b.closed {
			return
		}
	}
}

func NewBatcher(evaluator *Evaluator, inputLen int, MaxBatchSize, MaxBatchCounts int, MaxBatchWait time.Duration) *Batcher {
	batchConfig := BatcherConfig{
		MaxBatchSize:   MaxBatchSize,
		MaxBatchCounts: MaxBatchCounts,
		MaxBatchWait:   MaxBatchWait,
	}

	b := &Batcher{
		evaluator: evaluator,
		inputLen:  inputLen,

		BatcherConfig: batchConfig,

		bsPool: &sync.Pool{
			New: func() interface{} {
				return make([]SubBatch, MaxBatchCounts)
			},
		},

		abPool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, inputLen)
			},
		},

		q: make(chan InputBatch, MaxBatchCounts),

		closeChan: make(chan bool),
	}

	go b.Dispatcher()
	return b
}
