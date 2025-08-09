//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"notification-platform/internal/domain"
	auditevt "notification-platform/internal/event/audit"
	templateweb "notification-platform/internal/handler/template"
	auditmocks "notification-platform/internal/service/audit/mocks"
	providermocks "notification-platform/internal/service/provider/mocks"
	"notification-platform/internal/service/provider/sms/client"
	smsmocks "notification-platform/internal/service/provider/sms/client/mocks"
	"notification-platform/internal/test"
	templateioc "notification-platform/internal/test/integration/ioc/template"
	testioc "notification-platform/internal/test/ioc"
)

const (
	ownerID   = int64(234)
	ownerType = "person"
)

func TestTemplateHandlerTestSuite(t *testing.T) {
	t.Skip()
	suite.Run(t, new(TemplateHandlerTestSuite))
}

type TemplateHandlerTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
}

func (s *TemplateHandlerTestSuite) SetupSuite() {
	s.db = testioc.InitDBAndTables()
}

func (s *TemplateHandlerTestSuite) newService(ctrl *gomock.Controller) (templateSvc *templateioc.Service, providerSvc *providermocks.MockService, auditSvc *auditmocks.MockService, clients map[string]client.Client) {
	mockProviderSvc := providermocks.NewMockService(ctrl)
	mockAuditSvc := auditmocks.NewMockService(ctrl)
	mockClient1 := smsmocks.NewMockClient(ctrl)
	mockClient2 := smsmocks.NewMockClient(ctrl)

	clients = map[string]client.Client{
		"mock-provider-name-1": mockClient1,
		"mock-provider-name-2": mockClient2,
	}
	n := time.Now().UnixNano()
	addr := "localhost:9092"

	s.initTopic(addr, "audit_result_events")

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": addr,
	})
	s.NoError(err)

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  addr,
		"auto.offset.reset":  "earliest",
		"group.id":           fmt.Sprintf("mock-group-%d", n),
		"enable.auto.commit": "false",
	})
	s.NoError(err)

	svc, err := templateioc.Init(mockProviderSvc, mockAuditSvc, clients, producer, consumer, 10, 5*time.Second)
	s.NoError(err)
	return svc, mockProviderSvc, mockAuditSvc, clients
}

func (s *TemplateHandlerTestSuite) initTopic(addr, topic string) {
	// 创建 AdminClient
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": addr,
	})
	s.NoError(err)
	defer adminClient.Close()
	// 设置要创建的主题的配置信息
	numPartitions := 1
	replicationFactor := 1
	// 创建主题
	results, err := adminClient.CreateTopics(
		context.Background(),
		[]kafka.TopicSpecification{
			{
				Topic:             topic,
				NumPartitions:     numPartitions,
				ReplicationFactor: replicationFactor,
			},
		},
	)
	s.NoError(err)
	// 处理创建主题的结果
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() != kafka.ErrTopicAlreadyExists {
			fmt.Printf("创建topic失败 %s: %v\n", result.Topic, result.Error)
		} else {
			fmt.Printf("Topic %s 创建成功\n", result.Topic)
		}
	}
}

func (s *TemplateHandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `channel_templates`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `channel_template_versions`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `channel_template_providers`").Error
	s.NoError(err)
}

func (s *TemplateHandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `channel_templates`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `channel_template_versions`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `channel_template_providers`").Error
	s.NoError(err)
}

func (s *TemplateHandlerTestSuite) newGinServer(handler *templateweb.Handler) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	handler.PublicRoutes(server.Engine)
	return server
}

func (s *TemplateHandlerTestSuite) TestHandler_ListTemplates() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler
		req            templateweb.ListTemplatesReq
		wantCode       int
		wantResp       test.Result[templateweb.ListTemplatesResp]
	}{
		{
			name: "获取成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()

				svc, providerSvc, _, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
					{
						ID:      2,
						Name:    "mock-provider-name-2",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				_, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "list-templates-01",
					Description:  "list-templates-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.ListTemplatesReq{
				OwnerID:   ownerID,
				OwnerType: ownerType,
			},
			wantCode: 200,
			wantResp: test.Result[templateweb.ListTemplatesResp]{
				Data: templateweb.ListTemplatesResp{
					Templates: []templateweb.ChannelTemplate{
						{
							OwnerID:      ownerID,
							OwnerType:    ownerType,
							Name:         "list-templates-01",
							Description:  "list-templates-desc",
							Channel:      domain.ChannelSMS.String(),
							BusinessType: domain.BusinessTypePromotion.ToInt64(),
							Versions: []templateweb.ChannelTemplateVersion{
								{
									Name:        "版本名称，比如v1.0.0",
									Signature:   "提前配置好的可用的短信签名或者Email收件人",
									Content:     "模版变量使用${code}格式，也可以没有变量",
									Remark:      "模版使用场景或者用途说明，有利于供应商审核通过",
									AuditStatus: domain.AuditStatusPending.String(),
									Providers: []templateweb.ChannelTemplateProvider{
										{
											ProviderID:      1,
											ProviderName:    "mock-provider-name-1",
											ProviderChannel: domain.ChannelSMS.String(),
											AuditStatus:     domain.AuditStatusPending.String(),
										},
										{
											ProviderID:      2,
											ProviderName:    "mock-provider-name-2",
											ProviderChannel: domain.ChannelSMS.String(),
											AuditStatus:     domain.AuditStatusPending.String(),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/templates/list", iox.NewJSONReader(tc.req))

			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[templateweb.ListTemplatesResp]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))

			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)

			actual := recorder.MustScan()

			assert.Equal(t, len(tc.wantResp.Data.Templates), len(actual.Data.Templates))
			for i := range tc.wantResp.Data.Templates {
				s.assertTemplate(t, tc.wantResp.Data.Templates[i], actual.Data.Templates[i])
			}
		})
	}
}

func (s *TemplateHandlerTestSuite) TestHandler_CreateTemplate() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler
		req            templateweb.CreateTemplateReq
		wantCode       int
		wantResp       test.Result[templateweb.CreateTemplateResp]
	}{
		{
			name: "创建成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()

				svc, providerSvc, _, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.CreateTemplateReq{
				OwnerID:      ownerID,
				OwnerType:    ownerType,
				Name:         "create-template-test",
				Description:  "create-template-desc",
				Channel:      domain.ChannelSMS.String(),
				BusinessType: domain.BusinessTypePromotion.ToInt64(),
			},
			wantCode: 200,
			wantResp: test.Result[templateweb.CreateTemplateResp]{
				Data: templateweb.CreateTemplateResp{
					Template: templateweb.ChannelTemplate{
						OwnerID:      ownerID,
						OwnerType:    ownerType,
						Name:         "create-template-test",
						Description:  "create-template-desc",
						Channel:      domain.ChannelSMS.String(),
						BusinessType: domain.BusinessTypePromotion.ToInt64(),
						Versions: []templateweb.ChannelTemplateVersion{
							{
								Name:        "版本名称，比如v1.0.0",
								Signature:   "提前配置好的可用的短信签名或者Email收件人",
								Content:     "模版变量使用${code}格式，也可以没有变量",
								Remark:      "模版使用场景或者用途说明，有利于供应商审核通过",
								AuditStatus: domain.AuditStatusPending.String(),
								Providers: []templateweb.ChannelTemplateProvider{
									{
										ProviderID:      1,
										ProviderName:    "mock-provider-name-1",
										ProviderChannel: domain.ChannelSMS.String(),
										AuditStatus:     domain.AuditStatusPending.String(),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/templates/create", iox.NewJSONReader(tc.req))

			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[templateweb.CreateTemplateResp]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))

			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)

			actual := recorder.MustScan()

			s.assertTemplate(t, tc.wantResp.Data.Template, actual.Data.Template)
		})
	}
}

func (s *TemplateHandlerTestSuite) TestHandler_UpdateTemplate() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler
		req            templateweb.UpdateTemplateReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "更新模板成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()

				svc, providerSvc, _, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				_, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "original-name",
					Description:  "original-description",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.UpdateTemplateReq{
				TemplateID:   1, // 这里使用1，因为第一个创建的模板ID是1
				Name:         "updated-name",
				Description:  "updated-description",
				BusinessType: domain.BusinessTypeNotification.ToInt64(),
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},
		{
			name: "更新不存在的模板",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.UpdateTemplateReq{
				TemplateID:   9999, // 使用一个不存在的ID
				Name:         "updated-name",
				Description:  "updated-description",
				BusinessType: domain.BusinessTypeNotification.ToInt64(),
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/templates/update", iox.NewJSONReader(tc.req))

			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[any]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))

			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)

			actual := recorder.MustScan()
			assert.Equal(t, tc.wantResp.Code, actual.Code)
			assert.Equal(t, tc.wantResp.Msg, actual.Msg)
		})
	}
}

func (s *TemplateHandlerTestSuite) assertTemplate(t *testing.T, expected templateweb.ChannelTemplate, actual templateweb.ChannelTemplate) {
	assert.NotZero(t, actual.ID)
	assert.Equal(t, expected.OwnerID, actual.OwnerID)
	assert.Equal(t, expected.OwnerType, actual.OwnerType)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Channel, actual.Channel)
	assert.Equal(t, expected.BusinessType, actual.BusinessType)
	assert.Equal(t, expected.ActiveVersionID, actual.ActiveVersionID)
	assert.NotZero(t, actual.Ctime)
	assert.NotZero(t, actual.Utime)

	assert.Equal(t, len(expected.Versions), len(actual.Versions))
	for i := range expected.Versions {
		s.assertTemplateVersion(t, expected.Versions[i], actual.Versions[i])
	}
}

func (s *TemplateHandlerTestSuite) assertTemplateVersion(t *testing.T, expected templateweb.ChannelTemplateVersion, actual templateweb.ChannelTemplateVersion) {
	assert.NotZero(t, actual.ID)
	assert.NotZero(t, actual.ChannelTemplateID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Signature, actual.Signature)
	assert.Equal(t, expected.Content, actual.Content)
	assert.Equal(t, expected.Remark, actual.Remark)
	assert.Equal(t, expected.AuditID, actual.AuditID)
	assert.Equal(t, expected.AuditorID, actual.AuditorID)
	assert.Equal(t, expected.AuditTime, actual.AuditTime)
	assert.Equal(t, expected.AuditStatus, actual.AuditStatus)
	assert.Equal(t, expected.RejectReason, actual.RejectReason)
	assert.Equal(t, expected.LastReviewSubmissionTime, actual.LastReviewSubmissionTime)
	assert.NotZero(t, actual.Ctime)
	assert.NotZero(t, actual.Utime)

	assert.Equal(t, len(expected.Providers), len(actual.Providers))
	for i := range expected.Providers {
		s.assertTemplateProvider(t, expected.Providers[i], actual.Providers[i])
	}
}

func (s *TemplateHandlerTestSuite) assertTemplateProvider(t *testing.T, expected templateweb.ChannelTemplateProvider, actual templateweb.ChannelTemplateProvider) {
	assert.NotZero(t, actual.ID)
	assert.NotZero(t, actual.TemplateID)
	assert.NotZero(t, actual.TemplateVersionID)
	assert.Equal(t, expected.ProviderID, actual.ProviderID)
	assert.Equal(t, expected.ProviderName, actual.ProviderName)
	assert.Equal(t, expected.ProviderChannel, actual.ProviderChannel)
	assert.Equal(t, expected.RequestID, actual.RequestID)
	assert.Equal(t, expected.ProviderTemplateID, actual.ProviderTemplateID)
	assert.Equal(t, expected.AuditStatus, actual.AuditStatus)
	assert.Equal(t, expected.RejectReason, actual.RejectReason)
	assert.Equal(t, expected.LastReviewSubmissionTime, actual.LastReviewSubmissionTime)
	assert.NotZero(t, actual.Ctime)
	assert.NotZero(t, actual.Utime)
}

func (s *TemplateHandlerTestSuite) TestService_GetPendingOrInReviewProviders() {
	t := s.T()

	// 测试用例
	testCases := []struct {
		name      string
		setupMock func(t *testing.T) (*templateioc.Service, int64)
		offset    int
		limit     int
		expected  struct {
			count     int64
			providers []int64
		}
		errAssertFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, providers []domain.ChannelTemplateProvider, total int64)
	}{
		{
			name: "获取待审核供应商",
			setupMock: func(t *testing.T) (*templateioc.Service, int64) {
				t.Helper()

				// 创建模板和供应商
				svc, providerSvc, _, _ := s.newService(gomock.NewController(t))

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "pending-template",
					Description:  "pending description",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 1)

				// 设置供应商状态为PENDING
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusPending

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 返回未来时间点作为查询时间，确保能查到
				return svc, time.Now().Add(time.Hour).UnixMilli()
			},
			offset: 0,
			limit:  10,
			expected: struct {
				count     int64
				providers []int64
			}{
				count:     1,
				providers: []int64{}, // 会在after中验证
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider, total int64) {
				assert.Equal(t, int64(1), total, "应该有1个供应商")
				assert.Len(t, providers, 1, "应该返回1个供应商")
				assert.Equal(t, domain.AuditStatusPending, providers[0].AuditStatus, "供应商状态应该是PENDING")
			},
		},
		{
			name: "获取审核中供应商",
			setupMock: func(t *testing.T) (*templateioc.Service, int64) {
				t.Helper()

				// 创建模板和供应商
				svc, providerSvc, _, _ := s.newService(gomock.NewController(t))

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "inreview-template",
					Description:  "in review description",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 1)

				// 设置供应商状态为IN_REVIEW
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusInReview

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 返回未来时间点作为查询时间，确保能查到
				return svc, time.Now().Add(time.Hour).UnixMilli()
			},
			offset: 0,
			limit:  10,
			expected: struct {
				count     int64
				providers []int64
			}{
				count:     1,
				providers: []int64{}, // 会在after中验证
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider, total int64) {
				assert.Equal(t, int64(1), total, "应该有1个供应商")
				assert.Len(t, providers, 1, "应该返回1个供应商")
				assert.Equal(t, domain.AuditStatusInReview, providers[0].AuditStatus, "供应商状态应该是IN_REVIEW")
			},
		},
		{
			name: "时间范围限制 - 查不到供应商",
			setupMock: func(t *testing.T) (*templateioc.Service, int64) {
				t.Helper()

				// 创建模板和供应商
				svc, providerSvc, _, _ := s.newService(gomock.NewController(t))

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "timefilter-template",
					Description:  "time filter description",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 1)

				// 设置供应商状态为PENDING
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusPending

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 返回过去时间点作为查询时间，应该查不到刚创建的记录
				return svc, time.Now().Add(-24 * time.Hour).UnixMilli()
			},
			offset: 0,
			limit:  10,
			expected: struct {
				count     int64
				providers []int64
			}{
				count:     0,
				providers: []int64{},
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider, total int64) {
				assert.Equal(t, int64(0), total, "不应该有供应商")
				assert.Len(t, providers, 0, "不应该返回供应商")
			},
		},
		{
			name: "分页 - 偏移量测试",
			setupMock: func(t *testing.T) (*templateioc.Service, int64) {
				t.Helper()

				// 创建模板和供应商
				svc, providerSvc, _, _ := s.newService(gomock.NewController(t))

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil).Times(5) // 创建5个模板

				// 创建5个模板，每个模板有1个供应商
				for i := 0; i < 5; i++ {
					template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
						OwnerID:      ownerID,
						OwnerType:    ownerType,
						Name:         fmt.Sprintf("pagination-template-%d", i),
						Description:  fmt.Sprintf("pagination description %d", i),
						Channel:      domain.ChannelSMS,
						BusinessType: domain.BusinessTypePromotion,
					})
					require.NoError(t, err)

					// 获取模板版本和供应商
					templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
					require.NoError(t, err)
					require.Len(t, templateFromDB.Versions, 1)
					require.Len(t, templateFromDB.Versions[0].Providers, 1)

					// 设置供应商状态为PENDING
					providers := templateFromDB.Versions[0].Providers
					providers[0].AuditStatus = domain.AuditStatusPending

					// 更新供应商状态
					err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
					require.NoError(t, err)
				}

				// 返回未来时间点作为查询时间，确保能查到
				return svc, time.Now().Add(time.Hour).UnixMilli()
			},
			offset: 2, // 从第3个开始
			limit:  2, // 只获取2个
			expected: struct {
				count     int64
				providers []int64
			}{
				count:     5,         // 总共有5个
				providers: []int64{}, // 会在after中验证
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider, total int64) {
				assert.Equal(t, int64(5), total, "总共应该有5个供应商")
				assert.Len(t, providers, 2, "应该返回2个供应商") // 验证分页 - 只返回了2个
				for _, provider := range providers {
					assert.Equal(t, domain.AuditStatusPending, provider.AuditStatus, "供应商状态应该是PENDING")
				}
			},
		},
		{
			name: "供应商状态混合测试",
			setupMock: func(t *testing.T) (*templateioc.Service, int64) {
				t.Helper()

				// 创建模板和供应商
				svc, providerSvc, _, _ := s.newService(gomock.NewController(t))

				// 设置模拟数据 - 2个供应商
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
					{
						ID:      2,
						Name:    "mock-provider-name-2",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "mixed-status-template",
					Description:  "mixed status description",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 2)

				// 设置供应商状态 - 一个PENDING，一个IN_REVIEW
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusPending
				providers[1].AuditStatus = domain.AuditStatusInReview

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 创建另一个模板，设置供应商状态为已审核通过和拒绝（不应该被查询出来）
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
					{
						ID:      2,
						Name:    "mock-provider-name-2",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				template2, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "approved-rejected-template",
					Description:  "approved and rejected description",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				template2FromDB, err := svc.Svc.GetTemplateByID(t.Context(), template2.ID)
				require.NoError(t, err)
				require.Len(t, template2FromDB.Versions, 1)
				require.Len(t, template2FromDB.Versions[0].Providers, 2)

				// 设置供应商状态 - 一个APPROVED，一个REJECTED
				providers2 := template2FromDB.Versions[0].Providers
				providers2[0].AuditStatus = domain.AuditStatusApproved
				providers2[1].AuditStatus = domain.AuditStatusRejected

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers2)
				require.NoError(t, err)

				// 返回未来时间点作为查询时间，确保能查到
				return svc, time.Now().Add(time.Hour).UnixMilli()
			},
			offset: 0,
			limit:  10,
			expected: struct {
				count     int64
				providers []int64
			}{
				count:     2,         // 总共有2个符合条件的供应商
				providers: []int64{}, // 会在after中验证
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider, total int64) {
				assert.Equal(t, int64(2), total, "总共应该有2个供应商")
				assert.Len(t, providers, 2, "应该返回2个供应商")

				// 验证状态
				var pendingCount, inReviewCount int
				for _, provider := range providers {
					switch provider.AuditStatus {
					case domain.AuditStatusPending:
						pendingCount++
					case domain.AuditStatusInReview:
						inReviewCount++
					}
				}
				assert.Equal(t, 1, pendingCount, "应该有1个待审核供应商")
				assert.Equal(t, 1, inReviewCount, "应该有1个审核中供应商")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置模拟数据并获取查询时间
			svc, utime := tc.setupMock(t)

			// 执行测试
			providers, total, err := svc.Svc.GetPendingOrInReviewProviders(t.Context(), tc.offset, tc.limit, utime)

			// 验证结果
			tc.errAssertFunc(t, err)
			if err == nil {
				return
			}

			tc.after(t, providers, total)
		})
	}
}

func (s *TemplateHandlerTestSuite) TestService_BatchQueryAndUpdateProviderAuditInfo() {
	t := s.T()

	testCases := []struct {
		name          string
		setupMock     func(t *testing.T) (*templateioc.Service, []domain.ChannelTemplateProvider)
		errAssertFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, providers []domain.ChannelTemplateProvider)
	}{
		{
			name: "更新SMS供应商审核状态-通过",
			setupMock: func(t *testing.T) (*templateioc.Service, []domain.ChannelTemplateProvider) {
				t.Helper()

				// 创建服务实例和模拟客户端
				svc, providerSvc, _, clients := s.newService(gomock.NewController(t))
				mockClient := clients["mock-provider-name-1"].(*smsmocks.MockClient)

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "sms-approved-template",
					Description:  "SMS approved template",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 1)

				// 设置供应商状态为IN_REVIEW并设置模板ID
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusInReview
				providers[0].ProviderTemplateID = "template-id-1"

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 模拟供应商客户端响应 - 返回审核通过状态
				mockClient.EXPECT().
					BatchQueryTemplateStatus(gomock.Any()).
					DoAndReturn(func(req client.BatchQueryTemplateStatusReq) (client.BatchQueryTemplateStatusResp, error) {
						// 验证请求中包含正确的模板ID
						require.Contains(t, req.TemplateIDs, "template-id-1")

						// 返回审核通过的结果
						return client.BatchQueryTemplateStatusResp{
							Results: map[string]client.QueryTemplateStatusResp{
								"template-id-1": {
									RequestID:   "request-id-1",
									TemplateID:  "template-id-1",
									AuditStatus: client.AuditStatusApproved,
									Reason:      "",
								},
							},
						}, nil
					})

				return svc, providers
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider) {
				// 获取最新的供应商状态
				svc, _, _, _ := s.newService(gomock.NewController(t))

				// 验证供应商状态已更新为APPROVED
				updatedProviders, err := svc.Repo.GetProvidersByTemplateIDAndVersionID(t.Context(), providers[0].TemplateID, providers[0].TemplateVersionID)
				require.NoError(t, err)
				assert.Len(t, updatedProviders, 1)
				assert.Equal(t, domain.AuditStatusApproved, updatedProviders[0].AuditStatus)
				assert.Empty(t, updatedProviders[0].RejectReason)
			},
		},
		{
			name: "更新SMS供应商审核状态-拒绝",
			setupMock: func(t *testing.T) (*templateioc.Service, []domain.ChannelTemplateProvider) {
				t.Helper()

				// 创建服务实例和模拟客户端
				svc, providerSvc, _, clients := s.newService(gomock.NewController(t))
				mockClient := clients["mock-provider-name-1"].(*smsmocks.MockClient)

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "sms-rejected-template",
					Description:  "SMS rejected template",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 1)

				// 设置供应商状态为IN_REVIEW并设置模板ID
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusInReview
				providers[0].ProviderTemplateID = "template-id-2"

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 模拟供应商客户端响应 - 返回审核拒绝状态
				mockClient.EXPECT().
					BatchQueryTemplateStatus(gomock.Any()).
					DoAndReturn(func(req client.BatchQueryTemplateStatusReq) (client.BatchQueryTemplateStatusResp, error) {
						// 验证请求中包含正确的模板ID
						require.Contains(t, req.TemplateIDs, "template-id-2")

						// 返回审核拒绝的结果
						return client.BatchQueryTemplateStatusResp{
							Results: map[string]client.QueryTemplateStatusResp{
								"template-id-2": {
									RequestID:   "request-id-2",
									TemplateID:  "template-id-2",
									AuditStatus: client.AuditStatusRejected,
									Reason:      "内容不合规",
								},
							},
						}, nil
					})

				return svc, providers
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider) {
				// 获取最新的供应商状态
				svc, _, _, _ := s.newService(gomock.NewController(t))

				// 验证供应商状态已更新为REJECTED
				updatedProviders, err := svc.Repo.GetProvidersByTemplateIDAndVersionID(t.Context(), providers[0].TemplateID, providers[0].TemplateVersionID)
				require.NoError(t, err)
				assert.Len(t, updatedProviders, 1)
				assert.Equal(t, domain.AuditStatusRejected, updatedProviders[0].AuditStatus)
				assert.Equal(t, "内容不合规", updatedProviders[0].RejectReason)
			},
		},
		{
			name: "更新SMS供应商审核状态-审核中",
			setupMock: func(t *testing.T) (*templateioc.Service, []domain.ChannelTemplateProvider) {
				t.Helper()

				// 创建服务实例和模拟客户端
				svc, providerSvc, _, clients := s.newService(gomock.NewController(t))
				mockClient := clients["mock-provider-name-1"].(*smsmocks.MockClient)

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "sms-inreview-template",
					Description:  "SMS in review template",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 1)

				// 设置供应商状态为IN_REVIEW并设置模板ID
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusInReview
				providers[0].ProviderTemplateID = "template-id-3"

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 模拟供应商客户端响应 - 返回仍在审核中状态
				mockClient.EXPECT().
					BatchQueryTemplateStatus(gomock.Any()).
					DoAndReturn(func(req client.BatchQueryTemplateStatusReq) (client.BatchQueryTemplateStatusResp, error) {
						// 验证请求中包含正确的模板ID
						require.Contains(t, req.TemplateIDs, "template-id-3")

						// 返回仍在审核中的结果
						return client.BatchQueryTemplateStatusResp{
							Results: map[string]client.QueryTemplateStatusResp{
								"template-id-3": {
									RequestID:   "request-id-3",
									TemplateID:  "template-id-3",
									AuditStatus: client.AuditStatusPending,
									Reason:      "",
								},
							},
						}, nil
					})

				return svc, providers
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider) {
				// 获取最新的供应商状态
				svc, _, _, _ := s.newService(gomock.NewController(t))

				// 验证供应商状态仍然是IN_REVIEW
				updatedProviders, err := svc.Repo.GetProvidersByTemplateIDAndVersionID(t.Context(), providers[0].TemplateID, providers[0].TemplateVersionID)
				require.NoError(t, err)
				assert.Len(t, updatedProviders, 1)
				assert.Equal(t, domain.AuditStatusInReview, updatedProviders[0].AuditStatus)
				assert.Empty(t, updatedProviders[0].RejectReason)
			},
		},
		{
			name: "多个供应商混合状态更新",
			setupMock: func(t *testing.T) (*templateioc.Service, []domain.ChannelTemplateProvider) {
				t.Helper()

				// 创建服务实例和模拟客户端
				svc, providerSvc, _, clients := s.newService(gomock.NewController(t))
				mockClient1 := clients["mock-provider-name-1"].(*smsmocks.MockClient)
				mockClient2 := clients["mock-provider-name-2"].(*smsmocks.MockClient)

				// 设置模拟数据 - 两个不同的SMS供应商
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
					{
						ID:      2,
						Name:    "mock-provider-name-2",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "mixed-providers-template",
					Description:  "Mixed providers template",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 2)

				// 设置供应商状态为IN_REVIEW并设置模板ID
				providers := templateFromDB.Versions[0].Providers

				// 确保供应商名称正确，这样可以匹配到对应的mockClient
				for i := range providers {
					if providers[i].ProviderName == "mock-provider-name-1" {
						providers[i].AuditStatus = domain.AuditStatusInReview
						providers[i].ProviderTemplateID = "provider1-template-id"
					} else if providers[i].ProviderName == "mock-provider-name-2" {
						providers[i].AuditStatus = domain.AuditStatusInReview
						providers[i].ProviderTemplateID = "provider2-template-id"
					}
				}

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 模拟第一个供应商客户端响应 - 返回审核通过状态
				mockClient1.EXPECT().
					BatchQueryTemplateStatus(gomock.Any()).
					DoAndReturn(func(req client.BatchQueryTemplateStatusReq) (client.BatchQueryTemplateStatusResp, error) {
						return client.BatchQueryTemplateStatusResp{
							Results: map[string]client.QueryTemplateStatusResp{
								"provider1-template-id": {
									RequestID:   "request-id-1",
									TemplateID:  "provider1-template-id",
									AuditStatus: client.AuditStatusApproved,
									Reason:      "",
								},
							},
						}, nil
					})

				// 模拟第二个供应商客户端响应 - 返回审核拒绝状态
				mockClient2.EXPECT().
					BatchQueryTemplateStatus(gomock.Any()).
					DoAndReturn(func(req client.BatchQueryTemplateStatusReq) (client.BatchQueryTemplateStatusResp, error) {
						return client.BatchQueryTemplateStatusResp{
							Results: map[string]client.QueryTemplateStatusResp{
								"provider2-template-id": {
									RequestID:   "request-id-2",
									TemplateID:  "provider2-template-id",
									AuditStatus: client.AuditStatusRejected,
									Reason:      "内容不合规",
								},
							},
						}, nil
					})

				return svc, providers
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, providers []domain.ChannelTemplateProvider) {
				// 获取最新的供应商状态
				svc, _, _, _ := s.newService(gomock.NewController(t))
				for _, provider := range providers {
					// 验证供应商状态仍然是IN_REVIEW
					updatedProviders, err := svc.Repo.GetProvidersByTemplateIDAndVersionID(t.Context(), provider.TemplateID, provider.TemplateVersionID)
					assert.NoError(t, err)
					for i := range updatedProviders {
						updatedProvider := updatedProviders[i]
						if updatedProvider.ProviderName == "mock-provider-name-1" {
							// 第一个供应商应该被更新为审核通过
							assert.Equal(t, domain.AuditStatusApproved, updatedProvider.AuditStatus)
							assert.Empty(t, updatedProvider.RejectReason)
						} else if updatedProvider.ProviderName == "mock-provider-name-2" {
							// 第二个供应商应该被更新为审核拒绝
							assert.Equal(t, domain.AuditStatusRejected, updatedProvider.AuditStatus)
							assert.Equal(t, "内容不合规", updatedProvider.RejectReason)
						}
					}
				}
			},
		},
		{
			name: "客户端返回错误",
			setupMock: func(t *testing.T) (*templateioc.Service, []domain.ChannelTemplateProvider) {
				t.Helper()

				// 创建服务实例和模拟客户端
				svc, providerSvc, _, clients := s.newService(gomock.NewController(t))
				mockClient := clients["mock-provider-name-1"].(*smsmocks.MockClient)

				// 设置模拟数据
				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "error-template",
					Description:  "Error response template",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本和供应商
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)
				require.Len(t, templateFromDB.Versions[0].Providers, 1)

				// 设置供应商状态为IN_REVIEW并设置模板ID
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusInReview
				providers[0].ProviderTemplateID = "error-template-id"

				// 更新供应商状态
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 模拟供应商客户端响应 - 返回错误
				mockClient.EXPECT().
					BatchQueryTemplateStatus(gomock.Any()).
					Return(client.BatchQueryTemplateStatusResp{}, fmt.Errorf("模拟客户端错误"))

				return svc, providers
			},
			errAssertFunc: assert.Error,
			after:         func(t *testing.T, providers []domain.ChannelTemplateProvider) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置模拟数据
			svc, providers := tc.setupMock(t)

			// 执行测试
			err := svc.Svc.BatchQueryAndUpdateProviderAuditInfo(t.Context(), providers)

			// 验证结果
			tc.errAssertFunc(t, err)

			if err != nil {
				return
			}

			// 执行后置检查
			tc.after(t, providers)
		})
	}
}

func (s *TemplateHandlerTestSuite) TestHandler_PublishTemplate() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler
		req            templateweb.PublishTemplateReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "发布模板成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()

				svc, providerSvc, auditSvc, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "publish-template",
					Description:  "publish-template-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)

				// 设置供应商审核状态为通过
				providers := templateFromDB.Versions[0].Providers
				providers[0].AuditStatus = domain.AuditStatusApproved
				err = svc.Repo.BatchUpdateTemplateProvidersAuditInfo(t.Context(), providers)
				require.NoError(t, err)

				// 模拟审核服务
				auditSvc.EXPECT().CreateAudit(gomock.Any(), gomock.Any()).Return(1, nil)
				// 提交内部审核并设置为通过
				err = svc.Svc.SubmitForInternalReview(t.Context(), templateFromDB.Versions[0].ID)
				require.NoError(t, err)

				// 在数据库中更新版本状态为内部审核通过
				version := templateFromDB.Versions[0]
				version.AuditStatus = domain.AuditStatusApproved
				err = svc.Repo.BatchUpdateTemplateVersionAuditInfo(t.Context(), []domain.ChannelTemplateVersion{version})
				require.NoError(t, err)

				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.PublishTemplateReq{
				TemplateID: 1, // 第一个创建的模板ID
				VersionID:  1, // 第一个版本ID
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},
		{
			name: "发布不存在的模板",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.PublishTemplateReq{
				TemplateID: 9999, // 不存在的模板ID
				VersionID:  9999, // 不存在的版本ID
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 506001,
				Msg:  "系统错误",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/templates/publish", iox.NewJSONReader(tc.req))

			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[any]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))

			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)

			actual := recorder.MustScan()
			assert.Equal(t, tc.wantResp.Code, actual.Code)
			assert.Equal(t, tc.wantResp.Msg, actual.Msg)
		})
	}
}

func (s *TemplateHandlerTestSuite) TestHandler_ForkVersion() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler
		req            templateweb.ForkVersionReq
		wantCode       int
		wantResp       test.Result[templateweb.ForkVersionResp]
	}{
		{
			name: "分支版本成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()

				svc, providerSvc, _, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "fork-template",
					Description:  "fork-template-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)

				// 对原版本进行更新，添加一些内容
				version := templateFromDB.Versions[0]
				version.Content = "这是原始版本的内容"
				version.Signature = "原始签名"
				version.Remark = "原始备注"
				err = svc.Repo.UpdateTemplateVersion(t.Context(), version)
				require.NoError(t, err)

				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.ForkVersionReq{
				VersionID: 1, // 第一个版本ID
			},
			wantCode: 200,
			wantResp: test.Result[templateweb.ForkVersionResp]{
				Data: templateweb.ForkVersionResp{
					TemplateVersion: templateweb.ChannelTemplateVersion{
						Name:        "版本名称，比如v1.0.0",
						Signature:   "原始签名",
						Content:     "这是原始版本的内容",
						Remark:      "原始备注",
						AuditStatus: domain.AuditStatusPending.String(),
						Providers: []templateweb.ChannelTemplateProvider{
							{
								ProviderID:      1,
								ProviderName:    "mock-provider-name-1",
								ProviderChannel: domain.ChannelSMS.String(),
								AuditStatus:     domain.AuditStatusPending.String(),
							},
						},
					},
				},
			},
		},
		{
			name: "分支不存在的版本",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.ForkVersionReq{
				VersionID: 9999, // 不存在的版本ID
			},
			wantCode: 500,
			wantResp: test.Result[templateweb.ForkVersionResp]{
				Code: 506001,
				Msg:  "系统错误",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/templates/versions/fork", iox.NewJSONReader(tc.req))

			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[templateweb.ForkVersionResp]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))

			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)

			actual := recorder.MustScan()

			if tc.wantResp.Code != 0 {
				assert.Equal(t, tc.wantResp.Code, actual.Code)
				assert.Equal(t, tc.wantResp.Msg, actual.Msg)
			} else if actual.Code == 0 {
				// 成功情况下，验证响应内容
				assert.NotNil(t, actual.Data)
				assert.NotZero(t, actual.Data.TemplateVersion.ID)
				assert.Equal(t, tc.wantResp.Data.TemplateVersion.Signature, actual.Data.TemplateVersion.Signature)
				assert.Equal(t, tc.wantResp.Data.TemplateVersion.Content, actual.Data.TemplateVersion.Content)
				assert.Equal(t, tc.wantResp.Data.TemplateVersion.Remark, actual.Data.TemplateVersion.Remark)
				assert.Equal(t, tc.wantResp.Data.TemplateVersion.AuditStatus, actual.Data.TemplateVersion.AuditStatus)
				assert.Equal(t, len(tc.wantResp.Data.TemplateVersion.Providers), len(actual.Data.TemplateVersion.Providers))
			}
		})
	}
}

func (s *TemplateHandlerTestSuite) TestHandler_UpdateVersion() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler
		req            templateweb.UpdateVersionReq
		wantCode       int
		wantResp       test.Result[any]
		after          func(t *testing.T, expected templateweb.UpdateVersionReq, ctrl *gomock.Controller)
	}{
		{
			name: "更新版本成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()

				svc, providerSvc, _, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "update-version-template",
					Description:  "update-version-template-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)

				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.UpdateVersionReq{
				VersionID: 1, // 第一个版本ID
				Name:      "更新后的版本名称",
				Signature: "更新后的签名",
				Content:   "更新后的内容${code}",
				Remark:    "更新后的备注信息",
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, expected templateweb.UpdateVersionReq, ctrl *gomock.Controller) {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				version, err := svc.Repo.GetTemplateVersionByID(t.Context(), expected.VersionID)
				require.NoError(t, err)
				assert.Equal(t, expected.Name, version.Name)
				assert.Equal(t, expected.Signature, version.Signature)
				assert.Equal(t, expected.Content, version.Content)
				assert.Equal(t, expected.Remark, version.Remark)
			},
		},
		{
			name: "更新不存在的版本",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.UpdateVersionReq{
				VersionID: 9999, // 不存在的版本ID
				Name:      "更新后的版本名称",
				Signature: "更新后的签名",
				Content:   "更新后的内容${code}",
				Remark:    "更新后的备注信息",
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, expected templateweb.UpdateVersionReq, ctrl *gomock.Controller) {},
		},
		{
			name: "更新已审核通过的版本",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *templateweb.Handler {
				t.Helper()

				svc, providerSvc, _, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "approved-version-template",
					Description:  "approved-version-template-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)

				// 设置版本状态为已审核通过
				version := templateFromDB.Versions[0]
				version.AuditStatus = domain.AuditStatusApproved
				err = svc.Repo.BatchUpdateTemplateVersionAuditInfo(t.Context(), []domain.ChannelTemplateVersion{version})
				require.NoError(t, err)

				handler := templateweb.NewHandler(svc.Svc)
				return handler
			},
			req: templateweb.UpdateVersionReq{
				VersionID: 1, // 第一个版本ID
				Name:      "更新已审核版本",
				Signature: "更新后的签名",
				Content:   "更新后的内容${code}",
				Remark:    "更新后的备注信息",
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, expected templateweb.UpdateVersionReq, ctrl *gomock.Controller) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/templates/versions/update", iox.NewJSONReader(tc.req))

			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[any]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))

			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)

			actual := recorder.MustScan()
			assert.Equal(t, tc.wantResp.Code, actual.Code)
			assert.Equal(t, tc.wantResp.Msg, actual.Msg)

			// 如果更新成功，验证版本是否真的更新了
			if tc.wantResp.Code == 0 {
				tc.after(t, tc.req, ctrl)
			}
		})
	}
}

func (s *TemplateHandlerTestSuite) TestHandler_SubmitForInternalReview() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) (*templateweb.Handler, int64)
		req            templateweb.SubmitForInternalReviewReq
		wantCode       int
		wantResp       test.Result[any]
		after          func(t *testing.T, expected templateweb.SubmitForInternalReviewReq, ctrl *gomock.Controller)
	}{
		{
			name: "提交内部审核成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) (*templateweb.Handler, int64) {
				t.Helper()

				svc, providerSvc, auditSvc, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "internal-review-template",
					Description:  "internal-review-template-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)

				// 模拟审核服务
				auditSvc.EXPECT().CreateAudit(gomock.Any(), gomock.Any()).Return(1, nil)

				handler := templateweb.NewHandler(svc.Svc)
				return handler, templateFromDB.Versions[0].ID
			},
			req: templateweb.SubmitForInternalReviewReq{
				VersionID: 1, // 第一个版本ID
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, expected templateweb.SubmitForInternalReviewReq, ctrl *gomock.Controller) {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				version, err := svc.Repo.GetTemplateVersionByID(t.Context(), expected.VersionID)
				require.NoError(t, err)
				assert.Equal(t, domain.AuditStatusInReview, version.AuditStatus)
				assert.NotZero(t, version.AuditID)
				assert.NotZero(t, version.LastReviewSubmissionTime)
			},
		},
		{
			name: "提交不存在的版本进行内部审核",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) (*templateweb.Handler, int64) {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				handler := templateweb.NewHandler(svc.Svc)
				return handler, 0
			},
			req: templateweb.SubmitForInternalReviewReq{
				VersionID: 9999, // 不存在的版本ID
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 506001,
				Msg:  "系统错误",
			},
			after: func(t *testing.T, expected templateweb.SubmitForInternalReviewReq, ctrl *gomock.Controller) {},
		},
		{
			name: "重复提交内部审核",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) (*templateweb.Handler, int64) {
				t.Helper()

				svc, providerSvc, auditSvc, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "duplicate-review-template",
					Description:  "duplicate-review-template-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)

				// 模拟第一次提交审核
				auditSvc.EXPECT().CreateAudit(gomock.Any(), gomock.Any()).Return(1, nil)

				// 先执行一次提交内部审核
				err = svc.Svc.SubmitForInternalReview(t.Context(), templateFromDB.Versions[0].ID)
				require.NoError(t, err)

				// 第二次提交不需要mock审核服务，因为应该会在版本状态检查时就失败

				handler := templateweb.NewHandler(svc.Svc)
				return handler, templateFromDB.Versions[0].ID
			},
			req: templateweb.SubmitForInternalReviewReq{
				VersionID: 1, // 第一个版本ID
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, expected templateweb.SubmitForInternalReviewReq, ctrl *gomock.Controller) {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				version, err := svc.Repo.GetTemplateVersionByID(t.Context(), expected.VersionID)
				require.NoError(t, err)
				assert.Equal(t, domain.AuditStatusInReview, version.AuditStatus)
				assert.NotZero(t, version.AuditID)
				assert.NotZero(t, version.LastReviewSubmissionTime)
			},
		},
		{
			name: "审核服务返回错误",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) (*templateweb.Handler, int64) {
				t.Helper()

				svc, providerSvc, auditSvc, _ := s.newService(ctrl)

				providerSvc.EXPECT().GetByChannel(gomock.Any(), domain.ChannelSMS).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil)

				// 创建一个模板
				template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
					OwnerID:      ownerID,
					OwnerType:    ownerType,
					Name:         "error-review-template",
					Description:  "error-review-template-desc",
					Channel:      domain.ChannelSMS,
					BusinessType: domain.BusinessTypePromotion,
				})
				require.NoError(t, err)

				// 获取模板版本
				templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
				require.NoError(t, err)
				require.Len(t, templateFromDB.Versions, 1)

				// 模拟审核服务返回错误
				auditSvc.EXPECT().CreateAudit(gomock.Any(), gomock.Any()).Return(0, fmt.Errorf("模拟审核服务错误"))

				handler := templateweb.NewHandler(svc.Svc)
				return handler, templateFromDB.Versions[0].ID
			},
			req: templateweb.SubmitForInternalReviewReq{
				VersionID: 1, // 第一个版本ID
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 506001,
				Msg:  "系统错误",
			},
			after: func(t *testing.T, expected templateweb.SubmitForInternalReviewReq, ctrl *gomock.Controller) {
				t.Helper()
				svc, _, _, _ := s.newService(ctrl)
				version, err := svc.Repo.GetTemplateVersionByID(t.Context(), expected.VersionID)
				require.NoError(t, err)
				assert.Equal(t, domain.AuditStatusPending, version.AuditStatus)
				assert.NotZero(t, version.AuditID)
				assert.NotZero(t, version.LastReviewSubmissionTime)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, versionID := tc.newHandlerFunc(t, ctrl)
			tc.req.VersionID = versionID

			req, err := http.NewRequest(http.MethodPost,
				"/templates/versions/review/internal", iox.NewJSONReader(tc.req))

			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[any]()

			server := s.newGinServer(handler)

			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)

			actual := recorder.MustScan()
			assert.Equal(t, tc.wantResp.Code, actual.Code)
			assert.Equal(t, tc.wantResp.Msg, actual.Msg)

			// 如果提交成功，验证版本状态是否已更新
			if tc.wantResp.Code == 0 {
			}
		})
	}
}

func (s *TemplateHandlerTestSuite) TestEvent_AuditResultConsume() {
	t := s.T()

	testCases := []struct {
		name          string
		setupMock     func(t *testing.T, ctrl *gomock.Controller) (*templateioc.Service, []auditevt.CallbackResultEvent)
		errAssertFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, svc *templateioc.Service, events []auditevt.CallbackResultEvent)
	}{
		{
			name: "成功处理多条审核通过和审核拒绝消息",
			setupMock: func(t *testing.T, ctrl *gomock.Controller) (*templateioc.Service, []auditevt.CallbackResultEvent) {
				t.Helper()

				// 创建服务实例
				svc, providerSvc, _, clients := s.newService(ctrl)

				n := 6
				// 设置供应商审核预期
				providerSvc.EXPECT().GetByChannel(gomock.Any(), gomock.Any()).Return([]domain.Provider{
					{
						ID:      1,
						Name:    "mock-provider-name-1",
						Channel: domain.ChannelSMS,
						Status:  domain.ProviderStatusActive,
					},
				}, nil).Times(n)

				mockClient := clients["mock-provider-name-1"].(*smsmocks.MockClient)

				// 模拟客户端CreateTemplate方法，这是在submitForProviderReview中被调用的
				mockClient.EXPECT().
					CreateTemplate(gomock.Any()).
					Return(client.CreateTemplateResp{
						RequestID:  "mock-request-id",
						TemplateID: "mock-template-id",
					}, nil).
					AnyTimes()

				// 创建3个模板和版本，用于测试批量处理
				var versionIDs []int64
				for i := 0; i < n; i++ {
					template, err := svc.Svc.CreateTemplate(t.Context(), domain.ChannelTemplate{
						OwnerID:      ownerID,
						OwnerType:    ownerType,
						Name:         fmt.Sprintf("audit-approved-template-%d", i),
						Description:  fmt.Sprintf("audit approved template %d", i),
						Channel:      domain.ChannelSMS,
						BusinessType: domain.BusinessTypePromotion,
					})
					require.NoError(t, err)

					// 获取模板版本
					templateFromDB, err := svc.Svc.GetTemplateByID(t.Context(), template.ID)
					require.NoError(t, err)
					require.Len(t, templateFromDB.Versions, 1)

					// 提交内部审核
					versionIDs = append(versionIDs, templateFromDB.Versions[0].ID)
				}

				// 创建测试消息
				now := time.Now().Unix()
				events := make([]auditevt.CallbackResultEvent, 0, len(versionIDs))
				for i := range versionIDs {
					auditStatus := domain.AuditStatusApproved.String()
					if i%2 == 1 {
						auditStatus = domain.AuditStatusRejected.String()
					}
					events = append(events, auditevt.CallbackResultEvent{
						ResourceID:   versionIDs[i],
						ResourceType: domain.ResourceTypeTemplate,
						AuditID:      int64(i + 100),
						AuditorID:    int64(i + 1000),
						AuditTime:    now,
						AuditStatus:  auditStatus,
						RejectReason: "",
					})
				}

				// 使用AuditResultProducer发送消息
				for i := range events {
					err := svc.AuditResultProducer.Produce(t.Context(), events[i])
					require.NoError(t, err)
				}

				// 等待消息被消费
				time.Sleep(500 * time.Millisecond)

				return svc, events
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, svc *templateioc.Service, events []auditevt.CallbackResultEvent) {
				// 验证版本状态
				for i := range events {
					version, err := svc.Repo.GetTemplateVersionByID(t.Context(), events[i].ResourceID)
					require.NoError(t, err)
					assert.Equal(t, domain.AuditStatus(events[i].AuditStatus), version.AuditStatus)
					assert.Equal(t, events[i].AuditID, version.AuditID)
					assert.Equal(t, events[i].AuditorID, version.AuditorID)
					assert.Equal(t, events[i].AuditTime, version.AuditTime)

					// 验证供应商状态 - 因为审核状态是已通过，所以应该触发供应商审核
					assert.Len(t, version.Providers, 1)
					if i%2 == 1 {
						// 验证供应商状态 - 因为审核状态是已拒绝，所以不应该触发供应商审核
						assert.Equal(t, domain.AuditStatusPending, version.Providers[0].AuditStatus)
					} else {
						assert.Equal(t, domain.AuditStatusInReview, version.Providers[0].AuditStatus)
						assert.NotZero(t, version.Providers[0].LastReviewSubmissionTime)
					}
				}
			},
		},
		{
			name: "处理无效的ResourceType消息",
			setupMock: func(t *testing.T, ctrl *gomock.Controller) (*templateioc.Service, []auditevt.CallbackResultEvent) {
				t.Helper()

				// 创建服务实例
				svc, _, _, _ := s.newService(ctrl)

				// 创建测试消息 - 无效的ResourceType
				events := []auditevt.CallbackResultEvent{
					{
						ResourceID:   999,
						ResourceType: "INVALID_TYPE", // 无效的资源类型
						AuditID:      300,
						AuditorID:    3000,
						AuditTime:    time.Now().Unix(),
						AuditStatus:  domain.AuditStatusApproved.String(),
						RejectReason: "",
					},
				}

				// 使用AuditResultProducer发送消息
				for _, event := range events {
					err := svc.AuditResultProducer.Produce(t.Context(), event)
					require.NoError(t, err)
				}

				// 等待消息被消费
				time.Sleep(500 * time.Millisecond)

				return svc, events
			},
			errAssertFunc: assert.NoError, // 无效消息应该被跳过，不返回错误
			after: func(t *testing.T, svc *templateioc.Service, events []auditevt.CallbackResultEvent) {
				// 无效消息被跳过，不需要额外检查
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc, events := tc.setupMock(t, ctrl)

			err := svc.AuditResultConsumer.Consume(t.Context())
			tc.errAssertFunc(t, err)

			if err != nil {
				return
			}

			tc.after(t, svc, events)
		})
	}
}
