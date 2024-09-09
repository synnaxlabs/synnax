// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
)

//go:generate stringer -type=Command
type Command uint8

const AutoSpan = ts.AutoSpan

const (
	Next Command = iota + 1
	Prev
	SeekFirst
	SeekLast
	SeekLE
	SeekGE
	Valid
	Error
	SetBounds
)

// Request is a request to an Iterator.
type Request struct {
	// Command is the command to execute on the Iterator.
	Command Command `json:"command" msg:"command"`
	// Stamp should be set during calls to SeekLE and SeekGE.
	Stamp telem.TimeStamp `json:"stamp" msg:"stamp"`
	// Span should be set during calls to Next and Prev.
	Span telem.TimeSpan `json:"span" msg:"span"`
	// Bounds should be set during calls to SetBounds.
	Bounds telem.TimeRange `json:"bounds" msg:"bounds"`
	// Keys should only be set when opening the Iterator.
	Keys channel.Keys `json:"keys" msg:"keys"`
	// ChunkSize should only be set when opening the Iterator.
	ChunkSize int64 `json:"chunk_size" msg:"chunk_size"`
}

//go:generate stringer -type=ResponseVariant
type ResponseVariant uint8

const (
	// AckResponse is a response that indicates that an iteration request was acknowledged.
	AckResponse ResponseVariant = iota + 1
	// DataResponse is a response that indicates that an iteration request returned data.
	DataResponse
)

// Response is a response from a remote Iterator.
type Response struct {
	// Variant is the type of response returned.
	Variant ResponseVariant `json:"variant" msg:"variant"`
	// Command is non-zero when the
	Command Command `json:"command" msg:"command"`
	// Frame is only relevant for DataResponse. It is the data returned by the Iterator.
	Frame core.Frame `json:"frame" msg:"frame"`
	// NodeKey is the node Name where the remote Iterator lives.
	NodeKey dcore.NodeKey `json:"node_key" msg:"node_key"`
	// Ack is only relevant for variant AckResponse. Is true if the Iterator successfully
	// executed the request.
	Ack bool `json:"ack" msg:"ack"`
	// SeqNum
	SeqNum int `json:"seq_num" msg:"seq_num"`
	// Error is only relevant for variant AckResponse. It is an error returned during a call to
	// Iterator.Error
	Error error `json:"error" msg:"error"`
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
