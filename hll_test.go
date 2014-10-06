package main

import (
	"testing"
	"fmt"
)

func TestEncodeDecode(t *testing.T) {
	h, _ := NewHyperLogLogPP(8)
	i, r := h.decodeHash(h.encodeHash(0xffffff8000000000))
	if i != 0xff {
		t.Error(i)
	}

	if r != 1 {
		t.Error(r)
	}

	i, r = h.decodeHash(h.encodeHash(0xff00000000000000))
	if i != 0xff {
		t.Error(i)
	}

	if r != 57 {
		t.Error(r)
	}

	i, r = h.decodeHash(h.encodeHash(0xff30000000000000))
	if i != 0xff {
		t.Error(i)
	}

	if r != 3 {
		t.Error(r)
	}

	i, r = h.decodeHash(h.encodeHash(0xaa10000000000000))
	if i != 0xaa {
		t.Error(i)
	}

	if r != 4 {
		t.Error(r)
	}

	i, r = h.decodeHash(h.encodeHash(0xaa0f000000000000))
	if i != 0xaa {
		t.Error(i)
	}

	if r != 5 {
		t.Error(r)
	}
}

func BenchmarkHllpp(b *testing.B) {
	h, _ := NewHyperLogLogPP(8)
	for i := 0; i < b.N; i++ {
		h.Add(hash64(fmt.Sprintf("a", i)))
	}
}

func BenchmarkHll(b *testing.B) {
	h, _ := NewHyperLogLog(8)
	for i := 0; i < b.N; i++ {
		h.Add(hash32(fmt.Sprintf("a", i)))
	}
}
