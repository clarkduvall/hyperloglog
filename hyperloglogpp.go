package main

import (
	"math"
	"hash"
)

type hyperLogLogPP struct {
	bytes []byte
	m uint32
	ms uint32
	sparse bool
	tmp_set map[uint32]bool
	sparse_list []uint32
}

func (hll *hyperLogLogPP) encodeHash(x uint64) uint32 {
	ms_mask := uint64(hll.ms - 1)
	msm_mask := ms_mask - uint64(hll.m - 1)
	shifted_i := uint32((x & ms_mask) << 25)
	if (x & msm_mask) == 0 {
		return shifted_i | (uint32(countZeroBits(x | ms_mask)) << 1) | 1
	} else {
		return shifted_i
	}
}

func NewHyperLogLogPP(precision uint8) *hyperLogLogPP {
	hll := new(hyperLogLogPP)
	if precision > 16 || precision < 4 {
		panic("precision must be between 4 and 16")
	}
	hll.m = uint32(math.Exp2(float64(precision)))
	hll.ms = 1 << 25
	hll.sparse = false
	hll.tmp_set = make(map[uint32]bool)
	hll.bytes = make([]byte, hll.m)
	return hll
}

func countZeroBits(num uint64) byte {
	count := byte(0)
	for x := uint64(1 << 63); (x & num) == 0 && x != 0; x >>= 1 {
		count++
	}
	return count
}

func (hll *hyperLogLogPP) merge() {
}

func (hll *hyperLogLogPP) toNormal() {
}

func (hll *hyperLogLogPP) Add(item hash.Hash64) {
	x := item.Sum64()
	if hll.sparse {
		k := hll.encodeHash(x)
		hll.tmp_set[k] = true
		if uint32(len(hll.tmp_set)) * 100 > hll.m * 8 {
			hll.merge()
			hll.tmp_set = make(map[uint32]bool)
			if uint32(len(hll.sparse_list)) > hll.m * 8 {
				hll.sparse = false
				hll.toNormal()
			}
		}
	} else {
		mask := uint64(hll.m - 1)
		i := x & mask  // {x63,...,x64-p} First precision bits of hash
		w := x | mask  // {x64-p,...,x0}

		zeroBits := countZeroBits(w) + 1
		if zeroBits > hll.bytes[i] {
			hll.bytes[i] = zeroBits
		}
	}
}

func (hll *hyperLogLogPP) calculateE() float64 {
	sum := 0.0
	for _, val := range hll.bytes {
		sum += 1.0 / math.Exp2(float64(val))
	}

	m := float64(hll.m)
	return a(hll.m) * m * m / sum
}

func (hll *hyperLogLogPP) numZeroes() int {
	count := 0
	for _, val := range hll.bytes {
		if val == 0 {
			count++
		}
	}
	return count
}

func (hll *hyperLogLogPP) Estimate() uint64 {
	E := hll.calculateE()
	if E <= 2.5 * float64(hll.m) {
		V := hll.numZeroes()
		if V != 0 {
			return linearCounting(hll.m, uint32(V))
		}
		return uint64(E)
	} else if E < two32 / 30 {
		return uint64(E)
	}
	return -uint64(two32 * math.Log(1 - E / two32))
}
