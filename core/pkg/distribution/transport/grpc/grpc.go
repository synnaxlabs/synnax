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
	"github.com/synnaxlabs/freighter/grpc"
	dischannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	disframer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
)

// Transports bundles the gRPC-backed implementations of the distribution layer's
// node-to-node transports. Construct it with New; the zero value is not usable. It
// implements the distribution.Transport interface.
type Transports struct {
	// channelTransport forwards channel create, rename, and delete operations from a
	// gateway node to the leaseholder that owns the channels.
	channelTransport channel.Transport
	// framerTransport forwards frame write, iterate, relay, and delete operations from a
	// gateway node to the leaseholders that own the channels.
	framerTransport framer.Transport
}

// New constructs the distribution layer's gRPC transports, opening connections from
// pool.
func New(pool *grpc.Pool) Transports {
	return Transports{channelTransport: channel.New(pool), framerTransport: framer.New(pool)}
}

// Channel implements distribution.Transport.
func (t Transports) Channel() dischannel.Transport { return t.channelTransport }

// Framer implements distribution.Transport.
func (t Transports) Framer() disframer.Transport { return t.framerTransport }

// BindableTransports returns the transports as a slice for registration with the
// server's gRPC branch.
func (t Transports) BindableTransports() []grpc.BindableTransport {
	return []grpc.BindableTransport{t.channelTransport, t.framerTransport}
}
