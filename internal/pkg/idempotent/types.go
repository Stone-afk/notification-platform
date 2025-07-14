package idempotent

import "context"

type IdempotencyService interface {
	// Exists 这里的Exist是包含添加的语义的，返回true说明存在，返回false说明不存在 且已经将key添加了，下面的MExists也是同理
	Exists(ctx context.Context, key string) (bool, error)
	MExists(ctx context.Context, keys ...string) ([]bool, error)
}
