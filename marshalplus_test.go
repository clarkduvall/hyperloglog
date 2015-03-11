// Copyright (c) 2015, RetailNext, Inc.
// This material contains trade secrets and confidential information of
// RetailNext, Inc.  Any use, reproduction, disclosure or dissemination
// is strictly prohibited without the explicit written permission
// of RetailNext, Inc.
// All rights reserved.
package hyperloglog

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

// use a real hash so we actually fill up all the registers (rand.Int63()
// will leave the second half of your registers empty since it will never
// have the MSB set)
func realHash64(n int64) fakeHash64 {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(n))
	checksum := sha1.Sum(buf)
	return fakeHash64(binary.BigEndian.Uint64(checksum[:]))
}

func mustMarshal(h *HyperLogLogPlus) []byte {
	marshaled, err := h.Marshal()
	if err != nil {
		panic(err)
	}
	return marshaled
}

func hllpEqual(h1, h2 HyperLogLogPlus) bool {
	h1Sparse := h1.sparseList
	h1.sparseList = nil

	h2Sparse := h2.sparseList
	h2.sparseList = nil

	if !reflect.DeepEqual(h1, h2) {
		return false
	}

	if h1Sparse == nil && h2Sparse == nil {
		return true
	}

	if h1Sparse == nil || h2Sparse == nil {
		return false
	}

	return reflect.DeepEqual(*h1Sparse, *h2Sparse)
}

func marshalUnmarshal(h *HyperLogLogPlus) error {
	unmarshaled, err := UnmarshalPlus(mustMarshal(h))
	if err != nil {
		panic(err)
	}

	if !hllpEqual(*h, *unmarshaled) {
		return fmt.Errorf("Got %+v, expected %+v", unmarshaled, h)
	} else {
		return nil
	}
}

// Some white-box testing that we have the same hll after marshal/unmarshal

func TestMarshalSparse(t *testing.T) {
	h, _ := NewPlus(14)

	if err := marshalUnmarshal(h); err != nil {
		t.Error(err)
	}

	h.Add(realHash64(1))

	if err := marshalUnmarshal(h); err != nil {
		t.Error(err)
	}

	for i := 0; i < 100; i++ {
		h.Add(realHash64(rand.Int63()))
	}

	if !h.sparse {
		t.Error("expecting sparse!")
	}

	if err := marshalUnmarshal(h); err != nil {
		t.Error(err)
	}
}

func leadingZeroes(h *HyperLogLogPlus, x uint64) uint8 {
	w := x<<h.p | 1<<(h.p-1)
	return clz64(w) + 1
}

func TestMarshalDense(t *testing.T) {
	h, _ := NewPlus(14)

	// first make sure we don't have anything more than 15 leading zeroes so we
	// can test 4 bits per register
	for i := 0; i < 10000; i++ {
		x := realHash64(rand.Int63())
		if leadingZeroes(h, x.Sum64()) < 16 {
			h.Add(x)
		}
	}

	if h.sparse {
		t.Error("expecting dense")
	}

	if len(mustMarshal(h)) >= 5*len(h.reg)/8 {
		t.Error("Expected to compress below 5 bits to the byte")
	}

	if err := marshalUnmarshal(h); err != nil {
		t.Error(err)
	}

	// now add up to 31 leading zeroes so we can test 5 bit compression
	for i := 0; i < 100000; i++ {
		x := realHash64(rand.Int63())
		if leadingZeroes(h, x.Sum64()) < 32 {
			h.Add(x)
		}
	}

	// make sure we have at least one thing > 15 zeroes
	for {
		x := realHash64(rand.Int63())
		if leadingZeroes(h, x.Sum64()) < 16 {
			continue
		}
		h.Add(x)
		break
	}

	if len(mustMarshal(h)) >= 6*len(h.reg)/8 {
		t.Error("Expected to compress below 6 bits to the byte")
	}

	if err := marshalUnmarshal(h); err != nil {
		t.Error(err)
	}

	// force hash value with lots of leading zeroes, so can't compress registers
	// below 6 bits
	h.Add(fakeHash64(0))

	if err := marshalUnmarshal(h); err != nil {
		t.Error(err)
	}

	// fill up registers
	for i := 0; i < 1000000; i++ {
		h.Add(realHash64(rand.Int63()))
	}

	if err := marshalUnmarshal(h); err != nil {
		t.Error(err)
	}
}

// make sure we validate bytes before trying to unmarshal
func TestUnmarshalErrors(t *testing.T) {
	uh, err := UnmarshalPlus(nil)
	if uh != nil || err == nil {
		t.Error("Expected nil hll and some error")
	}

	uh, err = UnmarshalPlus([]byte{})
	if uh != nil || err == nil {
		t.Error("Expected nil hll and some error")
	}

	h, _ := NewPlus(14)
	for i := 0; i < 10000; i++ {
		h.Add(realHash64(rand.Int63()))
	}
	uh, err = UnmarshalPlus(mustMarshal(h)[0:100])
	if uh != nil || err == nil {
		t.Error("Expected nil hll and some error")
	}
}
