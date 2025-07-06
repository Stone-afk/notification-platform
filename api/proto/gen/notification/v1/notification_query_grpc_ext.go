package notificationv1

import (
	"github.com/redis/go-redis/v9"
)

type QueryCacheFirstService struct {
	NotificationQueryServiceClient
	rdb *redis.Client
}

// QueryNotification 叫做缓存前置
//func (g *QueryCacheFirstService) QueryNotification(ctx context.Context, in *QueryNotificationRequest, opts ...grpc.CallOption) (*QueryNotificationResponse, error) {
//	key := fmt.Sprintf("notification:%s", in.Key)
//	val, err := g.rdb.Get(ctx, key)
//	if err == nil {
//		// 找到了，直接返回
//	}
//	// 没找到
//	// 可以直接把请求发到服务端
//	res, err := g.NotificationQueryServiceClient.QueryNotification(ctx, in)
//}
