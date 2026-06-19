// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"go/types"

	"github.com/synnaxlabs/freighter"
)

// CreateMessage is the cluster-internal allocation request/response. It carries the
// channels to allocate local keys for and create storage for on the leaseholder, and
// returns those channels with their assigned keys.
type CreateMessage struct {
	// Channels is the set of channels being allocated. On a request it holds the
	// channels to create; on the response it holds those same channels populated with
	// their assigned keys.
	Channels []Channel
}

// RenameRequest is the cluster-internal request to rename channels on their leaseholder.
// Keys and Names are parallel: the channel identified by Keys[i] is renamed to Names[i],
// so the two slices must have equal length.
type RenameRequest struct {
	// Keys identifies the channels to rename, parallel to Names.
	Keys Keys
	// Names holds the new name for each channel, parallel to Keys.
	Names []string
}

// DeleteRequest is the cluster-internal request to delete channels on their leaseholder.
type DeleteRequest struct {
	// Keys identifies the channels to delete.
	Keys Keys
}

type (
	// CreateTransportClient sends channel allocation requests to a leaseholder and
	// receives the created channels, populated with their keys, in response.
	CreateTransportClient = freighter.UnaryClient[CreateMessage, CreateMessage]
	// CreateTransportServer handles channel allocation requests on the leaseholder,
	// returning the created channels populated with their keys.
	CreateTransportServer = freighter.UnaryServer[CreateMessage, CreateMessage]
	// DeleteTransportClient sends channel deletion requests to a leaseholder.
	DeleteTransportClient = freighter.UnaryClient[DeleteRequest, types.Nil]
	// DeleteTransportServer handles channel deletion requests on the leaseholder.
	DeleteTransportServer = freighter.UnaryServer[DeleteRequest, types.Nil]
	// RenameTransportServer handles channel rename requests on the leaseholder.
	RenameTransportServer = freighter.UnaryServer[RenameRequest, types.Nil]
	// RenameTransportClient sends channel rename requests to a leaseholder.
	RenameTransportClient = freighter.UnaryClient[RenameRequest, types.Nil]
)

// Transport bundles the unary client and server endpoints used to forward channel
// create, rename, and delete operations from a gateway node to the leaseholder that owns
// the channels. Implementations bind these endpoints over a concrete protocol (e.g.
// gRPC, or an in-memory mock for tests).
type Transport interface {
	// CreateClient returns the client used to send allocation requests to a leaseholder.
	CreateClient() CreateTransportClient
	// CreateServer returns the server that handles allocation requests on this node.
	CreateServer() CreateTransportServer
	// DeleteClient returns the client used to send deletion requests to a leaseholder.
	DeleteClient() DeleteTransportClient
	// DeleteServer returns the server that handles deletion requests on this node.
	DeleteServer() DeleteTransportServer
	// RenameClient returns the client used to send rename requests to a leaseholder.
	RenameClient() RenameTransportClient
	// RenameServer returns the server that handles rename requests on this node.
	RenameServer() RenameTransportServer
}
