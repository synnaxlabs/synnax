// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
)

type ReadRequest struct {
	Keys channel.Keys
}

type Data struct {
	Frame framer.Frame
	Error error
}

type (
	ServerStream    = freighter.ServerStream[ReadRequest, Data]
	ClientStream    = freighter.ClientStream[ReadRequest, Data]
	TransportServer = freighter.StreamServer[ReadRequest, Data]
	TransportClient = freighter.StreamClient[ReadRequest, Data]
)

type Transport interface {
	Server() TransportServer
	Client() TransportClient
}
