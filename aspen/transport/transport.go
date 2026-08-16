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
	"net"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

// Transport aggregates the client and server transports an aspen DB uses to reach its
// peers, and manages the underlying network resources.
type Transport interface {
	freighter.Transport
	// Configure prepares the transport for serving: it registers gRPC services and
	// binds the address. It does not start accepting connections. A non-nil lis is a
	// pre-bound listener to serve on instead of binding addr.
	Configure(
		addr address.Address,
		ins alamos.Instrumentation,
		external bool,
		lis net.Listener,
	) error
	// Address returns the configured address with the bound port substituted. It is
	// only valid after Configure. An address configured with port 0 binds to a port
	// the operating system chooses, so the two differ.
	Address() address.Address
	// Serve starts accepting connections on the bound address. All handlers must be
	// bound before calling Serve to prevent data races.
	Serve() error
	// Close gracefully stops the transport.
	Close() error
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
