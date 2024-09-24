package bit_array

import (
	"fmt"
	"strconv"
)

type BitArray struct {
	data []int64
	size uint64
	cap  uint64
}

func NewBitArray(w uint64) BitArray {
	size, opt := bitArrayBitBucketPosHelper(w)
	if opt != 0 {
		size++
	}
	return BitArray{
		data: make([]int64, size),
		size: size,
		cap:  uint64(size) * 64,
	}
}

func bitArrayBitBucketPosHelper(i uint64) (uint64, uint64) {
	return i >> 6, i & 63 // i / 64, i % 63
}

func (t *BitArray) PrintMask() string {
	bitsString := ""
	for i := len(t.data) - 1; i >= 0; i-- {
		bitString := strconv.FormatUint(uint64(t.data[i]), 2)
		for j := len(bitString); j < 64; j++ {
			bitString = "0" + bitString
		}
		bitsString += bitString
	}
	return bitsString
}

func (t *BitArray) GetData() []int64 {
	return t.data
}

func (t *BitArray) SetData(data []int64) {
	t.data = data
}

// max from NewBitArray(w uint64)
func (t *BitArray) SetBitMust(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	t.data[bitBucket] |= 1 << bitPos
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
	t.data[bitBucket] &= 0 << bitPos
}

func (t *BitArray) inverseBit(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	t.data[bitBucket] ^= 1 << bitPos
}

func (t *BitArray) Capacity() (size uint64) {
	return t.cap
}

func (t *BitArray) GetSize() (size uint64) {
	return t.size
}

func (t *BitArray) GetBit(i uint64) bool {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	if t.size < bitBucket {
		return false
	}
	if (t.data[bitBucket] & (1 << bitPos)) == 0 {
		return false
	}
	return true
}

func (t *BitArray) Intersection(bitArray BitArray) bool {
	if t.size != uint64(len(bitArray.data)) {
		return false
	}
	for i := uint64(0); i < t.size; i++ {
		if t.data[i]&bitArray.data[i] != bitArray.data[i] {
			return false
		}
	}
	return true
}
