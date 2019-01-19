package twlock

import (
	"context"
	"sync"
)

// あるリソースへのアクセスを表現するInterface
type ResourceFunc func(ctx context.Context, req interface{}) (res interface{}, ok bool, err error)

// リクエストの管理グループ名を返す
type GroupFunc func(req interface{}) string

type CacheStore interface {
	Get(ctx context.Context, req interface{}) (res interface{}, ok bool, err error)
	Set(req interface{}, res interface{}) error
}

func NewTWLock(g GroupFunc, c CacheStore, o ResourceFunc) *TWLock {
	return &TWLock{
		groupFunc:  g,
		cacheFunc:  c,
		originFunc: o,
		originLock: make(map[string]struct{}),
		cacheLock:  make(map[string]*sync.RWMutex),
		mux:        new(sync.Mutex),
	}
}

type TWLock struct {
	groupFunc  GroupFunc
	cacheFunc  CacheStore
	originFunc ResourceFunc
	originLock map[string]struct{}
	cacheLock  map[string]*sync.RWMutex
	mux        *sync.Mutex
}

// Request Sequence
func (l *TWLock) In(ctx context.Context, req interface{}) (interface{}, error) {
	// get cache group
	grName := l.groupFunc(req)

	// create cache lock
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
		res, ok, err := l.cacheFunc.Get(ctx, req)
		if err != nil {
			cl.RUnlock()
			return nil, err
		}
		if ok {
			cl.RUnlock()
			return res, nil
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
	// Origin request
	cl.Lock()
	defer func() {
		cl.Unlock()
		delete(l.originLock, grName)
	}()

	// origin route確認
	res, ok, err := l.originFunc(ctx, req)
	if err != nil {
		return nil, err
	}
	if !ok {
		return res, nil
	}
	// set cache
	err = l.cacheFunc.Set(req, res)
	return res, err
}
