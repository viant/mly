package buckets

// This contains common buckets for Prometheus metrics.
// We generally observe smaller models in 300 microsecond ranges, with most common occurring between 500 and 1000 microseconds.
// For inference, we usually do not care about inferences that take longer than 1 second, as we operate in RTB and the maximum acceptable latency is around 120 milliseconds.

var MicrosecondBuckets []float64 = []float64{
	100, 500,
	// 1 millisecond
	1000, 2000, 3000, 5000, 7500,
	10000, 20000, 30000, 50000, 75000,
	100000, 200000, 400000, 800000,
	// 1 second
	1000000, 2000000,
}

var MillisecondBuckets []float64 = []float64{
	1, 2, 3, 5, 8,
	10, 20, 40, 80,
	100, 200, 400, 800,
	1000, 2000, 4000, 8000,
}

var SecondBuckets []float64 = []float64{
	1e-4, 5e-4,
	1e-3, 2e-3, 3e-3, 5e-3, 7.5e-3,
	1e-2, 2e-2, 4e-2, 8e-2,
	1e-1, 2e-1, 4e-1, 8e-1,
	1,
}

var CommonSummaryObjectives = map[float64]float64{
	0.5:   0.05,
	0.9:   0.01,
	0.95:  0.005,
	0.99:  0.001,
	0.999: 0.001,
}
