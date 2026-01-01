// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
)

type Request struct {
	Keys channel.Keys
}

type Response struct {
	Frame core.Frame `json:"frame" msgpack:"frame"`
}

func reqToStorage(req Request) ts.StreamerRequest {
	return ts.StreamerRequest{Channels: req.Keys.Storage()}
}

func resFromStorage(res ts.StreamerResponse) Response {
	return Response{Frame: core.NewFrameFromStorage(res.Frame)}
}

type (
	ServerStream    = freighter.ServerStream[Request, Response]
	ClientStream    = freighter.ClientStream[Request, Response]
	TransportServer = freighter.StreamServer[Request, Response]
	TransportClient = freighter.StreamClient[Request, Response]
)

type Transport interface {
	Server() TransportServer
	Client() TransportClient
}
