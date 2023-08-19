package buffer

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkPool(b *testing.B) {
	sizes := []int{512, 1024, 2048, 4096, 32768}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size:%05d", size), func(b *testing.B) {
			var put uint64
			p := New(size, 1024)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					data := p.Get()

					if i%3 == 0 {
						go func() {
							time.Sleep(1 * time.Microsecond)
							p.Put(data)
							atomic.AddUint64(&put, 1)
						}()
					}

					i++
				}
			})

			b.ReportMetric(float64(p.Created), "created")
			b.ReportMetric(float64(p.Discarded), "discarded")
			b.ReportMetric(float64(put), "put")
		})
	}
}
