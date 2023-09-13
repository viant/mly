package permute

import (
	"time"
)

type Permuter struct {
	ints    [][]int
	durs    [][]time.Duration
	uint32s [][]uint32

	lastInc int
	maxInc  int
}

type Permutation struct {
	Ints    []int
	Durs    []time.Duration
	Uint32s []uint32
}

func NewPermuter(ints [][]int, durs [][]time.Duration, uint32s [][]uint32) *Permuter {
	max := 1
	for _, vs := range ints {
		max *= len(vs)
	}

	for _, vs := range durs {
		max *= len(vs)
	}

	for _, vs := range uint32s {
		max *= len(vs)
	}

	return &Permuter{
		ints:    ints,
		durs:    durs,
		uint32s: uint32s,
		maxInc:  max,
	}
}

func (p *Permuter) Next() (bool, Permutation) {
	lints := len(p.ints)
	ldurs := len(p.durs)
	luint32s := len(p.uint32s)

	it := Permutation{
		Ints:    make([]int, lints),
		Durs:    make([]time.Duration, ldurs),
		Uint32s: make([]uint32, luint32s),
	}

	div := 1
	curN := p.lastInc

	for i, v := range p.ints {
		lv := len(v)
		offset := curN / div % lv
		it.Ints[i] = v[offset]
		div *= lv
	}

	for i, v := range p.durs {
		lv := len(v)
		offset := curN / div % lv
		it.Durs[i] = v[offset]
		div *= lv
	}

	for i, v := range p.uint32s {
		lv := len(v)
		offset := curN / div % lv
		it.Uint32s[i] = v[offset]
		div *= lv
	}

	p.lastInc++
	return p.lastInc < p.maxInc, it
}
