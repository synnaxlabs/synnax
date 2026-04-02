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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

type Transport interface {
	freighter.Transport
	// Configure prepares the transport for serving (e.g. registers gRPC services).
	// It does not start accepting connections.
	Configure(addr address.Address, ins alamos.Instrumentation, external bool) error
	// Serve starts accepting connections on the configured address. All handlers
	// must be bound before calling Serve to prevent data races.
	Serve() error
	// Close gracefully stops the transport.
	Close() error
	PledgeServer() pledge.TransportServer
	PledgeClient() pledge.TransportClient
	GossipServer() gossip.TransportServer
	GossipClient() gossip.TransportClient
	TxServer() kv.TxTransportServer
	TxClient() kv.TxTransportClient
	LeaseServer() kv.LeaseTransportServer
	LeaseClient() kv.LeaseTransportClient
	FeedbackServer() kv.FeedbackTransportServer
	FeedbackClient() kv.FeedbackTransportClient
	RecoveryServer() kv.RecoveryTransportServer
	RecoveryClient() kv.RecoveryTransportClient
}
