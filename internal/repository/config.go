package repository

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/sqlx"
	"notification-platform/internal/repository/cache"
	"notification-platform/internal/repository/dao"
	"time"
)

type businessConfigRepository struct {
	dao        dao.BusinessConfigDAO
	localCache cache.ConfigCache
	redisCache cache.ConfigCache
	logger     *elog.Component
}

func (repo *businessConfigRepository) LoadCache(ctx context.Context) error {
	offset := 0
	const (
		limit       = 10
		loopTimeout = time.Second * 3
	)
	for {
		ctx, cancel := context.WithTimeout(ctx, loopTimeout)
		cnt, err := repo.loadCacheBatch(ctx, offset, limit)
		cancel()
		if err != nil {
			// 继续下一轮
			// 精细处理：比如说三个循环都是 error，你就判定数据库不可挽回了，你就中断
			repo.logger.Error("分批加载缓存失败", elog.FieldErr(err))
			continue
		}
		if cnt < limit {
			// 说明没了
			return nil
		}
		offset += cnt
	}
}

func (repo *businessConfigRepository) loadCacheBatch(ctx context.Context, offset, limit int) (int, error) {
	res, err := repo.Find(ctx, offset, limit)
	if err != nil {
		return 0, err
	}
	err = repo.redisCache.SetConfigs(ctx, res)
	if err != nil {
		repo.logger.Error("批量回写 Redis 缓存失败", elog.FieldErr(err))
	}
	err = repo.localCache.SetConfigs(ctx, res)
	if err != nil {
		repo.logger.Error("批量回写本地缓存失败", elog.FieldErr(err))
	}
	return len(res), err
}

func (repo *businessConfigRepository) Find(ctx context.Context, offset, limit int) ([]domain.BusinessConfig, error) {
	res, err := repo.dao.Find(ctx, offset, limit)
	return slice.Map(res, func(_ int, src dao.BusinessConfig) domain.BusinessConfig {
		return repo.toDomain(src)
	}), err
}

// GetByIDs 根据多个ID批量获取业务配置
// 用在异步请求调度的时候批量处理，批量执行，批量发送
func (repo *businessConfigRepository) GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error) {
	// 有两种思路，一种是整体从本地缓存，redis 缓存，数据库中取
	// 另外一种是从本地缓存取，没取到的从 Redis 取，再没取到的，从数据库中取.
	// 1. 先从本地缓存批量获取
	result, err := repo.localCache.GetConfigs(ctx, ids)
	if err != nil {
		repo.logger.Error("从本地缓存批量获取失败", elog.FieldErr(err))
		// 初始化 map，要注意指定容量，规避扩容引发的性能问题
		result = make(map[int64]domain.BusinessConfig, len(ids))
	}
	// 这边就是要尝试从 Redis 里面取
	// 取 result 当中没有的

	// 叠加可用性的设计，只查询本地缓存
	// if ctx.Value("downgrade") == true {
	//	return result, err
	// }

	missedIDs := repo.diffIDs(ids, result)
	if len(missedIDs) == 0 {
		// 一个都不缺，全找到了
		return result, nil
	}
	// 2. 从 Redis 里面获取
	// 相比之下可能需要查询更少的数据，Redis 传输的数据量也更少，性能会更好
	redisConfigs, err := repo.redisCache.GetConfigs(ctx, missedIDs)
	if err != nil {
		repo.logger.Error("从 Redis 中批量获取失败", elog.FieldErr(err))
	} else {
		// 尝试回写 local cache
		// 需要回写的，以及合并 redisConfigs 和 result
		// 这个是精确控制
		configToLocalCache := make([]domain.BusinessConfig, 0, len(redisConfigs))
		for id, conf := range redisConfigs {
			result[id] = conf
			configToLocalCache = append(configToLocalCache, conf)
		}
		// 全部回写，问题不大
		// b.localCache.SetConfigs(ctx, mapx.Values(result))
		err = repo.localCache.SetConfigs(ctx, configToLocalCache)
		if err != nil {
			repo.logger.Error("批量回写本地缓存失败", elog.FieldErr(err))
		}
	}

	// 叠加可用性的设计，查询 Redis 但是不查询数据库
	// if ctx.Value("downgrade") == true {
	// if ctx.Value("rate_limit") == true {
	// if ctx.Value("high_load") == true {
	//	return result, err
	// }

	// 从数据库中获取缓存未找到的配置
	missedIDs = repo.diffIDs(ids, result)
	// 精确控制，查询更少的 id，回表更少的次数
	configMap, err := repo.dao.GetByIDs(ctx, missedIDs)
	if err != nil {
		return nil, err
	}
	// 处理 configMap。回写 redis，回写本地缓存
	configs := make([]domain.BusinessConfig, 0, len(configMap))
	for id := range configMap {
		configs = append(configs, repo.toDomain(configMap[id]))
	}

	if len(configs) > 0 {
		err = repo.localCache.SetConfigs(ctx, configs)
		if err != nil {
			repo.logger.Error("批量回写本地缓存失败", elog.FieldErr(err))
		}

		err = repo.redisCache.SetConfigs(ctx, configs)
		if err != nil {
			repo.logger.Error("批量回写 Redis 缓存失败", elog.FieldErr(err))
		}
	}
	return result, nil
}

func (repo *businessConfigRepository) diffIDs(ids []int64, m map[int64]domain.BusinessConfig) []int64 {
	res := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := m[id]; !ok {
			res = append(res, id)
		}
	}
	return res
}

func (repo *businessConfigRepository) GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *businessConfigRepository) Delete(ctx context.Context, id int64) error {
	//TODO implement me
	panic("implement me")
}

func (repo *businessConfigRepository) SaveConfig(ctx context.Context, config domain.BusinessConfig) error {
	//TODO implement me
	panic("implement me")
}

func (repo *businessConfigRepository) toDomain(config dao.BusinessConfig) domain.BusinessConfig {
	domainCfg := domain.BusinessConfig{
		ID:        config.ID,
		OwnerID:   config.OwnerID,
		OwnerType: config.OwnerType,
		RateLimit: config.RateLimit,
		Ctime:     config.Ctime,
		Utime:     config.Utime,
	}
	if config.ChannelConfig.Valid {
		domainCfg.ChannelConfig = &config.ChannelConfig.Val
	}
	if config.TxnConfig.Valid {
		domainCfg.TxnConfig = &config.TxnConfig.Val
	}
	if config.Quota.Valid {
		domainCfg.Quota = &config.Quota.Val
	}
	if config.CallbackConfig.Valid {
		domainCfg.CallbackConfig = &config.CallbackConfig.Val
	}
	return domainCfg
}

func (repo *businessConfigRepository) toEntity(config domain.BusinessConfig) dao.BusinessConfig {
	businessConfig := dao.BusinessConfig{
		ID:        config.ID,
		OwnerID:   config.OwnerID,
		OwnerType: config.OwnerType,
		RateLimit: config.RateLimit,
		Ctime:     config.Ctime,
		Utime:     config.Utime,
	}

	if config.ChannelConfig != nil {
		businessConfig.ChannelConfig = sqlx.JSONColumn[domain.ChannelConfig]{
			Val:   *config.ChannelConfig,
			Valid: true,
		}
	}

	if config.TxnConfig != nil {
		businessConfig.TxnConfig = sqlx.JSONColumn[domain.TxnConfig]{
			Val:   *config.TxnConfig,
			Valid: true,
		}
	}

	if config.Quota != nil {
		businessConfig.Quota = sqlx.JSONColumn[domain.QuotaConfig]{
			Val:   *config.Quota,
			Valid: true,
		}
	}

	if config.CallbackConfig != nil {
		businessConfig.CallbackConfig = sqlx.JSONColumn[domain.CallbackConfig]{
			Val:   *config.CallbackConfig,
			Valid: true,
		}
	}

	return businessConfig
}

// NewBusinessConfigRepository 创建业务配置仓库实例
func NewBusinessConfigRepository(
	configDao dao.BusinessConfigDAO,
	localCache cache.ConfigCache,
	redisCache cache.ConfigCache,
) BusinessConfigRepository {
	res := &businessConfigRepository{
		dao:        configDao,
		localCache: localCache,
		redisCache: redisCache,
		logger:     elog.DefaultLogger,
	}
	// 复杂系统里面，启动非常慢，可以考虑开 goroutine
	go func() {
		const preloadTimeout = time.Minute
		ctx, cancel := context.WithTimeout(context.Background(), preloadTimeout)
		defer cancel()
		err := res.LoadCache(ctx)
		if err != nil {
			// 缓存预热失败，你可以中断
			res.logger.Error("缓存预热失败", elog.FieldErr(err))
		}
	}()
	return res
}
