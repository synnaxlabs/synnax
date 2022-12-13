package transport

import (
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
)

type Transport interface {
	Configure(ctx signal.Context, addr address.Address, external bool) error
	PledgeServer() pledge.TransportServer
	PledgeClient() pledge.TransportClient
	GossipServer() gossip.TransportServer
	GossipClient() gossip.TransportClient
	BatchServer() kv.BatchTransportServer
	BatchClient() kv.BatchTransportClient
	LeaseServer() kv.LeaseTransportServer
	LeaseClient() kv.LeaseTransportClient
	FeedbackServer() kv.FeedbackTransportServer
	FeedbackClient() kv.FeedbackTransportClient
}
