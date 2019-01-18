package twlock

import "context"

// あるリソースへのアクセスを表現するInterface
type ResourceFunc func(ctx context.Context, req interface{}) (res interface{}, ok bool, err error)

// リクエストの管理グループ名を返す
type GroupFunc func(ctx context.Context, req interface{}) string

type TWLock struct {
	groupFunc  GroupFunc
	cacheFunc  ResourceFunc
	originFunc ResourceFunc
}
