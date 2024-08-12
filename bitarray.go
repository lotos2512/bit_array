package bit_array

import (
	"fmt"
	"strconv"
)

type BitArray struct {
	Data []uint8
}

func NewBitArray(w uint64) BitArray {
	size, opt := bitArrayBitBucketPosHelper(w)
	if opt != 0 {
		size++
	}
	return BitArray{Data: make([]uint8, size)}
}

func bitArrayBitBucketPosHelper(i uint64) (uint64, uint64) {
	return i >> 3, i & 7 // i / 8, i % 8
}

func (t *BitArray) PrintMask() string {
	bitsString := ""
	for i := len(t.Data) - 1; i >= 0; i-- {
		bitString := strconv.FormatUint(uint64(t.Data[i]), 2)
		for j := len(bitString); j < 8; j++ {
			bitString = "0" + bitString
		}
		bitsString += bitString
	}
	return bitsString
}

func (t *BitArray) GetData() []uint8 {
	return t.Data
}

func (t *BitArray) SetData(data []uint8) {
	t.Data = data
}

// max from NewBitArray(w uint64)
func (t *BitArray) SetBitMust(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	t.Data[bitBucket] |= 1 << bitPos
}

var ErrorBadIndex = fmt.Errorf("index more then capacity")

func (t *BitArray) SetBit(i uint64) error {
	if i >= t.Capacity() {
		return ErrorBadIndex
	}
	t.SetBitMust(i)
	return nil
}

func (t *BitArray) unSetBit(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	t.Data[bitBucket] &= 0 << bitPos
}

func (t *BitArray) inverseBit(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	t.Data[bitBucket] ^= 1 << bitPos
}

func (t *BitArray) Capacity() (size uint64) {
	return uint64(len(t.Data)) * 8
}

func (t *BitArray) GetBit(i uint64) bool {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	if (t.Data[bitBucket] & (1 << bitPos)) == 0 {
		return false
	}
	return true
}

func (t *BitArray) Intersection(bitArray BitArray) bool {
	if len(t.Data) != len(bitArray.Data) {
		return false
	}
	for i := 0; i < len(t.Data); i++ {
		if t.Data[i]&bitArray.Data[i] != bitArray.Data[i] {
			return false
		}
	}
	return true
}
