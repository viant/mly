package batcher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/tfmodel/batcher/config"
	"github.com/viant/mly/service/tfmodel/evaluator/test"
	"github.com/viant/toolbox"
)

type mockEvaluator struct {
	Entered chan struct{}
	Exited  chan struct{}
}

func (m *mockEvaluator) Evaluate(ctx context.Context, params []interface{}) ([]interface{}, error) {
	m.Entered <- struct{}{}
	fmt.Println("mock trigger done")
	m.Exited <- struct{}{}
	return []interface{}{}, nil
}

func (m mockEvaluator) Close() error {
	return nil
}

func TestServiceConcurrencyMax(t *testing.T) {
	mockEval := &mockEvaluator{
		Entered: make(chan struct{}, 0),
		Exited:  make(chan struct{}, 0),
	}

	mockIn := mockEval.Entered
	mockOut := mockEval.Exited

	bc := config.BatcherConfig{
		MinBatchCounts:          1,
		MaxEvaluatorConcurrency: 1,
	}

	// TODO instead of using the batcher.Service API to test this, use the
	// dispatcher API
	batchSrv := NewBatcher(mockEval, 3, bc, createBSMeta())
	batchSrv.Verbose = &config.V{
		ID:     "test-concurrency",
		Input:  true,
		Output: true,
	}

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}, {"b"}})

	blockerWg := new(sync.WaitGroup)
	blockerWg.Add(1)
	go func() {
		sb, err := batchSrv.queue(feeds)
		assert.Nil(t, err)
		blockerWg.Done()

		select {
		case <-sb.channel:
			blockerWg.Done()
		case err = <-sb.ec:
		}

	}()

	fmt.Println("wait for blocker to queue")
	blockerWg.Wait()

	fmt.Println("wait for blocker to eval")
	<-mockIn

	fmt.Println("queue blocked")
	blockedWg := new(sync.WaitGroup)
	blockedWg.Add(1)
	go func() {
		sb, err := batchSrv.queue(feeds)
		assert.Nil(t, err)
		time.Sleep(1)
		blockedWg.Done()

		select {
		case <-sb.channel:
			blockedWg.Done()
		case err = <-sb.ec:
		}

	}()

	fmt.Println("wait for blocked to queue")
	blockedWg.Wait()

	fmt.Printf("blocked wg:%+v\n", blockedWg)
	// blockeR finishes
	blockerWg.Add(1)
	// blockeD finishes
	blockedWg.Add(1)

	// should trigger blocker to finish
	<-mockOut

	select {
	case <-mockIn:
		t.Error("dispatcher should not have sent")
	case <-mockOut:
		t.Error("weird ordering")
	default:
	}

	// triggered by blocker
	<-mockIn

	// should trigger blocked to finish
	<-mockOut

	blockedWg.Wait()
}

func TestServiceBatchMax(t *testing.T) {
	signature, evaluator, met := test.TLoadEvaluator(t, "example/model/string_lookups_int_model")
	batchSrv := NewBatcher(evaluator, len(signature.Inputs), config.BatcherConfig{
		MinBatchCounts: 3,
		MinBatchSize:   100,
		MinBatchWait:   time.Millisecond * 1,
	}, createBSMeta())

	batchSrv.Verbose = &config.V{
		ID:     "test",
		Input:  true,
		Output: true,
	}

	fmt.Printf("%+v\n", batchSrv)

	feeds := make([]interface{}, 0)
	feeds = append(feeds, [][]string{{"a"}, {"b"}})
	feeds = append(feeds, [][]string{{"c"}, {"d"}})
	singleValue, err := evaluator.Evaluate(context.Background(), feeds)
	assert.Nil(t, err)
	fmt.Printf("%v\n", singleValue)

	feeds3 := make([]interface{}, 0)
	feeds3 = append(feeds3, [][]string{{"e"}, {"f"}, {"g"}})
	feeds3 = append(feeds3, [][]string{{"h"}, {"i"}, {"j"}})
	sv3, err := evaluator.Evaluate(context.Background(), feeds3)
	assert.Nil(t, err)
	fmt.Printf("%v\n", sv3)

	preQ := time.Now()

	wg := new(sync.WaitGroup)

	var errors int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		wg.Add(1)

		go func() {
			sb, err := batchSrv.queue(feeds)
			assert.Nil(t, err)

			wait := time.Now()
			select {
			case r := <-sb.channel:
				if false {
					fmt.Printf("preQ:%s wait:%s\n", time.Since(preQ), time.Since(wait))
				}
				assert.Equal(t, singleValue, r)
			case err = <-sb.ec:
				fmt.Printf("%s\n", err)
				atomic.AddInt32(&errors, 1)
			}

			wg.Done()
		}()

		go func() {
			sb, err := batchSrv.queue(feeds3)
			assert.Nil(t, err)

			wait := time.Now()
			select {
			case r := <-sb.channel:
				if false {
					fmt.Printf("preQ:%s wait:%s\n", time.Since(preQ), time.Since(wait))
				}
				assert.Equal(t, sv3, r)
			case err = <-sb.ec:
				fmt.Printf("%s\n", err)
				atomic.AddInt32(&errors, 1)
			}

			wg.Done()
		}()
	}

	assert.Equal(t, int32(0), errors, "got errors")
	wg.Wait()
	toolbox.Dump(met)
	batchSrv.Close()
}

func createBSMeta() ServiceMeta {
	s := gmetric.New()
	return ServiceMeta{
		queueMetric:      s.OperationCounter("test", "queue", "", time.Microsecond, time.Minute, 2),
		dispatcherMetric: s.MultiOperationCounter("test", "dispatcher", "", time.Microsecond, time.Minute, 2, NewDispatcherP()),
	}
}

func BenchmarkServiceParallel(b *testing.B) {
	bnil := func(err error) {
		if err != nil {
			b.Error(err)
		}
	}

	signature, evaluator, _ := test.LoadEvaluator("example/model/string_lookups_int_model", bnil, bnil)

	bcfg := config.BatcherConfig{
		MinBatchSize:   100,
		MinBatchCounts: 80,
		MinBatchWait:   time.Millisecond * 1,
	}

	batcher := NewBatcher(evaluator, len(signature.Inputs), bcfg, createBSMeta())

	feeds2 := make([]interface{}, 0)
	feeds2 = append(feeds2, [][]string{{"a"}, {"b"}})
	feeds2 = append(feeds2, [][]string{{"c"}, {"d"}})

	evaluator.Evaluate(context.Background(), feeds2)

	preQ := time.Now()

	wg := new(sync.WaitGroup)

	var errors int32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg.Add(1)

			go func() {
				sb, err := batcher.queue(feeds2)
				wait := time.Now()
				select {
				case <-sb.channel:
					if false {
						fmt.Printf("preQ:%s wait:%s\n", time.Since(preQ), time.Since(wait))
					}
				case err = <-sb.ec:
					fmt.Printf("%s\n", err)
					atomic.AddInt32(&errors, 1)
				}

				wg.Done()
			}()
		}
		wg.Wait()
	})

	if errors > 0 {
		b.Error("had errors")
	}

	batcher.Close()
}
