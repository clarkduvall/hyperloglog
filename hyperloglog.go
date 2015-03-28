// Package hyperloglog implements the HyperLogLog and HyperLogLog++ cardinality
// estimation algorithms.
// These algorithms are used for accurately estimating the cardinality of a
// multiset using constant memory. HyperLogLog++ has multiple improvements over
// HyperLogLog, with a much lower error rate for smaller cardinalities.
//
// HyperLogLog is described here:
// http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf
//
// HyperLogLog++ is described here:
// http://research.google.com/pubs/pub40671.html
package hyperloglog

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

const two32 = 1 << 32

type HyperLogLog struct {
	reg []uint8
	m   uint32
	p   uint8
}

// New returns a new initialized HyperLogLog.
func New(precision uint8) (*HyperLogLog, error) {
	if precision > 16 || precision < 4 {
		return nil, errors.New("precision must be between 4 and 16")
	}

	h := &HyperLogLog{}
	h.p = precision
	h.m = 1 << precision
	h.reg = make([]uint8, h.m)
	return h, nil
}

// Clear sets HyperLogLog h back to its initial state.
func (h *HyperLogLog) Clear() {
	h.reg = make([]uint8, h.m)
}

// Add adds a new item to HyperLogLog h.
func (h *HyperLogLog) Add(item Hash32) {
	x := item.Sum32()
	i := eb32(x, 32, 32-h.p) // {x31,...,x32-p}
	w := x<<h.p | 1<<(h.p-1) // {x32-p,...,x0}

	zeroBits := clz32(w) + 1
	if zeroBits > h.reg[i] {
		h.reg[i] = zeroBits
	}
}

// Merge takes another HyperLogLog and combines it with HyperLogLog h.
func (h *HyperLogLog) Merge(other *HyperLogLog) error {
	if h.p != other.p {
		return errors.New("precisions must be equal")
	}

	for i, v := range other.reg {
		if v > h.reg[i] {
			h.reg[i] = v
		}
	}
	return nil
}

// Count returns the cardinality estimate.
func (h *HyperLogLog) Count() uint64 {
	est := calculateEstimate(h.reg)
	if est <= float64(h.m)*2.5 {
		if v := countZeros(h.reg); v != 0 {
			return uint64(linearCounting(h.m, v))
		}
		return uint64(est)
	} else if est < two32/30 {
		return uint64(est)
	}
	return uint64(-two32 * math.Log(1-est/two32))
}

// BinaryMarshaler marshals the HyperLogLog into binary form for storage
func (h *HyperLogLog) BinaryMarshaler() ([]byte, error) {
	data := make([]byte, 5+h.m) // length of p + m + reg

	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.LittleEndian, h.p); err != nil {
		return nil, errors.New("cannot write p")
	}
	if err := binary.Write(buf, binary.LittleEndian, h.m); err != nil {
		return nil, errors.New("cannot write m")
	}
	if err := binary.Write(buf, binary.LittleEndian, h.reg); err != nil {
		return nil, errors.New("cannot write reg")
	}

	if buf.Len() != len(data) {
		return nil, errors.New("buffer length not equal to expected length")
	}

	if _, err := buf.Read(data); err != nil {
		return nil, errors.New("cannot read buffer to []byte")
	}

	return data, nil
}

// BinaryUnmarshaler unmarshals binary data into this HyperLogLog
func (h *HyperLogLog) BinaryUnmarshaler(data []byte) error {
	buf := bytes.NewReader(data)

	if err := binary.Read(buf, binary.LittleEndian, &h.p); err != nil {
		return errors.New("cannot read p")
	}

	if err := binary.Read(buf, binary.LittleEndian, &h.m); err != nil {
		return errors.New("cannot read m")
	}

	if err := binary.Read(buf, binary.LittleEndian, &h.reg); err != nil {
		return errors.New("cannot read reg")
	}

	return nil
}
