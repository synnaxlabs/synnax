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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
)

// Request opens or updates a relay subscription, streaming live samples for the given
// channels. Sending a new Request replaces the previous set of channels.
type Request struct {
	// Keys are the channels to stream live samples from.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
}

// Response carries a batch of live samples delivered to a relay subscriber.
type Response struct {
	// Frame holds the live samples for the subscribed channels.
	Frame frame.Frame `json:"frame" msgpack:"frame"`
	// Group identifies the control group that produced the frame, used to filter
	// relayed frames (see relay ExcludeGroups).
	Group uint32 `json:"group" msgpack:"group"`
}

func reqToStorage(req Request) ts.StreamerRequest {
	return ts.StreamerRequest{Channels: req.Keys.Storage()}
}

func resFromStorage(res ts.StreamerResponse) Response {
	return Response{Frame: frame.NewFromStorage(res.Frame), Group: res.Group}
}

type (
	// ClientStream is the client-side of a relay stream, sending Requests to and
	// receiving Responses from a remote Core.
	ClientStream = freighter.ClientStream[Request, Response]
	// ServerStream is the server-side of a relay stream, receiving Requests from and
	// sending Responses to a remote Core.
	ServerStream = freighter.ServerStream[Request, Response]
	// Client is the client-side interface for opening a relay stream to a remote Core.
	Client = freighter.StreamClient[Request, Response]
	// Server is the server-side interface for handling relay streams from a remote
	// Core.
	Server = freighter.StreamServer[Request, Response]
)

// Transport is the interface for the relay transport.
type Transport interface {
	// Client returns the client-side interface for opening relay streams.
	Client() Client
	// Server returns the server-side interface for handling relay streams.
	Server() Server
}
