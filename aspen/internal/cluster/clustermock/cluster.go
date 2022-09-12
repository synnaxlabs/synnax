package clustermock

import (
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
)

type Builder struct {
	Configs     []cluster.Config
	GossipNet   *fmock.Network[gossip.Message, gossip.Message]
	PledgeNet   *fmock.Network[node.ID, node.ID]
	ClusterAPIs map[node.ID]cluster.Cluster
}

func NewBuilder(cfgs ...cluster.Config) *Builder {
	return &Builder{
		Configs:     cfgs,
		GossipNet:   fmock.NewNetwork[gossip.Message, gossip.Message](),
		PledgeNet:   fmock.NewNetwork[node.ID, node.ID](),
		ClusterAPIs: make(map[node.ID]cluster.Cluster),
	}
}

func (b *Builder) New(ctx signal.Context, cfgs ...cluster.Config) (cluster.Cluster, error) {
	gossipTransport := b.GossipNet.RouteUnary("")
	pledgeTransport := b.PledgeNet.RouteUnary(gossipTransport.Address)
	cfgs = append(b.Configs, cfgs...)
	cfgs = append(cfgs, cluster.Config{
		HostAddress: gossipTransport.Address,
		Gossip:      gossip.Config{Transport: gossipTransport},
		Pledge: pledge.Config{
			Transport: pledgeTransport,
			Peers:     b.MemberAddresses(),
		},
	})
	clust, err := cluster.Join(ctx, cfgs...)
	if err != nil {
		return nil, err
	}
	b.ClusterAPIs[clust.Host().ID] = clust
	return clust, err
}

func (b *Builder) MemberAddresses() (memberAddresses []address.Address) {
	for _, api := range b.ClusterAPIs {
		memberAddresses = append(memberAddresses, api.Host().Address)
	}
	return memberAddresses
}
