package main

import "fmt"

type iterable interface {
	Decode(i int, last uint32) (uint32, int)
	Len() int
	Iter() *iterator
}

type iterator struct {
	i int
	last uint32
	v iterable
}

func (iter *iterator) Next() uint32 {
	n, i := iter.v.Decode(iter.i, iter.last)
	iter.last = n
	iter.i = i
	return n
}

func (iter iterator) HasNext() bool {
	return iter.i < iter.v.Len()
}

type varLenDiff struct {
	b varLen
	last uint32
}

func (v *varLenDiff) Len() int {
	return len(v.b)
}

func (v *varLenDiff) Decode(i int, last uint32) (uint32, int) {
	n, i := v.b.Decode(i, last)
	return n + last, i
}

func (v *varLenDiff) Append(x uint32) {
	fmt.Println(x - v.last, x, v.last)
	v.b = v.b.Append(x - v.last)
	v.last = x
}

func (v *varLenDiff) Iter() *iterator {
	return &iterator{0, 0, v}
}

func NewVarLenDiff(size int) *varLenDiff {
	v := new(varLenDiff)
	v.b = make(varLen, 0, size)
	return v
}

type varLen []uint8

func (v varLen) Len() int {
	return len(v)
}

func (v *varLen) Iter() *iterator {
	return &iterator{0, 0, v}
}

func (v varLen) Decode(i int, last uint32) (uint32, int) {
	j := i
	for ; v[j] & 0x80 != 0 && j < len(v); j++ {}

	var n uint32
	for k := j; k >= i; k-- {
		n |= uint32(v[k] & 0x7f) << uint8(7 * (j - k))
	}
	return n, j + 1
}

func (v varLen) Append(x uint32) varLen {
	inserting := false
	for i := uint8(5); i > 0; i-- {
		b := eb32(x, i * 7, (i - 1) * 7)
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
