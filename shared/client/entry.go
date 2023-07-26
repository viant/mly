package client

import (
	"math"
	"math/big"
)

type entry struct {
	ints     map[int]bool
	float32s map[float32]bool // Deprecated: floats are never used for lookup
	strings  map[string]bool

	prec uint
	fm64 float64
}

func (e *entry) hasString(val string) bool {
	if len(e.strings) == 0 {
		return false
	}
	_, ok := e.strings[val]
	return ok
}

func (e *entry) hasInt(val int) bool {
	if len(e.ints) == 0 {
		return false
	}
	_, ok := e.ints[val]
	return ok
}

// bid.Accuracy can effectively be ignored in practice
// I am unsure of the "canonical" or "standard" way of rounding to a decimal in Golang.
func (e *entry) reduceFloat32(val float32) float32 {
	if e.fm64 > 0 {
		return reduceF32(val, e.fm64)
	}

	return val
}

// reference implementation with higher precision but slower performance
// most likely not needed until needing precision beyond float64
func reduceBigF32(val float32, m *big.Float) (float32, big.Accuracy, big.Accuracy) {
	f := big.NewFloat(float64(val))
	f = f.Mul(f, m)
	// post multiplication accuracy
	ff, pma := f.Float64()
	ff = math.Round(ff)
	f = big.NewFloat(float64(ff))
	f = f.Quo(f, m)
	// post division accuracy
	fo, pda := f.Float32()
	return fo, pma, pda
}

// implementation with lower precision but (~50x) higher performance
// this should be sufficient for most needs
func reduceF32(f float32, m float64) float32 {
	f64 := float64(f)
	f64 = f64 * m
	f64 = math.Round(f64)
	return float32(f64 / m)
}

// FloatEntry creates entry with prec digits after the decimal.
// A prec of less than or equal to 0 will be ignored.
func FloatEntry(prec int) *entry {
	e := new(entry)
	e.setPrec(prec)
	return e
}

func (e *entry) setPrec(prec int) {
	if prec > 0 {
		e.fm64 = math.Pow10(prec)
		e.prec = uint(prec)
	}
}
