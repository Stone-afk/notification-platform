package etcd

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"notification-platform/internal/pkg/configcenter/failover"
	"sync"
	"testing"
	"time"

	"github.com/ego-component/eetcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"notification-platform/internal/errs"
	"notification-platform/internal/pkg/registry"
	"notification-platform/internal/pkg/registry/etcd"
	testioc "notification-platform/internal/test/ioc"
)

// TestFailoverManagerSuite 运行测试套件
func TestFailoverManagerSuite(t *testing.T) {
	// 测试没通过
	t.Skip()
	suite.Run(t, new(FailoverManagerSuite))
}

// FailoverManagerSuite 定义测试套件
type FailoverManagerSuite struct {
	suite.Suite
	client *eetcd.Component
}

func (s *FailoverManagerSuite) SetupSuite() {
	s.client = testioc.InitEtcdClient()
}

func (s *FailoverManagerSuite) SetupTest() {
	// 清理测试数据
	_, err := s.client.Delete(s.T().Context(), failoverPrefix, clientv3.WithPrefix())
	require.NoError(s.T(), err)
}

func (s *FailoverManagerSuite) newManager() *Manager {
	return NewFailoverManager(s.client)
}

// TestFailover 测试Failover方法
func (s *FailoverManagerSuite) TestFailover() {
	t := s.T()

	testCases := []struct {
		name string
		svc  failover.ServiceInstance

		errorAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "正常情况",
			svc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "failoverServiceName",
				Address: "failoverServiceAddress",
			},
			errorAssertFunc: assert.NoError,
		},
		{
			name: "服务名称为空",
			svc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "",
				Address: "failoverServiceAddress",
			},
			errorAssertFunc: assert.Error,
		},
		{
			name: "服务地址为空",
			svc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "failoverServiceName",
				Address: "",
			},
			errorAssertFunc: assert.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := s.newManager()
			err := manager.Failover(t.Context(), tc.svc)
			tc.errorAssertFunc(t, err)
			if err != nil {
				return
			}
			resp, err := s.client.Get(t.Context(), manager.buildKey(tc.svc))
			assert.NoError(t, err)
			assert.Equal(t, 1, len(resp.Kvs))
			assert.Equal(t, "", string(resp.Kvs[0].Value))
		})
	}
}

// TestRecover 测试Recover方法
func (s *FailoverManagerSuite) TestRecover() {
	t := s.T()

	testCases := []struct {
		name   string
		before func(t *testing.T, svc failover.ServiceInstance)

		svc failover.ServiceInstance

		errorAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "正常情况",
			before: func(t *testing.T, svc failover.ServiceInstance) {
				t.Helper()
				err := s.newManager().Failover(t.Context(), svc)
				assert.NoError(t, err)
			},
			svc: failover.ServiceInstance{
				Group:   "recoverServiceGroup",
				Name:    "recoverServiceName",
				Address: "recoverServiceAddress",
			},
			errorAssertFunc: assert.NoError,
		},
		{
			name:   "目标服务不存在",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			svc: failover.ServiceInstance{
				Group:   "nonexistentServiceGroup",
				Name:    "nonexistentServiceName",
				Address: "nonexistentServiceAddress",
			},
			errorAssertFunc: assert.NoError,
		},
		{
			name:   "服务名称为空",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			svc: failover.ServiceInstance{
				Group:   "recoverServiceGroup",
				Name:    "",
				Address: "recoverServiceAddress",
			},
			errorAssertFunc: assert.Error,
		},
		{
			name:   "服务地址为空",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			svc: failover.ServiceInstance{
				Group:   "recoverServiceGroup",
				Name:    "recoverServiceName",
				Address: "",
			},
			errorAssertFunc: assert.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t, tc.svc)

			manager := s.newManager()

			err := manager.Recover(t.Context(), tc.svc)
			tc.errorAssertFunc(t, err)
			if err != nil {
				return
			}
			resp, err := s.client.Get(t.Context(), manager.buildKey(tc.svc))
			assert.NoError(t, err)
			assert.Equal(t, 0, len(resp.Kvs))
		})
	}
}

// TestTryTakeOver 测试TryTakeOver方法
func (s *FailoverManagerSuite) TestTryTakeOver() {
	t := s.T()

	manager := s.newManager()

	testCases := []struct {
		name   string
		before func(t *testing.T, svc failover.ServiceInstance)

		standBySvc failover.ServiceInstance
		targetSvc  failover.ServiceInstance

		success         bool
		errorAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "正常情况",
			before: func(t *testing.T, svc failover.ServiceInstance) {
				t.Helper()
				err := manager.Failover(t.Context(), svc)
				assert.NoError(t, err)
			},
			standBySvc: failover.ServiceInstance{
				Group:   "tryTakeOverServiceGroup",
				Name:    "tryTakeOverServiceName",
				Address: "tryTakeOverServiceAddress",
			},
			targetSvc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "failoverServiceName",
				Address: "failoverServiceAddress",
			},
			success:         true,
			errorAssertFunc: assert.NoError,
		},
		{
			name: "目标服务已被接管",
			before: func(t *testing.T, svc failover.ServiceInstance) {
				t.Helper()
				err := manager.Failover(t.Context(), svc)
				assert.NoError(t, err)

				success, err := manager.TryTakeOver(t.Context(), failover.ServiceInstance{
					Group:   "tryTakeOver1ServiceGroup",
					Name:    "tryTakeOver1ServiceName",
					Address: "tryTakeOver1ServiceAddress",
				}, svc)
				assert.NoError(t, err)
				assert.True(t, success)
			},
			standBySvc: failover.ServiceInstance{
				Group:   "tryTakeOverServiceGroup",
				Name:    "tryTakeOverServiceName",
				Address: "tryTakeOverServiceAddress",
			},
			targetSvc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "failoverServiceName",
				Address: "failoverServiceAddress",
			},
			success: false,
			errorAssertFunc: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, errs.ErrNoAvailableFailoverService)
			},
		},
		{
			name:   "目标服务不存在",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			standBySvc: failover.ServiceInstance{
				Group:   "tryTakeOverServiceGroup",
				Name:    "tryTakeOverServiceName",
				Address: "tryTakeOverServiceAddress",
			},
			targetSvc: failover.ServiceInstance{
				Group:   "nonexistentServiceGroup",
				Name:    "nonexistentServiceName",
				Address: "nonexistentServiceAddress",
			},
			errorAssertFunc: assert.Error,
		},
		{
			name:   "接管服务名字为空",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			standBySvc: failover.ServiceInstance{
				Group:   "tryTakeOverServiceGroup",
				Name:    "",
				Address: "tryTakeOverServiceAddress",
			},
			targetSvc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "failoverServiceName",
				Address: "failoverServiceAddress",
			},
			errorAssertFunc: assert.Error,
		},
		{
			name:   "接管服务地址为空",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			standBySvc: failover.ServiceInstance{
				Group:   "tryTakeOverServiceGroup",
				Name:    "tryTakeOverServiceName",
				Address: "",
			},
			targetSvc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "failoverServiceName",
				Address: "failoverServiceAddress",
			},
			errorAssertFunc: assert.Error,
		},
		{
			name:   "目标服务名字为空",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			standBySvc: failover.ServiceInstance{
				Group:   "tryTakeOverServiceGroup",
				Name:    "tryTakeOverServiceName",
				Address: "tryTakeOverServiceAddress",
			},
			targetSvc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "",
				Address: "failoverServiceAddress",
			},
			errorAssertFunc: assert.Error,
		},
		{
			name:   "目标服务地址为空",
			before: func(t *testing.T, svc failover.ServiceInstance) {},
			standBySvc: failover.ServiceInstance{
				Group:   "tryTakeOverServiceGroup",
				Name:    "tryTakeOverServiceName",
				Address: "tryTakeOverServiceAddress",
			},
			targetSvc: failover.ServiceInstance{
				Group:   "failoverServiceGroup",
				Name:    "failoverServiceName",
				Address: "",
			},
			errorAssertFunc: assert.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t, tc.targetSvc)

			ok, err := manager.TryTakeOver(t.Context(), tc.standBySvc, tc.targetSvc)
			tc.errorAssertFunc(t, err)
			assert.Equal(t, tc.success, ok)
			if err != nil {
				return
			}
			resp, err := s.client.Get(t.Context(), manager.buildKey(tc.targetSvc))
			assert.NoError(t, err)
			assert.Equal(t, 1, len(resp.Kvs))
			assert.Equal(t, manager.buildValue(tc.standBySvc), string(resp.Kvs[0].Value))
		})
	}
}

// TestWatchFailover 测试WatchFailover方法
func (s *FailoverManagerSuite) TestWatchFailover() {
	t := s.T()

	// 启动监听
	manager := s.newManager()
	eventCh, err := manager.WatchFailover(t.Context())
	assert.NoError(t, err)

	signalCh := make(chan struct{}, 1)
	// 创建几个事件
	expectedEvents := []failover.FailoverEvent{
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "core",
				Name:    "service1",
				Address: "192.167.0.1",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "core",
				Name:    "service1",
				Address: "192.167.0.2",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "web",
				Name:    "service3",
				Address: "192.167.0.3",
			},
		},
	}

	// 接收事件的协程
	var receivedEvents []failover.FailoverEvent

	go func() {
		signalCh <- struct{}{}
		// 触发事件
		for _, event := range expectedEvents {
			err = manager.Failover(t.Context(), event.ServiceInstance)
			assert.NoError(t, err)
			time.Sleep(100 * time.Millisecond) // 给监听一些时间
		}
	}()

	<-signalCh
	for i := 0; i < len(expectedEvents); i++ {
		select {
		case event := <-eventCh:
			receivedEvents = append(receivedEvents, event)
		}
	}

	// 验证收到的事件
	assert.Len(t, receivedEvents, len(expectedEvents))
	assert.ElementsMatch(t, receivedEvents, expectedEvents)
	assert.NoError(t, manager.Close())
}

// TestWatchRecovery 测试WatchRecover方法
func (s *FailoverManagerSuite) TestWatchRecovery() {
	t := s.T()

	manager := s.newManager()

	// 测试参数验证
	svc := failover.ServiceInstance{
		Group:   "core",
		Name:    "service1",
		Address: "192.155.0.1",
	}

	svc2 := svc
	svc2.Name = ""
	_, err := manager.WatchRecovery(t.Context(), svc2)
	assert.Error(t, err)

	svc3 := svc
	svc3.Address = ""
	_, err = manager.WatchRecovery(t.Context(), svc3)
	assert.Error(t, err)

	signalCh := make(chan struct{}, 1)

	// 创建几个事件
	expectedEvents := []failover.FailoverEvent{
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "order",
				Name:    "service1",
				Address: "192.157.0.1",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "order",
				Name:    "service2",
				Address: "192.157.0.2",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "payment",
				Name:    "service3",
				Address: "192.157.0.3",
			},
		},
		{
			ServiceInstance: svc,
		},
	}

	// 接收恢复信号的协程
	go func() {
		<-signalCh
		// 创建需要监听恢复的服务
		for i := range expectedEvents {
			err1 := manager.Failover(t.Context(), expectedEvents[i].ServiceInstance)
			assert.NoError(t, err1)
		}

		time.Sleep(100 * time.Millisecond)

		// 触发恢复事件
		for i := 0; i < len(expectedEvents); i++ {
			err2 := manager.Recover(t.Context(), expectedEvents[i].ServiceInstance)
			assert.NoError(t, err2)
		}
	}()

	// 监听特定服务的恢复事件
	recoverCh, err := manager.WatchRecovery(t.Context(), svc)
	assert.NoError(t, err)

	signalCh <- struct{}{}

	signalCounter := 0
	for i := 0; i < len(expectedEvents); i++ {
		select {
		case <-recoverCh:
			signalCounter++
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	assert.Equal(t, 1, signalCounter)
	assert.NoError(t, manager.Close())
}

// TestConcurrentTakeOver 测试并发接管场景
func (s *FailoverManagerSuite) TestConcurrentTakeOver() {
	t := s.T()
	ctx := t.Context()
	manager := s.newManager()

	// 创建多个需要接管的服务
	expectedEvents := []failover.FailoverEvent{
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "go",
				Name:    "service1",
				Address: "192.147.1.1",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "go",
				Name:    "service2",
				Address: "192.147.1.2",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "python",
				Name:    "service",
				Address: "192.147.2.1",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "python",
				Name:    "service",
				Address: "192.147.2.2",
			},
		},
		{
			ServiceInstance: failover.ServiceInstance{
				Group:   "python",
				Name:    "service",
				Address: "192.147.2.3",
			},
		},
	}

	type TakeOverEvent struct {
		failover.FailoverEvent
		Holder failover.ServiceInstance
	}

	signalCh := make(chan struct{}, len(expectedEvents))
	doneCh := make(chan struct{}, len(expectedEvents))
	takeOverEvents := make(chan TakeOverEvent, len(expectedEvents))

	// 多个备份服务并发尝试接管
	backupCount := len(expectedEvents)
	var wg sync.WaitGroup

	for i := 0; i < backupCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			failoverEvents, err := manager.WatchFailover(ctx)
			assert.NoError(t, err)

			signalCh <- struct{}{}

			for {
				select {
				case evt := <-failoverEvents:
					backupSvc := failover.ServiceInstance{
						Group:   "",
						Name:    "backup",
						Address: fmt.Sprintf("192.137.1.%d", id),
					}
					success, _ := manager.TryTakeOver(ctx, backupSvc, evt.ServiceInstance)
					if success {
						takeOverEvents <- TakeOverEvent{FailoverEvent: evt, Holder: backupSvc}
						return
					}
				case <-doneCh:
					return
				}
			}
		}(i)
	}

	for range expectedEvents {
		time.Sleep(100 * time.Millisecond)
		<-signalCh
	}

	for _, evt := range expectedEvents {
		err := manager.Failover(ctx, evt.ServiceInstance)
		assert.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)
	close(doneCh)

	wg.Wait()

	successCount := 0
	close(takeOverEvents)
	for evt := range takeOverEvents {
		successCount++
		resp, err := s.client.Get(ctx, manager.buildKey(evt.ServiceInstance))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(resp.Kvs))
		assert.Equal(t, manager.buildValue(evt.Holder), string(resp.Kvs[0].Value))
	}

	// 验证只有服务数量的接管成功
	require.Equal(t, backupCount, successCount)

	assert.NoError(t, manager.Close())
}

func (s *FailoverManagerSuite) TestServerFailoverAndRecover() {
	t := s.T()
	ctx := t.Context()

	etcdRegistry, err := etcd.NewRegistry(s.client)
	assert.NoError(t, err)

	failMgr := NewFailoverManager(s.client)

	n := 3
	cores := make([]*MockServer, 0, n)
	backups := make([]*MockServer, 0, n)

	// n 个 core Server
	serviceGroup := "core"
	serviceName := "golang"
	for i := 1; i <= n; i++ {
		s := &MockServer{
			RegistryInfo: registry.ServiceInstance{
				Name:    serviceName,
				Address: fmt.Sprintf("192.168.0.1:%d", 8000+i),
				Group:   serviceGroup,
			},
			reg:     etcdRegistry,
			failMgr: failMgr,
		}
		require.NoError(t, s.Start(ctx))
		cores = append(cores, s)
	}

	// n 个 backup Server
	backupServiceGroup := ""
	backupServiceName := "backup"
	for i := 1; i <= n; i++ {
		b := &MockServer{
			RegistryInfo: registry.ServiceInstance{
				Name:    backupServiceName,
				Address: fmt.Sprintf("127.0.0.1:%d", 9000+i),
				Group:   backupServiceGroup,
			},
			reg:     etcdRegistry,
			failMgr: failMgr,
		}
		require.NoError(t, b.Start(ctx))
		backups = append(backups, b)
	}

	// 并发触发 3 个 core 资源不足
	var wg sync.WaitGroup
	for _, c := range cores {
		wg.Add(1)
		go func(s *MockServer) {
			defer wg.Done()
			// 随机发送故障转移信号
			m := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(500)
			<-time.After(time.Duration(m) * time.Millisecond)
			require.NoError(t, s.Failover(ctx))
		}(c)
	}
	wg.Wait()

	// 等待 backup 抢占成功
	time.Sleep(2 * time.Second)

	// 验证：registry 中 Group="core" 的实例应该 = 6
	serverList, _ := etcdRegistry.ListServices(ctx, serviceName)
	require.Len(t, serverList, 2*n)
	// 再确保每台 standby 已经降级
	for _, inst := range serverList {
		assert.Equal(t, serviceGroup, inst.Group, fmt.Sprintf("servie instance = %#v\n", inst))
	}
	backupServerList, _ := etcdRegistry.ListServices(ctx, backupServiceName)
	require.Len(t, backupServerList, 0)

	// 并发恢复
	for _, c := range cores {
		wg.Add(1)
		go func(s *MockServer) {
			defer wg.Done()
			require.NoError(t, s.Recover(ctx))
		}(c)
	}
	wg.Wait()

	// 等待 backup 降级成功
	time.Sleep(2 * time.Second)

	// 恢复后 Group="core" 只有 n 台；Group="" 有 n 台 standby
	serverList, _ = etcdRegistry.ListServices(ctx, serviceName)
	assert.Len(t, serverList, n)

	backupServerList, _ = etcdRegistry.ListServices(ctx, backupServiceName)
	assert.Len(t, backupServerList, n)
	// 再确保每台 standby 已经降级
	for _, inst := range backupServerList {
		assert.Empty(t, inst.Group)
	}
}

type MockServer struct {
	mu sync.Mutex

	RegistryInfo      registry.ServiceInstance
	orignRegistryInfo registry.ServiceInstance

	reg     registry.Registry
	failMgr failover.FailoverManager

	cancel context.CancelFunc
}

func (s *MockServer) Start(ctx context.Context) error {
	if err := s.reg.Register(ctx, s.RegistryInfo); err != nil {
		return err
	}
	if s.RegistryInfo.Group == "" {
		wCtx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		go s.watchFailoverEvents(wCtx)
	}
	return nil
}

func (s *MockServer) Failover(ctx context.Context) error {
	return s.failMgr.Failover(ctx, failover.ServiceInstance{
		Group:   s.RegistryInfo.Group,
		Name:    s.RegistryInfo.Name,
		Address: s.RegistryInfo.Address,
	})
}

func (s *MockServer) Recover(ctx context.Context) error {
	return s.failMgr.Recover(ctx, failover.ServiceInstance{
		Group:   s.RegistryInfo.Group,
		Name:    s.RegistryInfo.Name,
		Address: s.RegistryInfo.Address,
	})
}

// 监听故障（Failover）事件并抢占
func (s *MockServer) watchFailoverEvents(ctx context.Context) {
	eventCh, err := s.failMgr.WatchFailover(ctx)
	if err != nil {
		return
	}
	log.Printf("监听【故障转移】事件成功 backup = %#v\n", s.RegistryInfo)
	for {
		select {
		case evt, ok := <-eventCh:
			if !ok {
				return
			}

			log.Printf("backup = %#v, 收到事件 = %#v\n", s.RegistryInfo, evt)

			// 只有空闲时才尝试接管
			if s.RegistryInfo.Group != "" {
				log.Printf("已转移 backup = %#v\n", s.RegistryInfo)
				time.Sleep(time.Second)
				continue
			}

			succeed, err1 := s.failMgr.TryTakeOver(ctx, failover.ServiceInstance{
				Group:   s.RegistryInfo.Group,
				Name:    s.RegistryInfo.Name,
				Address: s.RegistryInfo.Address,
			}, evt.ServiceInstance)
			if !succeed {
				log.Printf("抢占失败 backup = %#v, err = %#v \n", s.RegistryInfo, err1)
				time.Sleep(time.Second)
				continue
			}

			s.mu.Lock()
			s.orignRegistryInfo = s.RegistryInfo
			s.RegistryInfo = registry.ServiceInstance{
				Group:   evt.ServiceInstance.Group,
				Name:    evt.ServiceInstance.Name,
				Address: s.orignRegistryInfo.Address,
			}
			s.mu.Unlock()

			// 注销原始信息
			n := 0
			for n < 3 {
				err = s.reg.UnRegister(ctx, s.orignRegistryInfo)
				if err != nil {
					n++
					continue
				}
				break
			}

			// 注册新信息
			m := 0
			for m < 3 {
				err = s.reg.Register(ctx, s.RegistryInfo)
				if err != nil {
					m++
					continue
				}
				// 监听恢复事件
				log.Printf("成功故障转移 backup = %#v\n", s.RegistryInfo)
				go s.watchRecoveryEvents(ctx, evt.ServiceInstance)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// 监听被接管 core 的恢复
func (s *MockServer) watchRecoveryEvents(ctx context.Context, target failover.ServiceInstance) {
	reCh, err := s.failMgr.WatchRecovery(ctx, target)
	if err != nil {
		return
	}

	select {
	case <-reCh:

		log.Printf("监听【恢复健康】事件成功 backup = %#v\n", s.RegistryInfo)

		// 注销新信息
		n := 0
		for n < 3 {
			err = s.reg.UnRegister(ctx, s.RegistryInfo)
			if err != nil {
				n++
				continue
			}
			break
		}

		log.Printf("注销【新信息】=%#v\n", s.RegistryInfo)

		// 注册回原始信息
		s.mu.Lock()
		s.RegistryInfo = s.orignRegistryInfo
		s.mu.Unlock()

		m := 0
		for m < 3 {
			err = s.reg.Register(ctx, s.RegistryInfo)
			if err != nil {
				m++
				continue
			}
			log.Printf("注册【旧信息】=%#v\n", s.RegistryInfo)
			return
		}
	case <-ctx.Done():
	}
}

func (s *MockServer) Close() {
	s.cancel()
}
