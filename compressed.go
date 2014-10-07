package hyperloglog

type iterable interface {
	decode(i int, last uint32) (uint32, int)
	Len() int
	Iter() *iterator
}

type iterator struct {
	i    int
	last uint32
	v    iterable
}

func (iter *iterator) Next() uint32 {
	n, i := iter.v.decode(iter.i, iter.last)
	iter.last = n
	iter.i = i
	return n
}

func (iter *iterator) Peek() uint32 {
	n, _ := iter.v.decode(iter.i, iter.last)
	return n
}

func (iter iterator) HasNext() bool {
	return iter.i < iter.v.Len()
}

type compressedList struct {
	Count uint32
	b     variableLengthList
	last  uint32
}

func newCompressedList(size int) *compressedList {
	v := &compressedList{}
	v.b = make(variableLengthList, 0, size)
	return v
}

func (v *compressedList) Len() int {
	return len(v.b)
}

func (v *compressedList) decode(i int, last uint32) (uint32, int) {
	n, i := v.b.decode(i, last)
	return n + last, i
}

func (v *compressedList) Append(x uint32) {
	v.Count++
	v.b = v.b.Append(x - v.last)
	v.last = x
}

func (v *compressedList) Iter() *iterator {
	return &iterator{0, 0, v}
}

type variableLengthList []uint8

func (v variableLengthList) Len() int {
	return len(v)
}

func (v *variableLengthList) Iter() *iterator {
	return &iterator{0, 0, v}
}

func (v variableLengthList) decode(i int, last uint32) (uint32, int) {
	j := i
	for ; v[j]&0x80 != 0 && j < len(v); j++ {
	}

	var n uint32
	for k := j; k >= i; k-- {
		n |= uint32(v[k]&0x7f) << uint8(7*(j-k))
	}
	return n, j + 1
}

func (v variableLengthList) Append(x uint32) variableLengthList {
	inserting := false
	for i := uint8(5); i > 0; i-- {
		b := eb32(x, i*7, (i-1)*7)
		if inserting || b != 0 {
			inserting = true
			if i != 1 {
				b |= 0x80
			}
			v = append(v, uint8(b))
		}
	}
	return v
}
