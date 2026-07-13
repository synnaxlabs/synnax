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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

//go:generate stringer -type=Command

// Command is an operation that can be executed on a Writer.
type Command uint8

const (
	// CommandOpen represents opening the writer.
	CommandOpen Command = iota
	// CommandWrite represents a call to Writer.Write.
	CommandWrite
	// CommandCommit represents a call to Writer.Commit.
	CommandCommit
	// CommandSetAuthority represents a call to Writer.SetAuthority
	CommandSetAuthority
)

var validateCommand = validate.NewInclusiveBoundsChecker(CommandOpen, CommandSetAuthority)

// Mode configures the persistence and streaming behavior of a writer.
type Mode = ts.WriterMode

// Request represents a streaming call to a Writer.
type Request struct {
	// Config sets the configuration to use when opening the writer. Only used
	// internally when an open command is sent.
	Config Config
	// Frame is the telemetry frame. This field is only acknowledged during Write
	// commands.
	Frame frame.Frame
	// SeqNum is used to match the request with the response.
	SeqNum int
	// Command is the command to execute on the writer.
	Command Command
}

// Response represents a response to a streaming call to a Writer.
type Response struct {
	// Err contains an error that occurred when attempting to execute a request on a
	// writer.
	Err error
	// SeqNum is the current sequence number of the command. This value will correspond
	// to the Request.SeqNum that executed the command.
	SeqNum int
	// End is the end timestamp of the domain on commit. This value is only validate
	// during calls to WriterCommit.
	End telem.TimeStamp
	// NodeKey is the NodeKey of the node that sent the response.
	NodeKey node.Key
	// Command is the command that was executed on the writer.
	Command Command
	// Authorized flags whether the writer or commit operation was authorized. It is
	// only valid during calls to WriterWrite and WriterCommit.
	Authorized bool
}

type (
	// ClientStream is the client-side of a writer stream, sending Requests to and
	// receiving Responses from a remote Core.
	ClientStream = freighter.ClientStream[Request, Response]
	// ServerStream is the server-side of a writer stream, receiving Requests from and
	// sending Responses to a remote Core.
	ServerStream = freighter.ServerStream[Request, Response]
	// Client is the client-side interface for opening a writer stream to a remote Core.
	Client = freighter.StreamClient[Request, Response]
	// Server is the server-side interface for handling writer streams from a remote
	// Core.
	Server = freighter.StreamServer[Request, Response]
)

// Transport is the interface for the writer transport.
type Transport interface {
	// Client returns the client-side interface for opening writer streams.
	Client() Client
	// Server returns the server-side interface for handling writer streams.
	Server() Server
}
