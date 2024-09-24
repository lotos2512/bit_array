package bit_array

import (
	"sync"
	"sync/atomic"
)

type GlobalMap struct {
	keyMap      atomic.Value // map[uint64]uint64
	mutex       sync.RWMutex
	msisdnCount uint64
}

type MapGetter interface {
	GetMap() map[uint64]uint64
	GetLen() uint64
}

type MapUpdater interface{}

func (g *GlobalMap) GetMap() map[uint64]uint64 {
	v := g.keyMap.Load()
	if v != nil {
		return v.(map[uint64]uint64)
	}
	return nil
}

func (g *GlobalMap) GetLen() uint64 {
	return g.msisdnCount
}

type Cheker struct {
	keyMap   MapGetter
	bitArray BitArray
}

func New(msisdns []uint64, gm MapGetter) *Cheker {
	bitArray := NewBitArray((gm.GetLen()))
	gMap := gm.GetMap()
	for i := range msisdns {
		index, ok := gMap[msisdns[i]]
		if ok {
			bitArray.SetBitMust(index)
		}
	}

	r := &Cheker{
		keyMap:   gm,
		bitArray: bitArray,
	}

	return r
}

func (r *Cheker) CheckExists(msisdn uint64) bool {
	msisdnIndex, exists := r.keyMap.GetMap()[msisdn]
	if !exists {
		return false
	}
	if msisdnIndex > r.bitArray.Capacity()-1 {
		return false
	}

	return r.bitArray.GetBit(msisdnIndex)
}
