//go:build e2e

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"notification-platform/internal/errs"
	"notification-platform/internal/pkg/sqlx"

	"github.com/ego-component/egorm"
	ca "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"notification-platform/internal/pkg/retry"
	"notification-platform/internal/repository/dao"

	"notification-platform/internal/domain"
	"notification-platform/internal/service/config"
	configIoc "notification-platform/internal/test/integration/ioc/config"
	"notification-platform/internal/test/ioc"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BusinessConfigTestSuite struct {
	suite.Suite
	localCache *ca.Cache
	redisCache *redis.Client
	db         *egorm.Component
	svc        config.BusinessConfigService
}

func (s *BusinessConfigTestSuite) SetupSuite() {
	localCache := ca.New(10*time.Minute, 10*time.Minute)
	s.svc = configIoc.InitConfigService(localCache)
	s.localCache = localCache
	s.redisCache = ioc.InitRedisClient()
	s.db = ioc.InitDBAndTables()
}

func (s *BusinessConfigTestSuite) TearDownTest() {
	s.db.Exec("TRUNCATE TABLE `business_configs`")
}

func (s *BusinessConfigTestSuite) createTestConfig() domain.BusinessConfig {
	return domain.BusinessConfig{
		ID:        5,
		OwnerID:   1001,
		OwnerType: "person",
		ChannelConfig: &domain.ChannelConfig{
			Channels: []domain.ChannelItem{
				{
					Channel:  "SMS",
					Priority: 1,
					Enabled:  true,
				},
				{
					Channel:  "EMAIL",
					Priority: 2,
					Enabled:  true,
				},
			},
		},
		TxnConfig: &domain.TxnConfig{
			ServiceName:  "serviceName",
			InitialDelay: 10,
			RetryPolicy: &retry.Config{
				Type: "fixed",
				FixedInterval: &retry.FixedIntervalConfig{
					Interval:   10,
					MaxRetries: 3,
				},
			},
		},
		RateLimit: 2000,
		Quota: &domain.QuotaConfig{
			Monthly: domain.MonthlyConfig{
				SMS:   100,
				EMAIL: 100,
			},
		},
		CallbackConfig: &domain.CallbackConfig{
			ServiceName: "callbackName",
			RetryPolicy: &retry.Config{
				Type: "fixed",
				FixedInterval: &retry.FixedIntervalConfig{
					Interval:   10,
					MaxRetries: 3,
				},
			},
		},
	}
}

func (s *BusinessConfigTestSuite) createTestConfigList() []domain.BusinessConfig {
	list := []domain.BusinessConfig{
		{
			ID:        10001,
			OwnerID:   1001,
			OwnerType: "person",
			ChannelConfig: &domain.ChannelConfig{
				Channels: []domain.ChannelItem{
					{
						Channel:  "SMS",
						Priority: 1,
						Enabled:  true,
					},
					{
						Channel:  "EMAIL",
						Priority: 2,
						Enabled:  true,
					},
				},
			},
			TxnConfig: &domain.TxnConfig{
				ServiceName:  "serviceName",
				InitialDelay: 10,
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   10,
						MaxRetries: 3,
					},
				},
			},
			RateLimit: 2000,
			Quota: &domain.QuotaConfig{
				Monthly: domain.MonthlyConfig{
					SMS:   100,
					EMAIL: 100,
				},
			},
			CallbackConfig: &domain.CallbackConfig{
				ServiceName: "callbackName",
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   10,
						MaxRetries: 3,
					},
				},
			},
			Ctime: 1744274114000,
			Utime: 1744274114000,
		},
		{
			ID:        10002,
			OwnerID:   1002,
			OwnerType: "person",
			ChannelConfig: &domain.ChannelConfig{
				Channels: []domain.ChannelItem{
					{
						Channel:  "SMS",
						Priority: 1,
						Enabled:  true,
					},
					{
						Channel:  "EMAIL",
						Priority: 2,
						Enabled:  true,
					},
				},
			},
			TxnConfig: &domain.TxnConfig{
				ServiceName:  "serviceName",
				InitialDelay: 10,
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   10,
						MaxRetries: 3,
					},
				},
			},
			RateLimit: 2000,
			Quota: &domain.QuotaConfig{
				Monthly: domain.MonthlyConfig{
					SMS:   100,
					EMAIL: 100,
				},
			},
			CallbackConfig: &domain.CallbackConfig{
				ServiceName: "callbackName",
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   10,
						MaxRetries: 3,
					},
				},
			},
			Ctime: 1744274114000,
			Utime: 1744274114000,
		},
		{
			ID:        10003,
			OwnerID:   1003,
			OwnerType: "person",
			ChannelConfig: &domain.ChannelConfig{
				Channels: []domain.ChannelItem{
					{
						Channel:  "SMS",
						Priority: 1,
						Enabled:  true,
					},
					{
						Channel:  "EMAIL",
						Priority: 2,
						Enabled:  true,
					},
				},
			},
			TxnConfig: &domain.TxnConfig{
				ServiceName:  "serviceName",
				InitialDelay: 10,
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   10,
						MaxRetries: 3,
					},
				},
			},
			RateLimit: 2000,
			Quota: &domain.QuotaConfig{
				Monthly: domain.MonthlyConfig{
					SMS:   100,
					EMAIL: 100,
				},
			},
			Ctime: 1744274114000,
			Utime: 1744274114000,
		},
		{
			ID:        10004,
			OwnerID:   1004,
			OwnerType: "person",
			ChannelConfig: &domain.ChannelConfig{
				Channels: []domain.ChannelItem{
					{
						Channel:  "SMS",
						Priority: 1,
						Enabled:  true,
					},
					{
						Channel:  "EMAIL",
						Priority: 2,
						Enabled:  true,
					},
				},
			},
			TxnConfig: &domain.TxnConfig{
				ServiceName:  "serviceName",
				InitialDelay: 10,
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   10,
						MaxRetries: 3,
					},
				},
			},
			RateLimit: 2000,
			Quota: &domain.QuotaConfig{
				Monthly: domain.MonthlyConfig{
					SMS:   100,
					EMAIL: 100,
				},
			},
			Ctime: 1744274114000,
			Utime: 1744274114000,
		},
	}
	ans := slice.Map(list, func(_ int, src domain.BusinessConfig) dao.BusinessConfig {
		return s.toEntity(src)
	})
	err := s.db.WithContext(context.Background()).Create(&ans).Error
	require.NoError(s.T(), err)
	return list
}

// 创建配置并返回ID
func (s *BusinessConfigTestSuite) createConfigAndGetID(t *testing.T) int64 {
	t.Helper()
	testConfig := s.createTestConfig()
	testConfig.ID = 6
	// 创建配置
	err := s.svc.SaveConfig(t.Context(), testConfig)
	assert.NoError(t, err, "创建业务配置应成功")
	key := fmt.Sprintf("config:%d", testConfig.ID)
	err = s.redisCache.Del(t.Context(), key).Err()
	require.NoError(t, err)
	s.localCache.Delete(key)
	return 6
}

// TestServiceCreate 测试创建业务配置
func (s *BusinessConfigTestSuite) TestServiceSaveConfig() {
	t := s.T()
	ctx := context.Background()
	testcases := []struct {
		name    string
		before  func(t *testing.T)
		req     domain.BusinessConfig
		after   func(t *testing.T)
		wantErr error
	}{
		{
			name:   "新增",
			before: func(_ *testing.T) {},
			req:    s.createTestConfig(),
			after: func(t *testing.T) {
				t.Helper()
				s.checkBusinessConfig(ctx, t, s.createTestConfig())
				// 清理：删除创建的配置
				err := s.svc.Delete(ctx, 5)
				assert.NoError(t, err, "删除业务配置应成功")
				err = s.redisCache.Del(ctx, "config:5").Err()
				assert.NoError(t, err, "删除业务配置应成功")
				s.localCache.Delete("config:5")
			},
		},
		{
			name: "更新",
			before: func(t *testing.T) {
				t.Helper()
				testConfig := s.createTestConfig()
				err := s.svc.SaveConfig(ctx, testConfig)
				assert.NoError(t, err, "创建业务配置应成功")
			},
			req: domain.BusinessConfig{
				ID:        5,
				OwnerID:   1003,
				OwnerType: "person",
				ChannelConfig: &domain.ChannelConfig{
					Channels: []domain.ChannelItem{
						{
							Channel:  "SMS",
							Priority: 2,
							Enabled:  true,
						},
						{
							Channel:  "EMAIL",
							Priority: 3,
							Enabled:  true,
						},
					},
				},
				TxnConfig: &domain.TxnConfig{
					ServiceName:  "newServiceName",
					InitialDelay: 20,
					RetryPolicy: &retry.Config{
						Type: "fixed",
						FixedInterval: &retry.FixedIntervalConfig{
							Interval:   40,
							MaxRetries: 4,
						},
					},
				},
				RateLimit: 6000,
				Quota: &domain.QuotaConfig{
					Monthly: domain.MonthlyConfig{
						SMS:   200,
						EMAIL: 200,
					},
				},
				CallbackConfig: &domain.CallbackConfig{
					ServiceName: "newcallbackName",
					RetryPolicy: &retry.Config{
						Type: "fixed",
						FixedInterval: &retry.FixedIntervalConfig{
							Interval:   10,
							MaxRetries: 3,
						},
					},
				},
			},
			after: func(t *testing.T) {
				t.Helper()
				s.checkBusinessConfig(ctx, t, domain.BusinessConfig{
					ID:        5,
					OwnerID:   1003,
					OwnerType: "person",
					ChannelConfig: &domain.ChannelConfig{
						Channels: []domain.ChannelItem{
							{
								Channel:  "SMS",
								Priority: 2,
								Enabled:  true,
							},
							{
								Channel:  "EMAIL",
								Priority: 3,
								Enabled:  true,
							},
						},
					},
					TxnConfig: &domain.TxnConfig{
						ServiceName:  "newServiceName",
						InitialDelay: 20,
						RetryPolicy: &retry.Config{
							Type: "fixed",
							FixedInterval: &retry.FixedIntervalConfig{
								Interval:   40,
								MaxRetries: 4,
							},
						},
					},
					RateLimit: 6000,
					Quota: &domain.QuotaConfig{
						Monthly: domain.MonthlyConfig{
							SMS:   200,
							EMAIL: 200,
						},
					},
					CallbackConfig: &domain.CallbackConfig{
						ServiceName: "newcallbackName",
						RetryPolicy: &retry.Config{
							Type: "fixed",
							FixedInterval: &retry.FixedIntervalConfig{
								Interval:   10,
								MaxRetries: 3,
							},
						},
					},
				})

				// 清理：删除创建的配置
				err := s.svc.Delete(ctx, 5)
				assert.NoError(t, err, "删除业务配置应成功")
				err = s.redisCache.Del(ctx, "config:5").Err()
				assert.NoError(t, err, "删除业务配置应成功")
				s.localCache.Delete("config:5")
			},
		},
		{
			name:   "id为0",
			before: func(_ *testing.T) {},
			req: domain.BusinessConfig{
				ID: 0,
			},
			after:   func(_ *testing.T) {},
			wantErr: config.ErrIDNotSet,
		},
	}
	for idx := range testcases {
		tc := testcases[idx]
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			err := s.svc.SaveConfig(ctx, tc.req)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
			tc.after(t)
		})
	}
}

// TestBusinessConfigRead 测试读取业务配置
func (s *BusinessConfigTestSuite) TestServiceGetByID() {
	testcases := []struct {
		name       string
		before     func(t *testing.T) int64
		wantConfig domain.BusinessConfig
		wantErr    error
	}{
		{
			name: "成功获取",
			before: func(t *testing.T) int64 {
				t.Helper()
				return s.createConfigAndGetID(t)
			},
			wantConfig: domain.BusinessConfig{
				ID:        6,
				OwnerID:   1001,
				OwnerType: "person",
				ChannelConfig: &domain.ChannelConfig{
					Channels: []domain.ChannelItem{
						{
							Channel:  "SMS",
							Priority: 1,
							Enabled:  true,
						},
						{
							Channel:  "EMAIL",
							Priority: 2,
							Enabled:  true,
						},
					},
				},
				TxnConfig: &domain.TxnConfig{
					ServiceName:  "serviceName",
					InitialDelay: 10,
					RetryPolicy: &retry.Config{
						Type: "fixed",
						FixedInterval: &retry.FixedIntervalConfig{
							Interval:   10,
							MaxRetries: 3,
						},
					},
				},
				RateLimit: 2000,
				Quota: &domain.QuotaConfig{
					Monthly: domain.MonthlyConfig{
						SMS:   100,
						EMAIL: 100,
					},
				},
				CallbackConfig: &domain.CallbackConfig{
					ServiceName: "callbackName",
					RetryPolicy: &retry.Config{
						Type: "fixed",
						FixedInterval: &retry.FixedIntervalConfig{
							Interval:   10,
							MaxRetries: 3,
						},
					},
				},
			},
		},
		{
			name: "未存在的id",
			before: func(_ *testing.T) int64 {
				return 9999
			},
			wantErr: errs.ErrConfigNotFound,
		},
	}
	for idx := range testcases {
		tc := testcases[idx]
		s.T().Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			id := tc.before(t)
			conf, err := s.svc.GetByID(ctx, id)
			require.ErrorIs(t, err, tc.wantErr)
			if err != nil {
				return
			}
			assertBusinessConfig(t, tc.wantConfig, conf)
			s.checkBusinessConfig(ctx, t, conf)
			err = s.svc.Delete(ctx, id)
			assert.NoError(t, err, "删除业务配置应成功")
			key := fmt.Sprintf("config:%d", id)
			err = s.redisCache.Del(t.Context(), key).Err()
			assert.NoError(t, err, "删除业务配置应成功")
			s.localCache.Delete(key)
		})
	}
}

func (s *BusinessConfigTestSuite) TestServiceGetByIDs() {
	t := s.T()
	t.Skip()
	ctx := context.Background()
	defer func() {
		for i := 1; i <= 4; i++ {
			err := s.svc.Delete(ctx, int64(i+100000))
			require.NoError(t, err, "删除配置应成功")

		}
	}()

	// 1. 准备测试数据 - 创建4条测试配置
	configList := s.createTestConfigList()

	// 将config1仅添加到Redis缓存
	key1 := fmt.Sprintf("config:%d", configList[0].ID)
	configJSON, err := json.Marshal(configList[0])
	require.NoError(t, err, "序列化配置应成功")
	err = s.redisCache.Set(ctx, key1, string(configJSON), time.Minute*10).Err()
	require.NoError(t, err)
	// 将config2添加到Redis和本地缓存
	key2 := fmt.Sprintf("config:%d", configList[1].ID)
	configJSON, err = json.Marshal(configList[1])
	require.NoError(t, err, "序列化配置应成功")
	err = s.redisCache.Set(ctx, key2, string(configJSON), time.Minute*10).Err()
	require.NoError(t, err, "添加到Redis缓存应成功")

	ids := []int64{10001, 10002, 10003, 10004}
	configMap, err := s.svc.GetByIDs(ctx, ids)
	require.NoError(t, err, "获取配置应成功")
	require.Len(t, configMap, 4, "应返回4条配置")
	for _, wantConfig := range configList {
		v, ok := configMap[wantConfig.ID]
		require.True(t, ok, "返回结果应包含ID %d的配置", wantConfig.ID)
		assertBusinessConfig(t, wantConfig, v)
	}

	for _, cfg := range configList {
		key := fmt.Sprintf("config:%d", cfg.ID)
		cachedCfg, ok := s.localCache.Get(key)
		require.True(t, ok, "ID %d的配置应在本地缓存中", cfg.ID)
		if ok {
			vv, ok := cachedCfg.(domain.BusinessConfig)
			require.True(t, ok)
			assertBusinessConfig(t, cfg, vv)
		}
	}

	for _, cfg := range configList {
		key := fmt.Sprintf("config:%d", cfg.ID)
		result := s.redisCache.Get(ctx, key)
		require.NoError(t, result.Err(), "ID %d的配置应在Redis缓存中", cfg.ID)
		if result.Err() == nil {
			var redisCfg domain.BusinessConfig
			err := json.Unmarshal([]byte(result.Val()), &redisCfg)
			require.NoError(t, err, "Redis中的配置应能正确反序列化")
			assertBusinessConfig(t, cfg, redisCfg)
		}
	}
}

// TestBusinessConfigDelete 测试删除业务配置
func (s *BusinessConfigTestSuite) TestServiceDelete() {
	testcases := []struct {
		name    string
		id      int64
		before  func(t *testing.T)
		after   func(t *testing.T)
		wantErr error
	}{
		{
			name: "正常删除",
			id:   5,
			before: func(t *testing.T) {
				t.Helper()
				s.createConfigAndGetID(t)
			},
			after: func(t *testing.T) {
				t.Helper()
				_, err := s.svc.GetByID(t.Context(), 5)
				assert.ErrorIs(t, err, errs.ErrConfigNotFound, "应返回配置不存在错误")

				_, ok := s.localCache.Get("config:5")
				require.False(t, ok)
				res := s.redisCache.Get(t.Context(), "config:5")
				assert.Equal(t, redis.Nil, res.Err())
			},
		},
	}
	for idx := range testcases {
		tc := testcases[idx]
		s.T().Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			tc.before(t)
			err := s.svc.Delete(ctx, tc.id)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
			tc.after(t)
		})
	}
}

func TestBusinessConfigService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BusinessConfigTestSuite))
}

func assertBusinessConfig(t *testing.T, wantConfig, actualConfig domain.BusinessConfig) {
	t.Helper()
	require.True(t, actualConfig.Ctime > 0)
	require.True(t, actualConfig.Utime > 0)
	actualConfig.Ctime = 0
	wantConfig.Ctime = 0
	actualConfig.Utime = 0
	wantConfig.Utime = 0
	assert.Equal(t, wantConfig, actualConfig)
}

func (s *BusinessConfigTestSuite) checkBusinessConfig(ctx context.Context, t *testing.T, wantConfig domain.BusinessConfig) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var cfgDao dao.BusinessConfig
	err := s.db.WithContext(ctx).Where("id = ?", wantConfig.ID).First(&cfgDao).Error
	assert.NoError(t, err)
	conf := s.toDomain(cfgDao)

	key := fmt.Sprintf("config:%d", wantConfig.ID)
	v, ok := s.localCache.Get(key)
	require.True(t, ok)
	vv, ok := v.(domain.BusinessConfig)
	require.True(t, ok)
	assertBusinessConfig(t, conf, vv)

	res := s.redisCache.Get(ctx, key)
	require.NoError(t, res.Err())
	var redisConf domain.BusinessConfig
	configStr := res.Val()
	err = json.Unmarshal([]byte(configStr), &redisConf)
	require.NoError(t, err)
	assertBusinessConfig(t, conf, redisConf)
}

func (s *BusinessConfigTestSuite) toDomain(config dao.BusinessConfig) domain.BusinessConfig {
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

func (s *BusinessConfigTestSuite) toEntity(config domain.BusinessConfig) dao.BusinessConfig {
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
