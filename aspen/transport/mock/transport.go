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
	"go/types"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/x/address"
)

type Network struct {
	pledge     *mock.Network[pledge.Request, pledge.Response]
	cluster    *mock.Network[gossip.Message, gossip.Message]
	operations *mock.Network[kv.TxRequest, kv.TxRequest]
	lease      *mock.Network[kv.TxRequest, types.Nil]
	feedback   *mock.Network[kv.FeedbackMessage, types.Nil]
	recovery   *mock.Network[kv.RecoveryRequest, kv.RecoveryResponse]
}

func NewNetwork() *Network {
	return &Network{
		pledge:     mock.NewNetwork[pledge.Request, pledge.Response](),
		cluster:    mock.NewNetwork[gossip.Message, gossip.Message](),
		operations: mock.NewNetwork[kv.TxRequest, kv.TxRequest](),
		lease:      mock.NewNetwork[kv.TxRequest, types.Nil](),
		feedback:   mock.NewNetwork[kv.FeedbackMessage, types.Nil](),
		recovery:   mock.NewNetwork[kv.RecoveryRequest, kv.RecoveryResponse](),
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
	pledgeServer   *mock.UnaryServer[pledge.Request, pledge.Response]
	pledgeClient   *mock.UnaryClient[pledge.Request, pledge.Response]
	gossipServer   *mock.UnaryServer[gossip.Message, gossip.Message]
	gossipClient   *mock.UnaryClient[gossip.Message, gossip.Message]
	txServer       *mock.UnaryServer[kv.TxRequest, kv.TxRequest]
	txClient       *mock.UnaryClient[kv.TxRequest, kv.TxRequest]
	leaseServer    *mock.UnaryServer[kv.TxRequest, types.Nil]
	leaseClient    *mock.UnaryClient[kv.TxRequest, types.Nil]
	feedbackServer *mock.UnaryServer[kv.FeedbackMessage, types.Nil]
	feedbackClient *mock.UnaryClient[kv.FeedbackMessage, types.Nil]
	recoveryServer *mock.StreamServer[kv.RecoveryRequest, kv.RecoveryResponse]
	recoveryClient *mock.StreamClient[kv.RecoveryRequest, kv.RecoveryResponse]
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
