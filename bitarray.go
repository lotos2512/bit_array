// Package bit_array реализует массив битов с разрежённым хранением.
//
// Безопасен для параллельного чтения: несколько горутин могут одновременно
// вызывать GetBit, Capacity, GetSize, GetData, SparseBuckets, Intersection
// (и другие методы, не изменяющие массив) — гонки данных не возникает.
// Если одна горутина пишет (SetBit, unSetBit, inverseBit, SetData), а другая
// читает, синхронизацию (например, sync.RWMutex) должен обеспечить вызывающий.
package bit_array

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"sort"
	"strconv"
)

// sparseThreshold: switch to dense only when non-zero buckets exceed size/sparseThreshold.
// 2 = switch when >50% buckets used (fewer zeros in memory); was 8 (~12.5%), caused 1MB of zeros for step-500 pattern.
const sparseThreshold = 2

// indicesToMapThreshold: when set bit count exceeds this, switch from indices (8 bytes/bit) to sparse list.
const indicesToMapThreshold = 64

// linearScanThreshold: for sparseList length <= this, linear scan is faster than binary search (cache-friendly).
const linearScanThreshold = 32

// sparseEntry: один непустой bucket — 12 байт (uint32 bucket + int64 word).
// Bucket index fits in uint32 for capacity up to 2^38 bits.
type sparseEntry struct {
	b uint32 // bucket index
	w int64  // word (64 bits)
}

type BitArray struct {
	data            []int64       // nil when using sparse
	sparseList      []sparseEntry // sorted by b; used when data nil and sparseIndices/sparseIndices32 nil
	sparseIndices   []uint64      // sorted set bit indices; used when ≤64 bits (data nil, sparseList nil)
	sparseIndices32 []uint32      // sorted indices when >64 bits and one bit per bucket, cap≤2^32 (4 bytes/index)
	size            uint64
	cap             uint64
}

// NewBitArray creates a bit array with capacity for w bits. It does not allocate
// the full bucket array: storage is sparse (only non-zero buckets are stored)
// until the array becomes dense enough. E.g. NewBitArray(8000000) with only
// bits 1, 7999, 7000000 set uses 3 buckets instead of 125000.
func NewBitArray(w uint64) BitArray {
	size, opt := bitArrayBitBucketPosHelper(w)
	if opt != 0 {
		size++
	}
	return BitArray{
		size: size,
		cap:  uint64(size) * 64,
		// data, sparse, sparseIndices left nil/zero — sparse until toDense()
	}
}

// NewBitArrayFromData создаёт BitArray по старому плотному слайсу data []int64.
// Данные преобразуются в новую схему: хранятся только непустые bucket'ы (sparseList),
// полный слайс не сохраняется — вызывающий может отдать его и забыть.
func NewBitArrayFromData(data []int64) BitArray {
	if len(data) == 0 {
		return BitArray{}
	}
	size := uint64(len(data))
	capacity := size * 64
	list := make([]sparseEntry, 0, len(data)/4)
	for i, w := range data {
		if w != 0 {
			list = append(list, sparseEntry{b: uint32(i), w: w})
		}
	}
	return BitArray{
		size:       size,
		cap:        capacity,
		sparseList: list,
	}
}

func (t *BitArray) inIndicesMode() bool {
	return t.data == nil && t.sparseList == nil && (t.sparseIndices != nil || t.sparseIndices32 != nil)
}

func (t *BitArray) sparseListLookup(bucket uint64) (int64, bool) {
	list := t.sparseList
	if len(list) <= linearScanThreshold {
		for _, e := range list {
			if uint64(e.b) == bucket {
				return e.w, true
			}
		}
		return 0, false
	}
	i := sort.Search(len(list), func(j int) bool { return uint64(list[j].b) >= bucket })
	if i < len(list) && uint64(list[i].b) == bucket {
		return list[i].w, true
	}
	return 0, false
}

func (t *BitArray) sparseListSet(bucket uint64, word int64) {
	i := sort.Search(len(t.sparseList), func(j int) bool { return uint64(t.sparseList[j].b) >= bucket })
	if i < len(t.sparseList) && uint64(t.sparseList[i].b) == bucket {
		t.sparseList[i].w = word
		return
	}
	t.sparseList = append(t.sparseList, sparseEntry{})
	copy(t.sparseList[i+1:], t.sparseList[i:])
	t.sparseList[i] = sparseEntry{b: uint32(bucket), w: word}
}

func (t *BitArray) sparseListDelete(bucket uint64) {
	i := sort.Search(len(t.sparseList), func(j int) bool { return uint64(t.sparseList[j].b) >= bucket })
	if i < len(t.sparseList) && uint64(t.sparseList[i].b) == bucket {
		t.sparseList = append(t.sparseList[:i], t.sparseList[i+1:]...)
	}
}

func (t *BitArray) indicesToSparseList() {
	if t.sparseIndices == nil {
		return
	}
	// Собираем пары (bucket, pos), считаем уникальные bucket'ы
	type bp struct{ b uint64; p uint64 }
	pairs := make([]bp, len(t.sparseIndices))
	for i, idx := range t.sparseIndices {
		b, pos := bitArrayBitBucketPosHelper(idx)
		pairs[i] = bp{b, pos}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].b < pairs[j].b })
	uniqueBuckets := 0
	var prevB uint64 = ^uint64(0)
	for _, x := range pairs {
		if x.b != prevB {
			uniqueBuckets++
			prevB = x.b
		}
	}
	// Один бит на bucket и cap помещается в uint32 → компактный режим sparseIndices32 (4 B/индекс)
	if t.cap <= 1<<32 && uniqueBuckets == len(pairs) {
		t.sparseIndices32 = make([]uint32, len(t.sparseIndices))
		for i, idx := range t.sparseIndices {
			t.sparseIndices32[i] = uint32(idx)
		}
		t.sparseIndices = nil
		return
	}
	t.sparseList = make([]sparseEntry, 0, len(pairs))
	var cur uint64
	var word int64
	for _, x := range pairs {
		if x.b != cur && word != 0 {
			t.sparseList = append(t.sparseList, sparseEntry{b: uint32(cur), w: word})
		}
		if x.b != cur {
			cur, word = x.b, 0
		}
		word |= 1 << x.p
	}
	if word != 0 {
		t.sparseList = append(t.sparseList, sparseEntry{b: uint32(cur), w: word})
	}
	t.sparseIndices = nil
}

func (t *BitArray) toDense() {
	if t.data != nil {
		return
	}
	t.data = make([]int64, t.size)
	if t.sparseList != nil {
		for _, e := range t.sparseList {
			t.data[uint64(e.b)] = e.w
		}
		t.sparseList = nil
		t.sparseIndices = nil
		t.sparseIndices32 = nil
	} else if t.sparseIndices != nil {
		for _, i := range t.sparseIndices {
			b, pos := bitArrayBitBucketPosHelper(i)
			t.data[b] |= 1 << pos
		}
		t.sparseIndices = nil
		t.sparseIndices32 = nil
	} else if t.sparseIndices32 != nil {
		for _, idx := range t.sparseIndices32 {
			b, pos := bitArrayBitBucketPosHelper(uint64(idx))
			t.data[b] |= 1 << pos
		}
		t.sparseIndices = nil
		t.sparseIndices32 = nil
	}
}

func bitArrayBitBucketPosHelper(i uint64) (uint64, uint64) {
	return i >> 6, i & 63 // i / 64, i % 63
}

func (t *BitArray) wordAt(bucket uint64) int64 {
	if t.data != nil {
		return t.data[bucket]
	}
	if w, ok := t.sparseListLookup(bucket); ok {
		return w
	}
	if t.sparseList != nil {
		return 0
	}
	if t.sparseIndices32 != nil {
		var w int64
		for _, idx := range t.sparseIndices32 {
			b, pos := bitArrayBitBucketPosHelper(uint64(idx))
			if b == bucket {
				w |= 1 << pos
			}
		}
		return w
	}
	var w int64
	for _, i := range t.sparseIndices {
		b, pos := bitArrayBitBucketPosHelper(i)
		if b == bucket {
			w |= 1 << pos
		}
	}
	return w
}

func (t *BitArray) PrintMask() string {
	bitsString := ""
	for i := int(t.size) - 1; i >= 0; i-- {
		word := t.wordAt(uint64(i))
		bitString := strconv.FormatUint(uint64(word), 2)
		for j := len(bitString); j < 64; j++ {
			bitString = "0" + bitString
		}
		bitsString += bitString
	}
	return bitsString
}

func (t *BitArray) GetData() []int64 {
	if t.data != nil {
		return t.data
	}
	out := make([]int64, t.size)
	if t.sparseList != nil {
		for _, e := range t.sparseList {
			out[uint64(e.b)] = e.w
		}
	} else if t.sparseIndices32 != nil {
		for _, idx := range t.sparseIndices32 {
			b, pos := bitArrayBitBucketPosHelper(uint64(idx))
			out[b] |= 1 << pos
		}
	} else {
		for _, i := range t.sparseIndices {
			b, pos := bitArrayBitBucketPosHelper(i)
			out[b] |= 1 << pos
		}
	}
	return out
}

// SparseBuckets returns only non-zero buckets (bucket index -> word). Avoids
// allocating a full slice of size capacity/64. For a sparse array with only
// a few set bits this returns a small map (e.g. 3 entries for 3 set bits in
// different buckets).
func (t *BitArray) SparseBuckets() map[uint64]int64 {
	if t.data != nil {
		out := make(map[uint64]int64)
		for b := uint64(0); b < t.size; b++ {
			if t.data[b] != 0 {
				out[b] = t.data[b]
			}
		}
		return out
	}
	if t.sparseList != nil {
		out := make(map[uint64]int64, len(t.sparseList))
		for _, e := range t.sparseList {
			out[uint64(e.b)] = e.w
		}
		return out
	}
	if t.sparseIndices32 != nil {
		out := make(map[uint64]int64)
		for _, idx := range t.sparseIndices32 {
			b, pos := bitArrayBitBucketPosHelper(uint64(idx))
			out[b] |= 1 << pos
		}
		return out
	}
	out := make(map[uint64]int64)
	for _, i := range t.sparseIndices {
		b, pos := bitArrayBitBucketPosHelper(i)
		out[b] |= 1 << pos
	}
	return out
}

func (t *BitArray) SetData(data []int64) {
	t.data = data
	t.sparseList = nil
	t.sparseIndices = nil
	t.sparseIndices32 = nil
	if len(data) > 0 {
		t.size = uint64(len(data))
		t.cap = t.size * 64
	}
}

func indexOfSorted(s []uint64, x uint64) (int, bool) {
	i := sort.Search(len(s), func(j int) bool { return s[j] >= x })
	if i < len(s) && s[i] == x {
		return i, true
	}
	return i, false
}

func insertSorted(s []uint64, x uint64) []uint64 {
	i := sort.Search(len(s), func(j int) bool { return s[j] >= x })
	if i < len(s) && s[i] == x {
		return s
	}
	s = append(s, 0)
	copy(s[i+1:], s[i:])
	s[i] = x
	return s
}

func removeFromSorted(s []uint64, x uint64) []uint64 {
	i, ok := indexOfSorted(s, x)
	if !ok {
		return s
	}
	return append(s[:i], s[i+1:]...)
}

func indexOfSortedU32(s []uint32, x uint32) (int, bool) {
	i := sort.Search(len(s), func(j int) bool { return s[j] >= x })
	if i < len(s) && s[i] == x {
		return i, true
	}
	return i, false
}

func insertSortedU32(s []uint32, x uint32) []uint32 {
	i := sort.Search(len(s), func(j int) bool { return s[j] >= x })
	if i < len(s) && s[i] == x {
		return s
	}
	s = append(s, 0)
	copy(s[i+1:], s[i:])
	s[i] = x
	return s
}

func removeFromSortedU32(s []uint32, x uint32) []uint32 {
	i, ok := indexOfSortedU32(s, x)
	if !ok {
		return s
	}
	return append(s[:i], s[i+1:]...)
}

func (t *BitArray) sparseIndices32ToSparseList() {
	if t.sparseIndices32 == nil {
		return
	}
	t.sparseList = make([]sparseEntry, 0, len(t.sparseIndices32))
	var curB uint32 = ^uint32(0)
	var word int64
	for _, idx := range t.sparseIndices32 {
		b, pos := bitArrayBitBucketPosHelper(uint64(idx))
		bu := uint32(b)
		if bu != curB {
			if curB != ^uint32(0) && word != 0 {
				t.sparseList = append(t.sparseList, sparseEntry{b: curB, w: word})
			}
			curB, word = bu, 0
		}
		word |= 1 << pos
	}
	if word != 0 {
		t.sparseList = append(t.sparseList, sparseEntry{b: curB, w: word})
	}
	t.sparseIndices32 = nil
}

// max from NewBitArray(w uint64)
func (t *BitArray) SetBitMust(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	if t.data != nil {
		t.data[bitBucket] |= 1 << bitPos
		return
	}
	// Пустой массив: начинаем с sparseIndices (cap 16), чтобы при 65+ битах перейти в sparseIndices32 или sparseList
	if t.sparseList == nil && t.sparseIndices == nil && t.sparseIndices32 == nil {
		t.sparseIndices = make([]uint64, 0, 16)
		t.sparseIndices = insertSorted(t.sparseIndices, i)
		if len(t.sparseIndices) > indicesToMapThreshold {
			t.indicesToSparseList()
			if t.sparseList != nil && uint64(len(t.sparseList)) > t.size/sparseThreshold {
				t.toDense()
			}
		}
		return
	}
	if t.sparseIndices != nil {
		_, found := indexOfSorted(t.sparseIndices, i)
		if !found {
			t.sparseIndices = insertSorted(t.sparseIndices, i)
			if len(t.sparseIndices) > indicesToMapThreshold {
				t.indicesToSparseList()
				if t.sparseList != nil && uint64(len(t.sparseList)) > t.size/sparseThreshold {
					t.toDense()
				}
			}
		}
		return
	}
	if t.sparseIndices32 != nil {
		idx32 := uint32(i)
		if i >= t.cap {
			return
		}
		if _, found := indexOfSortedU32(t.sparseIndices32, idx32); found {
			return
		}
		// Adding: check if bucket already has another bit (binary search by index range)
		bucketStart := uint32(bitBucket << 6)
		bucketEnd := bucketStart + 63
		lo := sort.Search(len(t.sparseIndices32), func(j int) bool { return t.sparseIndices32[j] >= bucketStart })
		hi := sort.Search(len(t.sparseIndices32), func(j int) bool { return t.sparseIndices32[j] > bucketEnd })
		if lo < hi {
			t.sparseIndices32ToSparseList()
			word, _ := t.sparseListLookup(bitBucket)
			t.sparseListSet(bitBucket, word|(1<<bitPos))
			if uint64(len(t.sparseList)) > t.size/sparseThreshold {
				t.toDense()
			}
			return
		}
		t.sparseIndices32 = insertSortedU32(t.sparseIndices32, idx32)
		return
	}
	word, _ := t.sparseListLookup(bitBucket)
	t.sparseListSet(bitBucket, word|(1<<bitPos))
	if uint64(len(t.sparseList)) > t.size/sparseThreshold {
		t.toDense()
	}
}

var ErrorBadIndex = fmt.Errorf("index more then capacity")

func (t *BitArray) SetBit(i uint64) error {
	if i >= t.Capacity() {
		return ErrorBadIndex
	}
	t.SetBitMust(i)
	return nil
}

// SetBits sets all bits at the given indices in one go (faster than many SetBitMust).
// Indices beyond Capacity() are ignored. Duplicates in indices are fine.
// If the array is empty, builds sparseList or sparseIndices32 in one pass; otherwise falls back to per-bit set.
func (t *BitArray) SetBits(indices []uint64) {
	if len(indices) == 0 {
		return
	}
	cap := t.Capacity()
	buildFresh := t.sparseList == nil && t.sparseIndices == nil && t.sparseIndices32 == nil
	use32 := cap <= 1<<32

	if use32 && buildFresh && len(indices) > indicesToMapThreshold {
		// Single allocation: filter into []uint32, sort, dedupe; then either sparseIndices32 or sparseList
		sorted := make([]uint32, 0, len(indices))
		for _, i := range indices {
			if i < cap {
				sorted = append(sorted, uint32(i))
			}
		}
		if len(sorted) == 0 {
			return
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		wr := 0
		for _, idx := range sorted {
			if wr > 0 && sorted[wr-1] == idx {
				continue
			}
			sorted[wr] = idx
			wr++
		}
		sorted = sorted[:wr]
		if t.data != nil {
			for _, idx := range sorted {
				b, pos := bitArrayBitBucketPosHelper(uint64(idx))
				t.data[b] |= 1 << pos
			}
			return
		}
		var prevB uint32 = ^uint32(0)
		uniqueBuckets := 0
		for _, idx := range sorted {
			b, _ := bitArrayBitBucketPosHelper(uint64(idx))
			bu := uint32(b)
			if bu != prevB {
				uniqueBuckets++
				prevB = bu
			}
		}
		if uniqueBuckets == len(sorted) {
			t.sparseIndices32 = sorted
			return
		}
		t.sparseList = make([]sparseEntry, 0, len(sorted))
		var curB uint32 = ^uint32(0)
		var word int64
		for _, idx := range sorted {
			b, pos := bitArrayBitBucketPosHelper(uint64(idx))
			bu := uint32(b)
			if bu != curB {
				if curB != ^uint32(0) && word != 0 {
					t.sparseList = append(t.sparseList, sparseEntry{b: curB, w: word})
				}
				curB, word = bu, 0
			}
			word |= 1 << pos
		}
		if word != 0 {
			t.sparseList = append(t.sparseList, sparseEntry{b: curB, w: word})
		}
		if uint64(len(t.sparseList)) > t.size/sparseThreshold {
			t.toDense()
		}
		return
	}

	// General path: uint64 sorted slice
	sorted := make([]uint64, 0, len(indices))
	for _, i := range indices {
		if i < cap {
			sorted = append(sorted, i)
		}
	}
	if len(sorted) == 0 {
		return
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	wr := 0
	for _, idx := range sorted {
		if wr > 0 && sorted[wr-1] == idx {
			continue
		}
		sorted[wr] = idx
		wr++
	}
	sorted = sorted[:wr]
	if t.data != nil {
		for _, idx := range sorted {
			b, pos := bitArrayBitBucketPosHelper(idx)
			t.data[b] |= 1 << pos
		}
		return
	}
	if buildFresh && len(sorted) > indicesToMapThreshold {
		var prevB uint64 = ^uint64(0)
		uniqueBuckets := 0
		for _, idx := range sorted {
			b, _ := bitArrayBitBucketPosHelper(idx)
			if b != prevB {
				uniqueBuckets++
				prevB = b
			}
		}
		useIndices32 := cap <= 1<<32 && uniqueBuckets == len(sorted)
		if useIndices32 {
			t.sparseIndices32 = make([]uint32, len(sorted))
			for i, idx := range sorted {
				t.sparseIndices32[i] = uint32(idx)
			}
			return
		}
		t.sparseList = make([]sparseEntry, 0, len(sorted))
		var curB uint64 = ^uint64(0)
		var word int64
		for _, idx := range sorted {
			b, pos := bitArrayBitBucketPosHelper(idx)
			if b != curB {
				if curB != ^uint64(0) && word != 0 {
					t.sparseList = append(t.sparseList, sparseEntry{b: uint32(curB), w: word})
				}
				curB, word = b, 0
			}
			word |= 1 << pos
		}
		if word != 0 {
			t.sparseList = append(t.sparseList, sparseEntry{b: uint32(curB), w: word})
		}
		if uint64(len(t.sparseList)) > t.size/sparseThreshold {
			t.toDense()
		}
		return
	}
	if buildFresh && len(sorted) <= indicesToMapThreshold {
		t.sparseIndices = make([]uint64, len(sorted))
		copy(t.sparseIndices, sorted)
		return
	}
	for _, idx := range sorted {
		t.SetBitMust(idx)
	}
}

func (t *BitArray) unSetBit(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	mask := int64(1 << bitPos)
	if t.data != nil {
		t.data[bitBucket] &^= mask
		return
	}
	if t.sparseIndices != nil {
		t.sparseIndices = removeFromSorted(t.sparseIndices, i)
		return
	}
	if t.sparseIndices32 != nil {
		t.sparseIndices32 = removeFromSortedU32(t.sparseIndices32, uint32(i))
		return
	}
	word, ok := t.sparseListLookup(bitBucket)
	if !ok {
		return
	}
	word &^= mask
	if word == 0 {
		t.sparseListDelete(bitBucket)
	} else {
		t.sparseListSet(bitBucket, word)
	}
}

func (t *BitArray) inverseBit(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	mask := int64(1 << bitPos)
	if t.data != nil {
		t.data[bitBucket] ^= mask
		return
	}
	if t.sparseIndices != nil {
		if _, found := indexOfSorted(t.sparseIndices, i); found {
			t.sparseIndices = removeFromSorted(t.sparseIndices, i)
		} else {
			t.sparseIndices = insertSorted(t.sparseIndices, i)
			if len(t.sparseIndices) > indicesToMapThreshold {
				t.indicesToSparseList()
			}
		}
		return
	}
	if t.sparseIndices32 != nil {
		idx32 := uint32(i)
		if _, found := indexOfSortedU32(t.sparseIndices32, idx32); found {
			t.sparseIndices32 = removeFromSortedU32(t.sparseIndices32, idx32)
		} else {
			// Check if adding would create two bits in same bucket → convert to list first
			for _, idx := range t.sparseIndices32 {
				if uint64(idx)>>6 == bitBucket {
					t.sparseIndices32ToSparseList()
					word, _ := t.sparseListLookup(bitBucket)
					t.sparseListSet(bitBucket, word^mask)
					return
				}
			}
			t.sparseIndices32 = insertSortedU32(t.sparseIndices32, idx32)
		}
		return
	}
	word, _ := t.sparseListLookup(bitBucket)
	word ^= mask
	if word == 0 {
		t.sparseListDelete(bitBucket)
	} else {
		t.sparseListSet(bitBucket, word)
	}
}

func (t *BitArray) Capacity() (size uint64) {
	return t.cap
}

func (t *BitArray) GetSize() (size uint64) {
	return t.size
}

// Count returns the number of set bits (population count).
func (t *BitArray) Count() uint64 {
	if t.data != nil {
		var n uint64
		for _, w := range t.data {
			n += uint64(bits.OnesCount64(uint64(w)))
		}
		return n
	}
	if t.sparseList != nil {
		var n uint64
		for _, e := range t.sparseList {
			n += uint64(bits.OnesCount64(uint64(e.w)))
		}
		return n
	}
	if t.sparseIndices32 != nil {
		return uint64(len(t.sparseIndices32))
	}
	return uint64(len(t.sparseIndices))
}

func (t *BitArray) GetBit(i uint64) bool {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	if t.size <= bitBucket {
		return false
	}
	if t.data != nil {
		return (t.data[bitBucket] & (1 << bitPos)) != 0
	}
	if t.sparseIndices32 != nil {
		if i >= t.cap {
			return false
		}
		_, ok := indexOfSortedU32(t.sparseIndices32, uint32(i))
		return ok
	}
	if t.sparseIndices != nil {
		_, ok := indexOfSorted(t.sparseIndices, i)
		return ok
	}
	word, ok := t.sparseListLookup(bitBucket)
	if !ok {
		return false
	}
	return (word & (1 << bitPos)) != 0
}

func (t *BitArray) Intersection(other BitArray) bool {
	if t.size != other.size {
		return false
	}
	// Dense vs dense
	if t.data != nil && other.data != nil {
		for i := uint64(0); i < t.size; i++ {
			if t.data[i]&other.data[i] != other.data[i] {
				return false
			}
		}
		return true
	}
	// other in indices mode: every index in other must be set in t
	if other.sparseIndices32 != nil {
		for _, idx := range other.sparseIndices32 {
			if !t.GetBit(uint64(idx)) {
				return false
			}
		}
		return true
	}
	if other.sparseIndices != nil {
		for _, i := range other.sparseIndices {
			if !t.GetBit(i) {
				return false
			}
		}
		return true
	}
	// Check that every set bit in other is set in t (other ⊆ t)
	if other.data != nil {
		for i := uint64(0); i < other.size; i++ {
			ow := other.data[i]
			if ow == 0 {
				continue
			}
			var tw int64
			if t.data != nil {
				tw = t.data[i]
			} else if t.sparseList != nil {
				tw, _ = t.sparseListLookup(i)
			} else if t.sparseIndices32 != nil {
				for _, idx := range t.sparseIndices32 {
					bk, pos := bitArrayBitBucketPosHelper(uint64(idx))
					if bk == i {
						tw |= 1 << pos
					}
				}
			} else {
				for _, idx := range t.sparseIndices {
					b, pos := bitArrayBitBucketPosHelper(idx)
					if b == i {
						tw |= 1 << pos
					}
				}
			}
			if tw&ow != ow {
				return false
			}
		}
		return true
	}
	// other is sparse (list): iterate over other's buckets
	for _, e := range other.sparseList {
		b, ow := uint64(e.b), e.w
		var tw int64
		if t.data != nil {
			tw = t.data[b]
		} else if t.sparseList != nil {
			tw, _ = t.sparseListLookup(b)
		} else if t.sparseIndices32 != nil {
			for _, idx := range t.sparseIndices32 {
				bk, pos := bitArrayBitBucketPosHelper(uint64(idx))
				if bk == b {
					tw |= 1 << pos
				}
			}
		} else {
			for _, idx := range t.sparseIndices {
				bk, pos := bitArrayBitBucketPosHelper(idx)
				if bk == b {
					tw |= 1 << pos
				}
			}
		}
		if tw&ow != ow {
			return false
		}
	}
	return true
}

const serialMagic = 0x52524142 // "BARR" little-endian

// WriteTo serializes the BitArray to w (e.g. for saving to DB). Implements io.WriterTo.
func (t *BitArray) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte
	binary.LittleEndian.PutUint32(buf[:4], serialMagic)
	if _, err = w.Write(buf[:4]); err != nil {
		return int64(4), err
	}
	n = 4
	binary.LittleEndian.PutUint64(buf[:], t.size)
	if _, err = w.Write(buf[:8]); err != nil {
		return n + 8, err
	}
	n += 8
	binary.LittleEndian.PutUint64(buf[:], t.cap)
	if _, err = w.Write(buf[:8]); err != nil {
		return n + 8, err
	}
	n += 8
	if t.data != nil {
		buf[0] = 0
		if _, err = w.Write(buf[:1]); err != nil {
			return n + 1, err
		}
		n++
		for _, wd := range t.data {
			binary.LittleEndian.PutUint64(buf[:], uint64(wd))
			if _, err = w.Write(buf[:8]); err != nil {
				return n + 8, err
			}
			n += 8
		}
		return n, nil
	}
	if t.sparseList != nil {
		buf[0] = 1
		if _, err = w.Write(buf[:1]); err != nil {
			return n + 1, err
		}
		n++
		binary.LittleEndian.PutUint64(buf[:], uint64(len(t.sparseList)))
		if _, err = w.Write(buf[:8]); err != nil {
			return n + 8, err
		}
		n += 8
		for _, e := range t.sparseList {
			binary.LittleEndian.PutUint32(buf[:4], e.b)
			if _, err = w.Write(buf[:4]); err != nil {
				return n, err
			}
			n += 4
			binary.LittleEndian.PutUint64(buf[:], uint64(e.w))
			if _, err = w.Write(buf[:8]); err != nil {
				return n, err
			}
			n += 8
		}
		return n, nil
	}
	if t.sparseIndices32 != nil {
		buf[0] = 3
		if _, err = w.Write(buf[:1]); err != nil {
			return n + 1, err
		}
		n++
		binary.LittleEndian.PutUint64(buf[:], uint64(len(t.sparseIndices32)))
		if _, err = w.Write(buf[:8]); err != nil {
			return n + 8, err
		}
		n += 8
		for _, idx := range t.sparseIndices32 {
			binary.LittleEndian.PutUint32(buf[:4], idx)
			if _, err = w.Write(buf[:4]); err != nil {
				return n, err
			}
			n += 4
		}
		return n, nil
	}
	buf[0] = 2
	if _, err = w.Write(buf[:1]); err != nil {
		return n + 1, err
	}
	n++
	binary.LittleEndian.PutUint64(buf[:], uint64(len(t.sparseIndices)))
	if _, err = w.Write(buf[:8]); err != nil {
		return n + 8, err
	}
	n += 8
	for _, idx := range t.sparseIndices {
		binary.LittleEndian.PutUint64(buf[:], idx)
		if _, err = w.Write(buf[:8]); err != nil {
			return n, err
		}
		n += 8
	}
	return n, nil
}

// ReadFrom deserializes a BitArray from r (e.g. loaded from DB). Implements io.ReaderFrom.
func (t *BitArray) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [8]byte
	if _, err = io.ReadFull(r, buf[:4]); err != nil {
		return 0, err
	}
	n = 4
	if binary.LittleEndian.Uint32(buf[:4]) != serialMagic {
		return n, fmt.Errorf("bit_array: invalid magic")
	}
	if _, err = io.ReadFull(r, buf[:8]); err != nil {
		return n, err
	}
	n += 8
	t.size = binary.LittleEndian.Uint64(buf[:])
	if _, err = io.ReadFull(r, buf[:8]); err != nil {
		return n, err
	}
	n += 8
	t.cap = binary.LittleEndian.Uint64(buf[:])
	if _, err = io.ReadFull(r, buf[:1]); err != nil {
		return n, err
	}
	n++
	typ := buf[0]
	switch typ {
	case 0:
		t.data = make([]int64, t.size)
		for i := uint64(0); i < t.size; i++ {
			if _, err = io.ReadFull(r, buf[:8]); err != nil {
				return n, err
			}
			t.data[i] = int64(binary.LittleEndian.Uint64(buf[:]))
			n += 8
		}
		t.sparseList = nil
		t.sparseIndices = nil
		return n, nil
	case 1:
		if _, err = io.ReadFull(r, buf[:8]); err != nil {
			return n, err
		}
		n += 8
		L := binary.LittleEndian.Uint64(buf[:])
		t.sparseList = make([]sparseEntry, 0, L)
		for i := uint64(0); i < L; i++ {
			if _, err = io.ReadFull(r, buf[:4]); err != nil {
				return n, err
			}
			n += 4
			bu := binary.LittleEndian.Uint32(buf[:4])
			if _, err = io.ReadFull(r, buf[:8]); err != nil {
				return n, err
			}
			n += 8
			t.sparseList = append(t.sparseList, sparseEntry{b: bu, w: int64(binary.LittleEndian.Uint64(buf[:]))})
		}
		t.data = nil
		t.sparseIndices = nil
		t.sparseIndices32 = nil
		return n, nil
	case 2:
		if _, err = io.ReadFull(r, buf[:8]); err != nil {
			return n, err
		}
		n += 8
		L := binary.LittleEndian.Uint64(buf[:])
		t.sparseIndices = make([]uint64, 0, L)
		for i := uint64(0); i < L; i++ {
			if _, err = io.ReadFull(r, buf[:8]); err != nil {
				return n, err
			}
			n += 8
			t.sparseIndices = append(t.sparseIndices, binary.LittleEndian.Uint64(buf[:]))
		}
		t.data = nil
		t.sparseList = nil
		t.sparseIndices32 = nil
		return n, nil
	case 3:
		if _, err = io.ReadFull(r, buf[:8]); err != nil {
			return n, err
		}
		n += 8
		L := binary.LittleEndian.Uint64(buf[:])
		t.sparseIndices32 = make([]uint32, 0, L)
		for i := uint64(0); i < L; i++ {
			if _, err = io.ReadFull(r, buf[:4]); err != nil {
				return n, err
			}
			n += 4
			t.sparseIndices32 = append(t.sparseIndices32, binary.LittleEndian.Uint32(buf[:4]))
		}
		t.data = nil
		t.sparseList = nil
		t.sparseIndices = nil
		return n, nil
	default:
		return n, fmt.Errorf("bit_array: unknown type %d", typ)
	}
}
