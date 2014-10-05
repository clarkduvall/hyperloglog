package main

import (
	"math"
	"hash"
)

type hyperLogLog struct {
	bytes []byte
	m uint32
	p uint8
}

func NewHyperLogLog(precision uint8) *hyperLogLog {
	hll := new(hyperLogLog)
	if precision > 16 || precision < 4 {
		panic("precision must be between 4 and 16")
	}
	hll.p = precision
	hll.m = 1 << uint32(precision)
	hll.bytes = make([]byte, hll.m)
	return hll
}

func (hll *hyperLogLog) Add(item hash.Hash32) {
	x := item.Sum32()
	mask := hll.m - 1
	i := (x >> (32 - hll.p)) & mask  // {x31,...,x32-p} First precision bits of hash
	w := (x << hll.p) | mask  // {x32-p,...,x0}

	zeroBits := clz(w) + 1
	if zeroBits > hll.bytes[i] {
		hll.bytes[i] = zeroBits
	}
}

func (hll *hyperLogLog) calculateE() float64 {
	sum := 0.0
	for _, val := range hll.bytes {
		sum += 1.0 / float64(uint32(1) << val)
	}

	m := float64(hll.m)
	return a(hll.m) * m * m / sum
}

func (hll *hyperLogLog) numZeroes() int {
	count := 0
	for _, val := range hll.bytes {
		if val == 0 {
			count++
		}
	}
	return count
}

func (hll *hyperLogLog) Estimate() uint64 {
	E := hll.calculateE()
	if E <= 2.5 * float64(hll.m) {
		V := hll.numZeroes()
		if V != 0 {
			return uint64(linearCounting(hll.m, uint32(V)))
		}
		return uint64(E)
	} else if E < two32 / 30 {
		return uint64(E)
	}
	return -uint64(two32 * math.Log(1 - E / two32))
}
