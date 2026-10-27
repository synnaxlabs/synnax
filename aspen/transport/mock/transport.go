// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/freighter"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
)

type Network struct {
	pledge     *freightertestutil.Network[pledge.Request, pledge.Response]
	cluster    *freightertestutil.Network[gossip.Message, gossip.Message]
	operations *freightertestutil.Network[kv.TxRequest, kv.TxRequest]
	lease      *freightertestutil.Network[kv.TxRequest, struct{}]
	feedback   *freightertestutil.Network[kv.FeedbackMessage, struct{}]
	recovery   *freightertestutil.Network[kv.RecoveryRequest, kv.RecoveryResponse]
}

func NewNetwork() *Network {
	return &Network{
		pledge:     freightertestutil.NewNetwork[pledge.Request, pledge.Response](),
		cluster:    freightertestutil.NewNetwork[gossip.Message, gossip.Message](),
		operations: freightertestutil.NewNetwork[kv.TxRequest, kv.TxRequest](),
		lease:      freightertestutil.NewNetwork[kv.TxRequest, struct{}](),
		feedback:   freightertestutil.NewNetwork[kv.FeedbackMessage, struct{}](),
		recovery:   freightertestutil.NewNetwork[kv.RecoveryRequest, kv.RecoveryResponse](),
	}
}

// NewTransport constructs a Transport for a host node at addr. Its servers answer the
// requests peers on the Network send to addr.
func (n *Network) NewTransport(addr address.Address) aspen.Transport {
	return &transport{
		pledgeServer:   n.pledge.UnaryServer(addr),
		pledgeClient:   n.pledge.UnaryClient(),
		gossipServer:   n.cluster.UnaryServer(addr),
		gossipClient:   n.cluster.UnaryClient(),
		txServer:       n.operations.UnaryServer(addr),
		txClient:       n.operations.UnaryClient(),
		leaseServer:    n.lease.UnaryServer(addr),
		leaseClient:    n.lease.UnaryClient(),
		feedbackServer: n.feedback.UnaryServer(addr),
		feedbackClient: n.feedback.UnaryClient(),
		recoveryServer: n.recovery.StreamServer(addr),
		recoveryClient: n.recovery.StreamClient(),
	}
}

// transport is an in-memory, synchronous implementation of aspen.transport.
type transport struct {
	pledgeServer   *freightertestutil.UnaryServer[pledge.Request, pledge.Response]
	pledgeClient   *freightertestutil.UnaryClient[pledge.Request, pledge.Response]
	gossipServer   *freightertestutil.UnaryServer[gossip.Message, gossip.Message]
	gossipClient   *freightertestutil.UnaryClient[gossip.Message, gossip.Message]
	txServer       *freightertestutil.UnaryServer[kv.TxRequest, kv.TxRequest]
	txClient       *freightertestutil.UnaryClient[kv.TxRequest, kv.TxRequest]
	leaseServer    *freightertestutil.UnaryServer[kv.TxRequest, struct{}]
	leaseClient    *freightertestutil.UnaryClient[kv.TxRequest, struct{}]
	feedbackServer *freightertestutil.UnaryServer[kv.FeedbackMessage, struct{}]
	feedbackClient *freightertestutil.UnaryClient[kv.FeedbackMessage, struct{}]
	recoveryServer *freightertestutil.StreamServer[kv.RecoveryRequest, kv.RecoveryResponse]
	recoveryClient *freightertestutil.StreamClient[kv.RecoveryRequest, kv.RecoveryResponse]
}

func (t *transport) PledgeClient() pledge.TransportClient { return t.pledgeClient }

func (t *transport) PledgeServer() pledge.TransportServer { return t.pledgeServer }

func (t *transport) GossipClient() gossip.TransportClient { return t.gossipClient }

func (t *transport) GossipServer() gossip.TransportServer { return t.gossipServer }

func (t *transport) TxClient() kv.TxTransportClient { return t.txClient }

func (t *transport) TxServer() kv.TxTransportServer { return t.txServer }

func (t *transport) LeaseClient() kv.LeaseTransportClient { return t.leaseClient }

func (t *transport) LeaseServer() kv.LeaseTransportServer { return t.leaseServer }

func (t *transport) FeedbackClient() kv.FeedbackTransportClient {
	return t.feedbackClient
}

func (t *transport) FeedbackServer() kv.FeedbackTransportServer {
	return t.feedbackServer
}

func (t *transport) RecoveryClient() kv.RecoveryTransportClient {
	return t.recoveryClient
}

func (t *transport) RecoveryServer() kv.RecoveryTransportServer {
	return t.recoveryServer
}

func (t *transport) Use(middleware ...freighter.Middleware) {
	t.pledgeClient.Use(middleware...)
	t.pledgeServer.Use(middleware...)
	t.gossipClient.Use(middleware...)
	t.gossipServer.Use(middleware...)
	t.txClient.Use(middleware...)
	t.txServer.Use(middleware...)
	t.leaseClient.Use(middleware...)
	t.leaseServer.Use(middleware...)
	t.feedbackClient.Use(middleware...)
	t.feedbackServer.Use(middleware...)
	t.recoveryClient.Use(middleware...)
	t.recoveryServer.Use(middleware...)
}

func (t *transport) Report() alamos.Report { return t.pledgeClient.Report() }
