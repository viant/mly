package semaph

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCore(t *testing.T) {
	semaSize := 2
	s := NewSemaph(int32(semaSize))

	numWaiters := 3

	waitAcq := new(sync.WaitGroup)
	waitAcq.Add(numWaiters)

	waitAcqd := new(sync.WaitGroup)
	waitAcqd.Add(semaSize)

	ol := new(sync.Mutex)
	ol.Lock()

	dones := make([]bool, numWaiters)
	var doneL bool

	bctx := context.Background()

	// since semaphore size is 2 and num workers is 3, we should have 1 worker block
	for x := 1; x <= numWaiters; x++ {
		go func(id int, marker *bool) {
			waitAcq.Done()
			fmt.Printf("i%d acquiring\n", id)
			s.Acquire(bctx)
			defer s.Release()
			waitAcqd.Done()

			fmt.Printf("i%d wait for m0 lock\n", id)
			ol.Lock()
			defer ol.Unlock()
			*marker = true

			fmt.Printf("i%d done\n", id)
		}(x, &dones[x-1])
	}

	fmt.Printf("m0 wait for semaphores to be acquired by goroutines\n")
	waitAcq.Wait()
	fmt.Printf("m0 goroutines should have acquired\n")

	waitAcqd.Wait()

	// prevent error from calling wg.Done() too many times
	waitAcqd.Add(numWaiters - semaSize)

	assert.Equal(t, s.r, int32(0), "the sempahore should not be available")

	fmt.Printf("m0 unlock as enough goroutines should have started\n")
	ol.Unlock()
	fmt.Printf("m0 unlocked\n")

	s.Acquire(bctx)
	fmt.Printf("m0 acquire proceeded since a goroutine waiting on the lock finished\n")

	fl := new(sync.WaitGroup)
	fl.Add(1)
	go func() {
		fmt.Printf("iL wait for acquire might need to wait for prior 2 goroutines\n")
		s.Acquire(bctx)
		defer s.Release()

		assert.Equal(t, s.r, int32(0), "main and latest goroutine should have locked semaphore")
		doneL = true

		fmt.Printf("iL done\n")
		fl.Done()
	}()

	fmt.Printf("m0 wait for i3 goroutine to run\n")
	// once waiting, the original 2 goroutines should've completed
	fl.Wait()

	fmt.Printf("m0 i3 goroutine has unblocked waitgroup\n")
	s.Release()

	for x := 0; x < numWaiters; x++ {
		assert.True(t, dones[x], fmt.Sprintf("inner bool %d", x+1))
	}

	assert.True(t, doneL, "last bool")
}

// not really a test
func TestOrdering(t *testing.T) {
	tests := 8

	s := NewSemaph(1)

	wgs := new(sync.WaitGroup)
	wgs.Add(tests)

	wgd := new(sync.WaitGroup)
	wgd.Add(tests)

	s.Acquire(context.Background())
	for i := 1; i < tests+1; i += 1 {
		ii := i
		go func() {
			wgs.Done()
			fmt.Printf("%d-acquire\n", ii)
			s.Acquire(context.Background())
			defer s.Release()
			fmt.Printf("%d-done\n", ii)
			wgd.Done()
		}()
	}

	wgs.Wait()
	s.Release()
	wgd.Wait()
}

// There is a case that doesn't seem possible to test deterministically - this will attempt to hit it.
// In the documentation for sync.(*Cond).Signal(), there is a line:
// > Signal() does not affect goroutine scheduling priority; if other goroutines are attempting to lock c.L, they may be awoken before a "waiting" goroutine.
// This means that depending on how golang scheduling decides to work, 2 Release() calls could be invoked sequentially.
// This had a bug since there was an assumption that if Semaph.r was 0, there were goroutines Wait()-ing;
// but if 2 Release() calls happened before a goroutine Wait()-ing was woken, then on the second Release() call, there would be no Signal().
func TestBurst(t *testing.T) {
	sc := int32(3)

	gomaxthreads := int32(10000)

	s := NewSemaph(sc)
	nt := int(sc + gomaxthreads + 5000)

	wgs := new(sync.WaitGroup)
	wgs.Add(nt)

	wgr := new(sync.WaitGroup)
	wgr.Add(nt)

	l := new(sync.Mutex)
	l.Lock()

	var lockWait, lockDone, semaAcq, semaRel int32

	maxLock := new(sync.Mutex)
	semaLock := new(sync.Mutex)
	var maxLockWait, maxSemaAcq int32

	for i := 0; i < nt; i += 1 {
		ii := i
		go func() {
			maxLock.Lock()
			lockWait += 1
			clw := lockWait - lockDone

			if maxLockWait < clw {
				maxLockWait = clw
			}

			maxLock.Unlock()

			wgr.Wait()

			maxLock.Lock()
			lockDone += 1
			maxLock.Unlock()

			semaLock.Lock()
			semaAcq += 1
			sad := semaAcq - semaRel
			if maxSemaAcq < sad {
				maxSemaAcq = sad
			}
			semaLock.Unlock()

			s.acquireDebug(context.Background(), func(n int32) string {
				return fmt.Sprintf("%d, %d\n", ii, n)
			})
			wgs.Done()
			s.releaseDebug(func(n int32) string {
				return fmt.Sprintf("releasing %d, %d\n", ii, n)
			})

			semaLock.Lock()
			semaRel += 1
			semaLock.Unlock()

		}()
		wgr.Done()
	}

	l.Unlock()

	wgs.Wait()

	fmt.Printf("maxLockWait:%d maxSemaAcq:%d\n", maxLockWait, maxSemaAcq)
	assert.True(t, maxLockWait > gomaxthreads, "did not check if 10000+ goroutines locked trigger thread crash")
}

func TestCtx(t *testing.T) {
	s := NewSemaph(2)

	bctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	s.Acquire(bctx)
	s.Acquire(bctx)
	err := s.Acquire(ctx)

	assert.NotNil(t, err)
}
