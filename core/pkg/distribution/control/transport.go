// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

// RetrieveRequest asks a Core for the control state of a set of channels.
type RetrieveRequest struct {
	// Keys identifies the channels to retrieve control state for. An empty set requests
	// every channel the Core currently arbitrates.
	Keys channel.Keys
}

// RetrieveResponse carries the control state of the requested channels. Channels that
// no subject controls are absent.
type RetrieveResponse struct {
	// States holds one entry per controlled channel.
	States []State
}

// SubscribeRequest opens a subscription to a Core's control state. It carries no
// parameters: a subscriber receives the state of every channel that Core arbitrates.
type SubscribeRequest struct{}

// SubscribeResponse carries a Core's complete control state. A Core sends one on
// subscription and one after every change, so a subscriber that misses a message
// converges on the next one rather than diverging.
type SubscribeResponse struct {
	// States holds one entry per controlled channel.
	States []State
}

type (
	// RetrieveClient issues control retrieves to a remote Core.
	RetrieveClient = freighter.UnaryClient[RetrieveRequest, RetrieveResponse]
	// RetrieveServer handles control retrieves from a remote Core.
	RetrieveServer = freighter.UnaryServer[RetrieveRequest, RetrieveResponse]
	// SubscribeStream is the server side of a control subscription.
	SubscribeStream = freighter.ServerStream[SubscribeRequest, SubscribeResponse]
	// SubscribeClient opens control subscriptions to a remote Core.
	SubscribeClient = freighter.StreamClient[SubscribeRequest, SubscribeResponse]
	// SubscribeClientStream is the client side of a control subscription.
	SubscribeClientStream = freighter.ClientStream[SubscribeRequest, SubscribeResponse]
	// SubscribeServer handles control subscriptions from a remote Core.
	SubscribeServer = freighter.StreamServer[SubscribeRequest, SubscribeResponse]
)

// Transport bundles the node-to-node transports the control service uses to read and
// subscribe to the control state held by remote Cores.
type Transport interface {
	// RetrieveClient returns the client-side interface for sending control retrieves.
	RetrieveClient() RetrieveClient
	// RetrieveServer returns the server-side interface for handling control retrieves.
	RetrieveServer() RetrieveServer
	// SubscribeClient returns the client-side interface for opening control
	// subscriptions.
	SubscribeClient() SubscribeClient
	// SubscribeServer returns the server-side interface for handling control
	// subscriptions.
	SubscribeServer() SubscribeServer
}
