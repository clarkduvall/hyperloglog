package hyperloglog

import (
	"bytes"
	"encoding/gob"
	"errors"
	"sort"
)

const pPrime = 25
const mPrime = 1 << (pPrime - 1)

var threshold = []uint{
	10, 20, 40, 80, 220, 400, 900, 1800, 3100,
	6500, 11500, 20000, 50000, 120000, 350000,
}

type HyperLogLogPlus struct {
	reg        []uint8
	p          uint8
	m          uint32
	sparse     bool
	tmpSet     set
	sparseList *compressedList
}

// Encode a hash to be used in the sparse representation.
func (h *HyperLogLogPlus) encodeHash(x uint64) uint32 {
	idx := uint32(eb64(x, 64, 64-pPrime))

	if eb64(x, 64-h.p, 64-pPrime) == 0 {
		zeros := clz64((eb64(x, 64-pPrime, 0)<<pPrime)|(1<<pPrime-1)) + 1
		return idx<<7 | uint32(zeros<<1) | 1
	}
	return idx << 1
}

// Get the index of precision p from the sparse representation.
func (h *HyperLogLogPlus) getIndex(k uint32) uint32 {
	if k&1 == 1 {
		return eb32(k, 32, 32-h.p)
	}
	return eb32(k, pPrime+1, pPrime-h.p+1)
}

// Decode a hash from the sparse representation.
func (h *HyperLogLogPlus) decodeHash(k uint32) (uint32, uint8) {
	var r uint8
	if k&1 == 1 {
		r = uint8(eb32(k, 7, 1)) + pPrime - h.p
	} else {
		r = clz32(k<<(32-pPrime+h.p-1)) + 1
	}
	return h.getIndex(k), r
}

// Merge tmpSet and sparseList in the sparse representation.
func (h *HyperLogLogPlus) mergeSparse() {
	if len(h.tmpSet) == 0 {
		return
	}

	keys := make(sortableSlice, 0, len(h.tmpSet))
	for k := range h.tmpSet {
		keys = append(keys, k)
	}
	sort.Sort(keys)

	newList := newCompressedList(int(h.m))
	for iter, i := h.sparseList.Iter(), 0; iter.HasNext() || i < len(keys); {
		if !iter.HasNext() {
			newList.Append(keys[i])
			i++
			continue
		}

		if i >= len(keys) {
			newList.Append(iter.Next())
			continue
		}

		x1, x2 := iter.Peek(), keys[i]
		if x1 == x2 {
			newList.Append(iter.Next())
			i++
		} else if x1 > x2 {
			newList.Append(x2)
			i++
		} else {
			newList.Append(iter.Next())
		}
	}

	h.sparseList = newList
	h.tmpSet = set{}
}

// NewPlus returns a new initialized HyperLogLogPlus that uses the HyperLogLog++
// algorithm.
func NewPlus(precision uint8) (*HyperLogLogPlus, error) {
	h, err := createPlus(precision)
	if err != nil {
		return nil, err
	}
	h.Clear()
	return h, nil
}

// createPlus returns a new, empty HyperLogLogPlus w/o inititalizing internal
// buffers, for use as a deserialization target.
func createPlus(precision uint8) (*HyperLogLogPlus, error) {
	if precision > 18 || precision < 4 {
		return nil, errors.New("precision must be between 4 and 18")
	}

	h := &HyperLogLogPlus{}
	h.p = precision
	h.m = 1 << precision
	return h, nil
}

// Clear sets HyperLogLogPlus h back to its initial state.
func (h *HyperLogLogPlus) Clear() {
	h.sparse = true
	h.tmpSet = set{}
	h.sparseList = newCompressedList(int(h.m))
	h.reg = nil
}

// Converts HyperLogLogPlus h to the normal representation from the sparse
// representation.
func (h *HyperLogLogPlus) toNormal() {
	h.mergeSparse()

	h.reg = make([]uint8, h.m)
	for iter := h.sparseList.Iter(); iter.HasNext(); {
		i, r := h.decodeHash(iter.Next())
		if h.reg[i] < r {
			h.reg[i] = r
		}
	}

	h.sparse = false
	h.tmpSet = nil
	h.sparseList = nil
}

// Add adds a new item to HyperLogLogPlus h.
func (h *HyperLogLogPlus) Add(item Hash64) {
	x := item.Sum64()
	if h.sparse {
		h.tmpSet.Add(h.encodeHash(x))
		h.maybeToNormal()
	} else {
		i := eb64(x, 64, 64-h.p) // {x63,...,x64-p}
		w := x<<h.p | 1<<(h.p-1) // {x63-p,...,x0}

		zeroBits := clz64(w) + 1
		if zeroBits > h.reg[i] {
			h.reg[i] = zeroBits
		}
	}
}

// Merge takes another HyperLogLogPlus and combines it with HyperLogLogPlus h.
func (h *HyperLogLogPlus) Merge(other *HyperLogLogPlus) error {
	if h.p != other.p {
		return errors.New("precisions must be equal")
	}

	if h.sparse && other.sparse {
		for k := range other.tmpSet {
			h.tmpSet.Add(k)
		}
		for iter := other.sparseList.Iter(); iter.HasNext(); {
			h.tmpSet.Add(iter.Next())
		}
		h.maybeToNormal()
		return nil
	}

	if h.sparse {
		h.toNormal()
	}

	if other.sparse {
		for k := range other.tmpSet {
			i, r := other.decodeHash(k)
			if r > h.reg[i] {
				h.reg[i] = r
			}
		}

		for iter := other.sparseList.Iter(); iter.HasNext(); {
			i, r := other.decodeHash(iter.Next())
			if r > h.reg[i] {
				h.reg[i] = r
			}
		}
	} else {
		for i, v := range other.reg {
			if v > h.reg[i] {
				h.reg[i] = v
			}
		}
	}
	return nil
}

// Converts to normal if the sparse list is too large.
func (h *HyperLogLogPlus) maybeToNormal() {
	if uint32(len(h.tmpSet))*100 > h.m {
		h.mergeSparse()
		if uint32(h.sparseList.Len()) > h.m {
			h.toNormal()
		}
	}
}

// Estimates the bias using empirically determined values.
func (h *HyperLogLogPlus) estimateBias(est float64) float64 {
	estTable, biasTable := rawEstimateData[h.p-4], biasData[h.p-4]

	if estTable[0] > est {
		return biasTable[0]
	}

	lastEstimate := estTable[len(estTable)-1]
	if lastEstimate < est {
		return biasTable[len(biasTable)-1]
	}

	var i int
	for i = 0; i < len(estTable) && estTable[i] < est; i++ {
	}

	e1, b1 := estTable[i-1], biasTable[i-1]
	e2, b2 := estTable[i], biasTable[i]

	c := (est - e1) / (e2 - e1)
	return b1*(1-c) + b2*c
}

// Count returns the cardinality estimate.
func (h *HyperLogLogPlus) Count() uint64 {
	if h.sparse {
		h.mergeSparse()
		return uint64(linearCounting(mPrime, mPrime-uint32(h.sparseList.Count)))
	}

	est := calculateEstimate(h.reg)
	if est <= float64(h.m)*5.0 {
		est -= h.estimateBias(est)
	}

	if v := countZeros(h.reg); v != 0 {
		lc := linearCounting(h.m, v)
		if lc <= float64(threshold[h.p-4]) {
			return uint64(lc)
		}
	}
	return uint64(est)
}

// Encode HyperLogLogPlus into a gob
func (h *HyperLogLogPlus) GobEncode() ([]byte, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(h.reg); err != nil {
		return nil, err
	}
	if err := enc.Encode(h.m); err != nil {
		return nil, err
	}
	if err := enc.Encode(h.p); err != nil {
		return nil, err
	}
	if err := enc.Encode(h.sparse); err != nil {
		return nil, err
	}
	if h.sparse {
		if err := enc.Encode(h.tmpSet); err != nil {
			return nil, err
		}
		if err := enc.Encode(h.sparseList.Count); err != nil {
			return nil, err
		}
		if err := enc.Encode(h.sparseList.b); err != nil {
			return nil, err
		}
		if err := enc.Encode(h.sparseList.last); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// Decode gob into a HyperLogLogPlus structure
func (h *HyperLogLogPlus) GobDecode(b []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&h.reg); err != nil {
		return err
	}
	if err := dec.Decode(&h.m); err != nil {
		return err
	}
	if err := dec.Decode(&h.p); err != nil {
		return err
	}
	if err := dec.Decode(&h.sparse); err != nil {
		return err
	}
	if h.sparse {
		if err := dec.Decode(&h.tmpSet); err != nil {
			return err
		}
		h.sparseList = newCompressedList(int(h.m))
		if err := dec.Decode(&h.sparseList.Count); err != nil {
			return err
		}
		if err := dec.Decode(&h.sparseList.b); err != nil {
			return err
		}
		if err := dec.Decode(&h.sparseList.last); err != nil {
			return err
		}
	}
	return nil
}

// Use if you want to do your own serialization of HyperLogLogPlus data.
// Note this is intended for performance-sensitive cases and depends on
// HyperLogLogPlus's internal data structure. It may change in the future.
type PlusEncodable interface {
	SetP(uint8)
	SetB([]uint8)
	SetSparse(bool)
	SetCount(uint32)
	SetLast(uint32)
}

// Encode stores internal values into serializable dest.
// For performance reasons, pointer values are NOT copied.
// This means you data will be corrupted if you change this hll before copying
// out the resulting data.
func (h *HyperLogLogPlus) Encode(dest PlusEncodable) {
	h.mergeSparse()

	dest.SetP(h.p)
	dest.SetSparse(h.sparse)
	if h.sparse {
		dest.SetCount(h.sparseList.Count)
		dest.SetB(h.sparseList.b)
		dest.SetLast(h.sparseList.last)
	} else {
		dest.SetB(h.reg)
	}
}

type PlusDecodable interface {
	GetP() uint8
	GetB() []uint8
	GetSparse() bool
	GetCount() uint32
	GetLast() uint32
}

// DecodePlus returns a new HyperLogLogPlus with values from src.
// For performance reasons, pointer values are NOT copied.
// This means that the maps and slices in src must not be re-used.
func DecodePlus(src PlusDecodable) (*HyperLogLogPlus, error) {
	h, err := createPlus(src.GetP())
	if err != nil {
		return nil, err
	}
	h.sparse = src.GetSparse()

	if h.sparse {
		h.tmpSet = set{}

		h.sparseList = &compressedList{
			Count: src.GetCount(),
			b:     src.GetB(),
			last:  src.GetLast(),
		}
		if h.sparseList.b == nil {
			h.sparseList.b = make(variableLengthList, 0, h.m)
		}
	} else {
		h.reg = src.GetB()
	}
	return h, nil
}
