package semaph

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	s := NewSemaph(2)
	wg := new(sync.WaitGroup)
	wg.Add(2)

	ol := new(sync.Mutex)
	ol.Lock()

	var done1, done2, done3 bool

	go func() {
		// 2 -> 1
		s.Acquire()
		defer s.Release()
		wg.Done()

		fmt.Printf("i1 wait for m0 lock\n")
		ol.Lock()
		defer ol.Unlock()
		done1 = true

		fmt.Printf("i1 done\n")
	}()

	go func() {
		// 1 -> 0
		s.Acquire()
		defer s.Release()
		wg.Done()

		fmt.Printf("i2 wait for m0 lock\n")
		ol.Lock()
		defer ol.Unlock()
		done2 = true

		fmt.Printf("i2 done\n")
	}()

	fmt.Printf("m0 wait for semaphores to be acquired by goroutines\n")
	wg.Wait()

	fmt.Printf("m0 unlock as both goroutes should have acquired\n")
	assert.Equal(t, s.i, int32(0), "the sempahore should not be available")
	ol.Unlock()

	s.Acquire()
	fmt.Printf("m0 acquire proceeded since a goroutine waiting on the lock finished\n")
	// should have either 0 or 1 available in semaphore

	fl := new(sync.WaitGroup)
	fl.Add(1)
	go func() {
		fmt.Printf("i3 wait for acquire might need to wait for prior 2 goroutines\n")
		s.Acquire()
		defer s.Release()

		assert.Equal(t, s.i, int32(0), "main and latest goroutine should have locked semaphore")
		done3 = true

		fmt.Printf("i3 done\n")
		fl.Done()
	}()

	fmt.Printf("m0 wait for i3 goroutine to run\n")
	// once waiting, the original 2 goroutines should've completed
	fl.Wait()

	fmt.Printf("m0 i3 goroutine has unblocked waitgroup\n")
	s.Release()

	assert.True(t, done1, "inner bool 1")
	assert.True(t, done2, "inner bool 2")
	assert.True(t, done3, "last bool")
}

// not really a test
func TestOrdering(t *testing.T) {
	tests := 8

	s := NewSemaph(1)

	wgs := new(sync.WaitGroup)
	wgs.Add(tests)

	wgd := new(sync.WaitGroup)
	wgd.Add(tests)

	s.Acquire()
	for i := 1; i < tests+1; i += 1 {
		ii := i
		go func() {
			wgs.Done()
			fmt.Printf("%d-acquire\n", ii)
			s.Acquire()
			defer s.Release()
			fmt.Printf("%d-done\n", ii)
			wgd.Done()
		}()
	}

	wgs.Wait()
	s.Release()
	wgd.Wait()
}
