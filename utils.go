package main

import "math"

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

func countZeroBits2(num uint32, start uint) byte {
	count := byte(1)
	for x := uint32(1 << (start - 1)); (x & num) == 0 && x != 0; x >>= 1 {
		count++
	}
	return count
}

func countZeroBits(num uint64) byte {
	count := byte(1)
	for x := uint64(1 << 63); (x & num) == 0 && x != 0; x >>= 1 {
		count++
	}
	return count
}

// Extract bits from uint32 using LSB 0 numbering, including lo
func eb32(bits uint32, hi uint, lo uint) uint32 {
	m := uint32(((1 << (hi - lo)) - 1) << lo)
	return (bits & m) >> lo
}

// Extract bits from uint64 using LSB 0 numbering, including lo
func eb64(bits uint64, hi uint, lo uint) uint64 {
	m := uint64(((1 << (hi - lo)) - 1) << lo)
	return (bits & m) >> lo
}

func linearCounting(m uint32, v uint32) float64 {
	fm := float64(m)
	return fm * math.Log(fm / float64(v))
}
