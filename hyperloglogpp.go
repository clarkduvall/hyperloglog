package main

import (
	"fmt"
	"math"
	"hash"
	"sort"
)

const pPrime byte = 25
const mPrime uint32 = 1 << (uint32(pPrime) - 1)
const mPrimeMask uint32 = mPrime - 1

var threshold = []uint {
	10, 20, 40, 80, 220, 400, 900, 1800, 3100, 6500, 11500, 20000, 50000, 120000, 350000,
}

type set map[uint32]bool
func (s set) Add(i uint32) { s[i] = true }

type uintSlice []uint32
func (p uintSlice) Len() int           { return len(p) }
func (p uintSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p uintSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type hyperLogLogPP struct {
	bytes []byte
	p byte
	m uint32
	sparse bool
	tmp_set set
	sparse_list []uint32
}

func (hll *hyperLogLogPP) encodeHash(x uint64) uint32 {
	mask := uint64(mPrimeMask - (hll.m - 1))
	shifted := uint32(x & uint64(mPrimeMask)) << 7
	if x & mask == 0 {
		return shifted | (uint32(countZeroBits(x | uint64(mPrimeMask))) << uint32(1)) | 1
	}
	return shifted
}

func (hll *hyperLogLogPP) getIndex(k uint32) uint32 {
	mask := hll.m - 1
	return (k & (mask << 7)) >> 7
}

func (hll *hyperLogLogPP) decodeHash(k uint32) (uint32, byte) {
	r := byte(0)
	if k & 1 == 1 {
		r = byte((k & ((1 << 7) - 2)) >> 1) + (pPrime - hll.p)
	} else {
		r = clz(k)
	}
	return hll.getIndex(k), r
}

func (hll *hyperLogLogPP) insertInSparse(item uint32, i int) {
	hll.sparse_list = append(hll.sparse_list, 0)
	copy(hll.sparse_list[i+1:], hll.sparse_list[i:])
	hll.sparse_list[i] = item
}

func (hll *hyperLogLogPP) merge() {
	mask := mPrimeMask << 7

	keys := make(uintSlice, 0, len(hll.tmp_set))
	for k := range hll.tmp_set {
		keys = append(keys, k)
	}
	sort.Sort(keys)

	var keyLess = func(a uint32, b uint32) bool {
		return a & mask < b & mask
	}

	var keyEqual = func(a uint32, b uint32) bool {
		return a & mask == b & mask
	}

	i := 0
	for _, k := range keys {
		for ; i < len(hll.sparse_list) && keyLess(hll.sparse_list[i], k); i++ {}

		if i >= len(hll.sparse_list) {
			hll.sparse_list = append(hll.sparse_list, k)
			continue
		}

		list_item := hll.sparse_list[i]
		if k > list_item {
			if keyEqual(k, list_item) {
				hll.sparse_list[i] = k
			} else {
				hll.insertInSparse(k, i + 1)
			}
		} else if keyLess(k, list_item) {
			hll.insertInSparse(k, i)
		}
		i++
	}
}

func NewHyperLogLogPP(precision byte, sparse bool) *hyperLogLogPP {
	hll := new(hyperLogLogPP)
	if precision > 16 || precision < 4 {
		panic("precision must be between 4 and 16")
	}
	hll.p = precision
	hll.m = uint32(math.Exp2(float64(precision)))
	hll.sparse = sparse
	if sparse {
		hll.tmp_set = make(set)
	} else {
		hll.bytes = make([]byte, hll.m)
	}
	return hll
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

func (hll *hyperLogLogPP) toNormal() {
	hll.bytes = make([]byte, hll.m)
	for _, k := range hll.sparse_list {
		i, r := hll.decodeHash(k)
		if hll.bytes[i] < r {
			hll.bytes[i] = r
		}
	}
}

func (hll *hyperLogLogPP) Add(item hash.Hash64) {
	x := item.Sum64()
	if hll.sparse {
		k := hll.encodeHash(x)
		hll.tmp_set.Add(k)
		if uint32(len(hll.tmp_set)) * 100 > hll.m * 8 {
			hll.merge()
			hll.tmp_set = make(set)
			if uint32(len(hll.sparse_list)) > hll.m * 8 {
				fmt.Println("SWITCHING ", hll.Estimate())
				hll.sparse = false
				hll.toNormal()
				fmt.Println("SWITCHING ", hll.Estimate())
			}
		}
	} else {
		mask := uint64(hll.m - 1)
		// i := (x >> (64 - hll.p)) & mask  // {x63,...,x64-p} First precision bits of hash
		// w := (x << hll.p) | mask  // {x64-p,...,x0}
		i := x & mask  // {x63,...,x64-p} First precision bits of hash
		w := x | mask  // {x64-p,...,x0}

		zeroBits := countZeroBits(w)
		if zeroBits > hll.bytes[i] {
			hll.bytes[i] = zeroBits
		}
	}
}

func (hll *hyperLogLogPP) estimateBias(E float64) float64 {
	rawEstimate := rawEstimateData[hll.p - 4]
	bias := biasData[hll.p - 4]

	if rawEstimate[0] > E {
		return rawEstimate[0] - bias[0]
	}

	lastEstimate := rawEstimate[len(rawEstimate)-1]
	if lastEstimate < E {
		return lastEstimate - bias[len(bias)-1]
	}

	i := 1
	for ; i < len(rawEstimate) && rawEstimate[i] < E; i++ {}

	e1 := rawEstimate[i - 1]
	e2 := rawEstimate[i]
	b1 := bias[i - 1]
	b2 := bias[i]

	c := (E - e1) / (e2 - e1)
	return b1 * c + b2 * (1 - c)
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
	if hll.sparse {
		fmt.Println("case 1")
		hll.merge()
		return uint64(linearCounting(mPrime, mPrime - uint32(len(hll.sparse_list))))
	}

	E := hll.calculateE()
	if E <= float64(hll.m) * 5.0 {
		fmt.Println("case 2")
		E -= hll.estimateBias(E)
	}
	V := hll.numZeroes()
	H := E
	if V != 0 {
		fmt.Println("case 3")
		H = linearCounting(hll.m, uint32(V))
	}

	if H <= float64(threshold[hll.p - 4]) {
		fmt.Println("case 4")
		return uint64(H)
	}
	fmt.Println("ret", H)
	return uint64(E)
}
