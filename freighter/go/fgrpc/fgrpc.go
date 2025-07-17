// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fgrpc

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"google.golang.org/grpc"
)

// BindableTransport is a transport that can be bound to a gRPC service registrar.
type BindableTransport interface {
	freighter.Transport
	// BindTo binds the transport to the given gRPC service registrar.
	BindTo(grpc.ServiceRegistrar)
}

type CompoundBindableTransport []BindableTransport

var _ BindableTransport = CompoundBindableTransport{}

func (cbt CompoundBindableTransport) Use(middleware ...freighter.Middleware) {
	for _, t := range cbt {
		t.Use(middleware...)
	}
}

func (cbt CompoundBindableTransport) Report() alamos.Report { return cbt[0].Report() }

func (cbt CompoundBindableTransport) BindTo(reg grpc.ServiceRegistrar) {
	for _, t := range cbt {
		t.BindTo(reg)
	}
}

var Reporter = freighter.Reporter{
	Protocol:  "grpc",
	Encodings: []string{"protobuf"},
}
