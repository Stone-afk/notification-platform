package loadbalancer

import (
	"errors"
	"github.com/ecodeclub/ekit/slice"
	"golang.org/x/net/context"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"io"
	"sync"
)

type CtxKey string

const (
	readWeightStr         = "read_weight"
	writeWeightStr        = "write_weight"
	groupStr              = "group"
	nodeStr               = "nodeStr"
	RequestType    CtxKey = "requestType"
)

type groupKey struct{}

type rwServiceNode struct {
	mutex                *sync.RWMutex
	conn                 balancer.SubConn
	readWeight           int32
	curReadWeight        int32
	efficientReadWeight  int32
	writeWeight          int32
	curWriteWeight       int32
	efficientWriteWeight int32
	group                string
}

type RWPicker struct {
	nodes []*rwServiceNode
}

func (r *RWPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(r.nodes) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	// 过滤出候选节点
	candidates := slice.FilterMap(r.nodes, func(_ int, src *rwServiceNode) (*rwServiceNode, bool) {
		src.mutex.RLock()
		nodeGroup := src.group
		src.mutex.RUnlock()
		return src, r.getGroup(info.Ctx) == nodeGroup
	})

	var totalWeight int32
	var selectedNode *rwServiceNode
	ctx := info.Ctx
	isWrite := r.isWrite(ctx)
	for _, node := range candidates {
		node.mutex.Lock()
		if isWrite {
			totalWeight += node.efficientWriteWeight
			node.curWriteWeight += node.efficientWriteWeight
			if selectedNode == nil || selectedNode.curWriteWeight < node.curWriteWeight {
				selectedNode = node
			}
		} else {
			totalWeight += node.efficientReadWeight
			node.curReadWeight += node.efficientReadWeight
			if selectedNode == nil || selectedNode.curReadWeight < node.curReadWeight {
				selectedNode = node
			}

		}
		node.mutex.Unlock()
	}

	if selectedNode == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	selectedNode.mutex.Lock()
	if r.isWrite(ctx) {
		selectedNode.curWriteWeight -= totalWeight
	} else {
		selectedNode.curReadWeight -= totalWeight
	}
	selectedNode.mutex.Unlock()
	return balancer.PickResult{
		SubConn: selectedNode.conn,
		Done: func(info balancer.DoneInfo) {
			selectedNode.mutex.Lock()
			defer selectedNode.mutex.Unlock()
			isDecrementError := info.Err != nil && (errors.Is(info.Err, context.DeadlineExceeded) || errors.Is(info.Err, io.EOF))
			if r.isWrite(ctx) {
				if isDecrementError && selectedNode.efficientWriteWeight > 0 {
					selectedNode.efficientWriteWeight--
				} else if info.Err == nil {
					selectedNode.efficientWriteWeight++
				}
			} else {
				if isDecrementError && selectedNode.efficientReadWeight > 0 {
					selectedNode.efficientReadWeight--
				} else if info.Err == nil {
					selectedNode.efficientReadWeight++
				}
			}
		},
	}, nil
}

func (r *RWPicker) isWrite(ctx context.Context) bool {
	val := ctx.Value(RequestType)
	if val == nil {
		return false
	}
	vv, ok := val.(int)
	if !ok {
		return false
	}
	return vv == 1
}

func (r *RWPicker) getGroup(ctx context.Context) string {
	val := ctx.Value(groupKey{})
	if val == nil {
		return ""
	}
	vv, ok := val.(string)
	if !ok {
		return ""
	}
	return vv
}

type WeightPickerBuilder struct {
	// 存储已有的节点，使用SubConn的地址作为键
	nodeCache map[string]*rwServiceNode
	mu        *sync.RWMutex
}

func (w *WeightPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	// 创建一个新的节点列表，用于存储当前可用的节点
	nodes := make([]*rwServiceNode, 0, len(info.ReadySCs))
	// 记录当前批次的SubConn地址，用于后续清理不再存在的节点
	currentConns := make(map[string]struct{})
	w.mu.Lock()
	defer w.mu.Unlock()
	for sub, subInfo := range info.ReadySCs {
		readWeight, ok := subInfo.Address.Attributes.Value(readWeightStr).(int32)
		if !ok {
			continue
		}
		writeWeight, ok := subInfo.Address.Attributes.Value(writeWeightStr).(int32)
		if !ok {
			continue
		}
		group, ok := subInfo.Address.Attributes.Value(groupStr).(string)
		if !ok {
			continue
		}
		nodeName, ok := subInfo.Address.Attributes.Value(nodeStr).(string)
		if !ok {
			continue
		}
		currentConns[nodeName] = struct{}{}
		// 检查缓存中是否存在该节点
		if cachedNode, exists := w.nodeCache[nodeName]; exists {
			// 存在则更新连接和组信息，但保留权重状态
			cachedNode.mutex.Lock()
			cachedNode.group = group
			cachedNode.mutex.Unlock()

			if cachedNode.readWeight != readWeight || cachedNode.writeWeight != writeWeight {
				cachedNode = newRwServiceNode(sub, readWeight, writeWeight, group)
				w.nodeCache[nodeName] = cachedNode
			}
			// 将已有节点添加到当前节点列表
			nodes = append(nodes, cachedNode)
		} else {
			// 不存在则创建新节点
			newNode := newRwServiceNode(sub, readWeight, writeWeight, group)
			// 缓存新节点
			w.nodeCache[nodeName] = newNode
			nodes = append(nodes, newNode)
		}
	}

	// 清理不再存在的节点
	for connKey := range w.nodeCache {
		if _, exists := currentConns[connKey]; !exists {
			delete(w.nodeCache, connKey)
		}
	}

	return &RWPicker{
		nodes: nodes,
	}

}

func NewWeightPickerBuilder() *WeightPickerBuilder {
	return &WeightPickerBuilder{
		nodeCache: make(map[string]*rwServiceNode),
		mu:        &sync.RWMutex{},
	}
}

func WithGroup(ctx context.Context, group string) context.Context {
	return context.WithValue(ctx, groupKey{}, group)
}

func newRwServiceNode(conn balancer.SubConn, readWeight, writeWeight int32, group string) *rwServiceNode {
	return &rwServiceNode{
		mutex:                &sync.RWMutex{},
		conn:                 conn,
		readWeight:           readWeight,
		curReadWeight:        readWeight,
		efficientReadWeight:  readWeight,
		writeWeight:          writeWeight,
		curWriteWeight:       writeWeight,
		efficientWriteWeight: writeWeight,
		group:                group,
	}
}
