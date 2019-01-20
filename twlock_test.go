package twlock_test

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	twlock "github.com/uzuna/go-twlock"
)

func TestTway(t *testing.T) {
	length := 100000 * 6
	cacheLifeCount := length / 10
	c := newDummyCache(cacheLifeCount)
	o := &DummyOrigin{
		mutex: new(sync.Mutex),
	}

	l := twlock.NewTWLock(c, o.Get)

	wg := &sync.WaitGroup{}
	f := func(i int) {
		defer wg.Done()
		req := &DummyRequest{
			name: "X",
			Wait: time.Millisecond,
			Has:  true,
		}
		ctx := context.Background()
		var res DummyRequest
		err := l.Serve(ctx, req, &res)
		if err != nil && res.Wait < 1 {
			log.Println(res, err)
			t.Fail()
		}
	}

	ogirinCountExpect := length / cacheLifeCount
	for i := 0; i < length; i++ {
		wg.Add(1)
		go f(i)
	}
	wg.Wait()

	assert.Equal(t, length, c.counter+o.counter)
	assert.Equal(t, ogirinCountExpect, o.counter)
}

func newDummyCache(lifeCount int) *DummyCache {
	return &DummyCache{
		data:        make(map[string]*DummyRequest),
		accessCount: make(map[string]int),
		lifeCount:   lifeCount,
		cacheLock:   new(sync.RWMutex),
	}
}

type DummyCache struct {
	data        map[string]*DummyRequest
	accessCount map[string]int
	lifeCount   int
	cacheLock   *sync.RWMutex
	counter     int
}

func (d *DummyCache) Set(req twlock.Named, res interface{}) error {
	name := req.Name()
	d.cacheLock.Lock()
	d.data[name] = res.(*DummyRequest)
	d.cacheLock.Unlock()
	return nil
}
func (d *DummyCache) Get(ctx context.Context, req twlock.Named, res interface{}) (ok bool, err error) {
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
		d.counter++
		d.accessCount[name]++
		if d.accessCount[name] > d.lifeCount {
			log.Println("UnCache", d.counter)
			d.accessCount[name] = 0
			delete(d.data, name)
		}
		d.cacheLock.Unlock()

		err := twlock.WriteToInterface(res, x)
		return true, err
	}
	d.cacheLock.RUnlock()
	return ok, nil
}

type DummyRequest struct {
	name  string
	Wait  time.Duration
	Has   bool
	Error error
}

func (r *DummyRequest) Name() string {
	return r.name
}

type DummyOrigin struct {
	mutex   *sync.Mutex
	counter int
}

func (d *DummyOrigin) Get(ctx context.Context, req interface{}, res interface{}) (bool, error) {
	d.mutex.Lock()
	d.counter++
	d.mutex.Unlock()
	x := req.(*DummyRequest)
	if x.Error != nil {
		return false, x.Error
	}

	timer := time.NewTimer(x.Wait)
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-timer.C:
		err := twlock.WriteToInterface(res, x)
		return x.Has, err
	}
}
