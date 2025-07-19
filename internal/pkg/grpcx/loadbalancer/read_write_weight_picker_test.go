package loadbalancer

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"io"
	"sync"
	"testing"
)

func TestRWPicker_isWrite(t *testing.T) {
	bala := &RWPicker{}
	t.Parallel()
	// 测试用例
	testCases := []struct {
		name     string
		context  context.Context
		expected bool
	}{
		{
			name:     "空值",
			context:  t.Context(),
			expected: false,
		},
		{
			name:     "非整数值",
			context:  context.WithValue(t.Context(), RequestType, "not an int"),
			expected: false,
		},
		{
			name:     "整数 0 (读操作)",
			context:  context.WithValue(t.Context(), RequestType, 0),
			expected: false,
		},
		{
			name:     "整数 1 (写操作)",
			context:  context.WithValue(t.Context(), RequestType, 1),
			expected: true,
		},
		{
			name:     "整数 2 (非写操作)",
			context:  context.WithValue(t.Context(), RequestType, 2),
			expected: false,
		},
	}

	// 运行测试用例
	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := bala.isWrite(tc.context)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// mockSubConn 是 balancer.SubConn 的测试实现
type mockSubConn struct {
	balancer.SubConn
	name string
}

func (m *mockSubConn) Name() string {
	return m.name
}

// createTestNode 创建一个具有指定权重和连接的测试 rwServiceNode
func createTestNode(name string, readWeight, writeWeight int32) *rwServiceNode {
	return &rwServiceNode{
		conn:                 &mockSubConn{name: name},
		mutex:                &sync.RWMutex{},
		readWeight:           readWeight,
		curReadWeight:        readWeight,
		efficientReadWeight:  readWeight,
		writeWeight:          writeWeight,
		curWriteWeight:       writeWeight,
		efficientWriteWeight: writeWeight,
	}
}

func TestRWBalancer_Pick(t *testing.T) {
	t.Parallel()
	// 为了验证测试功能，先手动创建RWBalancer实例，以便可以预测权重变化
	nodes := []*rwServiceNode{
		createTestNode("weight-4", 1, 4), // 低读权重，高写权重
		createTestNode("weight-3", 2, 3), // 中低读权重，中高写权重
		createTestNode("weight-2", 3, 2), // 中高读权重，中低写权重
		createTestNode("weight-1", 4, 1), // 高读权重，低写权重
	}

	// 基于RWBalancer实现的预期行为
	operations := []struct {
		requestType  int    // 0 = 读，1 = 写
		expectedName string // 预期选择的节点名称
		err          error  // 在 DoneInfo 中返回的错误
		description  string // 此操作的描述
	}{
		{0, "weight-1", nil, "第一次读操作选择weight-1（初始读权重最高）"},
		{1, "weight-4", context.DeadlineExceeded, "第一次写操作选择weight-4（初始写权重最高）"},
		{0, "weight-2", nil, "第二次读操作选择weight-2（weight-1的curReadWeight在第一次被减少）"},
		{1, "weight-3", io.EOF, "第二次写操作选择weight-3（weight-4的curWriteWeight在第一次被减少）"},
		{0, "weight-3", nil, "第三次读操作选择weight-3（之前节点的curReadWeight较低）"},
		{1, "weight-2", nil, "第三次写操作选择weight-2（之前节点的curWriteWeight较低）"},
		{0, "weight-1", nil, "第四次读操作选择weight-1（完成轮询回到第一个）"},
		{1, "weight-4", nil, "第四次写操作选择weight-4（完成轮询回到第一个）"},
		{0, "weight-2", nil, "第五次读操作选择weight-2（继续轮询）"},
		{1, "weight-3", nil, "第五次写操作选择weight-3（继续轮询）"},
	}

	b := &RWPicker{nodes: nodes}

	// 顺序执行所有操作以验证负载均衡
	for i := range operations {
		op := operations[i]
		// 打印当前操作描述
		t.Logf("执行操作 %d: %s", i, op.description)

		// 打印权重，便于调试
		t.Logf("== 操作 %d 前权重 ==", i)
		for _, node := range nodes {
			conn, _ := node.conn.(*mockSubConn)
			t.Logf("Node %s: readWeight=%d, efficientReadWeight=%d, curReadWeight=%d, writeWeight=%d, efficientWriteWeight=%d, curWriteWeight=%d",
				conn.Name(), node.readWeight, node.efficientReadWeight, node.curReadWeight,
				node.writeWeight, node.efficientWriteWeight, node.curWriteWeight)
		}

		// 创建具有适当请求类型的上下文
		ctx := context.WithValue(t.Context(), RequestType, op.requestType)

		// 执行 Pick 方法
		pickRes, err := b.Pick(balancer.PickInfo{Ctx: ctx})

		// 验证在选择过程中没有发生错误
		require.NoError(t, err, "操作 %d (%s) 在 Pick 期间不应失败", i, op.description)

		// 验证选择了正确的节点
		selectedConn, ok := pickRes.SubConn.(*mockSubConn)
		require.True(t, ok, "操作 %d (%s): SubConn 应该是 mockSubConn 类型", i, op.description)
		assert.Equal(t, op.expectedName, selectedConn.Name(), "操作 %d (%s): 选择了错误的节点", i, op.description)

		// 调用 Done 来模拟操作完成，带有指定的错误
		pickRes.Done(balancer.DoneInfo{Err: op.err})

		// 打印操作后的权重
		t.Logf("== 操作 %d 后权重 ==", i)
		for _, node := range nodes {
			conn, _ := node.conn.(*mockSubConn)
			t.Logf("Node %s: readWeight=%d, efficientReadWeight=%d, curReadWeight=%d, writeWeight=%d, efficientWriteWeight=%d, curWriteWeight=%d",
				conn.Name(), node.readWeight, node.efficientReadWeight, node.curReadWeight,
				node.writeWeight, node.efficientWriteWeight, node.curWriteWeight)
		}
	}
}

func TestWeightBalancerBuilder_Build(t *testing.T) {
	t.Parallel()

	// 测试用例
	testCases := []struct {
		name            string
		readySCs        map[balancer.SubConn]base.SubConnInfo
		expectedNodes   int
		expectedReadWt  map[string]int32
		expectedWriteWt map[string]int32
		expectedGroups  map[string]string
	}{
		{
			name: "正常构建多个节点",
			readySCs: map[balancer.SubConn]base.SubConnInfo{
				&mockSubConn{name: "node1"}: {
					Address: resolver.Address{
						Addr: "addr1",
						Attributes: attributes.New(readWeightStr, int32(5)).
							WithValue(writeWeightStr, int32(3)).
							WithValue(groupStr, "group1").
							WithValue(nodeStr, "node1"),
					},
				},
				&mockSubConn{name: "node2"}: {
					Address: resolver.Address{
						Addr: "addr2",
						Attributes: attributes.New(readWeightStr, int32(2)).
							WithValue(writeWeightStr, int32(7)).
							WithValue(groupStr, "group2").
							WithValue(nodeStr, "node2"),
					},
				},
			},
			expectedNodes:   2,
			expectedReadWt:  map[string]int32{"node1": 5, "node2": 2},
			expectedWriteWt: map[string]int32{"node1": 3, "node2": 7},
			expectedGroups:  map[string]string{"node1": "group1", "node2": "group2"},
		},
		{
			name: "缺少属性的节点被忽略",
			readySCs: map[balancer.SubConn]base.SubConnInfo{
				&mockSubConn{name: "node1"}: {
					Address: resolver.Address{
						Addr: "addr1",
						Attributes: attributes.New(readWeightStr, int32(5)).
							WithValue(writeWeightStr, int32(3)).
							WithValue(groupStr, "group1").
							WithValue(nodeStr, "node1"),
					},
				},
				&mockSubConn{name: "missing-read-weight"}: {
					Address: resolver.Address{
						Addr: "addr2",
						Attributes: attributes.New(writeWeightStr, int32(7)).
							WithValue(groupStr, "group2").
							WithValue(nodeStr, "missing-read-weight"),
						// 缺少readWeightStr
					},
				},
				&mockSubConn{name: "missing-write-weight"}: {
					Address: resolver.Address{
						Addr: "addr3",
						Attributes: attributes.New(readWeightStr, int32(4)).
							WithValue(groupStr, "group3").
							WithValue(nodeStr, "missing-write-weight"),
						// 缺少writeWeightStr
					},
				},
				&mockSubConn{name: "missing-group"}: {
					Address: resolver.Address{
						Addr: "addr4",
						Attributes: attributes.New(readWeightStr, int32(6)).
							WithValue(writeWeightStr, int32(2)).
							WithValue(nodeStr, "missing-group"),
						// 缺少groupStr
					},
				},
				&mockSubConn{name: "missing-node"}: {
					Address: resolver.Address{
						Addr: "addr5",
						Attributes: attributes.New(readWeightStr, int32(3)).
							WithValue(writeWeightStr, int32(8)).
							WithValue(groupStr, "group5"),
						// 缺少nodeStr
					},
				},
			},
			expectedNodes:   1,
			expectedReadWt:  map[string]int32{"node1": 5},
			expectedWriteWt: map[string]int32{"node1": 3},
			expectedGroups:  map[string]string{"node1": "group1"},
		},
		{
			name:            "空的readySCs",
			readySCs:        map[balancer.SubConn]base.SubConnInfo{},
			expectedNodes:   0,
			expectedReadWt:  map[string]int32{},
			expectedWriteWt: map[string]int32{},
			expectedGroups:  map[string]string{},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 模拟base.PickerBuildInfo
			buildInfo := base.PickerBuildInfo{
				ReadySCs: tc.readySCs,
			}
			// 创建WeightPickerBuilder实例
			builder := NewWeightPickerBuilder()

			// 执行Build方法
			picker := builder.Build(buildInfo)

			// 验证返回的是RWBalancer类型
			rwBalancer, ok := picker.(*RWPicker)
			require.True(t, ok, "返回的picker应该是*RWBalancer类型")

			// 验证节点数量
			assert.Equal(t, tc.expectedNodes, len(rwBalancer.nodes), "节点数量不匹配")

			// 验证每个节点的权重和组信息
			for _, node := range rwBalancer.nodes {
				conn, ok := node.conn.(*mockSubConn)
				require.True(t, ok)
				nodeName := conn.Name()

				// 验证读权重
				expectedReadWt, exists := tc.expectedReadWt[nodeName]
				require.True(t, exists, "找不到节点 %s 的预期读权重", nodeName)
				assert.Equal(t, expectedReadWt, node.readWeight, "节点 %s 的读权重不匹配", nodeName)
				assert.Equal(t, expectedReadWt, node.curReadWeight, "节点 %s 的当前读权重不匹配", nodeName)
				assert.Equal(t, expectedReadWt, node.efficientReadWeight, "节点 %s 的高效读权重不匹配", nodeName)

				// 验证写权重
				expectedWriteWt, exists := tc.expectedWriteWt[nodeName]
				require.True(t, exists, "找不到节点 %s 的预期写权重", nodeName)
				assert.Equal(t, expectedWriteWt, node.writeWeight, "节点 %s 的写权重不匹配", nodeName)
				assert.Equal(t, expectedWriteWt, node.curWriteWeight, "节点 %s 的当前写权重不匹配", nodeName)
				assert.Equal(t, expectedWriteWt, node.efficientWriteWeight, "节点 %s 的高效写权重不匹配", nodeName)

				// 验证组信息
				expectedGroup, exists := tc.expectedGroups[nodeName]
				require.True(t, exists, "找不到节点 %s 的预期组信息", nodeName)
				assert.Equal(t, expectedGroup, node.group, "节点 %s 的组信息不匹配", nodeName)
			}
		})
	}
}
