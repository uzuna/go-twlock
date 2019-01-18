package twlock_test

import (
	"context"
	"testing"
	"time"
)

func TestTway(t testing.T) {

}

type DummyCache struct {
	data map[string]interface{}
}

func (d *DummyCache) Set(key string, v interface{}, expire time.Duration) error {
	return nil
}
func (d *DummyCache) Get(ctx context.Context, key string) (v interface{}, ok bool, err error) {
	return nil, false, nil
}

type DummyRequest struct {
	Wait   time.Duration
	HasNot bool
	Error  error
}

type DummyOrigin struct{}

func (d *DummyOrigin) Get(ctx context.Context, x DummyRequest) (v interface{}, ok bool, err error) {
	if x.Error != nil {
		return nil, ok, x.Error
	}

	timer := time.NewTimer(x.Duration)
	select {
	case <-ctx.Done():
		return nil, ok, ctx.Err()
	case <-timer.C:
		return x.Wait, x.HasNot, nil
	}
}
