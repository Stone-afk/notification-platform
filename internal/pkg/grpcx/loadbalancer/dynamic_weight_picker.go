package loadbalancer

import (
	"context"
	"errors"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"sync"
)

type DynamicWeightPicker struct {
	connections []*dynamicWeightConn
	// mutex sync.Mutex
}

func (p *DynamicWeightPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.connections) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	var totalWeight uint32
	var res *dynamicWeightConn
	for _, c := range p.connections {
		c.mutex.Lock()
		totalWeight += c.efficientWeight
		c.currentWeight += c.efficientWeight
		if res == nil || res.currentWeight < c.currentWeight {
			res = c
		}
		c.mutex.Unlock()
	}
	res.mutex.Lock()
	res.currentWeight -= totalWeight
	res.mutex.Unlock()
	return balancer.PickResult{
		SubConn: res.c,
		Done: func(info balancer.DoneInfo) {
			res.mutex.Lock()
			defer res.mutex.Unlock()
			if info.Err == nil {
				res.efficientWeight++
				// 要注意可以控制上限，假设说不能超过两倍
				const twice = 2
				res.efficientWeight = max(res.efficientWeight, twice*res.weight)
				return
			}

			// 分错误处理
			if errors.Is(info.Err, context.DeadlineExceeded) {
				// 超时了
				res.efficientWeight--
				return
			}

			if errors.Is(info.Err, io.EOF) || errors.Is(info.Err, io.ErrUnexpectedEOF) {
				res.efficientWeight = 1
				return
			}

			// 利用 gRPC 的 status 和 codes 包来传递特殊错误码
			code, _ := status.FromError(info.Err)
			switch code.Code() {
			case codes.Unavailable:
				// 一般用来表达熔断
				res.efficientWeight = 1
			default:
				res.efficientWeight--
			}
		},
	}, nil
}

type DynamicWeightPickerBuilder struct{}

func (w *DynamicWeightPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	cs := make([]*dynamicWeightConn, 0, len(info.ReadySCs))
	for sub, subInfo := range info.ReadySCs {
		weight, _ := subInfo.Address.Attributes.Value("weight").(uint32)
		cs = append(cs, &dynamicWeightConn{
			c:               sub,
			weight:          weight,
			currentWeight:   weight,
			efficientWeight: weight,
		})
	}
	return &DynamicWeightPicker{
		connections: cs,
	}
}

type dynamicWeightConn struct {
	mutex         sync.Mutex
	c             balancer.SubConn
	weight        uint32
	currentWeight uint32
	// 我根据你的调用情况，动态权重
	// 取代 weight 参与平滑加权轮询算法
	efficientWeight uint32
	// 维持住连续超时的次数
}
