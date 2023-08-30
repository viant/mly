package batcher

import (
	"log"
	"time"

	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

// dispatcher represents a state after an iteration of pulling from
// the dispatcher queue.
type dispatcher struct {
	busy     bool
	deadline *time.Timer

	active        []interface{}
	subBatches    []subBatch
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
	BatchTooMany = batchFull(2)
)

func (d *dispatcher) run() {
	go d.Service.run(predictionBatch{d.active, d.subBatches, d.curBatchCount})
}

func (d *dispatcher) full() batchFull {
	if d.curBatchRows >= d.MinBatchSize {
		return BatchIsFull
	}

	if d.curBatchCount >= d.MinBatchCounts {
		return BatchTooMany
	}

	return BatchNotFull
}

// TODO change it so that we check if we're not busy, rather than checking that
// we are busy, as we know we are busy when we run a prediction.
func (d *dispatcher) checkBusy() bool {
	if d.busy {
		return true
	}

	d.busy = d.Service.isBusy()
	return d.busy
}

func (d *dispatcher) appendStat(key string) {
	d.stats.Append(mkbs(key, d.curBatchRows, d.curBatchCount))
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

func (d *dispatcher) debug(action string) {
	if d.Service.Verbose != nil {
		log.Printf("[%s dispatcher] %s timeout:%v busy:%v size:%d, count:%d", d.Service.Verbose.ID, action,
			d.timeout, d.busy, d.curBatchRows, d.curBatchCount)
	}
}

func (d *dispatcher) submit() {
	d.wg.Add(1)
	d.run()
	d.endStats()

	d.curBatchCount = 0
	d.curBatchRows = 0

	d.active = d.Service.getActive()
	d.subBatches = d.Service.getSubBatches()
}

// return value represents if we should terminate the loop
//
// notBusyNoDeadline -> terminate
// notBusyNoDeadline -> notBusyNoDeadline + empty
// notBusyNoDeadline -> busyWithDeadline + start
// notBusyNoDeadline -> notBusyWithDeadline + start
//
// notBusyWithDeadline -> terminate
// notBusyWithDeadline -> notBusyNoDeadline + empty
// notBusyWithDeadline -> notBusyWithDeadline + active
// notBusyWithDeadline -> busyNoDeadline + active
// notBusyWithDeadline -> busyWithDeadline
//
// busyNoDeadline -> notBusyNoDeadline + empty
// busyNoDeadline -> busyWithDeadline
//
// busyWithDeadline -> notBusyWithDeadline + empty
// busyWithDeadline -> busyNoDeadline
// busyWithDeadline -> busyWithDeadline (repeat)
func (d *dispatcher) dispatch() bool {
	if !d.busy && d.deadline == nil {
		d.debug("notBusyNoDeadline")
		// in all cases we reach this point, the active batch should be empty
		select {
		case batch := <-d.q:
			if batch.closed {
				// terminate
				return true
			}

			d.startStats()
			d.appendBatch(batch)

			if f := d.full(); f != BatchNotFull && !d.checkBusy() {
				d.debug("push")
				d.appendStat(FullElements)
				if d.Adjust != nil {
					d.Adjust.Count()
				}
				d.submit()
				d.resetStats()
				// should go back to notBusyNoDeadline
				return false
			}

			// we know we're not full or busy
			// go to either busyWithDeadline or notBusyWithDeadline
			d.newDeadline()
		}
	} else if !d.busy && d.deadline != nil {
		d.debug("notBusyWithDeadline")

		// We are waiting for a timeout, will collect requests if
		// they come in often enough.
		select {
		case batch := <-d.q:
			if batch.closed {
				if d.checkBusy() {
					d.deadline = nil
					// go to busyNoDeadline
					return false
				}

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

			if f := d.full(); f != BatchNotFull && !d.checkBusy() {
				d.debug("go")

				// TODO wrong stat
				d.appendStat(MaxBatches)
				if d.Adjust != nil {
					d.Adjust.Count()
				}
				d.submit()
				d.resetStats()
				d.deadline = nil
				// in this case we would go to notBusyNoDeadline
			}
			// else we go to busyWithDeadline or back to notBusyWithDeadline
		case <-d.deadline.C:
			// d.curBatchCount can be 0 if we were busy and now we're not but
			// the deadline hadn't expired when we received <-d.free
			if d.curBatchCount > 0 {
				if !d.checkBusy() {
					d.debug("timeout")
					d.appendStat(stat.Timeout)
					if d.Adjust != nil {
						d.Adjust.IncTimeout()
					}

					d.submit()
					d.resetStats()
					// in this case we'd end in notBusyNoDeadline
				}
				// we'd end in busyNoDeadline
			} else {
				// TODO track times we get 0 batch timeouts
				// end in notBusyNoDeadline
			}

			d.deadline = nil
		}
	} else if d.busy && d.deadline == nil {
		d.debug("busyNoDeadline")

		// A deadline expired yet we are still busy, or we're closed and busy.
		select {
		case <-d.free:
			// as far as we know we are no longer busy
			d.busy = false
			d.debug("waited")
			d.appendStat(Waiting)
			if d.Adjust != nil {
				d.Adjust.IncOverloaded()
			}
			d.submit()
			d.resetStats()

			// go to notBusyNoDeadline
		case batch := <-d.q:
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
				// TODO track that we should've submitted a batch but couldn't
			}

			d.newDeadline()
			// go to busyWithDeadline
		}
	} else {
		d.debug("busyWithDeadline")

		// We are still busy yet a deadline has not expired.
		// This happens because:
		// 1. we the max batch size or count cap
		// 2. we were busy and got another request which initates a deadline

		// We will not reset the deadline since if we weren't busy,
		// the deadline should've expired anyway.

		select {
		case <-d.free:
			d.busy = false
			d.debug("waited")
			d.appendStat(Waiting)
			if d.Adjust != nil {
				d.Adjust.IncOverloaded()
			}
			d.submit()
			d.resetStats()

			// whenever we full submit, we reset the deadline
			d.deadline = nil
			// go to notBusyNoDeadline
		case batch := <-d.q:
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
				// TODO track that we should've submitted a batch but couldn't
			}

			// go back to busyWithDeadline
		case <-d.deadline.C:
			// just let the deadline expire
			d.deadline = nil
			// go to busyNoDeadline
		}
	}

	return false
}
