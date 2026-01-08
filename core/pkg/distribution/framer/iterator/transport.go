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
	Keys             channel.Keys    `json:"keys" msgpack:"keys"`
	Bounds           telem.TimeRange `json:"bounds" msgpack:"bounds"`
	Stamp            telem.TimeStamp `json:"stamp" msgpack:"stamp"`
	Span             telem.TimeSpan  `json:"span" msgpack:"span"`
	ChunkSize        int64           `json:"chunk_size" msgpack:"chunk_size"`
	DownsampleFactor int             `json:"downsample_factor" msgpack:"downsample_factor"`
	SeqNum           int
	Command          Command `json:"command" msgpack:"command"`
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
	// Error is only relevant for variant AckResponse. It is an error returned during a call to
	// Iterator.Error
	Error error `json:"error" msgpack:"error"`
	// Frame is only relevant for DataResponse. It is the data returned by the Iterator.
	Frame frame.Frame `json:"frame" msgpack:"frame"`
	// SeqNum
	SeqNum int `json:"seq_num" msgpack:"seq_num"`
	// NodeKey is the node Name where the remote Iterator lives.
	NodeKey cluster.NodeKey `json:"node_key" msgpack:"node_key"`
	// Variant is the type of response returned.
	Variant ResponseVariant `json:"variant" msgpack:"variant"`
	// Command is non-zero when the
	Command Command `json:"command" msgpack:"command"`
	// Ack is only relevant for variant AckResponse. Is true if the Iterator successfully
	// executed the request.
	Ack bool `json:"ack" msgpack:"ack"`
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
