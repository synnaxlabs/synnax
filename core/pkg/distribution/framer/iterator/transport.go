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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
)

//go:generate stringer -type=Command

// Command is an operation that can be executed on a remote Iterator.
type Command uint8

// AutoSpan instructs Next and Prev to automatically size each chunk so the iterator
// returns all data in the current view in a single step.
const AutoSpan = ts.AutoSpan

const (
	// CommandNext advances the iterator forward by a span and returns the data in
	// range.
	CommandNext Command = iota + 1
	// CommandPrev moves the iterator backward by a span and returns the data in range.
	CommandPrev
	// CommandSeekFirst positions the iterator at the first sample.
	CommandSeekFirst
	// CommandSeekLast positions the iterator at the last sample.
	CommandSeekLast
	// CommandSeekLE positions the iterator at the last sample at or before a timestamp.
	CommandSeekLE
	// CommandSeekGE positions the iterator at the first sample at or after a timestamp.
	CommandSeekGE
	// CommandValid reports whether the iterator is currently positioned on valid data.
	CommandValid
	// CommandError returns any error accumulated by the iterator.
	CommandError
	// CommandSetBounds sets the time bounds the iterator operates over.
	CommandSetBounds
)

// Request is a request to an Iterator.
type Request struct {
	// Keys should only be set when opening the Iterator.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Bounds should be set during calls to SetBounds.
	Bounds telem.TimeRange `json:"bounds" msgpack:"bounds"`
	// Stamp should be set during calls to SeekLE and SeekGE.
	Stamp telem.TimeStamp `json:"stamp" msgpack:"stamp"`
	// Span should be set during calls to Next and Prev.
	Span telem.TimeSpan `json:"span" msgpack:"span"`
	// ChunkSize should only be set when opening the Iterator.
	ChunkSize int64 `json:"chunk_size" msgpack:"chunk_size"`
	// DownsampleFactor should only be set when opening the Iterator.
	DownsampleFactor int `json:"downsample_factor" msgpack:"downsample_factor"`
	// SeqNum is the sequence number of the request (starting at 0). This is used to
	// match responses to requests. Each request should increment the sequence number by
	// 1.
	SeqNum int
	// Command is the command to execute on the Iterator.
	Command Command `json:"command" msgpack:"command"`
}

//go:generate stringer -type=ResponseVariant

// ResponseVariant is the kind of Response returned by a remote Iterator.
type ResponseVariant uint8

const (
	// ResponseVariantAck is a response that indicates that an iteration request was
	// acknowledged.
	ResponseVariantAck ResponseVariant = iota + 1
	// ResponseVariantData is a response that indicates that an iteration request
	// returned data.
	ResponseVariantData
)

// Response is a response from a remote Iterator.
type Response struct {
	// Error is only relevant for variant AckResponse. It is an error returned during a
	// call to Iterator.Error
	Error error `json:"error" msgpack:"error"`
	// Frame is only relevant for DataResponse. It is the data returned by the Iterator.
	Frame frame.Frame `json:"frame" msgpack:"frame"`
	// SeqNum matches the response to the Request.SeqNum that triggered it.
	SeqNum int `json:"seq_num" msgpack:"seq_num"`
	// NodeKey is the node Name where the remote Iterator lives.
	NodeKey node.Key `json:"node_key" msgpack:"node_key"`
	// Variant is the type of response returned.
	Variant ResponseVariant `json:"variant" msgpack:"variant"`
	// Command is the command that produced this response. It is non-zero only for
	// acknowledgement responses.
	Command Command `json:"command" msgpack:"command"`
	// Ack is only relevant for variant AckResponse. Is true if the Iterator
	// successfully executed the request.
	Ack bool `json:"ack" msgpack:"ack"`
}

type (
	// ClientStream is the client-side of an iterator stream, sending Requests to and
	// receiving Responses from a remote Core.
	ClientStream = freighter.ClientStream[Request, Response]
	// ServerStream is the server-side of an iterator stream, receiving Requests from
	// and sending Responses to a remote Core.
	ServerStream = freighter.ServerStream[Request, Response]
	// Client is the client-side interface for opening an iterator stream to a remote
	// Core.
	Client = freighter.StreamClient[Request, Response]
	// Server is the server-side interface for handling iterator streams from a remote
	// Core.
	Server = freighter.StreamServer[Request, Response]
)

// Transport is the interface for the iterator transport.
type Transport interface {
	// Client returns the client-side interface for opening iterator streams.
	Client() Client
	// Server returns the server-side interface for handling iterator streams.
	Server() Server
}
