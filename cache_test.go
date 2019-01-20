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

func TestCache(t *testing.T) {

	length := 100000
	split := 10
	cacheLifeCount := length / split

	// cache option
	var opts []twlock.MemoryCacheOption

	opts = append(opts,
		twlock.WithLifeCount(cacheLifeCount),
		twlock.WithLifeTime(time.Millisecond*900),
	)

	c := twlock.NewMemoryRequest(opts...)
	o := &CopyOrigin{
		mutex: new(sync.Mutex),
	}
	l := twlock.NewTWLock(c, o.Get)

	wg := &sync.WaitGroup{}
	f := func(i int) {
		defer wg.Done()
		req := &waitRequest{
			name: "X",
			wait: time.Millisecond,
		}
		ctx := context.Background()
		var res WaitReply
		err := l.Serve(ctx, req, &res)
		// log.Println(res)
		if err != nil && !res.atTime.IsZero() {
			log.Println(res, err)
			t.Fail()
		}
	}

	// ogirinCountExpect := length / cacheLifeCount
	for i := 0; i < length; i++ {
		wg.Add(1)
		go f(i)
	}
	wg.Wait()

	assert.Equal(t, split, o.counter)
	wg.Add(1)
	f(1)
	assert.Equal(t, split, o.counter)

	time.Sleep(time.Second)
	wg.Add(1)
	f(1)
	assert.Equal(t, split+1, o.counter)
}

type waitRequest struct {
	name string
	wait time.Duration
}

func (r *waitRequest) WaitTime() time.Duration {
	return r.wait
}
func (r *waitRequest) Name() string {
	return r.name
}

type WaitRequest interface {
	WaitTime() time.Duration
}

type WaitReply struct {
	time   time.Duration
	atTime time.Time
}

type CopyOrigin struct {
	mutex   *sync.Mutex
	counter int
}

func (d *CopyOrigin) Get(ctx context.Context, req interface{}, res interface{}) (bool, error) {
	d.mutex.Lock()
	d.counter++
	d.mutex.Unlock()

	x := req.(WaitRequest)
	timer := time.NewTimer(x.WaitTime())

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-timer.C:
		x := &WaitReply{
			time:   x.WaitTime(),
			atTime: time.Now(),
		}
		err := twlock.WriteToInterface(res, x)
		return true, err
	}
}
