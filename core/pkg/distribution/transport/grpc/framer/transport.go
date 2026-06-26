// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	distdeleter "github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	distiterator "github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	distrelay "github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	distwriter "github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/writer"
	"google.golang.org/grpc"
)

// Transport is a gRPC-backed implementation of the framer.Transport interface. It
// bundles the per-operation gRPC transports for the deleter, iterator, relay, and
// writer.
type Transport struct {
	// ReportProvider provides a report for the transport.
	alamos.ReportProvider
	deleter  deleter.Transport
	iterator iterator.Transport
	relay    relay.Transport
	writer   writer.Transport
}

var (
	_ framer.Transport        = Transport{}
	_ fgrpc.BindableTransport = Transport{}
)

// New creates a new gRPC Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		ReportProvider: fgrpc.Reporter,
		deleter:        deleter.New(pool),
		iterator:       iterator.New(pool),
		relay:          relay.New(pool),
		writer:         writer.New(pool),
	}
}

// Deleter implements the framer.Transport interface.
func (t Transport) Deleter() distdeleter.Transport { return t.deleter }

// Iterator implements the framer.Transport interface.
func (t Transport) Iterator() distiterator.Transport { return t.iterator }

// Relay implements the framer.Transport interface.
func (t Transport) Relay() distrelay.Transport { return t.relay }

// Writer implements the framer.Transport interface.
func (t Transport) Writer() distwriter.Transport { return t.writer }

// BindTo implements the grpc.BindableTransport interface, registering each
// per-operation server with the given gRPC service registrar.
func (t Transport) BindTo(server grpc.ServiceRegistrar) {
	t.deleter.BindTo(server)
	t.iterator.BindTo(server)
	t.relay.BindTo(server)
	t.writer.BindTo(server)
}

// Use implements the freighter.Transport interface, binding the given middleware to
// every per-operation transport.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.deleter.Use(middleware...)
	t.iterator.Use(middleware...)
	t.relay.Use(middleware...)
	t.writer.Use(middleware...)
}
