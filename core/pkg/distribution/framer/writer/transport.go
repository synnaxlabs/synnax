// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

//go:generate stringer -type=Command
type Command uint8

const (
	CommandOpen Command = iota
	// CommandWrite represents a call to Writer.Write.
	CommandWrite
	// CommandCommit represents a call to Writer.Commit.
	CommandCommit
	// CommandSetAuthority represents a call to Writer.SetAuthority
	CommandSetAuthority
)

var validateCommand = validate.NewInclusiveBoundsChecker(CommandOpen, CommandSetAuthority)

type Mode = ts.WriterMode

// Request represents a streaming call to a Writer.
type Request struct {
	// Config sets the configuration to use when opening the writer. Only used internally
	// when an open command is sent.
	Config Config `json:"config" msgpack:"config"`
	// Frame is the telemetry frame. This field is only acknowledged during Write commands.
	Frame frame.Frame `json:"frame" msgpack:"keys"`
	// SeqNum is used to match the request with the response.
	SeqNum int `json:"seq_num" msgpack:"seq_num"`
	// Command is the command to execute on the writer.
	Command Command `json:"command" msgpack:"command"`
}

// Response represents a response to a streaming call to a Writer.
type Response struct {
	// Err contains an error that occurred when attempting to execute a request on
	// a writer.
	Err error `json:"err" msgpack:"err"`
	// SeqNum is the current sequence number of the command. This value will
	// correspond to the Request.SeqNum that executed the command.
	SeqNum int `json:"seq_num" msgpack:"seq_num"`
	// End is the end timestamp of the domain on commit. This value is only
	// validate during calls to WriterCommit.
	End telem.TimeStamp `json:"end" msgpack:"end"`
	// The NodeKey of the node that sent the response.
	NodeKey cluster.NodeKey `json:"node_key" msgpack:"node_key"`
	// Command is the command that was executed on the writer.
	Command Command `json:"command" msgpack:"command"`
	// Authorized flags whether the writer or commit operation was authorized. It is only
	// valid during calls to WriterWrite and WriterCommit.
	Authorized bool `json:"authorized" msgpack:"authorized"`
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
