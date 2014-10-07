package hyperloglog

import "testing"

type fakeHash32 uint32

func (f fakeHash32) Write(p []byte) (n int, err error) { return 0, nil }
func (f fakeHash32) Sum(b []byte) []byte               { return b }
func (f fakeHash32) Reset()                            {}
func (f fakeHash32) BlockSize() int                    { return 1 }
func (f fakeHash32) Size() int                         { return 1 }
func (f fakeHash32) Sum32() uint32                     { return uint32(f) }

func TestHLLAdd(t *testing.T) {
	h, _ := New(16)

	h.Add(fakeHash32(0x00010fff))
	n := h.reg[1]
	if n != 5 {
		t.Error(n)
	}

	h.Add(fakeHash32(0x0002ffff))
	n = h.reg[2]
	if n != 1 {
		t.Error(n)
	}

	h.Add(fakeHash32(0x00030000))
	n = h.reg[3]
	if n != 17 {
		t.Error(n)
	}

	h.Add(fakeHash32(0x00030001))
	n = h.reg[3]
	if n != 17 {
		t.Error(n)
	}

	h.Add(fakeHash32(0xff037000))
	n = h.reg[0xff03]
	if n != 2 {
		t.Error(n)
	}

	h.Add(fakeHash32(0xff030800))
	n = h.reg[0xff03]
	if n != 5 {
		t.Error(n)
	}
}

func TestHLLCardinality(t *testing.T) {
	h, _ := New(16)

	n := h.Count()
	if n != 0 {
		t.Error(n)
	}

	h.Add(fakeHash32(0x00010fff))
	h.Add(fakeHash32(0x00020fff))
	h.Add(fakeHash32(0x00030fff))
	h.Add(fakeHash32(0x00040fff))
	h.Add(fakeHash32(0x00050fff))
	h.Add(fakeHash32(0x00050fff))

	n = h.Count()
	if n != 5 {
		t.Error(n)
	}
}

func TestHLLClear(t *testing.T) {
	h, _ := New(16)
	h.Add(fakeHash32(0x00010fff))

	n := h.Count()
	if n != 1 {
		t.Error(n)
	}
	h.Clear()

	n = h.Count()
	if n != 0 {
		t.Error(n)
	}

	h.Add(fakeHash32(0x00010fff))
	n = h.Count()
	if n != 1 {
		t.Error(n)
	}
}

func TestHLLPrecision(t *testing.T) {
	h, _ := New(4)

	h.Add(fakeHash32(0x1fffffff))
	n := h.reg[1]
	if n != 1 {
		t.Error(n)
	}

	h.Add(fakeHash32(0xffffffff))
	n = h.reg[0xf]
	if n != 1 {
		t.Error(n)
	}

	h.Add(fakeHash32(0x00ffffff))
	n = h.reg[0]
	if n != 5 {
		t.Error(n)
	}
}

func TestHLLError(t *testing.T) {
	_, err := New(3)
	if err == nil {
		t.Error("precision 3 should return error")
	}

	_, err = New(17)
	if err == nil {
		t.Error("precision 17 should return error")
	}
}
