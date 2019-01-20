package twlock

import (
	"context"
	"sync"
)

// あるリソースへのアクセスを表現するInterface
type ResourceFunc func(ctx context.Context, req interface{}, res interface{}) (ok bool, err error)

// CacheStore is cache module interface
// Must request has "name" for identify
// Cache module store the responce from origin by identity of "name"
type CacheStore interface {
	// When has not data then return false into ok variable
	Get(ctx context.Context, req Named, res interface{}) (ok bool, err error)
	Set(req Named, res interface{}) error
}

// NewTWLock return TWLock
func NewTWLock(c CacheStore, o ResourceFunc) *TWLock {
	return &TWLock{
		cacheFunc:  c,
		originFunc: o,
		originLock: make(map[string]struct{}),
		cacheLock:  make(map[string]*sync.RWMutex),
		mux:        new(sync.Mutex),
	}
}

// TWLock handle the request and control responce from cache or origin
type TWLock struct {
	cacheFunc  CacheStore
	originFunc ResourceFunc
	originLock map[string]struct{}      // Mutex for originFunc
	cacheLock  map[string]*sync.RWMutex // Mutex for cacheFunc
	mux        *sync.Mutex              // Mutex for access originLock
}

// Serve is controll request and responce
func (l *TWLock) Serve(ctx context.Context, req Named, res interface{}) error {
	// get request identity
	grName := req.Name()

	// check and create cache lock
	cl, ok := l.cacheLock[grName]
	if !ok {
		l.mux.Lock()
		if l.cacheLock[grName] == nil {
			l.cacheLock[grName] = new(sync.RWMutex)
		}
		cl = l.cacheLock[grName]
		l.mux.Unlock()
	}

	for {
		// cache read and wait
		cl.RLock()
		ok, err := l.cacheFunc.Get(ctx, req, res)
		if err != nil {
			cl.RUnlock()
			return err
		}
		if ok {
			cl.RUnlock()
			return nil
		}

		// When has not cache data
		// 1 request through to origin resource and lock resource.
		l.mux.Lock()
		if _, ok := l.originLock[grName]; ok {
			// If request isafter resource mutex flow that back to read lock
			l.mux.Unlock()
			cl.RUnlock()
			continue
		}
		l.originLock[grName] = struct{}{}
		l.mux.Unlock()
		cl.RUnlock()
		break
	}

	// Request to Origin
	cl.Lock()
	defer func() {
		cl.Unlock()
		l.mux.Lock()
		delete(l.originLock, grName)
		l.mux.Unlock()
	}()

	// origin route確認
	ok, err := l.originFunc(ctx, req, res)
	if err != nil {
		// @todo error発生時におなじジョブが流れるのを許すかどうか
		// 再開可能にするとしたらどれだけ時間を空けるか
		return err
	}
	if !ok {
		return nil
	}
	// set cache
	err = l.cacheFunc.Set(req, res)
	return err
}
