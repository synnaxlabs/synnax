// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package transport

import (
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/freighter"
)

// Transport aggregates the client and server transports an aspen DB uses to reach its
// peers. The DB binds a handler to every server, so the caller must serve them on a
// network for its peers to reach the host.
type Transport interface {
	freighter.Transport
	// PledgeServer returns the server transport for node pledge requests.
	PledgeServer() pledge.TransportServer
	// PledgeClient returns the client transport for node pledge requests.
	PledgeClient() pledge.TransportClient
	// GossipServer returns the server transport for cluster state gossip.
	GossipServer() gossip.TransportServer
	// GossipClient returns the client transport for cluster state gossip.
	GossipClient() gossip.TransportClient
	// TxServer returns the server transport for KV transactions.
	TxServer() kv.TxTransportServer
	// TxClient returns the client transport for KV transactions.
	TxClient() kv.TxTransportClient
	// LeaseServer returns the server transport for leaseholder operations.
	LeaseServer() kv.LeaseTransportServer
	// LeaseClient returns the client transport for leaseholder operations.
	LeaseClient() kv.LeaseTransportClient
	// FeedbackServer returns the server transport for lease feedback.
	FeedbackServer() kv.FeedbackTransportServer
	// FeedbackClient returns the client transport for lease feedback.
	FeedbackClient() kv.FeedbackTransportClient
	// RecoveryServer returns the server transport for KV recovery.
	RecoveryServer() kv.RecoveryTransportServer
	// RecoveryClient returns the client transport for KV recovery.
	RecoveryClient() kv.RecoveryTransportClient
}
