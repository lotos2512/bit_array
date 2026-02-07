package bit_array

import (
	"fmt"
	"sort"
	"strconv"
)

// sparseThreshold: switch to dense only when non-zero buckets exceed size/sparseThreshold.
// 2 = switch when >50% buckets used (fewer zeros in memory); was 8 (~12.5%), caused 1MB of zeros for step-500 pattern.
const sparseThreshold = 2

// indicesToMapThreshold: when set bit count exceeds this, switch from indices (8 bytes/bit) to sparse list.
const indicesToMapThreshold = 64

// sparseEntry: один непустой bucket — только 16 байт, без накладных расходов map.
type sparseEntry struct {
	b uint64 // bucket index
	w int64  // word (64 bits)
}

type BitArray struct {
	data          []int64       // nil when using sparse
	sparseList    []sparseEntry // sorted by b; used when data nil and sparseIndices nil (compact, no map overhead)
	sparseIndices []uint64      // sorted set bit indices; used when data nil and sparseList nil (minimal memory)
	size          uint64
	cap           uint64
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
			list = append(list, sparseEntry{b: uint64(i), w: w})
		}
	}
	return BitArray{
		size:       size,
		cap:        capacity,
		sparseList: list,
	}
}

func (t *BitArray) inIndicesMode() bool {
	return t.data == nil && t.sparseList == nil
}

func (t *BitArray) sparseListLookup(bucket uint64) (int64, bool) {
	i := sort.Search(len(t.sparseList), func(j int) bool { return t.sparseList[j].b >= bucket })
	if i < len(t.sparseList) && t.sparseList[i].b == bucket {
		return t.sparseList[i].w, true
	}
	return 0, false
}

func (t *BitArray) sparseListSet(bucket uint64, word int64) {
	i := sort.Search(len(t.sparseList), func(j int) bool { return t.sparseList[j].b >= bucket })
	if i < len(t.sparseList) && t.sparseList[i].b == bucket {
		t.sparseList[i].w = word
		return
	}
	t.sparseList = append(t.sparseList, sparseEntry{})
	copy(t.sparseList[i+1:], t.sparseList[i:])
	t.sparseList[i] = sparseEntry{b: bucket, w: word}
}

func (t *BitArray) sparseListDelete(bucket uint64) {
	i := sort.Search(len(t.sparseList), func(j int) bool { return t.sparseList[j].b >= bucket })
	if i < len(t.sparseList) && t.sparseList[i].b == bucket {
		t.sparseList = append(t.sparseList[:i], t.sparseList[i+1:]...)
	}
}

func (t *BitArray) indicesToSparseList() {
	if t.sparseIndices == nil {
		return
	}
	// Собираем пары (bucket, pos), сортируем по bucket, склеиваем в слова — без временной map
	type bp struct{ b uint64; p uint64 }
	pairs := make([]bp, len(t.sparseIndices))
	for i, idx := range t.sparseIndices {
		b, pos := bitArrayBitBucketPosHelper(idx)
		pairs[i] = bp{b, pos}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].b < pairs[j].b })
	t.sparseList = make([]sparseEntry, 0, len(pairs))
	var cur uint64
	var word int64
	for _, x := range pairs {
		if x.b != cur && word != 0 {
			t.sparseList = append(t.sparseList, sparseEntry{b: cur, w: word})
		}
		if x.b != cur {
			cur, word = x.b, 0
		}
		word |= 1 << x.p
	}
	if word != 0 {
		t.sparseList = append(t.sparseList, sparseEntry{b: cur, w: word})
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
			t.data[e.b] = e.w
		}
		t.sparseList = nil
	} else {
		for _, i := range t.sparseIndices {
			b, pos := bitArrayBitBucketPosHelper(i)
			t.data[b] |= 1 << pos
		}
		t.sparseIndices = nil
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
			out[e.b] = e.w
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
			out[e.b] = e.w
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

// max from NewBitArray(w uint64)
func (t *BitArray) SetBitMust(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	if t.data != nil {
		t.data[bitBucket] |= 1 << bitPos
		return
	}
	if t.inIndicesMode() {
		_, found := indexOfSorted(t.sparseIndices, i)
		if !found {
			t.sparseIndices = insertSorted(t.sparseIndices, i)
			if len(t.sparseIndices) > indicesToMapThreshold {
				t.indicesToSparseList()
				if uint64(len(t.sparseList)) > t.size/sparseThreshold {
					t.toDense()
				}
			}
		}
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

func (t *BitArray) unSetBit(i uint64) {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	mask := int64(1 << bitPos)
	if t.data != nil {
		t.data[bitBucket] &^= mask
		return
	}
	if t.inIndicesMode() {
		t.sparseIndices = removeFromSorted(t.sparseIndices, i)
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
	if t.inIndicesMode() {
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

func (t *BitArray) GetBit(i uint64) bool {
	bitBucket, bitPos := bitArrayBitBucketPosHelper(i)
	if t.size <= bitBucket {
		return false
	}
	if t.data != nil {
		return (t.data[bitBucket] & (1 << bitPos)) != 0
	}
	if t.inIndicesMode() {
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
	if other.inIndicesMode() {
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
		b, ow := e.b, e.w
		var tw int64
		if t.data != nil {
			tw = t.data[b]
		} else if t.sparseList != nil {
			tw, _ = t.sparseListLookup(b)
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
