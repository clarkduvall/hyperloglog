package main

import (
	"fmt"
	"testing"
	"bytes"
)

func TestAppend(t *testing.T) {
	l := make(varLen, 0, 100)

	l = l.Append(106903)

	l2 := []uint8{134, 195, 23}
	if bytes.Compare(l, l2) != 0 {
		t.Error(l)
	}

	l = l.Append(0x7f)
	l2 = append(l2, 0x7f)
	if bytes.Compare(l, l2) != 0 {
		t.Error(l)
	}

	l = l.Append(0xff)
	l2 = append(l2, 0x81, 0x7f)
	if bytes.Compare(l, l2) != 0 {
		t.Error(l)
	}

	l = l.Append(0xffffffff)
	l2 = append(l2, 0x8f, 0xff, 0xff, 0xff, 0x7f)
	if bytes.Compare(l, l2) != 0 {
		t.Error(l)
	}

	iter := l.Iter()

	n := iter.Next()
	if n != 106903 {
		t.Error(n)
	}

	n = iter.Next()
	if n != 0x7f {
		t.Error(n)
	}

	n = iter.Next()
	if n != 0xff {
		t.Error(n)
	}

	n = iter.Next()
	if n != 0xffffffff {
		t.Error(n)
	}
}

func TestVarLenDiff(t *testing.T) {
	l := NewVarLenDiff(100)

	l.Append(0xff)

	iter := l.Iter()
	n := iter.Next()
	if n != 0xff {
		t.Error(n)
	}

	l.Append(0xffffffff)
	n = iter.Next()
	if n != 0xffffffff {
		t.Error(n)
	}

	l.Append(0xffff)
	n = iter.Next()
	if n != 0xffff {
		t.Error(n)
	}

	l.Append(0xb0af1000)
	n = iter.Next()
	if n != 0xb0af1000 {
		t.Error(n)
	}

	iter = l.Iter()
	n = iter.Next()
	if n != 0xff {
		t.Error(n)
	}
	n = iter.Next()
	if n != 0xffffffff {
		t.Error(n)
	}
	n = iter.Next()
	if n != 0xffff {
		t.Error(n)
	}
	n = iter.Next()
	if n != 0xb0af1000 {
		t.Error(n)
	}
}

func TestLen(t *testing.T) {
	l1 := make(varLen, 0, 100)
	l2 := NewVarLenDiff(100)
	for i := uint32(0xffffff00); i < 0xffffffff; i++ {
		l1 = l1.Append(i)
		l2.Append(i)
	}
	fmt.Println(l1.Len(), l2.Len())
}
