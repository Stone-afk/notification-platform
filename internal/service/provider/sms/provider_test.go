package sms

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/service/provider/sms/client"
	smsmocks "notification-platform/internal/service/provider/sms/client/mocks"
	templatemocks "notification-platform/internal/service/template/mocks"
	"testing"
)

func TestNewSMSProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerName string
	}{
		{
			name:         "创建SMS供应商成功",
			providerName: "aliyun",
		},
		{
			name:         "创建另一个SMS供应商成功",
			providerName: "tencent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockTemplateSvc := templatemocks.NewMockChannelTemplateService(ctrl)
			mockClient := smsmocks.NewMockClient(ctrl)

			provider := NewSMSProvider(tt.providerName, mockTemplateSvc, mockClient)

			assert.NotNil(t, provider)
		})
	}
}

func TestSmsProvider_Send(t *testing.T) {
	t.Parallel()

	ErrGetTemplateFailed := errors.New("获取模板失败")
	ErrSendSMSFailed := errors.New("发送短信失败")

	testNotification := domain.Notification{
		ID:      uint64(12345),
		Channel: domain.ChannelSMS,
		Template: domain.Template{
			ID:        1,
			VersionID: 1,
			Params:    map[string]string{"code": "123456"},
		},
		Receivers: []string{"13800138000"},
	}

	tests := []struct {
		name      string
		setupMock func(ctrl *gomock.Controller, provider *smsProvider)
		wantErr   error
	}{
		{
			name: "获取模板失败",
			setupMock: func(ctrl *gomock.Controller, provider *smsProvider) {
				mockTemplateSvc := provider.templateSvc.(*templatemocks.MockChannelTemplateService)

				// 模拟获取模板失败
				mockTemplateSvc.EXPECT().
					GetTemplateByIDAndProviderInfo(gomock.Any(), testNotification.Template.ID, "aliyun", domain.ChannelSMS).
					Return(domain.ChannelTemplate{}, fmt.Errorf("%w: 供应商%d", ErrGetTemplateFailed, 1))
			},
			wantErr: errs.ErrSendNotificationFailed,
		},
		{
			name: "无已发布模版",
			setupMock: func(ctrl *gomock.Controller, provider *smsProvider) {
				mockTemplateSvc := provider.templateSvc.(*templatemocks.MockChannelTemplateService)

				// 模拟返回没有活跃版本的模板
				mockTemplateSvc.EXPECT().
					GetTemplateByIDAndProviderInfo(gomock.Any(), testNotification.Template.ID, "aliyun", domain.ChannelSMS).
					Return(domain.ChannelTemplate{
						ID:       testNotification.Template.ID,
						Channel:  domain.ChannelSMS,
						Versions: []domain.ChannelTemplateVersion{},
					}, nil)
			},
			wantErr: errs.ErrSendNotificationFailed,
		},
		{
			name: "发送短信失败",
			setupMock: func(ctrl *gomock.Controller, provider *smsProvider) {
				mockTemplateSvc := provider.templateSvc.(*templatemocks.MockChannelTemplateService)
				mockClient := provider.client.(*smsmocks.MockClient)

				// 模拟返回有活跃版本的模板
				activeVersion := domain.ChannelTemplateVersion{
					ID:                1,
					ChannelTemplateID: testNotification.Template.ID,
					Name:              "验证码模板",
					Signature:         "测试签名",
					Content:           "您的验证码是：${code}",
					AuditStatus:       domain.AuditStatusApproved,
					Providers: []domain.ChannelTemplateProvider{
						{
							ID:                 1,
							TemplateID:         1,
							TemplateVersionID:  1,
							ProviderName:       "aliyun",
							ProviderTemplateID: "SMS_123456",
							AuditStatus:        domain.AuditStatusApproved,
						},
					},
				}

				mockTemplateSvc.EXPECT().
					GetTemplateByIDAndProviderInfo(gomock.Any(), testNotification.Template.ID, "aliyun", domain.ChannelSMS).
					Return(domain.ChannelTemplate{
						ID:              testNotification.Template.ID,
						Channel:         domain.ChannelSMS,
						Versions:        []domain.ChannelTemplateVersion{activeVersion},
						ActiveVersionID: activeVersion.ID,
					}, nil)

				// 模拟发送短信失败
				mockClient.EXPECT().
					Send(client.SendReq{
						PhoneNumbers:  testNotification.Receivers,
						SignName:      activeVersion.Signature,
						TemplateID:    activeVersion.Providers[0].ProviderTemplateID,
						TemplateParam: testNotification.Template.Params,
					}).
					Return(client.SendResp{}, fmt.Errorf("%w: 供应商%d", ErrSendSMSFailed, 1))
			},
			wantErr: errs.ErrSendNotificationFailed,
		},
		{
			name: "发送短信状态码不为OK",
			setupMock: func(ctrl *gomock.Controller, provider *smsProvider) {
				mockTemplateSvc := provider.templateSvc.(*templatemocks.MockChannelTemplateService)
				mockClient := provider.client.(*smsmocks.MockClient)

				// 模拟返回有活跃版本的模板
				activeVersion := domain.ChannelTemplateVersion{
					ID:                1,
					ChannelTemplateID: testNotification.Template.ID,
					Name:              "验证码模板",
					Signature:         "测试签名",
					Content:           "您的验证码是：${code}",
					AuditStatus:       domain.AuditStatusApproved,
					Providers: []domain.ChannelTemplateProvider{
						{
							ID:                 1,
							TemplateID:         1,
							TemplateVersionID:  1,
							ProviderName:       "aliyun",
							ProviderTemplateID: "SMS_123456",
							AuditStatus:        domain.AuditStatusApproved,
						},
					},
				}

				mockTemplateSvc.EXPECT().
					GetTemplateByIDAndProviderInfo(gomock.Any(), testNotification.Template.ID, "aliyun", domain.ChannelSMS).
					Return(domain.ChannelTemplate{
						ID:              testNotification.Template.ID,
						Channel:         domain.ChannelSMS,
						Versions:        []domain.ChannelTemplateVersion{activeVersion},
						ActiveVersionID: activeVersion.ID,
					}, nil)

				// 模拟发送短信成功，但状态码不是OK
				mockClient.EXPECT().
					Send(client.SendReq{
						PhoneNumbers:  testNotification.Receivers,
						SignName:      activeVersion.Signature,
						TemplateID:    activeVersion.Providers[0].ProviderTemplateID,
						TemplateParam: testNotification.Template.Params,
					}).
					Return(client.SendResp{
						PhoneNumbers: map[string]client.SendRespStatus{
							"13800138000": {
								Code:    "ERROR",
								Message: "发送失败",
							},
						},
					}, nil)
			},
			wantErr: errs.ErrSendNotificationFailed,
		},
		{
			name: "发送短信成功",
			setupMock: func(ctrl *gomock.Controller, provider *smsProvider) {
				mockTemplateSvc := provider.templateSvc.(*templatemocks.MockChannelTemplateService)
				mockClient := provider.client.(*smsmocks.MockClient)

				// 模拟返回有活跃版本的模板
				activeVersion := domain.ChannelTemplateVersion{
					ID:                1,
					ChannelTemplateID: testNotification.Template.ID,
					Name:              "验证码模板",
					Signature:         "测试签名",
					Content:           "您的验证码是：${code}",
					AuditStatus:       domain.AuditStatusApproved,
					Providers: []domain.ChannelTemplateProvider{
						{
							ID:                 1,
							TemplateID:         1,
							TemplateVersionID:  1,
							ProviderName:       "aliyun",
							ProviderTemplateID: "SMS_123456",
							AuditStatus:        domain.AuditStatusApproved,
						},
					},
				}

				mockTemplateSvc.EXPECT().
					GetTemplateByIDAndProviderInfo(gomock.Any(), testNotification.Template.ID, "aliyun", domain.ChannelSMS).
					Return(domain.ChannelTemplate{
						ID:              testNotification.Template.ID,
						Channel:         domain.ChannelSMS,
						Versions:        []domain.ChannelTemplateVersion{activeVersion},
						ActiveVersionID: activeVersion.ID,
					}, nil)

				// 模拟发送短信成功，且状态码为OK
				mockClient.EXPECT().
					Send(client.SendReq{
						PhoneNumbers:  testNotification.Receivers,
						SignName:      activeVersion.Signature,
						TemplateID:    activeVersion.Providers[0].ProviderTemplateID,
						TemplateParam: testNotification.Template.Params,
					}).
					Return(client.SendResp{
						PhoneNumbers: map[string]client.SendRespStatus{
							"13800138000": {
								Code:    "OK",
								Message: "发送成功",
							},
						},
					}, nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 直接创建provider并传入mock服务
			mockTemplateSvc := templatemocks.NewMockChannelTemplateService(ctrl)
			mockClient := smsmocks.NewMockClient(ctrl)

			provider := &smsProvider{
				name:        "aliyun",
				templateSvc: mockTemplateSvc,
				client:      mockClient,
			}

			// 设置mock期望
			tt.setupMock(ctrl, provider)

			// 执行测试
			resp, err := provider.Send(context.Background(), testNotification)

			// 验证结果
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testNotification.ID, resp.NotificationID)
			assert.Equal(t, domain.SendStatusSucceeded, resp.Status)
		})
	}
}
