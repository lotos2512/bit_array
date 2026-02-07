package bit_array

import (
	"testing"
)

const benchCap = 8_000_000 // 8M bits, 125k buckets when dense

// BenchmarkNewBitArray measures creation only (no allocation for sparse).
func BenchmarkNewBitArray(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewBitArray(benchCap)
	}
}

// BenchmarkSetBitMust_Sparse: few bits set in large array (stays sparse).
func BenchmarkSetBitMust_Sparse(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(benchCap)
	// Pre-set a few bits so we're in sparse mode
	ba.SetBitMust(1)
	ba.SetBitMust(7999)
	ba.SetBitMust(7_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ba.SetBitMust(uint64(i % int(ba.Capacity())))
	}
}

// BenchmarkSetBitMust_FirstBits: set bits 0..N-1 (stays in sparseIndices until 65+).
func BenchmarkSetBitMust_FirstBits(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(benchCap)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ba.SetBitMust(uint64(i % 128))
	}
}

// BenchmarkGetBit_Sparse: lookup on sparse array (few buckets).
func BenchmarkGetBit_Sparse(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(benchCap)
	ba.SetBitMust(1)
	ba.SetBitMust(7999)
	ba.SetBitMust(7_000_000)
	indices := []uint64{0, 1, 2, 7998, 7999, 8000, 6_999_999, 7_000_000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba.GetBit(indices[i%len(indices)])
	}
}

// BenchmarkGetBit_Dense: lookup on dense array (full slice).
func BenchmarkGetBit_Dense(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(64 * 1000) // 64k bits
	for i := uint64(0); i < ba.Capacity(); i += 2 {
		ba.SetBitMust(i) // enough to trigger toDense
	}
	indices := []uint64{0, 1, 100, 1000, 10000, 50000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba.GetBit(indices[i%len(indices)])
	}
}

// BenchmarkSparseBuckets: return only non-zero buckets (no full slice alloc).
func BenchmarkSparseBuckets(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(benchCap)
	ba.SetBitMust(1)
	ba.SetBitMust(7999)
	ba.SetBitMust(7_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba.SparseBuckets()
	}
}

// BenchmarkGetData_Sparse: full slice alloc when sparse (for comparison with SparseBuckets).
func BenchmarkGetData_Sparse(b *testing.B) {
	b.ReportAllocs()
	ba := NewBitArray(benchCap)
	ba.SetBitMust(1)
	ba.SetBitMust(7999)
	ba.SetBitMust(7_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba.GetData()
	}
}

// BenchmarkIntersection_Sparse: both arrays sparse.
func BenchmarkIntersection_Sparse(b *testing.B) {
	b.ReportAllocs()
	ba1 := NewBitArray(benchCap)
	ba1.SetBitMust(1)
	ba1.SetBitMust(7999)
	ba1.SetBitMust(7_000_000)
	ba2 := NewBitArray(benchCap)
	ba2.SetBitMust(1)
	ba2.SetBitMust(7999)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba1.Intersection(ba2)
	}
}

// BenchmarkIntersection_Dense: both arrays dense.
func BenchmarkIntersection_Dense(b *testing.B) {
	b.ReportAllocs()
	n := uint64(64 * 1000)
	ba1 := NewBitArray(n)
	ba2 := NewBitArray(n)
	for i := uint64(0); i < n; i += 2 {
		ba1.SetBitMust(i)
		ba2.SetBitMust(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba1.Intersection(ba2)
	}
}

// BenchmarkMemory_Step500: 8M бит, каждый 500-й установлен (16k bucket'ов).
// Новая схема: только sparseList ~256 KB. Старая (dense): 1 MB нулей.
func BenchmarkMemory_Step500(b *testing.B) {
	b.ReportAllocs()
	const capBits = 8_000_000
	const step = 500
	ba := NewBitArray(capBits)
	for i := uint64(0); i < ba.Capacity(); i += step {
		ba.SetBitMust(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ba.SparseBuckets() // не аллоцирует полный slice
	}
}

// BenchmarkMemory_FromData: загрузка старых данных в новую схему (только непустые bucket'ы).
func BenchmarkMemory_FromData(b *testing.B) {
	b.ReportAllocs()
	// Имитация старых данных: 125k слов, непустых 16k (каждый ~8-й)
	oldData := make([]int64, 125000)
	for i := 0; i < len(oldData); i += 8 {
		oldData[i] = 1
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ba := NewBitArrayFromData(oldData)
		_ = ba.Capacity()
	}
}
