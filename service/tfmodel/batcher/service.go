package batcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/service/tfmodel/evaluator"
)

// Service sits on top of an Evaluator and collects predictions calls
// and attempts to merge multiple calls into a single call, if they occur
// often enough.
type Service struct {
	closed    bool
	closeChan chan bool
	wg        sync.WaitGroup

	evaluator *evaluator.Service
	inputLen  int

	bsPool *sync.Pool
	abPool *sync.Pool

	q chan InputBatch

	config.BatcherConfig
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

// Waits for any queued batches to complete then closes underlying resources.
func (b *Service) Close() error {
	b.closeChan <- true
	b.closed = true
	if b.evaluator != nil {
		// wait for any queued batches to evaluate and return
		b.wg.Wait()
		b.evaluator.Close()
	}
	return nil
}

// len([]SubBatch) is fixed to MaxBatchCounts.
func (b *Service) getSubBatches() []SubBatch {
	return b.bsPool.Get().([]SubBatch)
}

// this isn't a really helpful pool?
func (b *Service) getActive() []interface{} {
	return b.abPool.Get().([]interface{})
}

func (b *Service) Evaluate(ctx context.Context, inputs []interface{}) ([]interface{}, error) {
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

func (b *Service) Queue(inputs []interface{}) (*SubBatch, error) {
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

func (b *Service) run(batch predictionBatch) {
	if b.Verbose != nil {
		log.Printf("[%s run] inputData :%v", b.Verbose.ID, batch.inputData)
	}

	ctx := context.TODO()

	b.wg.Add(1)
	results, err := b.evaluator.Evaluate(ctx, batch.inputData)
	b.wg.Done()

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
func (b *Service) Dispatcher() {
	active := b.getActive()
	subBatches := b.getSubBatches()

	curBatches := 0
	curBatchRows := 0

	maxBatchWait := b.MaxBatchWait
	var deadline *time.Timer

	for {
		if deadline == nil {
			// empty batch queue
			select {
			case <-b.closeChan:
				// do nothing
			case batch := <-b.q:
				curBatchRows = batch.batchSize
				subBatches[curBatches] = batch.SubBatch
				curBatches = 1

				for o, inSlice := range batch.inputData {
					active[o] = inSlice
				}

				if curBatchRows >= b.MaxBatchSize || b.MaxBatchCounts == 1 {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] go curBatched:%d", b.Verbose.ID, curBatchRows)
					}

					go b.run(predictionBatch{active, subBatches, curBatches})

					curBatches = 0
					curBatchRows = 0

					active = b.getActive()
					subBatches = b.getSubBatches()
				} else {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] wait curBatched:%d", b.Verbose.ID, curBatchRows)
					}

					deadline = time.NewTimer(maxBatchWait)
				}
			}
		} else {
			select {
			case <-b.closeChan:
				if curBatches > 0 {
					go b.run(predictionBatch{active, subBatches, curBatches})
				}
			case batch := <-b.q:
				batchSize := batch.batchSize
				if curBatchRows+batchSize > b.MaxBatchSize {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] go expectedBatchSize:%d", b.Verbose.ID, curBatchRows+batchSize)
					}

					// run current batch, use new batch
					go b.run(predictionBatch{active, subBatches, curBatches})

					curBatches = 0
					curBatchRows = 0

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

				curBatchRows += batchSize
				subBatches[curBatches] = batch.SubBatch
				curBatches++

				if curBatchRows >= b.MaxBatchSize || curBatches >= b.MaxBatchCounts {
					if b.Verbose != nil {
						log.Printf("[%s Dispatcher] go curBatched:%d curBatches:%d", b.Verbose.ID, curBatchRows, curBatches)
					}

					go b.run(predictionBatch{active, subBatches, curBatches})

					curBatches = 0
					curBatchRows = 0

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
					curBatchRows = 0

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

func NewBatcher(evaluator *evaluator.Service, inputLen int, batchConfig config.BatcherConfig) *Service {
	b := &Service{
		evaluator: evaluator,
		inputLen:  inputLen,

		BatcherConfig: batchConfig,

		bsPool: &sync.Pool{
			New: func() interface{} {
				return make([]SubBatch, batchConfig.MaxBatchCounts)
			},
		},

		abPool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, inputLen)
			},
		},

		q: make(chan InputBatch, batchConfig.MaxBatchCounts),

		closeChan: make(chan bool),
	}

	go b.Dispatcher()
	return b
}
