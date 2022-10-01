package mock

import (
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
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

// transport is an in-memory, synchronous implementation of aspen.transport.
type transport struct {
	net        *Network
	pledge     *fmock.UnaryServer[node.ID, node.ID]
	cluster    *fmock.UnaryServer[gossip.Message, gossip.Message]
	operations *fmock.UnaryServer[kv.BatchRequest, kv.BatchRequest]
	lease      *fmock.UnaryServer[kv.BatchRequest, types.Nil]
	feedback   *fmock.UnaryServer[kv.FeedbackMessage, types.Nil]
}

// Configure implements aspen.transport.
func (t *transport) Configure(ctx signal.Context, addr address.Address, external bool) error {
	t.pledge = t.net.pledge.UnaryServer(addr)
	t.cluster = t.net.cluster.UnaryServer(addr)
	t.operations = t.net.operations.UnaryServer(addr)
	t.lease = t.net.lease.UnaryServer(addr)
	t.feedback = t.net.feedback.UnaryServer(addr)
	return nil
}

// Pledge implements aspen.transport.
func (t *transport) Pledge() pledge.Transport { return t.pledge }

// Cluster implements aspen.transport.
func (t *transport) Cluster() gossip.TransportClient { return t.cluster }

// Operations implements aspen.transport.
func (t *transport) Operations() kv.BatchTransport { return t.operations }

// Lease implements aspen.transport.
func (t *transport) Lease() kv.LeaseTransportClient { return t.lease }

// Feedback implements aspen.transport.
func (t *transport) Feedback() kv.FeedbackTransportClient { return t.feedback }
