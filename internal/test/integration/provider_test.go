//go:build e2e

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	providersvc "notification-platform/internal/service/provider/manage"
	providerioc "notification-platform/internal/test/integration/ioc/provider"
	testioc "notification-platform/internal/test/ioc"
)

func TestProviderServiceSuite(t *testing.T) {
	suite.Run(t, new(ProviderServiceTestSuite))
}

type ProviderServiceTestSuite struct {
	suite.Suite
	db  *egorm.Component
	svc providersvc.Service
}

func (s *ProviderServiceTestSuite) SetupSuite() {
	s.db = testioc.InitDBAndTables()
	s.svc = providerioc.Init()
}

func (s *ProviderServiceTestSuite) TearDownTest() {
	// 每个测试后清空表数据
	s.db.Exec("TRUNCATE TABLE `providers`")
}

// 创建测试用供应商对象
func (s *ProviderServiceTestSuite) createTestProvider(channel domain.Channel) domain.Provider {
	now := time.Now().UnixNano()
	return domain.Provider{
		Name:             fmt.Sprintf("测试供应商-%d", now),
		Channel:          channel,
		Endpoint:         "https://api.test-provider.com",
		APIKey:           "test-api-key",
		APISecret:        "test-api-secret",
		Weight:           100,
		QPSLimit:         200,
		DailyLimit:       10000,
		AuditCallbackURL: "https://callback.test-provider.com",
		Status:           domain.ProviderStatusActive,
	}
}

func (s *ProviderServiceTestSuite) TestCreate() {
	t := s.T()

	provider := s.createTestProvider(domain.ChannelSMS)

	created, err := s.svc.Create(t.Context(), provider)
	require.NoError(t, err)

	s.assertProvider(t, provider, created)
}

func (s *ProviderServiceTestSuite) TestCreateFailed() {
	t := s.T()

	tests := []struct {
		name          string
		provider      domain.Provider
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name: "名称为空",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelSMS)
				p.Name = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "渠道类型不支持",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelInApp)
				p.Channel = "UNKNOWN"
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "API入口地址为空",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelInApp)
				p.Endpoint = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "API Key为空",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelEmail)
				p.APIKey = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "API Secret为空",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelSMS)
				p.APISecret = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "权重小于等于0",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelSMS)
				p.Weight = 0
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "每秒请求数限制小于等于0",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelEmail)
				p.QPSLimit = 0
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "每日请求数限制小于等于0",
			provider: func() domain.Provider {
				p := s.createTestProvider(domain.ChannelInApp)
				p.DailyLimit = 0
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.svc.Create(t.Context(), tt.provider)
			tt.assertErrFunc(t, err)
		})
	}
}

// 测试更新供应商
func (s *ProviderServiceTestSuite) TestUpdate() {
	t := s.T()

	// 先创建一个供应商
	provider := s.createTestProvider(domain.ChannelSMS)
	created, err := s.svc.Create(t.Context(), provider)
	require.NoError(t, err)

	// 修改供应商信息
	created.Name = "更新后的供应商名称"
	created.Endpoint = "https://new-api.test-provider.com"
	created.APIKey = "new-api-key"
	created.APISecret = "new-api-secret"
	created.Weight = 200
	created.QPSLimit = 300
	created.DailyLimit = 20000
	created.AuditCallbackURL = "https://new-callback.test-provider.com"

	// 更新供应商
	err = s.svc.Update(t.Context(), created)
	require.NoError(t, err)

	// 获取更新后的供应商并验证
	updated, err := s.svc.GetByID(t.Context(), created.ID)
	require.NoError(t, err)

	s.assertProvider(t, created, updated)
}

// 测试更新供应商失败的情况
func (s *ProviderServiceTestSuite) TestUpdateFailed() {
	t := s.T()

	// 先创建一个供应商
	provider := s.createTestProvider(domain.ChannelSMS)
	created, err := s.svc.Create(t.Context(), provider)
	require.NoError(t, err)

	tests := []struct {
		name          string
		provider      domain.Provider
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name: "名称为空",
			provider: func() domain.Provider {
				p := created
				p.Name = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "渠道类型不支持",
			provider: func() domain.Provider {
				p := created
				p.Channel = "UNKNOWN"
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "API入口地址为空",
			provider: func() domain.Provider {
				p := created
				p.Endpoint = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "API Key为空",
			provider: func() domain.Provider {
				p := created
				p.APIKey = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "API Secret为空",
			provider: func() domain.Provider {
				p := created
				p.APISecret = ""
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "权重不能小于等于0",
			provider: func() domain.Provider {
				p := created
				p.Weight = 0
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "每秒请求数限制小于等于0",
			provider: func() domain.Provider {
				p := created
				p.QPSLimit = 0
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "每日请求数限制小于等于0",
			provider: func() domain.Provider {
				p := created
				p.DailyLimit = 0
				return p
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.svc.Update(t.Context(), tt.provider)
			tt.assertErrFunc(t, err)
		})
	}
}

func (s *ProviderServiceTestSuite) TestGetByID() {
	t := s.T()

	provider := s.createTestProvider(domain.ChannelSMS)
	created, err := s.svc.Create(t.Context(), provider)
	require.NoError(t, err)

	found, err := s.svc.GetByID(t.Context(), created.ID)
	require.NoError(t, err)

	s.assertProvider(t, provider, found)
}

func (s *ProviderServiceTestSuite) TestGetByIDFailed() {
	t := s.T()

	tests := []struct {
		name          string
		id            int64
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name: "ID为0",
			id:   0,
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "ID为负数",
			id:   -1,
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name: "ID不存在",
			id:   9999,
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrProviderNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.svc.GetByID(t.Context(), tt.id)
			tt.assertErrFunc(t, err)
		})
	}
}

func (s *ProviderServiceTestSuite) TestGetByChannel() {
	t := s.T()

	tests := []struct {
		name          string
		before        func(t *testing.T) []domain.Provider
		channel       domain.Channel
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name: "单个SMS渠道",
			before: func(t *testing.T) []domain.Provider {
				t.Helper()
				n := 1
				providers := make([]domain.Provider, n)
				for i := 0; i < n; i++ {
					created, err := s.svc.Create(t.Context(), s.createTestProvider(domain.ChannelSMS))
					require.NoError(t, err)
					providers[i] = created
				}
				return providers
			},
			channel:       domain.ChannelSMS,
			assertErrFunc: assert.NoError,
		},
		{
			name: "多个Email渠道",
			before: func(t *testing.T) []domain.Provider {
				t.Helper()
				n := 2
				providers := make([]domain.Provider, n)
				for i := 0; i < n; i++ {
					created, err := s.svc.Create(t.Context(), s.createTestProvider(domain.ChannelEmail))
					require.NoError(t, err)
					providers[i] = created
				}
				return providers
			},
			channel:       domain.ChannelEmail,
			assertErrFunc: assert.NoError,
		},
		{
			name: "零个InApp渠道",
			before: func(t *testing.T) []domain.Provider {
				t.Helper()
				return nil
			},
			channel:       domain.ChannelInApp,
			assertErrFunc: assert.NoError,
		},
		{
			name: "未知渠道",
			before: func(t *testing.T) []domain.Provider {
				t.Helper()
				return nil
			},
			channel: domain.Channel("Unknown"),
			assertErrFunc: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expected := tt.before(t)

			actual, err := s.svc.GetByChannel(t.Context(), tt.channel)
			tt.assertErrFunc(t, err)
			if err != nil {
				return
			}

			assert.ElementsMatch(t, expected, actual)
		})
	}
}

func (s *ProviderServiceTestSuite) assertProvider(t *testing.T, expected, actual domain.Provider) {
	assert.NotZero(t, actual.ID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Channel, actual.Channel)
	assert.Equal(t, expected.Endpoint, actual.Endpoint)
	assert.Equal(t, expected.APIKey, actual.APIKey)
	assert.Equal(t, expected.APISecret, actual.APISecret)
	assert.Equal(t, expected.Weight, actual.Weight)
	assert.Equal(t, expected.QPSLimit, actual.QPSLimit)
	assert.Equal(t, expected.DailyLimit, actual.DailyLimit)
	assert.Equal(t, expected.AuditCallbackURL, actual.AuditCallbackURL)
	assert.Equal(t, expected.Status, actual.Status)
}
