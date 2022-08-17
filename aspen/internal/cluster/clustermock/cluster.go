package clustermock

import (
	"github.com/arya-analytics/aspen/internal/cluster"
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
)

type Builder struct {
	BaseCfg     cluster.Config
	GossipNet   *fmock.Network[gossip.Message, gossip.Message]
	PledgeNet   *fmock.Network[node.ID, node.ID]
	ClusterAPIs map[node.ID]cluster.Cluster
}

func NewBuilder(baseCfg cluster.Config) *Builder {
	return &Builder{
		BaseCfg:     baseCfg.Merge(cluster.DefaultConfig()),
		GossipNet:   fmock.NewNetwork[gossip.Message, gossip.Message](),
		PledgeNet:   fmock.NewNetwork[node.ID, node.ID](),
		ClusterAPIs: make(map[node.ID]cluster.Cluster),
	}
}

func (b *Builder) New(ctx signal.Context, cfg cluster.Config) (cluster.Cluster,
	error) {
	gossipTransport := b.GossipNet.RouteUnary("")
	pledgeTransport := b.PledgeNet.RouteUnary(gossipTransport.Address)
	cfg.Gossip.Transport = gossipTransport
	cfg.Pledge.Transport = pledgeTransport
	cfg = cfg.Merge(b.BaseCfg)
	clust, err := cluster.Join(ctx, gossipTransport.Address, b.MemberAddresses(), cfg)
	b.ClusterAPIs[clust.Host().ID] = clust
	return clust, err
}

func (b *Builder) MemberAddresses() (memberAddresses []address.Address) {
	for _, api := range b.ClusterAPIs {
		memberAddresses = append(memberAddresses, api.Host().Address)
	}
	return memberAddresses
}
