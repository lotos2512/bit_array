package bit_array

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitArray_printMask1(t *testing.T) {
	bitMap := NewBitArray(127)
	bitMap.SetBitMust(1)
	bitMap.SetBitMust(2)
	bitMap.SetBitMust(5)
	bitMap.SetBitMust(62)
	bitMap.SetBitMust(63)
	bitMap.SetBitMust(65)
	bitMap.SetBitMust(65)
	bitMap.SetBitMust(99)
	bitMap.SetBitMust(100)
	bitMap.SetBitMust(101)
	bitMap.SetBitMust(126)
	bitMap.SetBitMust(127)
	assert.Equal(t, "11000000000000000000000000111000000000000000000000000000000000101100000000000000000000000000000000000000000000000000000000100110", bitMap.PrintMask())
}

func TestBitArray_printMask8(t *testing.T) {
	bitMap := NewBitArray(127)
	bitMap.SetBitMust(8)
	bitMap.SetBitMust(9)
	bitMap.SetBitMust(10)

	bitMap.SetBitMust(126)
	bitMap.SetBitMust(127)
	assert.Equal(t, "11000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000011100000000", bitMap.PrintMask())
}

func TestBitArray_GetData(t *testing.T) {
	bitMap := NewBitArray(8)
	bitMap.SetBitMust(1)
	bitMap.SetBitMust(2)
	assert.Equal(t, bitMap.data, []int64{0x6})
}

func TestBitArray_GetData63(t *testing.T) {
	bitMap := NewBitArray(63)
	bitMap.SetBitMust(1)
	bitMap.SetBitMust(2)
	bitMap.SetBitMust(63)
	assert.Equal(t, bitMap.GetData(), []int64{-9223372036854775802})
}

func TestBitArray_SetData(t *testing.T) {
	bitMap := BitArray{data: []int64{0b110}}
	assert.Equal(t, bitMap.GetBit(1), true)
	assert.Equal(t, bitMap.GetBit(2), true)
}

func TestBitArray_SetData2(t *testing.T) {
	bitMap := NewBitArray(64)
	bitMap.SetData([]int64{0b110})
	assert.Equal(t, bitMap.GetBit(1), true)
	assert.Equal(t, bitMap.GetBit(2), true)
}

func TestBitArray_SetBit(t *testing.T) {
	bitMap := NewBitArray(64)
	e := bitMap.SetBit(1)
	assert.Equal(t, e, nil)
	e = bitMap.SetBit(2)
	assert.Equal(t, e, nil)

	e = bitMap.SetBit(99)
	assert.Equal(t, e, ErrorBadIndex)
}

func TestBitArray_printMask2(t *testing.T) {
	bitMap := NewBitArray(63)
	bitMap.SetBitMust(1)
	bitMap.SetBitMust(2)
	bitMap.SetBitMust(5)
	bitMap.SetBitMust(62)
	assert.Equal(t, "0100000000000000000000000000000000000000000000000000000000100110", bitMap.PrintMask())
}

func TestBitArray_GetBit(t *testing.T) {
	bitMap := NewBitArray(10)
	bitMap.SetBitMust(1)
	bitMap.SetBitMust(2)
	bitMap.SetBitMust(5)
	bitMap.SetBitMust(10)

	assert.Equal(t, true, bitMap.GetBit(5))
	assert.Equal(t, true, bitMap.GetBit(1))
	assert.Equal(t, true, bitMap.GetBit(10))
	assert.Equal(t, true, bitMap.GetBit(2))
}

func TestBitArray_UnsetBit(t *testing.T) {
	bitMap := NewBitArray(10)
	bitMap.SetBitMust(1)
	assert.Equal(t, true, bitMap.GetBit(1))
	bitMap.unSetBit(1)
	assert.Equal(t, false, bitMap.GetBit(1))
}

func TestBitArray_inverseBit(t *testing.T) {
	bitMap := NewBitArray(10)
	bitMap.inverseBit(1)
	assert.Equal(t, true, bitMap.GetBit(1))
	bitMap.inverseBit(1)
	assert.Equal(t, false, bitMap.GetBit(1))
	bitMap.inverseBit(1)
	assert.Equal(t, true, bitMap.GetBit(1))
}

func TestBitArray_Capacity1(t *testing.T) {
	bitMap := NewBitArray(10)
	assert.Equal(t, uint64(64), bitMap.Capacity())
}

func TestBitArray_Capacity2(t *testing.T) {
	bitMap := NewBitArray(120)
	assert.Equal(t, uint64(128), bitMap.Capacity())
}

func TestBitArray_Capacity3(t *testing.T) {
	bitMap := NewBitArray(160)
	assert.Equal(t, uint64(192), bitMap.Capacity())
}

func TestBitArray_Intersection(t *testing.T) {
	bitMap1 := NewBitArray(160)
	bitMap1.SetBitMust(1)
	bitMap1.SetBitMust(2)
	bitMap1.SetBitMust(3)

	bitMap2 := NewBitArray(160)
	bitMap2.SetBitMust(1)
	bitMap2.SetBitMust(2)
	bitMap2.SetBitMust(3)

	assert.Equal(t, true, bitMap1.Intersection(bitMap2))
}
