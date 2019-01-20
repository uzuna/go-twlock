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
	g := func(req interface{}) string {
		x := req.(*DummyRequest)
		return x.Name
	}

	length := 100000 * 6
	cacheLifeCount := length / 10
	c := newDummyCache(cacheLifeCount)
	o := &DummyOrigin{
		mutex: new(sync.Mutex),
	}

	l := twlock.NewTWLock(g, c, o.Get)

	wg := &sync.WaitGroup{}
	f := func(i int) {
		defer wg.Done()
		req := &DummyRequest{
			Name: "X",
			Wait: time.Millisecond,
			Has:  true,
		}
		ctx := context.Background()
		var res int
		err := l.In(ctx, req, &res)
		if err != nil && res < 1 {
			log.Println(res, err)
			panic(err)
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

	// table := []*DummyRequest{
	// 	&DummyRequest{
	// 		Name: "X",
	// 		Wait: time.Millisecond * 10,
	// 		Has:  true,
	// 	},
	// }

	// for _, v := range table {
	// 	ctx := context.Background()
	// 	res, err := l.In(ctx, v)
	// 	log.Println(res, err)
	// }
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

func (d *DummyCache) Set(req interface{}, res interface{}) error {
	x := req.(*DummyRequest)
	d.cacheLock.Lock()
	d.data[x.Name] = x
	d.cacheLock.Unlock()
	return nil
}
func (d *DummyCache) Get(ctx context.Context, req interface{}, res interface{}) (ok bool, err error) {
	x := req.(*DummyRequest)

	d.cacheLock.RLock()
	if x, ok := d.data[x.Name]; ok {
		d.cacheLock.RUnlock()
		// ReCheck
		d.cacheLock.Lock()
		if _, ok := d.data[x.Name]; !ok {
			d.cacheLock.Unlock()
			return ok, nil
		}
		d.counter++
		d.accessCount[x.Name]++
		if d.accessCount[x.Name] > d.lifeCount {
			log.Println("UnCache", d.counter)
			d.accessCount[x.Name] = 0
			delete(d.data, x.Name)
		}
		d.cacheLock.Unlock()

		err := twlock.WriteToInterface(res, int(x.Wait))
		return true, err
	}
	d.cacheLock.RUnlock()
	return ok, nil
}

type DummyRequest struct {
	Name  string
	Wait  time.Duration
	Has   bool
	Error error
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
		err := twlock.WriteToInterface(res, int(x.Wait))
		return x.Has, err
	}
}
