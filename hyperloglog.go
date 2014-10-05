package main

import (
	"math"
	"hash"
)

const a16 = 0.673
const a32 = 0.697
const a64 = 0.709
const two32 = 1 << 32

func a(m uint32) float64 {
	if m == 16 {
		return a16
	} else if m == 32 {
		return a32
	} else if m == 64 {
		return a64
	}
	return 0.7213 / (1 + 1.079 / float64(m))
}

var clzLookup = [...]byte {
	32, 31, 30, 30, 29, 29, 29, 29, 28, 28, 28, 28, 28, 28, 28, 28,
}

// http://embeddedgurus.com/state-space/2014/09/fast-deterministic-and-portable-counting-leading-zeros/
// func countZeroBits(num uint32) byte {
// 	count := byte(0)
// 	for x := uint32(1 << 31); (x & num) == 0 && x != 0; x >>= 1 {
// 		count++
// 	}
// 	return count
// }
func clz(x uint32) byte {
	n := byte(0)

	if x >= (1 << 16) {
		if x >= (1 << 24) {
			if x >= (1 << 28) { n = 28 } else { n = 24 }
		} else {
			if x >= (1 << 20) { n = 20 } else { n = 16 }
		}
	} else {
		if x >= (1 << 8) {
			if x >= (1 << 12) { n = 12 } else { n = 8 }
		} else {
			if x >= (1 << 4) { n = 4 } else { n = 0 }
		}
	}
	return clzLookup[x >> n] - n;
}

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
	hll.m = uint32(math.Exp2(float64(precision)))
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
		sum += 1.0 / math.Exp2(float64(val))
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

func linearCounting(m uint32, v uint32) float64 {
	fm := float64(m)
	return fm * math.Log(fm / float64(v))
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
