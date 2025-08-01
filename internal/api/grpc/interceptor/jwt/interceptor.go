package jwt

import (
	"context"
	"notification-platform/internal/errs"
)

func GetBizIDFromContext(ctx context.Context) (int64, error) {
	val := ctx.Value(BizIDName)
	if val == nil {
		return 0, errs.ErrBizIDNotFound
	}
	v, ok := val.(int64)
	if !ok {
		return 0, errs.ErrBizIDNotFound
	}
	return v, nil
}
