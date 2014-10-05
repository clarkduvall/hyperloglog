package main

import "testing"

type test testing.T

func assert(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("%s != %s", a, b)
	}
}

func TestMerge(t *testing.T) {
	hll := NewHyperLogLogPP(byte(4))
	hll.tmp_set.Add(5)
	hll.merge()
	assert(t, len(hll.sparse_list), 1)
	assert(t, hll.sparse_list[0], uint32(5))

	hll.tmp_set.Add(6)
	hll.merge()
	assert(t, len(hll.sparse_list), 2)
	assert(t, hll.sparse_list[0], uint32(5))
	assert(t, hll.sparse_list[1], uint32(6))

	hll.tmp_set.Add(4)
	hll.merge()
	assert(t, len(hll.sparse_list), 3)
	assert(t, hll.sparse_list[0], uint32(4))
	assert(t, hll.sparse_list[1], uint32(5))
	assert(t, hll.sparse_list[2], uint32(6))
}
