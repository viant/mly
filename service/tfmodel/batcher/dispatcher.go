package batcher

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

// dispatcher represents a state after an iteration of pulling from
// the dispatcher queue.
type dispatcher struct {
	deadline *time.Timer

	// shared with Service
	bsPool *sync.Pool
	abPool *sync.Pool

	active     []interface{}
	subBatches []subBatch

	curBatchCount int
	curBatchRows  int

	onDone counter.OnDone
	stats  *stat.Values

	timeout *time.Duration
	*Service
}

type batchFull uint32

const (
	BatchNotFull = batchFull(0)
	BatchIsFull  = batchFull(1)
	BatchMaxHit  = batchFull(2)
)

func (d *dispatcher) full() batchFull {
	if d.curBatchRows >= d.Service.BatcherConfig.MaxBatchSize {
		return BatchIsFull
	}

	return BatchNotFull
}

func (d *dispatcher) checkBusy() bool {
	return d.Service.BatcherConfig.MaxQueuedBatches > 0 &&
		len(d.Service.batchQ) >= d.Service.BatcherConfig.MaxQueuedBatches
}

func (d *dispatcher) appendStat(key string) {
	d.stats.Append(mkbs(key, d.curBatchRows, d.curBatchCount))
}

func (d *dispatcher) getSubBatches() []subBatch {
	return d.bsPool.Get().([]subBatch)
}

func (d *dispatcher) getActive() []interface{} {
	return d.abPool.Get().([]interface{})
}

func (d *dispatcher) startStats() {
	d.onDone = d.Service.dispatcherMetric.Begin(time.Now())
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

func (d *dispatcher) newDeadline() {
	d.deadline = time.NewTimer(*d.timeout)
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
	if d.Service.Verbose != nil {
		if len(pfvargs) > 0 {
			desc = fmt.Sprintf(desc, pfvargs...)
		}

		log.Printf("[%s dispatcher] %s timeout:%v size:%d, count:%d",
			d.Service.Verbose.ID, desc, d.timeout, d.curBatchRows, d.curBatchCount)
	}
}

func (d *dispatcher) resetSlices() {
	d.active = d.getActive()
	d.subBatches = d.getSubBatches()
}

func (d *dispatcher) submit() {
	d.wg.Add(1)

	d.Service.queueBatch(predictionBatch{d.active, d.subBatches, d.curBatchCount})
	d.endStats()

	d.curBatchCount = 0
	d.curBatchRows = 0

	d.resetSlices()
}

// return value represents if we should terminate the loop
func (d *dispatcher) dispatch() bool {
	isBusy := d.checkBusy()
	hasDeadline := d.deadline != nil

	d.debug("busy:%v hasDeadline:%v", isBusy, hasDeadline)
	if !isBusy && !hasDeadline {
		// in all cases we reach this point, the active batch should be empty
		select {
		case batch := <-d.inputQ:
			if batch.closed {
				// terminate
				return true
			}

			d.startStats()
			d.appendBatch(batch)

			if f := d.full(); f != BatchNotFull || d.zeroDeadline() {
				d.debug("push")
				// TODO stats
				d.appendStat(FullElements)
				d.submit()
				d.resetStats()

				// go to (back to) notBusyNoDeadline or busyNoDeadline
				return false
			}

			d.newDeadline()
			// go to notBusyWithDeadline
		}
	} else if !isBusy && hasDeadline {
		// aka notBusyWithDeadline

		// We are waiting for a timeout, will collect requests if
		// they come in often enough.
		select {
		case batch := <-d.inputQ:
			if batch.closed {
				if d.curBatchCount > 0 {
					d.debug("closed")
					d.appendStat(Closing)
					d.submit()
				}

				// terminate
				return true
			}

			if err := d.appendBatch(batch); err != nil {
				return false
			}

			if f := d.full(); f != BatchNotFull {
				d.debug("go")

				// TODO stat
				d.appendStat(MaxBatches)
				d.submit()
				d.resetStats()

				d.deadline = nil
				// go to notBusyNoDeadline or busyNoDeadline
			}
			// else go to (back to) notBusyWithDeadline or busyWithDeadline
		case <-d.deadline.C:
			// d.curBatchCount can be 0 if we were busy and now we're not but
			// the deadline hadn't expired when we received <-d.bqFree
			if d.curBatchCount > 0 {
				d.debug("timeout")
				d.appendStat(stat.Timeout)
				d.submit()
				d.resetStats()
				// go to notBusyNoDeadline or busyNoDeadline
			} else {
				// TODO track times we get 0 batch timeouts
				// go to notBusyNoDeadline
			}

			d.deadline = nil
		}
	} else if isBusy && !hasDeadline {
		// aka busyNoDeadline
		// A deadline expired yet we are still busy, or we're closed and busy.
		select {
		case <-d.bqFree:
			d.debug("waited")
			d.appendStat(Waiting)
			d.submit()
			d.resetStats()
			// go to notBusyNoDeadline or busyNoDeadline
		case batch := <-d.inputQ:
			if batch.closed {
				// do nothing since we need to wait for a "free worker"
				// go to busyNoDeadline
				return false
			}

			err := d.appendBatch(batch)
			if err != nil {
				return false
			}

			if f := d.full(); f != BatchNotFull {
				if f == BatchMaxHit {
					// TODO enter shedding
					d.setShedding(true)
				}
				// TODO track that we should've submitted a batch but couldn't
			}

			d.newDeadline()
			// go to busyWithDeadline
		}
	} else {
		// aka busyWithDeadline

		// We are still busy yet a deadline has not expired.
		// This happens because:
		// 1. we the max batch size or count cap
		// 2. we were busy and got another request which initates a deadline

		// We will not reset the deadline since if we weren't busy,
		// the deadline should've expired anyway.

		select {
		case <-d.bqFree:
			d.debug("waited")
			d.appendStat(Waiting)
			d.submit()
			d.resetStats()

			// whenever we full submit, we reset the deadline
			d.deadline = nil
			// go to notBusyNoDeadline or busyNoDeadline
		case batch := <-d.inputQ:
			if batch.closed {
				d.deadline = nil
				// go to busyNoDeadline
				return false
			}

			err := d.appendBatch(batch)
			if err != nil {
				return false
			}

			if f := d.full(); f != BatchNotFull {
				if f == BatchMaxHit {
					// TODO enter shedding
					d.setShedding(true)
				}
				// TODO track that we should've submitted a batch but couldn't
			}

			// go back to busyWithDeadline
		case <-d.deadline.C:
			// just let the deadline expire
			// TODO count
			d.deadline = nil
			// go to busyNoDeadline
		}
	}

	return false
}
