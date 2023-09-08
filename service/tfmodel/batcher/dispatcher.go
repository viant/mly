package batcher

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/shared/stat"
)

// dispatcher represents a state after an iteration of pulling from
// the dispatcher queue.
type dispatcher struct {
	timer *time.Timer

	wg     *sync.WaitGroup
	inputQ chan inputBatch
	blockQ chan blockQDebug
	batchQ chan predictionBatch

	// shared with Service
	bsPool *sync.Pool
	abPool *sync.Pool

	active        []interface{}
	subBatches    []subBatch
	curBatchCount int
	curBatchRows  int

	onDone counter.OnDone
	stats  *stat.Values

	timeout *time.Duration

	queueBatch func(predictionBatch) bool

	config.BatcherConfig
	ServiceMeta

	runN uint64
}

type batchFull uint32

const (
	BatchNotFull = batchFull(0)
	BatchIsFull  = batchFull(1)
)

func (d *dispatcher) full() batchFull {
	if d.BatcherConfig.MaxBatchSize > 0 && d.curBatchRows >= d.BatcherConfig.MaxBatchSize {
		return BatchIsFull
	}

	return BatchNotFull
}

func (d *dispatcher) checkBatchQFull() bool {
	return d.BatcherConfig.MaxQueuedBatches > 0 && len(d.batchQ) >= d.BatcherConfig.MaxQueuedBatches
}

func (d *dispatcher) getSubBatches() []subBatch {
	return d.bsPool.Get().([]subBatch)
}

func (d *dispatcher) getActive() []interface{} {
	return d.abPool.Get().([]interface{})
}

func (d *dispatcher) startStats() {
	d.onDone = d.ServiceMeta.dispatcherMetric.Begin(time.Now())
	d.stats = stat.NewValues()
}

func (d *dispatcher) resetStats() {
	d.onDone = nil
	d.stats = nil
}

func (d *dispatcher) endStats() {
	d.onDone(time.Now(), d.stats.Values()...)
}

func (d *dispatcher) zeroDeadline() bool {
	return *d.timeout <= 1
}

func (d *dispatcher) newTimer() {
	d.timer = time.NewTimer(*d.timeout)
}

func (d *dispatcher) appendBatch(batch inputBatch) error {
	err := modifyInterfaceSlice(d.active, batch.inputData)
	if err != nil {
		// TODO this may result in the batch having inconsistent state
		// TODO record
		batch.subBatch.ec <- err
		return err
	}

	d.curBatchRows += batch.batchSize
	if len(d.subBatches) <= d.curBatchCount {
		d.subBatches = append(d.subBatches, batch.subBatch)
	} else {
		d.subBatches[d.curBatchCount] = batch.subBatch
	}

	d.curBatchCount++

	return err
}

func (d *dispatcher) debug(desc string, pfvargs ...interface{}) {
	if d.Verbose != nil {
		if len(pfvargs) > 0 {
			desc = fmt.Sprintf(desc, pfvargs...)
		}

		log.Printf("[%s dispatcher %d] %s timeout:%v size:%d, count:%d",
			d.Verbose.ID, d.runN, desc, d.timeout, d.curBatchRows, d.curBatchCount)
	}
}

func (d *dispatcher) resetSlices() {
	d.active = d.getActive()
	d.subBatches = d.getSubBatches()
}

func (d *dispatcher) submit(statKey string) {
	d.stats.Append(mkbs(statKey, d.curBatchRows, d.curBatchCount))
	d.wg.Add(1)

	d.queueBatch(predictionBatch{d.active, d.subBatches, d.curBatchCount})
	d.endStats()

	d.curBatchCount = 0
	d.curBatchRows = 0

	d.resetSlices()
}

type blockQDebug struct {
	start time.Time
}

func (d *dispatcher) clearBlockQ() {
	d.blockQ <- blockQDebug{time.Now()}
}

// dispatch runs a single input queue read or single batch submission.
// The return value represents if we should terminate the loop.
// See Service.queue() for the producer side.
// See Service.queueBatch() and queueService.dispatch() for consumer side.
// TODO maybe consider design where the deadline is passed via channel and
// then it enters a "with deadline" loop otherwise just be synchronous
func (d *dispatcher) dispatch() bool {
	if d.Verbose != nil {
		dlod := d.dispatcherLoop.Begin(time.Now())
		defer func() { dlod(time.Now()) }()
	}

	d.runN++
	hasDeadline := d.timer != nil
	d.debug("hasDeadline:%v", hasDeadline)
	if !hasDeadline {
		select {
		case batch := <-d.inputQ:
			if d.Verbose != nil {
				onDone := d.inputQDelay.Begin(batch.created)
				onDone(time.Now())
			}

			defer d.clearBlockQ()
			if batch.closed() {
				return true
			}

			d.startStats()
			d.appendBatch(batch)

			if f := d.full(); f != BatchNotFull || d.zeroDeadline() {
				d.debug("instantQ")
				d.submit(InstantQ)
				d.resetStats()
				return false
			}

			d.newTimer()

		}
	} else {
		// We are waiting for a timeout, will collect requests if
		// they come in often enough.
		select {
		case batch := <-d.inputQ:
			if d.Verbose != nil {
				onDone := d.inputQDelay.Begin(batch.created)
				onDone(time.Now())
			}

			defer d.clearBlockQ()

			if batch.closed() {
				if d.curBatchCount > 0 {
					d.debug("closed")
					d.submit(Closing)
				}

				// terminate
				return true
			}

			if err := d.appendBatch(batch); err != nil {
				return false
			}

			if f := d.full(); f != BatchNotFull {
				d.debug("full")
				d.submit(FullBatch)
				d.resetStats()
				d.timer = nil
			}
		case <-d.timer.C:
			// TODO if batch Q is full but max size is not just don't submit?
			if d.curBatchCount > 0 {
				d.debug("timeout")
				d.submit(stat.Timeout)
				d.resetStats()
			}

			d.timer = nil
		}
	}

	return false
}
