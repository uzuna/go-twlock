package twlock

import (
	"context"
	"sync"
)

type Named interface {
	Name() string
}

func NewMemoryRequest() *MemoryCache {
	return &MemoryCache{
		data:      make(map[string]interface{}),
		cacheLock: new(sync.RWMutex),
	}
}

type MemoryCache struct {
	data      map[string]interface{}
	cacheLock *sync.RWMutex
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
		// ReCheck
		d.cacheLock.Lock()
		if _, ok := d.data[name]; !ok {
			d.cacheLock.Unlock()
			return ok, nil
		}
		d.cacheLock.Unlock()
		err := WriteToInterface(res, x)
		return true, err
	}
	d.cacheLock.RUnlock()
	return ok, nil
}
