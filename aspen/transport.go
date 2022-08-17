package aspen

import (
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/aspen/internal/kv"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
)

type Transport interface {
	Configure(ctx signal.Context, addr address.Address, external bool) error
	Pledge() pledge.Transport
	Cluster() gossip.Transport
	Operations() kv.BatchTransport
	Lease() kv.LeaseTransport
	Feedback() kv.FeedbackTransport
}
