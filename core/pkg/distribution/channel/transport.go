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
	CreateTransportClient = freighter.UnaryClient[CreateMessage, CreateMessage]
	CreateTransportServer = freighter.UnaryServer[CreateMessage, CreateMessage]
	DeleteTransportClient = freighter.UnaryClient[DeleteRequest, types.Nil]
	DeleteTransportServer = freighter.UnaryServer[DeleteRequest, types.Nil]
	RenameTransportServer = freighter.UnaryServer[RenameRequest, types.Nil]
	RenameTransportClient = freighter.UnaryClient[RenameRequest, types.Nil]
)

type Transport interface {
	CreateClient() CreateTransportClient
	CreateServer() CreateTransportServer
	DeleteClient() DeleteTransportClient
	DeleteServer() DeleteTransportServer
	RenameClient() RenameTransportClient
	RenameServer() RenameTransportServer
}

type CreateMessage struct {
	Channels []Channel
	Opts     CreateOptions
}

type RenameRequest struct {
	Keys  Keys
	Names []string
}

type DeleteRequest struct {
	Keys Keys
}
