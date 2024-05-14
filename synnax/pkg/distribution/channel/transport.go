// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"github.com/synnaxlabs/freighter"
	"go/types"
)

type (
	CreateTransportClient = freighter.UnaryClient[CreateMessage, CreateMessage]
	CreateTransportServer = freighter.UnaryServer[CreateMessage, CreateMessage]
	DeleteTransportClient = freighter.UnaryClient[DeleteRequest, types.Nil]
	DeleteTransportServer = freighter.UnaryServer[DeleteRequest, types.Nil]
)

type Transport interface {
	CreateClient() CreateTransportClient
	CreateServer() CreateTransportServer
	DeleteClient() DeleteTransportClient
	DeleteServer() DeleteTransportServer
}

type CreateMessage struct {
	Channels             []Channel
	RetrieveIfNameExists bool
}

type DeleteRequest struct {
	Keys []Key
}
