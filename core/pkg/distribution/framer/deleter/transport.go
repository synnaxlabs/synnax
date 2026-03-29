// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter

import (
	"go/types"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
)

type (
	// TransportServer is the server side interface for receiving and processing a
	// delete request from a remote Core.
	TransportServer = freighter.UnaryServer[Request, types.Nil]
	// TransportClient is the client side interface for sending a delete request to a
	// remote Core.
	TransportClient = freighter.UnaryClient[Request, types.Nil]
)

// Transport is the interface for the deleter transport.
type Transport interface {
	// Server returns the server side interface for receiving and processing a
	// delete request from a remote Core.
	Server() TransportServer
	// Client returns the client side interface for sending a delete request to a
	// remote Core.
	Client() TransportClient
}

// Request is the request for deleting data from a Synnax cluster through deleting
// certain time ranges from channels.
type Request struct {
	// Keys is the list of keys to delete data from.
	Keys channel.Keys
	// Bounds is the time range to delete data from.
	Bounds telem.TimeRange
}
