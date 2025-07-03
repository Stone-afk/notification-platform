package channel

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/service/provider"
	providermocks "notification-platform/internal/service/provider/mocks"
	"testing"
)

func TestSMSSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SMSTestSuite))
}

type SMSTestSuite struct {
	suite.Suite
}

func (s *SMSTestSuite) TestNewSMSChannel() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBuilder := providermocks.NewMockSelectorBuilder(ctrl)

	channel := NewSMSChannel(mockBuilder)
	assert.NotNil(t, channel)

	_, ok := channel.(*smsChannel)
	assert.True(t, ok)
}

func (s *SMSTestSuite) TestSMSChannelSend() {
	t := s.T()
	t.Parallel()

	// 定义错误变量
	ErrBuildSelectorFailed := errors.New("构建选择器失败")
	ErrNoAvailableProvider := errors.New("没有可用的供应商")
	ErrProviderSendFailed := errors.New("供应商发送失败")

	// 定义测试通知
	testNotification := domain.Notification{
		ID:      uint64(17387),
		Channel: domain.ChannelSMS,
		Template: domain.Template{
			ID:        1,
			VersionID: 1,
			Params: map[string]string{
				"code": "123456",
			},
		},
		Receivers: []string{"13800138000"},
	}

	tests := []struct {
		name         string
		setupMocks   func(ctrl *gomock.Controller) (provider.SelectorBuilder, *providermocks.MockSelector, []*providermocks.MockProvider)
		expectedResp domain.SendResponse
		assertFunc   assert.ErrorAssertionFunc
	}{
		{
			name: "发送成功",
			setupMocks: func(ctrl *gomock.Controller) (provider.SelectorBuilder, *providermocks.MockSelector, []*providermocks.MockProvider) {
				mockBuilder := providermocks.NewMockSelectorBuilder(ctrl)
				mockSelector := providermocks.NewMockSelector(ctrl)
				mockProvider := providermocks.NewMockProvider(ctrl)

				mockBuilder.EXPECT().Build().Return(mockSelector, nil)

				mockSelector.EXPECT().Next(gomock.Any(), testNotification).Return(mockProvider, nil)

				expectedResp := domain.SendResponse{
					NotificationID: testNotification.ID,
					Status:         domain.SendStatusSucceeded,
				}
				mockProvider.EXPECT().Send(gomock.Any(), testNotification).Return(expectedResp, nil)

				return mockBuilder, mockSelector, []*providermocks.MockProvider{mockProvider}
			},
			expectedResp: domain.SendResponse{
				NotificationID: testNotification.ID,
				Status:         domain.SendStatusSucceeded,
			},
			assertFunc: assert.NoError,
		},
		{
			name: "构建选择器失败",
			setupMocks: func(ctrl *gomock.Controller) (provider.SelectorBuilder, *providermocks.MockSelector, []*providermocks.MockProvider) {
				mockBuilder := providermocks.NewMockSelectorBuilder(ctrl)
				mockBuilder.EXPECT().Build().Return(nil, fmt.Errorf("%w", ErrBuildSelectorFailed))
				return mockBuilder, nil, nil
			},
			expectedResp: domain.SendResponse{},
			assertFunc: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, errs.ErrSendNotificationFailed, msgAndArgs...)
			},
		},
		{
			name: "没有可用供应商",
			setupMocks: func(ctrl *gomock.Controller) (provider.SelectorBuilder, *providermocks.MockSelector, []*providermocks.MockProvider) {
				mockBuilder := providermocks.NewMockSelectorBuilder(ctrl)
				mockSelector := providermocks.NewMockSelector(ctrl)
				mockBuilder.EXPECT().Build().Return(mockSelector, nil)
				mockSelector.EXPECT().Next(gomock.Any(), testNotification).
					Return(nil, fmt.Errorf("%w", ErrNoAvailableProvider))
				return mockBuilder, mockSelector, nil
			},
			expectedResp: domain.SendResponse{},
			assertFunc: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, ErrNoAvailableProvider, msgAndArgs...)
			},
		},
		{
			name: "首次发送失败,重试成功",
			setupMocks: func(ctrl *gomock.Controller) (provider.SelectorBuilder, *providermocks.MockSelector, []*providermocks.MockProvider) {
				mockBuilder := providermocks.NewMockSelectorBuilder(ctrl)
				mockSelector := providermocks.NewMockSelector(ctrl)
				failedProvider := providermocks.NewMockProvider(ctrl)
				successProvider := providermocks.NewMockProvider(ctrl)

				mockBuilder.EXPECT().Build().Return(mockSelector, nil)

				mockSelector.EXPECT().Next(gomock.Any(), testNotification).Return(failedProvider, nil)
				failedProvider.EXPECT().Send(gomock.Any(), testNotification).
					Return(domain.SendResponse{}, fmt.Errorf("%w: 供应商1", ErrProviderSendFailed))

				mockSelector.EXPECT().Next(gomock.Any(), testNotification).Return(successProvider, nil)
				successResp := domain.SendResponse{
					NotificationID: testNotification.ID,
					Status:         domain.SendStatusSucceeded,
				}
				successProvider.EXPECT().Send(gomock.Any(), testNotification).Return(successResp, nil)
				return mockBuilder, mockSelector, []*providermocks.MockProvider{failedProvider, successProvider}
			},
			expectedResp: domain.SendResponse{
				NotificationID: testNotification.ID,
				Status:         domain.SendStatusSucceeded,
			},
			assertFunc: assert.NoError,
		},
		{
			name: "连续多个供应商失败后成功",
			setupMocks: func(ctrl *gomock.Controller) (provider.SelectorBuilder, *providermocks.MockSelector, []*providermocks.MockProvider) {
				mockBuilder := providermocks.NewMockSelectorBuilder(ctrl)
				mockSelector := providermocks.NewMockSelector(ctrl)
				failedProvider1 := providermocks.NewMockProvider(ctrl)
				failedProvider2 := providermocks.NewMockProvider(ctrl)
				successProvider := providermocks.NewMockProvider(ctrl)

				mockBuilder.EXPECT().Build().Return(mockSelector, nil)

				// 第一个供应商失败
				mockSelector.EXPECT().Next(gomock.Any(), testNotification).Return(failedProvider1, nil)
				failedProvider1.EXPECT().Send(gomock.Any(), testNotification).
					Return(domain.SendResponse{}, fmt.Errorf("%w: 供应商1", ErrProviderSendFailed))

				// 第二个供应商失败
				mockSelector.EXPECT().Next(gomock.Any(), testNotification).Return(failedProvider2, nil)
				failedProvider2.EXPECT().Send(gomock.Any(), testNotification).
					Return(domain.SendResponse{}, fmt.Errorf("%w: 供应商2", ErrProviderSendFailed))

				// 第三个供应商成功
				mockSelector.EXPECT().Next(gomock.Any(), testNotification).Return(successProvider, nil)
				successResp := domain.SendResponse{
					NotificationID: testNotification.ID,
					Status:         domain.SendStatusSucceeded,
				}
				successProvider.EXPECT().Send(gomock.Any(), testNotification).Return(successResp, nil)

				return mockBuilder, mockSelector, []*providermocks.MockProvider{failedProvider1, failedProvider2, successProvider}
			},
			expectedResp: domain.SendResponse{
				NotificationID: testNotification.ID,
				Status:         domain.SendStatusSucceeded,
			},
			assertFunc: assert.NoError,
		},
		{
			name: "所有供应商都失败",
			setupMocks: func(ctrl *gomock.Controller) (provider.SelectorBuilder, *providermocks.MockSelector, []*providermocks.MockProvider) {
				mockBuilder := providermocks.NewMockSelectorBuilder(ctrl)
				mockSelector := providermocks.NewMockSelector(ctrl)
				var failedProviders []*providermocks.MockProvider

				mockBuilder.EXPECT().Build().Return(mockSelector, nil)

				// 尝试三次,最后返回没有更多供应商
				for i := 0; i < 3; i++ {
					failedProvider := providermocks.NewMockProvider(ctrl)
					failedProviders = append(failedProviders, failedProvider)
					mockSelector.EXPECT().Next(gomock.Any(), testNotification).Return(failedProvider, nil)
					failedProvider.EXPECT().Send(gomock.Any(), testNotification).
						Return(domain.SendResponse{}, fmt.Errorf("%w: 供应商%d", ErrProviderSendFailed, i+1))
				}

				// 最后没有更多供应商
				mockSelector.EXPECT().Next(gomock.Any(), testNotification).
					Return(nil, fmt.Errorf("%w", ErrNoAvailableProvider))

				return mockBuilder, mockSelector, failedProviders
			},
			expectedResp: domain.SendResponse{},
			assertFunc: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, ErrNoAvailableProvider, msgAndArgs...)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockBuilder, _, _ := tt.setupMocks(ctrl)
			channel := NewSMSChannel(mockBuilder)

			resp, err := channel.Send(t.Context(), testNotification)

			tt.assertFunc(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.expectedResp, resp)
		})
	}
}
