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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distcontrol "github.com/synnaxlabs/synnax/pkg/distribution/control"
	distframer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
	"google.golang.org/grpc"
)

// Transport bundles the gRPC-backed implementations of the distribution layer's
// node-to-node transports. Construct it with New; the zero value is not usable. It
// implements the distribution.Transport interface.
type Transport struct {
	// ReportProvider provides a report for the transport.
	alamos.ReportProvider
	channel channel.Transport
	framer  framer.Transport
	control control.Transport
}

var (
	_ distribution.Transport  = Transport{}
	_ fgrpc.BindableTransport = Transport{}
)

// New constructs the distribution layer's gRPC transports, opening connections from
// pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		ReportProvider: fgrpc.Reporter,
		channel:        channel.New(pool),
		framer:         framer.New(pool),
		control:        control.New(pool),
	}
}

// Channel implements distribution.Transport.
func (t Transport) Channel() distchannel.Transport { return t.channel }

// Framer implements distribution.Transport.
func (t Transport) Framer() distframer.Transport { return t.framer }

// Control implements distribution.Transport.
func (t Transport) Control() distcontrol.Transport { return t.control }

// Use implements the freighter.Transport interface, binding the given middleware to the
// channel, framer, and control transports.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.channel.Use(middleware...)
	t.framer.Use(middleware...)
	t.control.Use(middleware...)
}

// BindTo implements the grpc.BindableTransport interface, registering the channel,
// framer, and control transports with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	t.channel.BindTo(reg)
	t.framer.BindTo(reg)
	t.control.BindTo(reg)
}
