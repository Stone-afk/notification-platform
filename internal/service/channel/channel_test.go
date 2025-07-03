package channel

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	channelmocks "notification-platform/internal/service/channel/mocks"
)

func TestChannelSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ChannelTestSuite))
}

type ChannelTestSuite struct {
	suite.Suite
}

func (s *ChannelTestSuite) TestNewDispatcher() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name            string
		getChannelsFunc func(ctrl *gomock.Controller) map[domain.Channel]Channel
		wantLen         int
	}{
		{
			name: "初始化空map",
			getChannelsFunc: func(_ *gomock.Controller) map[domain.Channel]Channel {
				return make(map[domain.Channel]Channel)
			},
			wantLen: 0,
		},
		{
			name: "初始化包含Channel的map",
			getChannelsFunc: func(ctrl *gomock.Controller) map[domain.Channel]Channel {
				return map[domain.Channel]Channel{
					domain.ChannelSMS:   channelmocks.NewMockChannel(ctrl),
					domain.ChannelEmail: channelmocks.NewMockChannel(ctrl),
				}
			},
			wantLen: 2,
		},
		{
			name: "初始化只包含SMS的map",
			getChannelsFunc: func(ctrl *gomock.Controller) map[domain.Channel]Channel {
				return map[domain.Channel]Channel{
					domain.ChannelSMS: channelmocks.NewMockChannel(ctrl),
				}
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			channels := tt.getChannelsFunc(ctrl)

			dispatcher := NewDispatcher(channels)

			assert.NotNil(t, dispatcher)
			assert.Equal(t, channels, dispatcher.channels)
			assert.Len(t, dispatcher.channels, tt.wantLen)
		})
	}
}

func (s *ChannelTestSuite) TestDispatcherSend() {
	t := s.T()
	t.Parallel()
	ErrSMSSendFailed := errors.New("短信发送失败")
	ErrEMAILSendFailed := errors.New("EMAIL发送失败")
	tests := []struct {
		name            string
		notification    domain.Notification
		getChannelsFunc func(ctrl *gomock.Controller) map[domain.Channel]Channel
		expectedResp    domain.SendResponse
		assertFunc      assert.ErrorAssertionFunc
	}{
		{
			name: "发送成功_SMS",
			notification: domain.Notification{
				Channel: domain.ChannelSMS,
			},
			getChannelsFunc: func(ctrl *gomock.Controller) map[domain.Channel]Channel {
				mockChannel := channelmocks.NewMockChannel(ctrl)
				mockChannel.EXPECT().Send(gomock.Any(), gomock.Any()).Return(
					domain.SendResponse{
						Status: domain.SendStatusSucceeded,
					}, nil)

				return map[domain.Channel]Channel{
					domain.ChannelSMS: mockChannel,
				}
			},
			expectedResp: domain.SendResponse{
				Status: domain.SendStatusSucceeded,
			},
			assertFunc: assert.NoError,
		},
		{
			name: "发送失败_SMS",
			notification: domain.Notification{
				Channel: domain.ChannelSMS,
			},
			getChannelsFunc: func(ctrl *gomock.Controller) map[domain.Channel]Channel {
				mockChannel := channelmocks.NewMockChannel(ctrl)
				mockChannel.EXPECT().Send(gomock.Any(), gomock.Any()).Return(
					domain.SendResponse{}, fmt.Errorf("%w", ErrSMSSendFailed))
				return map[domain.Channel]Channel{
					domain.ChannelSMS: mockChannel,
				}
			},
			expectedResp: domain.SendResponse{},
			assertFunc: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, ErrSMSSendFailed, msgAndArgs...)
			},
		},
		{
			name: "发送成功_Email",
			notification: domain.Notification{
				Channel: domain.ChannelEmail,
			},
			getChannelsFunc: func(ctrl *gomock.Controller) map[domain.Channel]Channel {
				mockChannel := channelmocks.NewMockChannel(ctrl)
				mockChannel.EXPECT().Send(gomock.Any(), gomock.Any()).Return(
					domain.SendResponse{
						Status: domain.SendStatusSucceeded,
					}, nil)

				return map[domain.Channel]Channel{
					domain.ChannelEmail: mockChannel,
				}
			},
			expectedResp: domain.SendResponse{
				Status: domain.SendStatusSucceeded,
			},
			assertFunc: assert.NoError,
		},
		{
			name: "发送失败_Email",
			notification: domain.Notification{
				Channel: domain.ChannelEmail,
			},
			getChannelsFunc: func(ctrl *gomock.Controller) map[domain.Channel]Channel {
				mockChannel := channelmocks.NewMockChannel(ctrl)
				mockChannel.EXPECT().Send(gomock.Any(), gomock.Any()).Return(
					domain.SendResponse{}, fmt.Errorf("%w", ErrEMAILSendFailed))
				return map[domain.Channel]Channel{
					domain.ChannelEmail: mockChannel,
				}
			},
			expectedResp: domain.SendResponse{},
			assertFunc: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, ErrEMAILSendFailed, msgAndArgs...)
			},
		},
		{
			name: "渠道不存在",
			notification: domain.Notification{
				Channel: "不存在的渠道",
			},
			getChannelsFunc: func(_ *gomock.Controller) map[domain.Channel]Channel {
				return make(map[domain.Channel]Channel)
			},
			expectedResp: domain.SendResponse{},
			assertFunc: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorIs(t, err, errs.ErrNoAvailableChannel, msgAndArgs...)
			},
		},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			channels := tests[i].getChannelsFunc(ctrl)
			dispatcher := NewDispatcher(channels)

			resp, err := dispatcher.Send(t.Context(), tests[i].notification)
			tests[i].assertFunc(t, err)

			if err != nil {
				return
			}
			assert.Equal(t, tests[i].expectedResp, resp)
		})
	}
}
