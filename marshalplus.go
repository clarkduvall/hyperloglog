package hyperloglog

import (
	"encoding/binary"
	"fmt"
)

// Marshal HyperLogLogPlus to and from []byte to make it easy to persist
// to the datastore of your choice.

// There is a version field so backwards compatibility can be maintained in the
// future if the marshal format is changed.

/*

Here is a diagram of the marshal header:

    0               1               2               3
    0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |         Marshal Version       |            Length             |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |             Flags             |       p       |      p'       |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |  Data... (differs for sparse/dense case)
   +-+-+-+-+-+-+-+-+-+-+-+-+-

*/

const (
	marshalVersion    = 1
	marshalHeaderSize = 2 + 2 + 2 + 1 + 1

	marshalFlagSparse = 1
)

// Marshal serializes h into a byte slice that can be deserialized via
// UnmarshalPlus. Marshal is optimized to produce compact serializations
// when possible.
func (h *HyperLogLogPlus) Marshal() []byte {
	bufSize := marshalHeaderSize

	var (
		regSize uint8
		flags   uint16
	)

	if h.sparse {
		h.mergeSparse()
		bufSize += 4 + 4 + len(h.sparseList.b)
		flags |= marshalFlagSparse
	} else {
		// one byte to store regSize
		bufSize++

		var maxReg uint8
		for _, r := range h.reg {
			if r > maxReg {
				maxReg = r
				if maxReg >= 32 {
					break
				}
			}
		}

		regSize = 6

		// use 5 or 4 bits per register if possible
		if maxReg < 16 {
			regSize = 4
		} else if maxReg < 32 {
			regSize = 5
		}

		bufSize += int(regSize) * len(h.reg) / 8
		if (int(regSize)*len(h.reg))%8 > 0 {
			bufSize++
		}
	}

	buf := make([]byte, bufSize)

	offset := 0

	binary.BigEndian.PutUint16(buf[offset:], marshalVersion)
	offset += 2

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(buf)))
	offset += 2

	binary.BigEndian.PutUint16(buf[offset:], flags)
	offset += 2

	buf[offset] = h.p
	offset += 1

	// add pPrime in case it becomes configurable
	buf[offset] = pPrime
	offset += 1

	if h.sparse {
		binary.BigEndian.PutUint32(buf[offset:], h.sparseList.Count)
		offset += 4

		binary.BigEndian.PutUint32(buf[offset:], h.sparseList.last)
		offset += 4

		copy(buf[offset:], h.sparseList.b)
	} else {

		buf[offset] = regSize
		offset++

		var (
			currentByte byte

			// how many bits have we used of current compressed byte
			bitsUsed uint8
		)
		for _, reg := range h.reg {
			if bitsUsed == 8 {
				buf[offset] = currentByte
				offset++
				currentByte, bitsUsed = 0, 0
			}

			if bitsUsed <= (8 - regSize) {
				// can fit register in this byte's remaining bits
				currentByte |= reg << ((8 - regSize) - bitsUsed)
				bitsUsed += regSize
			} else {
				// can't fit entire register, so complete the current byte with the
				// first bits of our register, then put remaining register bits in
				// currentByte
				buf[offset] = currentByte | reg>>(bitsUsed-(8-regSize))
				offset++
				currentByte = reg << (8 - (bitsUsed - (8 - regSize)))
				bitsUsed = regSize - (8 - bitsUsed)
			}
		}

		if bitsUsed > 0 {
			buf[offset] = currentByte
			offset++
		}
	}

	return buf
}

// UnmarshalPlus deserializes the result of Marshal back into a HyperLogLogPlus
// object.
func UnmarshalPlus(data []byte) (*HyperLogLogPlus, error) {
	if len(data) < marshalHeaderSize {
		return nil, fmt.Errorf("data too short (%d bytes)", len(data))
	}

	offset := 0

	version := binary.BigEndian.Uint16(data[offset:])
	offset += 2

	if version != marshalVersion {
		return nil, fmt.Errorf("unknown version: %d", version)
	}

	length := binary.BigEndian.Uint16(data[offset:])
	offset += 2

	if int(length) != len(data) {
		return nil, fmt.Errorf("length mismatch: header says %d, was %d", length, len(data))
	}

	flags := binary.BigEndian.Uint16(data[offset:])
	offset += 2

	p := data[offset]
	offset++

	pp := data[offset]
	offset++

	// for now check that pPrime is the default value
	if pp != pPrime {
		return nil, fmt.Errorf("unexpected p' value: %d", pp)
	}

	h, err := NewPlus(p)
	if err != nil {
		return nil, err
	}

	if flags&marshalFlagSparse > 0 {
		h.sparseList.Count = binary.BigEndian.Uint32(data[offset:])
		offset += 4

		h.sparseList.last = binary.BigEndian.Uint32(data[offset:])
		offset += 4

		h.sparseList.b = h.sparseList.b[:len(data)-offset]
		copy(h.sparseList.b, data[offset:])
	} else {
		regSize := data[offset]
		offset++

		h.sparse = false
		h.tmpSet = nil
		h.sparseList = nil

		h.reg = make([]uint8, h.m)

		var (
			regIdx int

			// current register we are decompressing
			byteSoFar uint8

			// number of bits we need to make next complete register
			numBitsLeft uint8 = regSize
		)

		for _, b := range data[offset:] {
			h.reg[regIdx] = byteSoFar | (b >> (8 - numBitsLeft))
			regIdx++

			byteSoFar = (b << numBitsLeft) >> (8 - regSize)

			if numBitsLeft <= (8 - regSize) {
				// we know byteSoFar holds the complete register
				h.reg[regIdx] = byteSoFar
				regIdx++
				byteSoFar = (b << (numBitsLeft + regSize)) >> (8 - regSize)
				numBitsLeft = regSize - (8 - (numBitsLeft + regSize))
			} else {
				numBitsLeft = regSize - (8 - numBitsLeft)
			}
		}
	}

	return h, nil
}
