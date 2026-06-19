// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
)

// Transports bundles the gRPC-backed implementations of the distribution layer's
// node-to-node transports. Construct it with New; the zero value is not usable.
type Transports struct {
	// Channel forwards channel create, rename, and delete operations from a gateway
	// node to the leaseholder that owns the channels.
	Channel channel.Transport
	// Framer forwards frame write, iterate, relay, and delete operations from a gateway
	// node to the leaseholders that own the channels.
	Framer framer.Transport
}

// New constructs the distribution layer's gRPC transports, opening connections from
// pool.
func New(pool *fgrpc.Pool) Transports {
	return Transports{Channel: channel.New(pool), Framer: framer.New(pool)}
}

// BindableTransports returns the transports as a slice for registration with the
// server's gRPC branch.
func (t Transports) BindableTransports() []fgrpc.BindableTransport {
	return []fgrpc.BindableTransport{t.Channel, t.Framer}
}
