// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
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
	Command Command `json:"command" msgpack:"command"`
	// Stamp should be set during calls to SeekLE and SeekGE.
	Stamp telem.TimeStamp `json:"stamp" msgpack:"stamp"`
	// Span should be set during calls to Next and Prev.
	Span telem.TimeSpan `json:"span" msgpack:"span"`
	// Bounds should be set during calls to SetBounds.
	Bounds telem.TimeRange `json:"bounds" msgpack:"bounds"`
	// Keys should only be set when opening the Iterator.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// ChunkSize should only be set when opening the Iterator.
	ChunkSize int64 `json:"chunk_size" msgpack:"chunk_size"`
	// DownsampleFactor should only be set when opening the Iterator.
	DownsampleFactor int `json:"downsample_factor" msgpack:"downsample_factor"`
	// SeqNum is the sequence number of the request (starting at 1). This is used to
	// match responses to requests. Each request should increment the sequence number
	// by 1.
	SeqNum int
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
	Variant ResponseVariant `json:"variant" msgpack:"variant"`
	// Command is non-zero when the
	Command Command `json:"command" msgpack:"command"`
	// Frame is only relevant for DataResponse. It is the data returned by the Iterator.
	Frame frame.Frame `json:"frame" msgpack:"frame"`
	// NodeKey is the node Name where the remote Iterator lives.
	NodeKey cluster.NodeKey `json:"node_key" msgpack:"node_key"`
	// Ack is only relevant for variant AckResponse. Is true if the Iterator successfully
	// executed the request.
	Ack bool `json:"ack" msgpack:"ack"`
	// SeqNum
	SeqNum int `json:"seq_num" msgpack:"seq_num"`
	// Error is only relevant for variant AckResponse. It is an error returned during a call to
	// Iterator.Error
	Error error `json:"error" msgpack:"error"`
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
