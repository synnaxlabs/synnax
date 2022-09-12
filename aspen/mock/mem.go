package mock

import (
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/transport/mock"
	"time"
)

func NewMemBuilder(defaultOpts ...aspen.Option) *Builder {
	propConfig := aspen.PropagationConfig{
		PledgeRetryInterval:   10 * time.Millisecond,
		PledgeRetryScale:      1,
		ClusterGossipInterval: 50 * time.Millisecond,
		KVGossipInterval:      50 * time.Millisecond,
	}
	return &Builder{
		DefaultOptions: append([]aspen.Option{
			aspen.WithTransport(mock.NewNetwork().NewTransport()),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(propConfig),
		}, defaultOpts...),
		Nodes:     make(map[aspen.NodeID]NodeInfo),
		memBacked: true,
	}
}
