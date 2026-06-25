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

type (
	// CreateTransportClient issues channel create requests to the leaseholder node and
	// returns the created channels.
	CreateTransportClient = freighter.UnaryClient[CreateMessage, CreateMessage]
	// CreateTransportServer handles channel create requests forwarded from a gateway
	// node.
	CreateTransportServer = freighter.UnaryServer[CreateMessage, CreateMessage]
	// DeleteTransportClient issues channel delete requests to the leaseholder node.
	DeleteTransportClient = freighter.UnaryClient[DeleteRequest, types.Nil]
	// DeleteTransportServer handles channel delete requests forwarded from a gateway
	// node.
	DeleteTransportServer = freighter.UnaryServer[DeleteRequest, types.Nil]
	// RenameTransportServer handles channel rename requests forwarded from a gateway
	// node.
	RenameTransportServer = freighter.UnaryServer[RenameRequest, types.Nil]
	// RenameTransportClient issues channel rename requests to the leaseholder node.
	RenameTransportClient = freighter.UnaryClient[RenameRequest, types.Nil]
)

// Transport bundles the node-to-node transports used to forward channel create, delete,
// and rename operations from a gateway node to the leaseholder that owns the affected
// channels. Implementations are provided for gRPC (production) and in-memory (testing).
type Transport interface {
	// CreateClient returns the client used to issue create requests to a leaseholder.
	CreateClient() CreateTransportClient
	// CreateServer returns the server that handles incoming create requests.
	CreateServer() CreateTransportServer
	// DeleteClient returns the client used to issue delete requests to a leaseholder.
	DeleteClient() DeleteTransportClient
	// DeleteServer returns the server that handles incoming delete requests.
	DeleteServer() DeleteTransportServer
	// RenameClient returns the client used to issue rename requests to a leaseholder.
	RenameClient() RenameTransportClient
	// RenameServer returns the server that handles incoming rename requests.
	RenameServer() RenameTransportServer
}

// CreateMessage is the request and response payload for a channel create operation. As a
// request it carries the channels to create; as a response it carries those same
// channels populated with their assigned keys.
type CreateMessage struct {
	// Channels are the channels to create on the request, and the created channels with
	// their assigned keys on the response.
	Channels []Channel
	// Opts controls create behavior when a channel with the same name already exists.
	Opts CreateOptions
}

// DeleteRequest is the payload for a channel delete operation.
type DeleteRequest struct {
	// Keys identifies the channels to delete.
	Keys Keys
}

// RenameRequest is the payload for a channel rename operation. Keys and Names are
// positional: the channel at Keys[i] is renamed to Names[i], so both slices must have
// the same length.
type RenameRequest struct {
	// Keys identifies the channels to rename.
	Keys Keys
	// Names holds the new name for each channel in Keys, by position.
	Names []string
}
