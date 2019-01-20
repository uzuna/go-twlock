package twlock

import (
	"context"
	"sync"
)

type Named interface {
	Name() string
}

type MemoryCacheOption func(o *MemoryCache)

func NewMemoryRequest(opts ...MemoryCacheOption) *MemoryCache {
	c := &MemoryCache{
		data:      make(map[string]interface{}),
		cacheLock: new(sync.RWMutex),
	}
	for _, v := range opts {
		v(c)
	}
	return c
}

type MemoryCache struct {
	data      map[string]interface{}
	cacheLock *sync.RWMutex

	// for LifeCount
	lifeCount    int
	lifeCountMap map[string]int
}

func (d *MemoryCache) Set(req Named, res interface{}) error {
	name := req.Name()
	d.cacheLock.Lock()
	d.data[name] = res
	d.cacheLock.Unlock()
	return nil
}

func (d *MemoryCache) Get(ctx context.Context, req Named, res interface{}) (ok bool, err error) {
	name := req.Name()
	d.cacheLock.RLock()
	if x, ok := d.data[name]; ok {
		d.cacheLock.RUnlock()
		// non-expires
		if d.lifeCount < 1 {
			err := WriteToInterface(res, x)
			return true, err
		}

		// ReCheck
		d.cacheLock.Lock()
		// Confirm whether it is being processed in concurrent process
		if _, ok := d.data[name]; !ok {
			d.cacheLock.Unlock()
			return ok, nil
		}
		// life count
		d.lifeCountMap[name]++
		if d.lifeCountMap[name] > d.lifeCount {
			d.lifeCountMap[name] = 0
			delete(d.data, name)
		}
		d.cacheLock.Unlock()
		err := WriteToInterface(res, x)
		return true, err
	}
	d.cacheLock.RUnlock()
	return ok, nil
}

// WithLifeCount is set life count to memory cache
func WithLifeCount(count int) MemoryCacheOption {
	return func(o *MemoryCache) {
		o.lifeCount = count
		o.lifeCountMap = make(map[string]int)
	}
}
