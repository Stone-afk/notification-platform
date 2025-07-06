package loopjob

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	loopjobmocks "notification-platform/internal/pkg/loopjob/mocks"
)

var (
	errNewLock      = errors.New("创建锁失败")
	errLock         = errors.New("获取锁失败")
	errRefresh      = errors.New("续约锁失败")
	errUnlock       = errors.New("释放锁失败")
	errBizExecution = errors.New("业务执行失败")
)

func TestLoopJobSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LoopJobTestSuite))
}

type LoopJobTestSuite struct {
	suite.Suite
}

// TestNewInfiniteLoop 测试InfiniteLoop的构造函数
func (s *LoopJobTestSuite) TestNewInfiniteLoop() {
	t := s.T()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := loopjobmocks.NewMockClient(ctrl)

	// 测试默认构造函数
	loop := NewInfiniteLoop(mockClient, func(_ context.Context) error { return nil }, "test-key")
	assert.Equal(t, mockClient, loop.dclient)
	assert.Equal(t, "test-key", loop.key)
	assert.Equal(t, time.Minute, loop.retryInterval)
	assert.Equal(t, 3*time.Second, loop.defaultTimeout)

	// 测试带自定义参数的构造函数
	customInterval := 500 * time.Millisecond
	defaultTimeout := 3 * time.Second
	loop = newInfiniteLoop(mockClient, func(_ context.Context) error { return nil }, "test-key", customInterval, defaultTimeout)
	assert.Equal(t, mockClient, loop.dclient)
	assert.Equal(t, "test-key", loop.key)
	assert.Equal(t, customInterval, loop.retryInterval)
	assert.Equal(t, defaultTimeout, loop.defaultTimeout)
}

// TestRunWithNewLockError 测试创建锁失败时的行为
func (s *LoopJobTestSuite) TestRunWithNewLockError() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建一个控制测试终止的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 设置模拟行为：创建锁失败，允许调用多次
	mockClient := loopjobmocks.NewMockClient(ctrl)
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(nil, errNewLock).AnyTimes()

	// 使用非常短的重试间隔
	retryInterval := 2 * time.Millisecond
	defaultTimeout := 3 * time.Millisecond
	var bizExecuted atomic.Bool

	// 创建测试完成通知通道
	done := make(chan struct{})

	go func() {
		defer close(done)

		loop := newInfiniteLoop(mockClient,
			func(_ context.Context) error {
				bizExecuted.Store(true)
				return nil
			},
			"test-key",
			retryInterval,
			defaultTimeout)

		loop.Run(ctx)
	}()

	// 等待测试完成或强制超时
	select {
	case <-done:
		// 测试正常完成
	case <-time.After(100 * time.Millisecond):
		t.Log("测试强制结束")
		cancel()
	}

	// 业务逻辑不应该被执行
	assert.False(t, bizExecuted.Load(), "业务逻辑不应该被执行")
}

// TestRunWithLockError 测试获取锁失败时的行为
func (s *LoopJobTestSuite) TestRunWithLockError() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建一个控制测试终止的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 设置模拟行为：创建锁成功，但获取锁失败
	mockClient := loopjobmocks.NewMockClient(ctrl)
	mockLock := loopjobmocks.NewMockLock(ctrl)
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(mockLock, nil).AnyTimes()
	mockLock.EXPECT().Lock(gomock.Any()).Return(errLock).AnyTimes()

	// 使用非常短的重试间隔
	retryInterval := 2 * time.Millisecond
	defaultTimeout := 3 * time.Millisecond
	var bizExecuted atomic.Bool

	// 创建测试完成通知通道
	done := make(chan struct{})

	go func() {
		defer close(done)

		loop := newInfiniteLoop(mockClient,
			func(_ context.Context) error {
				bizExecuted.Store(true)
				return nil
			},
			"test-key",
			retryInterval,
			defaultTimeout)

		loop.Run(ctx)
	}()

	// 等待测试完成或强制超时
	select {
	case <-done:
		// 测试正常完成
	case <-time.After(100 * time.Millisecond):
		t.Log("测试强制结束")
		cancel()
	}

	// 业务逻辑不应该被执行
	assert.False(t, bizExecuted.Load(), "业务逻辑不应该被执行")
}

// TestRunWithRefreshError 测试续约锁失败时的行为
func (s *LoopJobTestSuite) TestRunWithRefreshError() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建一个控制测试终止的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 设置模拟行为
	mockClient := loopjobmocks.NewMockClient(ctrl)
	mockLock := loopjobmocks.NewMockLock(ctrl)

	// 1. 创建锁成功
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(mockLock, nil).AnyTimes()
	// 2. 获取锁成功
	mockLock.EXPECT().Lock(gomock.Any()).Return(nil).AnyTimes()
	// 3. 业务执行成功后，续约失败
	mockLock.EXPECT().Refresh(gomock.Any()).Return(errRefresh).AnyTimes()
	// 4. 释放锁
	mockLock.EXPECT().Unlock(gomock.Any()).Return(nil).AnyTimes()

	// 使用非常短的重试间隔
	retryInterval := 2 * time.Millisecond
	defaultTimeout := 3 * time.Millisecond
	var bizExecuted atomic.Bool
	var bizExecCount atomic.Int32

	// 创建测试完成通知通道
	done := make(chan struct{})

	go func() {
		defer close(done)

		loop := newInfiniteLoop(mockClient,
			func(_ context.Context) error {
				bizExecuted.Store(true)
				bizExecCount.Add(1)
				return nil
			},
			"test-key",
			retryInterval,
			defaultTimeout)

		loop.Run(ctx)
	}()

	// 等待测试完成或强制超时
	select {
	case <-done:
		// 测试正常完成
	case <-time.After(100 * time.Millisecond):
		t.Log("测试强制结束")
		cancel()
	}

	// 业务逻辑应该被执行过
	assert.True(t, bizExecuted.Load(), "业务逻辑应该被执行")
	assert.GreaterOrEqual(t, bizExecCount.Load(), int32(1), "业务逻辑应该至少执行一次")
}

// TestRunWithUnlockError 测试释放锁失败时的行为
func (s *LoopJobTestSuite) TestRunWithUnlockError() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建一个控制测试终止的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 设置模拟行为
	mockClient := loopjobmocks.NewMockClient(ctrl)
	mockLock := loopjobmocks.NewMockLock(ctrl)

	// 1. 创建锁成功
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(mockLock, nil).AnyTimes()
	// 2. 获取锁成功
	mockLock.EXPECT().Lock(gomock.Any()).Return(nil).AnyTimes()
	// 3. 业务执行成功后，续约失败
	mockLock.EXPECT().Refresh(gomock.Any()).Return(errRefresh).AnyTimes()
	// 4. 释放锁失败
	mockLock.EXPECT().Unlock(gomock.Any()).Return(errUnlock).AnyTimes()

	// 使用非常短的重试间隔
	retryInterval := 2 * time.Millisecond
	defaultTimeout := 3 * time.Millisecond
	var bizExecuted atomic.Bool

	// 创建测试完成通知通道
	done := make(chan struct{})

	go func() {
		defer close(done)

		loop := newInfiniteLoop(mockClient,
			func(_ context.Context) error {
				bizExecuted.Store(true)
				return nil
			},
			"test-key",
			retryInterval,
			defaultTimeout)

		loop.Run(ctx)
	}()

	// 等待测试完成或强制超时
	select {
	case <-done:
		// 测试正常完成
	case <-time.After(100 * time.Millisecond):
		t.Log("测试强制结束")
		cancel()
	}

	// 业务逻辑应该被执行
	assert.True(t, bizExecuted.Load(), "业务逻辑应该被执行")
}

// TestRunWithBizError 测试业务逻辑执行失败时的行为
func (s *LoopJobTestSuite) TestRunWithBizError() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建一个控制测试终止的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 设置模拟行为
	mockClient := loopjobmocks.NewMockClient(ctrl)
	mockLock := loopjobmocks.NewMockLock(ctrl)

	// 1. 创建锁成功
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(mockLock, nil).AnyTimes()
	// 2. 获取锁成功
	mockLock.EXPECT().Lock(gomock.Any()).Return(nil).AnyTimes()
	// 3. 续约可能被调用多次
	mockLock.EXPECT().Refresh(gomock.Any()).Return(nil).AnyTimes()
	// 4. 最终释放锁
	mockLock.EXPECT().Unlock(gomock.Any()).Return(nil).AnyTimes()

	// 使用非常短的重试间隔
	retryInterval := 2 * time.Millisecond
	defaultTimeout := 3 * time.Millisecond
	var bizExecCount atomic.Int32

	// 创建测试完成通知通道
	done := make(chan struct{})

	go func() {
		defer close(done)

		loop := newInfiniteLoop(mockClient,
			func(_ context.Context) error {
				bizExecCount.Add(1)
				// 执行几次后取消上下文，模拟业务执行过程中的取消
				if bizExecCount.Load() >= int32(3) {
					cancel()
				}
				return errBizExecution
			},
			"test-key",
			retryInterval,
			defaultTimeout)

		loop.Run(ctx)
	}()

	// 等待测试完成或强制超时
	select {
	case <-done:
		// 测试正常完成
	case <-time.After(100 * time.Millisecond):
		t.Log("测试强制结束")
		cancel()
	}

	// 业务逻辑应该被执行了至少3次
	assert.GreaterOrEqual(t, bizExecCount.Load(), int32(3), "业务逻辑应该至少执行3次")
}

// TestRunWithContextCancel 测试上下文取消时的行为
func (s *LoopJobTestSuite) TestRunWithContextCancel() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建一个控制测试终止的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 设置模拟行为
	mockClient := loopjobmocks.NewMockClient(ctrl)
	mockLock := loopjobmocks.NewMockLock(ctrl)

	// 1. 创建锁成功
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(mockLock, nil).AnyTimes()
	// 2. 获取锁成功
	mockLock.EXPECT().Lock(gomock.Any()).Return(nil).AnyTimes()
	// 3. 续约可能被调用多次
	mockLock.EXPECT().Refresh(gomock.Any()).Return(nil).AnyTimes()
	// 4. 最终释放锁
	mockLock.EXPECT().Unlock(gomock.Any()).Return(nil).AnyTimes()

	// 使用非常短的重试间隔
	retryInterval := 2 * time.Millisecond
	defaultTimeout := 3 * time.Millisecond
	var bizExecCount atomic.Int32

	// 创建测试完成通知通道
	done := make(chan struct{})

	go func() {
		defer close(done)

		loop := newInfiniteLoop(mockClient,
			func(ctx context.Context) error {
				bizExecCount.Add(1)
				if bizExecCount.Load() >= int32(3) {
					cancel()         // 执行几次后取消上下文
					return ctx.Err() // 返回上下文错误
				}
				return nil
			},
			"test-key",
			retryInterval,
			defaultTimeout)

		loop.Run(ctx)
	}()

	// 等待测试完成或强制超时
	select {
	case <-done:
		// 测试正常完成
	case <-time.After(100 * time.Millisecond):
		t.Log("测试强制结束")
		cancel()
	}

	// 业务逻辑应该被执行了至少3次
	assert.GreaterOrEqual(t, bizExecCount.Load(), int32(3), "业务逻辑应该至少执行3次")
}

// TestBizLoop 测试业务循环逻辑的各种场景
func (s *LoopJobTestSuite) TestBizLoop() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name              string                            // 测试名称
		setupMock         func(lock *loopjobmocks.MockLock) // 设置Mock预期行为
		bizFunc           func(ctx context.Context) error   // 业务逻辑函数
		ctxTimeout        time.Duration                     // 上下文超时时间
		expectedExecCount int                               // 预期业务执行次数
		expectedError     error                             // 预期返回的错误
	}{
		{
			name: "业务执行成功，续约成功",
			setupMock: func(lock *loopjobmocks.MockLock) {
				// 由于上下文超时控制，续约可能会被调用多次
				lock.EXPECT().Refresh(gomock.Any()).Return(nil).AnyTimes()
			},
			bizFunc: func(_ context.Context) error {
				return nil
			},
			ctxTimeout:        20 * time.Millisecond,
			expectedExecCount: 1,
			expectedError:     context.DeadlineExceeded,
		},
		{
			name: "业务执行失败，续约成功",
			setupMock: func(lock *loopjobmocks.MockLock) {
				// 由于上下文超时控制，续约可能会被调用多次
				lock.EXPECT().Refresh(gomock.Any()).Return(nil).AnyTimes()
			},
			bizFunc: func(_ context.Context) error {
				return errBizExecution
			},
			ctxTimeout:        20 * time.Millisecond,
			expectedExecCount: 1,
			expectedError:     context.DeadlineExceeded,
		},
		{
			name: "业务执行成功，续约失败",
			setupMock: func(lock *loopjobmocks.MockLock) {
				lock.EXPECT().Refresh(gomock.Any()).Return(errRefresh).AnyTimes()
			},
			bizFunc: func(_ context.Context) error {
				return nil
			},
			ctxTimeout:        20 * time.Millisecond,
			expectedExecCount: 1,
			expectedError:     fmt.Errorf("分布式锁续约失败 %w", errRefresh),
		},
		{
			name: "上下文取消",
			setupMock: func(lock *loopjobmocks.MockLock) {
				// 由于mock创建方式的变化，可能会被调用
				lock.EXPECT().Refresh(gomock.Any()).Return(nil).AnyTimes()
			},
			bizFunc: func(_ context.Context) error {
				// 直接返回上下文取消错误
				return context.Canceled
			},
			ctxTimeout:        20 * time.Millisecond,
			expectedExecCount: 1,
			expectedError:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // 允许子测试并行运行

			// 为每个子测试创建新的Mock
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockLock := loopjobmocks.NewMockLock(mockCtrl)
			mockClient := loopjobmocks.NewMockClient(mockCtrl)

			// 设置锁的Mock行为
			tt.setupMock(mockLock)

			// 创建上下文
			ctx, cancel := context.WithTimeout(t.Context(), tt.ctxTimeout)
			defer cancel()

			// 准备计数器
			var execCount atomic.Int32

			// 创建InfiniteLoop实例
			defaultTimeout := 3 * time.Millisecond
			retryInterval := 1 * time.Millisecond
			loop := newInfiniteLoop(mockClient,
				func(ctx context.Context) error {
					execCount.Add(1)
					return tt.bizFunc(ctx)
				},
				"test-key",
				retryInterval,
				defaultTimeout)

			// 创建测试完成通道
			done := make(chan error)

			// 在goroutine中执行bizLoop
			go func() {
				done <- loop.bizLoop(ctx, mockLock)
			}()

			// 等待bizLoop完成或强制超时
			var err error
			select {
			case err = <-done:
				// bizLoop正常结束了
			case <-time.After(50 * time.Millisecond):
				t.Log("测试超时，强制结束")
				cancel()
				err = <-done
			}

			// 验证结果
			if tt.expectedError != nil {
				if errors.Is(tt.expectedError, context.DeadlineExceeded) ||
					errors.Is(tt.expectedError, context.Canceled) {
					assert.Error(t, err)
					assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
						errors.Is(err, context.Canceled),
						"应该返回上下文错误")
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			assert.GreaterOrEqual(t, execCount.Load(), int32(tt.expectedExecCount), "业务函数执行次数不符合预期")
		})
	}
}

// TestInfiniteLoopWithInterruption 测试快速创建和销毁InfiniteLoop的情况
func (s *LoopJobTestSuite) TestInfiniteLoopWithInterruption() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := loopjobmocks.NewMockClient(ctrl)
	mockLock := loopjobmocks.NewMockLock(ctrl)

	// 设置模拟行为
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(mockLock, nil).AnyTimes()
	mockLock.EXPECT().Lock(gomock.Any()).Return(nil).AnyTimes()
	mockLock.EXPECT().Refresh(gomock.Any()).Return(nil).AnyTimes()
	mockLock.EXPECT().Unlock(gomock.Any()).Return(nil).AnyTimes()

	// 创建和取消多个上下文，测试Loop对上下文取消的处理
	for i := 0; i < 3; i++ {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()

			defaultTimeout := 3 * time.Millisecond
			retryInterval := 2 * time.Millisecond

			// 创建测试完成通知通道
			done := make(chan struct{})

			go func() {
				defer close(done)

				loop := newInfiniteLoop(mockClient,
					func(_ context.Context) error {
						return nil
					},
					"test-key",
					retryInterval,
					defaultTimeout)

				loop.Run(ctx)
			}()

			// 等待测试完成或强制超时
			select {
			case <-done:
				// 测试正常完成
			case <-time.After(50 * time.Millisecond):
				t.Log("测试强制结束")
				cancel()
			}
		}()
	}
}

// TestInfiniteLoopSpecificErrors 测试特定错误情况
func (s *LoopJobTestSuite) TestInfiniteLoopSpecificErrors() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := loopjobmocks.NewMockClient(ctrl)

	// 测试失败后重试
	mockClient.EXPECT().NewLock(gomock.Any(), "test-key", gomock.Any()).
		Return(nil, errors.New("failed to acquire lock")).AnyTimes()

	// 创建带有短超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	defaultTimeout := 3 * time.Millisecond
	retryInterval := 2 * time.Millisecond

	// 创建测试完成通知通道
	done := make(chan struct{})

	go func() {
		defer close(done)

		loop := newInfiniteLoop(mockClient,
			func(_ context.Context) error {
				return nil
			},
			"test-key",
			retryInterval,
			defaultTimeout)

		loop.Run(ctx)
	}()

	// 等待测试完成或强制超时
	select {
	case <-done:
		// 测试正常完成
	case <-time.After(50 * time.Millisecond):
		t.Log("测试强制结束")
		cancel()
	}
}
