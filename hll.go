package main

import (
	"errors"
	"hash"
	"math"
)

var two32 float64 = 1 << 32

type hyperLogLog struct {
	reg []uint8
	m   uint32
	p   uint8
}

func NewHyperLogLog(precision uint8) (*hyperLogLog, error) {
	if precision > 16 || precision < 4 {
		return nil, errors.New("precision must be between 4 and 16")
	}

	h := new(hyperLogLog)
	h.p = precision
	h.m = 1 << precision
	h.reg = make([]uint8, h.m)
	return h, nil
}

func (h *hyperLogLog) Clear() {
	h.reg = make([]uint8, h.m)
}

func (h *hyperLogLog) Add(item hash.Hash32) {
	x := item.Sum32()
	i := eb32(x, 32, 32 - h.p)      // {x31,...,x32-p}
	w := x << h.p | 1 << (h.p - 1)  // {x32-p,...,x0}

	zeroBits := clz32(w) + 1
	if zeroBits > h.reg[i] {
		h.reg[i] = zeroBits
	}
}

func (h *hyperLogLog) Estimate() uint64 {
	est := calculateEstimate(h.reg)
	if est <= float64(h.m) * 2.5 {
		if v := countZeros(h.reg); v != 0 {
			return uint64(linearCounting(h.m, v))
		}
		return uint64(est)
	} else if est < two32 / 30 {
		return uint64(est)
	}
	return -uint64(two32 * math.Log(1 - est / two32))
}
