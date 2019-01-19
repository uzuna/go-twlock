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

	c := newDummyCache()
	o := &DummyOrigin{
		mutex: new(sync.Mutex),
	}

	l := twlock.NewTWLock(g, c, o.Get)

	wg := &sync.WaitGroup{}
	f := func(i int) {
		defer wg.Done()
		req := &DummyRequest{
			Name: "X",
			Wait: time.Millisecond * time.Duration(i) * 100,
			Has:  true,
		}
		ctx := context.Background()
		res, err := l.In(ctx, req)
		if err != nil {
			log.Println(res, err)
		}
	}
	length := 10000
	for i := 0; i < length; i++ {
		wg.Add(1)
		go f(i)
	}
	wg.Wait()

	assert.Equal(t, length, c.counter)
	assert.Equal(t, 1, o.counter)

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

func newDummyCache() *DummyCache {
	return &DummyCache{
		data:      make(map[string]*DummyRequest),
		mutex:     new(sync.Mutex),
		cacheLock: new(sync.RWMutex),
	}
}

type DummyCache struct {
	data      map[string]*DummyRequest
	mutex     *sync.Mutex
	cacheLock *sync.RWMutex
	counter   int
}

func (d *DummyCache) Set(req interface{}, res interface{}) error {
	x := req.(*DummyRequest)
	d.cacheLock.Lock()
	d.data[x.Name] = x
	d.cacheLock.Unlock()
	go func() {
		log.Println("Cache")
		time.Sleep(time.Millisecond)
		d.cacheLock.Lock()
		delete(d.data, x.Name)
		d.cacheLock.Unlock()
	}()
	return nil
}
func (d *DummyCache) Get(ctx context.Context, req interface{}) (v interface{}, ok bool, err error) {
	d.mutex.Lock()
	d.counter++
	d.mutex.Unlock()
	x := req.(*DummyRequest)
	d.cacheLock.RLock()
	defer d.cacheLock.RUnlock()
	if v, ok := d.data[x.Name]; ok {
		return v, true, nil
	}
	return nil, ok, nil
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

func (d *DummyOrigin) Get(ctx context.Context, v interface{}) (interface{}, bool, error) {
	d.mutex.Lock()
	d.counter++
	d.mutex.Unlock()
	x := v.(*DummyRequest)
	if x.Error != nil {
		return nil, false, x.Error
	}

	timer := time.NewTimer(x.Wait)
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	case <-timer.C:
		return x.Wait, x.Has, nil
	}
}
