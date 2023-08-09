package semaph

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type Semaph struct {
	l   sync.Mutex // locks for modifying r
	c   *sync.Cond
	r   int32 // i.e. remaining tickets
	max int32

	stats Stats
}

type Stats struct {
	Acquired,
	Waited,
	Canceled,
	CanceledDone,
	WaitDone,
	WaitCanceled uint64
}

func NewSemaph(max int32) *Semaph {
	s := new(Semaph)
	s.r = max
	s.max = max

	s.l = sync.Mutex{}
	s.c = sync.NewCond(&s.l)

	return s
}

func (s *Semaph) Internals() (int32, Stats) {
	return s.r, s.stats
}

// Acquire will block if there are no more "tickets" left; otherwise will decrement number of tickets and continue.
// Caller must call Release() later, unless an error is returned, which should always be context.Context.Err().
func (s *Semaph) Acquire(ctx context.Context) error {
	s.l.Lock()

	atomic.AddUint64(&s.stats.Acquired, 1)

	l := new(sync.Mutex)
	c := make(chan bool, 1)
	for s.r <= 0 {
		var done, canceled bool
		go func(cc *bool) {
			// this should unlock
			s.c.Wait()
			// this would lock
			l.Lock()
			defer l.Unlock()

			if *cc {
				atomic.AddUint64(&s.stats.WaitCanceled, 1)

				// outer routine would exist without unlocking
				defer s.l.Unlock()
				// "pass the torch" to next thing Wait()-ing
				s.c.Signal()

				return
			}

			atomic.AddUint64(&s.stats.WaitDone, 1)
			done = true
			c <- true
		}(&canceled)

		select {
		case <-ctx.Done():
			l.Lock()
			defer l.Unlock()

			if done {
				// while we were waiting, the lock was released
				defer s.l.Unlock()
				s.c.Signal()
				atomic.AddUint64(&s.stats.CanceledDone, 1)
				return ctx.Err()
			}

			// this may still incur the Wait call completing
			canceled = true
			atomic.AddUint64(&s.stats.Canceled, 1)

			return ctx.Err()
		case <-c:
			atomic.AddUint64(&s.stats.Waited, 1)
			// should've slept and locked
			// but we can wait again due to s.r == 0
		}
	}

	defer s.l.Unlock()
	s.r -= 1
	return nil
}

func (s *Semaph) acquireDebug(ctx context.Context, f func(n int32) string) error {
	s.l.Lock()

	fmt.Printf("acquiring %s", f(s.r))

	c := make(chan bool, 1)
	for s.r <= 0 {
		canceled := false
		go func(cc *bool) {
			s.c.Wait()
			if *cc {
				fmt.Printf("waited but canceled %s", f(s.r))
				defer s.l.Unlock()
				s.c.Signal()
				return
			}
			c <- true
		}(&canceled)

		select {
		case <-ctx.Done():
			fmt.Printf("canceled %s", f(s.r))
			canceled = true
			return ctx.Err()
		case <-c:
			fmt.Printf("waited %s", f(s.r))
		}
	}

	defer s.l.Unlock()
	s.r -= 1
	return nil
}

// Release will free up a "ticket", and if there were any waiting goroutines, will Signal() (one of) them.
func (s *Semaph) Release() {
	s.l.Lock()
	defer s.l.Unlock()

	if s.r < s.max {
		s.r += 1
		s.c.Signal()
	}
}

func (s *Semaph) releaseDebug(f func(n int32) string) {
	s.l.Lock()
	defer s.l.Unlock()

	fmt.Printf("%s", f(s.r))

	if s.r < s.max {
		s.r += 1
		s.c.Signal()
	}
}
