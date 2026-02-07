// Сравнение производительности с bitset и Roaring bitmap.
// Запуск: go test -bench=BenchmarkCompare -benchmem -count=2 ./...

package bit_array

import (
	"testing"

	"github.com/RoaringBitmap/roaring"
	"github.com/bits-and-blooms/bitset"
)

const compareCap = 80_000_000

// --- Создание (ёмкость 8M бит) ---

func BenchmarkCompare_New_Ours(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewBitArray(compareCap)
	}
}

func BenchmarkCompare_New_Bitset(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bitset.New(compareCap)
	}
}

// --- SetBit: 3 бита (разрежённый сценарий) ---

func BenchmarkCompare_SetBit3_Ours(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(compareCap)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ba.SetBitMust(1)
		ba.SetBitMust(7999)
		ba.SetBitMust(7_000_000)
	}
}

func BenchmarkCompare_SetBit3_Bitset(b *testing.B) {
	b.ReportAllocs()
	bs := bitset.New(compareCap)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs.Set(1)
		bs.Set(7999)
		bs.Set(7_000_000)
	}
}

// --- GetBit: разрежённый массив (3 bucket'а у нас, у bitset — плотный slice) ---

func BenchmarkCompare_GetBit_Sparse_Ours(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(compareCap)
	ba.SetBitMust(1)
	ba.SetBitMust(7999)
	ba.SetBitMust(7_000_000)
	indices := []uint64{0, 1, 2, 7998, 7999, 8000, 6_999_999, 7_000_000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba.GetBit(indices[i%len(indices)])
	}
}

func BenchmarkCompare_GetBit_Sparse_Bitset(b *testing.B) {
	b.ReportAllocs()
	bs := bitset.New(compareCap)
	bs.Set(1)
	bs.Set(7999)
	bs.Set(7_000_000)
	indices := []uint{0, 1, 2, 7998, 7999, 8000, 6_999_999, 7_000_000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.Test(indices[i%len(indices)])
	}
}

// --- GetBit: плотный массив (каждый 2-й бит, оба в dense) ---

func BenchmarkCompare_GetBit_Dense_Ours(b *testing.B) {
	b.ReportAllocs()
	n := uint64(64 * 1000)
	ba := NewBitArray(n)
	for i := uint64(0); i < n; i += 2 {
		ba.SetBitMust(i)
	}
	ba.SetBitMust(32000)
	idx := []uint64{0, 1, 100, 1000, 10000, 50000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba.GetBit(idx[i%len(idx)])
	}
}

func BenchmarkCompare_GetBit_Dense_Bitset(b *testing.B) {
	b.ReportAllocs()
	n := uint(64 * 1000)
	bs := bitset.New(n)
	for i := uint(0); i < n; i += 2 {
		bs.Set(i)
	}
	bs.Set(32000)
	idx := []uint{0, 1, 100, 1000, 10000, 50000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.Test(idx[i%len(idx)])
	}
}

// --- Заполнение "каждый 500-й" (16k бит): у нас sparseList, у bitset — плотный slice ---

func BenchmarkCompare_FillEvery500_Ours(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(compareCap)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ba2 := NewBitArray(compareCap)
		for j := uint64(0); j < ba2.Capacity(); j += 500 {
			ba2.SetBitMust(j)
		}
		_ = ba2
	}
	_ = ba
}

func BenchmarkCompare_FillEvery500_Bitset(b *testing.B) {
	b.ReportAllocs()
	bs := bitset.New(compareCap)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs2 := bitset.New(compareCap)
		for j := uint(0); j < compareCap; j += 500 {
			bs2.Set(j)
		}
		_ = bs2
	}
	_ = bs
}

// --- Intersection (оба разрежённые, 3 бита) ---

func BenchmarkCompare_Intersection_Sparse_Ours(b *testing.B) {
	b.ReportAllocs()
	ba1 := NewBitArray(compareCap)
	ba1.SetBitMust(1)
	ba1.SetBitMust(7999)
	ba1.SetBitMust(7_000_000)
	ba2 := NewBitArray(compareCap)
	ba2.SetBitMust(1)
	ba2.SetBitMust(7999)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba1.Intersection(ba2)
	}
}

func BenchmarkCompare_Intersection_Sparse_Bitset(b *testing.B) {
	b.ReportAllocs()
	bs1 := bitset.New(compareCap)
	bs1.Set(1).Set(7999).Set(7_000_000)
	bs2 := bitset.New(compareCap)
	bs2.Set(1).Set(7999)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs1.Intersection(bs2)
	}
}

// --- Roaring bitmap (github.com/RoaringBitmap/roaring): сжатый битмап, uint32 ---

func BenchmarkCompare_New_Roaring(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = roaring.New()
	}
}

func BenchmarkCompare_SetBit3_Roaring(b *testing.B) {
	b.ReportAllocs()
	rb := roaring.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Clear()
		rb.Add(1)
		rb.Add(7999)
		rb.Add(7_000_000)
	}
}

func BenchmarkCompare_GetBit_Sparse_Roaring(b *testing.B) {
	b.ReportAllocs()
	rb := roaring.New()
	rb.Add(1)
	rb.Add(7999)
	rb.Add(7_000_000)
	indices := []uint32{0, 1, 2, 7998, 7999, 8000, 6_999_999, 7_000_000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Contains(indices[i%len(indices)])
	}
}

func BenchmarkCompare_FillEvery500_Roaring(b *testing.B) {
	b.ReportAllocs()
	rb := roaring.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb2 := roaring.New()
		for j := uint32(0); j < compareCap; j += 500 {
			rb2.Add(j)
		}
		_ = rb2
	}
	_ = rb
}

func BenchmarkCompare_Intersection_Sparse_Roaring(b *testing.B) {
	b.ReportAllocs()
	rb1 := roaring.New()
	rb1.Add(1)
	rb1.Add(7999)
	rb1.Add(7_000_000)
	rb2 := roaring.New()
	rb2.Add(1)
	rb2.Add(7999)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = roaring.And(rb1, rb2)
	}
}

// --- Память в разных сценариях (B/op = память на одну структуру "ёмкость 8M, K установленных бит") ---

func BenchmarkCompare_Memory_3bits_Ours(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ba := NewBitArray(compareCap)
		ba.SetBitMust(1)
		ba.SetBitMust(7999)
		ba.SetBitMust(7_000_000)
		_ = ba.Count()
	}
}

func BenchmarkCompare_Memory_3bits_Roaring(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rb := roaring.New()
		rb.Add(1)
		rb.Add(7999)
		rb.Add(7_000_000)
		_ = rb.GetCardinality()
	}
}

func BenchmarkCompare_Memory_100bits_Ours(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ba := NewBitArray(compareCap)
		for j := uint64(0); j < 100; j++ {
			ba.SetBitMust(j * 80000)
		}
		_ = ba.Count()
	}
}

func BenchmarkCompare_Memory_100bits_Roaring(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rb := roaring.New()
		for j := uint32(0); j < 100; j++ {
			rb.Add(j * 80000)
		}
		_ = rb.GetCardinality()
	}
}

func BenchmarkCompare_Memory_1000bits_Ours(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ba := NewBitArray(compareCap)
		for j := uint64(0); j < 1000; j++ {
			ba.SetBitMust(j * 8000)
		}
		_ = ba.Count()
	}
}

func BenchmarkCompare_Memory_1000bits_Roaring(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rb := roaring.New()
		for j := uint32(0); j < 1000; j++ {
			rb.Add(j * 8000)
		}
		_ = rb.GetCardinality()
	}
}

func BenchmarkCompare_Memory_16kbits_Ours(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ba := NewBitArray(compareCap)
		for j := uint64(0); j < ba.Capacity(); j += 500 {
			ba.SetBitMust(j)
		}
		_ = ba.Count()
	}
}

// FillEvery500 using SetBits (bulk) — should be much faster than SetBitMust loop.
func BenchmarkCompare_FillEvery500_SetBits_Ours(b *testing.B) {
	b.ReportAllocs()
	indices := make([]uint64, 0, compareCap/500)
	for j := uint64(0); j < compareCap; j += 500 {
		indices = append(indices, j)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ba := NewBitArray(compareCap)
		ba.SetBits(indices)
		_ = ba.Capacity()
	}
}

func BenchmarkCompare_Memory_16kbits_Roaring(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rb := roaring.BitmapOf()
		for j := uint32(0); j < compareCap; j += 500 {
			rb.Add(j)
		}
		_ = rb.GetCardinality()
	}
}

// Память уже инициализированной структуры: 64k бит, 50% заполнено (каждый 2-й бит).
// Соответствует сценарию GetBit_Dense — после построения кто занимает больше.
func BenchmarkCompare_Memory_Dense50pct_Ours(b *testing.B) {
	b.ReportAllocs()
	n := uint64(64 * 1000)
	for i := 0; i < b.N; i++ {
		ba := NewBitArray(n)
		for j := uint64(0); j < n; j += 2 {
			ba.SetBitMust(j)
		}
		ba.SetBitMust(32000)
		_ = ba.Count()
	}
}

func BenchmarkCompare_Memory_Dense50pct_Bitset(b *testing.B) {
	b.ReportAllocs()
	n := uint(64 * 1000)
	for i := 0; i < b.N; i++ {
		bs := bitset.New(n)
		for j := uint(0); j < n; j += 2 {
			bs.Set(j)
		}
		bs.Set(32000)
		_ = bs.Count()
	}
}
