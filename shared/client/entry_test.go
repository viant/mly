package client

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tcrf32 struct {
	expect float64
	prec   int
	input  float64
	delta  float64
	skip   bool
}

func TestReduceFloat32(t *testing.T) {
	p3d := float64(0.0005)
	p4d := float64(0.00005)
	p5d := float64(0.000005)
	p6d := float64(0.0000005)
	p7d := float64(0.00000005)

	tcs := []tcrf32{
		{1.235, 3, 1.23456, p3d, false},
		{2.001, 3, 2.0009, p3d, false},

		{3.0001, 4, 3.00008, p4d, false},
		{4.9998, 4, 4.99978, p4d, false},

		{5.00009, 5, 5.0000949, p5d, false},

		// on linux intel this and below starts to fail
		// float32(5.00000949) =~ 5.0000095367
		{5.000009, 6, 5.00000949, p6d, true},
		{5.0000009, 7, 5.000000949, p7d, true},
	}

	for i, tc := range tcs {
		fe := FloatEntry(tc.prec)
		m := fmt.Sprintf("%d: prec:%d input:%0.10f delta:%0.10f fm64:%0.10f", i, tc.prec, tc.input, tc.delta, fe.fm64)
		actual := fe.reduceFloat32(float32(tc.input))
		if tc.skip {
			continue
		}
		assert.InDelta(t, tc.expect, actual, tc.delta, m)
	}
}

func TestFloatSize(t *testing.T) {
	fmt.Printf("32:%0.10f 64:%0.10f\n", float32(1.00000949), float64(1.00000949))
}

func TestReduceF32(t *testing.T) {
	assert.InDelta(t, 1.235, reduceF32(1.23456, 1000), 0.0005, "float32")

	var f float32
	f, _, _ = reduceBigF32(1.23456, big.NewFloat(1000))
	assert.InDelta(t, 1.235, f, 0.0005, "big.Float")

	f, _, _ = reduceBigF32(1.23456, big.NewFloat(10000))
	assert.InDelta(t, 1.2346, f, 0.00005, "big.Float")
}

func BenchmarkReduceBigF32Parallel(b *testing.B) {
	fm := big.NewFloat(10000)

	var trs float32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rf, _, _ := reduceBigF32(1.23456, fm)
			trs += rf
		}
	})
}

func BenchmarkReduceF32Parallel(b *testing.B) {
	fm := float64(10000)

	var trs float32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rf := reduceF32(1.23456, fm)
			trs += rf
		}
	})
}

func BenchmarkHasInt(b *testing.B) {
	ivs := []int{0, 1, 2, 3, 4, 5}
	ivm := make(map[int]bool, len(ivs))
	for _, v := range ivs {
		ivm[v] = true
	}

	e := entry{
		ints: ivm,
	}

	var trs int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rf := e.hasInt(3)
			if rf {
				trs += 1
			}
		}
	})
}

func BenchmarkHasString(b *testing.B) {
	ivs := []string{"a", "b", "cde", "verylongdomainname.com"}
	ivm := make(map[string]bool, len(ivs))
	for _, v := range ivs {
		ivm[v] = true
	}

	e := entry{
		strings: ivm,
	}

	var trs int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rf := e.hasString("verylongdomainname.com")
			if rf {
				trs += 1
			}
		}
	})
}
