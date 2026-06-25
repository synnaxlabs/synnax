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

// CreateMessage is the request and response payload for a channel create operation. As
// a request it carries the channels to create; as a response it carries those same
// channels populated with their assigned keys.
type CreateMessage struct {
	// Channels are the channels to create on the request, and the created channels with
	// their assigned keys on the response.
	Channels []Channel
	// Opts controls create behavior when a channel with the same name already exists.
	Opts CreateOptions
}

type (
	// CreateClient issues channel create requests to a remote Core and returns the
	// created channels.
	CreateClient = freighter.UnaryClient[CreateMessage, CreateMessage]
	// CreateServer handles incoming channel create requests from a remote Core.
	CreateServer = freighter.UnaryServer[CreateMessage, CreateMessage]
)

// DeleteRequest is the payload for a channel delete operation.
type DeleteRequest struct {
	// Keys identifies the channels to delete.
	Keys Keys
}

type (
	// DeleteClient issues channel delete requests to a remote Core.
	DeleteClient = freighter.UnaryClient[DeleteRequest, types.Nil]
	// DeleteServer handles incoming channel delete requests from a remote Core.
	DeleteServer = freighter.UnaryServer[DeleteRequest, types.Nil]
)

// RenameRequest is the payload for a channel rename operation. Keys and Names are
// positional: the channel at Keys[i] is renamed to Names[i], so both slices must have
// the same length.
type RenameRequest struct {
	// Keys identifies the channels to rename.
	Keys Keys
	// Names holds the new name for each channel in Keys, by position.
	Names []string
}

type (
	// RenameClient issues channel rename requests to a remote Core.
	RenameClient = freighter.UnaryClient[RenameRequest, types.Nil]
	// RenameServer handles incoming channel rename requests from a remote Core.
	RenameServer = freighter.UnaryServer[RenameRequest, types.Nil]
)

// Transport bundles the node-to-node transports used to forward channel create, delete,
// and rename operations to a remote Core that owns the affected channels and handle
// incoming requests from remote Cores.
type Transport interface {
	// CreateClient returns the client-side interface for sending channel create
	// requests.
	CreateClient() CreateClient
	// CreateServer returns the server-side interface for handling channel create
	// requests.
	CreateServer() CreateServer
	// DeleteClient returns the client-side interface for sending channel delete
	// requests.
	DeleteClient() DeleteClient
	// DeleteServer returns the server-side interface for handling channel delete
	// requests.
	DeleteServer() DeleteServer
	// RenameClient returns the client-side interface for sending channel rename
	// requests.
	RenameClient() RenameClient
	// RenameServer returns the server-side interface for handling channel rename
	// requests.
	RenameServer() RenameServer
}
