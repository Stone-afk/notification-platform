package loadbalancer

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

type SmoothWeightPicker struct {
	connections []*weightConn
	// mutex       sync.Mutex
	totalWeight uint32
}

func (p *SmoothWeightPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.connections) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	// w.mutex.Lock()
	// defer w.mutex.Unlock()

	res := p.connections[0]
	for _, c := range p.connections {
		// c.mutex
		c.currentWeight = c.currentWeight + c.weight
		// 取 < 则是相同权重取第一个
		// <= 则是取最后一个
		if res.currentWeight <= c.currentWeight {
			res = c
		}
	}

	res.currentWeight -= p.totalWeight
	return balancer.PickResult{
		SubConn: res.c,
		Done: func(_ balancer.DoneInfo) {
			// 调用结果
			// info.Err => 你发起调用有没有出错
			// 根据调用结果调整权重
			//switch info.Err {
			//case nil:
			//	res.weight += res.weight
			//case context.DeadlineExceeded:
			//	res.weight -= res.weight
			//case io.EOF:
			//	// 这个节点不可用
			//	res.weight = 0
			//}
		},
	}, nil
}

type SmoothWeightPickerBuilder struct{}

func (w *SmoothWeightPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	cs := make([]*weightConn, 0, len(info.ReadySCs))
	totalWight := uint32(0)
	for sub, subInfo := range info.ReadySCs {
		weight, _ := subInfo.Address.Attributes.Value("weight").(uint32)
		totalWight += weight
		cs = append(cs, &weightConn{
			c:             sub,
			weight:        weight,
			currentWeight: weight,
		})
	}
	return &SmoothWeightPicker{
		totalWeight: totalWight,
		connections: cs,
	}
}

type weightConn struct {
	// mutex         sync.Mutex
	c             balancer.SubConn
	weight        uint32
	currentWeight uint32
}
