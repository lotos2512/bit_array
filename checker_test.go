package bit_array

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCheckExists(t *testing.T) {
	lenMsisdns := uint64(600000)
	keyMap := make(map[uint64]uint64, int(float64(lenMsisdns)*1.3))
	var msisdnsAll uint64
	msisdnsSegments := []uint64{}

	for i := uint64(0); i < lenMsisdns; i++ {
		keyMap[msisdnsAll] = i
		if i%1000 == 0 {
			msisdnsSegments = append(msisdnsSegments, msisdnsAll)
		}
		msisdnsAll++
	}

	v := atomic.Value{}
	v.Store(keyMap)

	gm := &GlobalMap{
		keyMap:      v,
		mutex:       sync.RWMutex{},
		msisdnCount: msisdnsAll,
	}

	r := New(msisdnsSegments, gm)

	for i := range msisdnsSegments {
		v := r.CheckExists(msisdnsSegments[i])
		assert.Equal(t, true, v)
	}

}

// BenchmarkCheckExists-8   	185780337	         6.422 ns/op	       0 B/op	       0 allocs/op
func BenchmarkCheckExists(b *testing.B) {
	msisdns := []uint64{17952592, 17952593, 17952594, 17952595, 17952596, 179525}
	keyMap := map[uint64]uint64{
		17952592: 0,
		17952593: 1,
		17952594: 2,
		17952595: 3,
		17952596: 4,
		179525:   5,
	}
	v := atomic.Value{}
	v.Store(keyMap)

	gm := &GlobalMap{
		keyMap:      v,
		mutex:       sync.RWMutex{},
		msisdnCount: uint64(len(keyMap)),
	}

	r := New(msisdns, gm)
	var res bool
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res = r.CheckExists(17952593)
	}
	b.StopTimer()
	if res {

	}
}
