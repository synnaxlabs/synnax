package mock

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/aspen/internal/kv"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
	"go/types"
)

type Network struct {
	pledge     *fmock.Network[node.ID, node.ID]
	cluster    *fmock.Network[gossip.Message, gossip.Message]
	operations *fmock.Network[kv.BatchRequest, kv.BatchRequest]
	lease      *fmock.Network[kv.BatchRequest, types.Nil]
	feedback   *fmock.Network[kv.FeedbackMessage, types.Nil]
}

func NewNetwork() *Network {
	return &Network{
		pledge:     fmock.NewNetwork[node.ID, node.ID](),
		cluster:    fmock.NewNetwork[gossip.Message, gossip.Message](),
		operations: fmock.NewNetwork[kv.BatchRequest, kv.BatchRequest](),
		lease:      fmock.NewNetwork[kv.BatchRequest, types.Nil](),
		feedback:   fmock.NewNetwork[kv.FeedbackMessage, types.Nil](),
	}
}

func (n *Network) NewTransport() aspen.Transport { return &transport{net: n} }

// transport is an in-memory, synchronous implementation of aspen.Transport.
type transport struct {
	net        *Network
	pledge     *fmock.Unary[node.ID, node.ID]
	cluster    *fmock.Unary[gossip.Message, gossip.Message]
	operations *fmock.Unary[kv.BatchRequest, kv.BatchRequest]
	lease      *fmock.Unary[kv.BatchRequest, types.Nil]
	feedback   *fmock.Unary[kv.FeedbackMessage, types.Nil]
}

// Configure implements aspen.Transport.
func (t *transport) Configure(ctx signal.Context, addr address.Address, external bool) error {
	t.pledge = t.net.pledge.RouteUnary(addr)
	t.cluster = t.net.cluster.RouteUnary(addr)
	t.operations = t.net.operations.RouteUnary(addr)
	t.lease = t.net.lease.RouteUnary(addr)
	t.feedback = t.net.feedback.RouteUnary(addr)
	return nil
}

// Pledge implements aspen.Transport.
func (t *transport) Pledge() pledge.Transport { return t.pledge }

// Cluster implements aspen.Transport.
func (t *transport) Cluster() gossip.Transport { return t.cluster }

// Operations implements aspen.Transport.
func (t *transport) Operations() kv.BatchTransport { return t.operations }

// Lease implements aspen.Transport.
func (t *transport) Lease() kv.LeaseTransport { return t.lease }

// Feedback implements aspen.Transport.
func (t *transport) Feedback() kv.FeedbackTransport { return t.feedback }
