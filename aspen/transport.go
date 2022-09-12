package aspen

import (
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
)

type Transport interface {
	Configure(ctx signal.Context, addr address.Address, external bool) error
	Pledge() pledge.Transport
	Cluster() gossip.Transport
	Operations() kv.BatchTransport
	Lease() kv.LeaseTransport
	Feedback() kv.FeedbackTransport
}
