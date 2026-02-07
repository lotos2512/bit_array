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
	// Sparse storage: logical content via GetData() equals one bucket with bits 1 and 2 set (0x6).
	assert.Equal(t, bitMap.GetData(), []int64{0x6})
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
	bitMap.size = uint64(len(bitMap.data))
	assert.Equal(t, bitMap.GetBit(1), true)
	assert.Equal(t, bitMap.GetBit(2), true)
}

func TestBitArray_SetData2(t *testing.T) {
	bitMap := NewBitArray(64)
	bitMap.SetData([]int64{0b110})
	assert.Equal(t, bitMap.GetBit(1), true)
	assert.Equal(t, bitMap.GetBit(2), true)
}

// NewBitArrayFromData: старые данные []int64 преобразуются в новую схему (только непустые bucket'ы).
func TestBitArray_NewBitArrayFromData(t *testing.T) {
	// data[i] = bucket i (биты i*64 .. i*64+63). 0x6 = биты 1 и 2 в bucket 0.
	oldData := []int64{0x6, 0, 0, 0}
	ba := NewBitArrayFromData(oldData)
	assert.Equal(t, uint64(256), ba.Capacity())
	assert.True(t, ba.GetBit(1))
	assert.True(t, ba.GetBit(2))
	assert.False(t, ba.GetBit(0))
	assert.False(t, ba.GetBit(64))
	assert.False(t, ba.GetBit(65))
	buckets := ba.SparseBuckets()
	assert.Len(t, buckets, 1)
	assert.Equal(t, int64(0x6), buckets[0])
	assert.Equal(t, oldData, ba.GetData())
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

func TestBitArrayWithOwerFlow(t *testing.T) {
	bitMap1 := NewBitArray(82_074_042)
	err := bitMap1.SetBit(82_074_100)
	assert.Equal(t, ErrorBadIndex, err)

	err = bitMap1.SetBit(0)
	assert.Nil(t, err)

	err = bitMap1.SetBit(2)
	assert.Nil(t, err)

	err = bitMap1.SetBit(3)
	assert.Nil(t, err)

	assert.Equal(t, false, bitMap1.GetBit(82_074_100))
	assert.Equal(t, true, bitMap1.GetBit(0))
	assert.Equal(t, true, bitMap1.GetBit(2))
	assert.Equal(t, true, bitMap1.GetBit(3))
}

func TestBitArrayMinPos(t *testing.T) {
	bitMap1 := NewBitArray(1)

	err := bitMap1.SetBit(0)
	assert.Nil(t, err)

	err = bitMap1.SetBit(1)
	assert.Nil(t, err)

	assert.Equal(t, true, bitMap1.GetBit(0))
	assert.Equal(t, true, bitMap1.GetBit(1))
	assert.Equal(t, false, bitMap1.GetBit(2))
}

// TestBitArray_SparseStorage: при большой ёмкости и малом числе установленных битов
// хранятся только непустые bucket'ы (например, 3 вместо 125000 для 8M бит).
func TestBitArray_SparseStorage(t *testing.T) {
	const capBits = 8_000_000
	ba := NewBitArray(capBits)
	ba.SetBitMust(1)
	ba.SetBitMust(7999)
	ba.SetBitMust(7_000_000)

	assert.Equal(t, uint64(capBits), ba.Capacity())

	assert.True(t, ba.GetBit(1))
	assert.True(t, ba.GetBit(7999))
	assert.True(t, ba.GetBit(7_000_000))
	assert.False(t, ba.GetBit(0))
	assert.False(t, ba.GetBit(2))
	assert.False(t, ba.GetBit(8000))
	assert.False(t, ba.GetBit(6_999_999))

	buckets := ba.SparseBuckets()
	assert.Len(t, buckets, 3, "должно быть ровно 3 непустых bucket'а")

	// bucket 0: бит 1 -> word 2
	assert.Equal(t, int64(2), buckets[0])
	// bucket 124: бит 7999 (124*64+63) -> word 1<<63 (как int64 — минимальное значение)
	assert.Equal(t, int64(-9223372036854775808), buckets[124])
	// bucket 109375: бит 7_000_000 (109375*64+0) -> word 1
	assert.Equal(t, int64(1), buckets[109375])
}

// TestBitArray_FillEvery500th_8M: заполняем каждый 500-й бит (0, 500, 1000, ...).
// Установлено 16_000 бит — массив остаётся sparse (нет полного slice с нулями).
func TestBitArray_FillEvery500th_8M(t *testing.T) {
	const capBits = 8_000_000
	const step = 500
	ba := NewBitArray(capBits)
	for i := uint64(0); i < ba.Capacity(); i += step {
		ba.SetBitMust(i)
	}
	setCount := capBits / step
	assert.Equal(t, uint64(16000), uint64(setCount))

	// Установленные: каждый 500-й
	assert.True(t, ba.GetBit(0))
	assert.True(t, ba.GetBit(500))
	assert.True(t, ba.GetBit(7999500)) // последний кратный 500 < 8M
	// Не установленные: между ними
	assert.False(t, ba.GetBit(1))
	assert.False(t, ba.GetBit(499))
	assert.False(t, ba.GetBit(501))
	assert.False(t, ba.GetBit(capBits-1))

	// Проверяем все 16_000 установленных индексов
	for i := uint64(0); i < ba.Capacity(); i += step {
		assert.True(t, ba.GetBit(i), "бит %d должен быть установлен", i)
	}
}

// TestBitArray_FillAllBits: заполняем все биты (0 .. Capacity()-1), проверяем что все установлены.
func TestBitArray_FillAllBits(t *testing.T) {
	const capBits = 8_000_000
	ba := NewBitArray(capBits)
	for i := uint64(0); i < ba.Capacity(); i++ {
		ba.SetBitMust(i)
	}
	// Выборочно: начало, середина, конец
	assert.True(t, ba.GetBit(0))
	assert.True(t, ba.GetBit(1))
	assert.True(t, ba.GetBit(capBits/2))
	assert.True(t, ba.GetBit(capBits-1))
	// Все слова в dense — полные (-1), т.к. 8_000_000 кратно 64
	data := ba.GetData()
	assert.Len(t, data, 125000)
	for i, w := range data {
		assert.Equal(t, int64(-1), w, "bucket %d должен быть заполнен (все биты 1)", i)
	}
}
