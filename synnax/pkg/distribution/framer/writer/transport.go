// Copyright 2022 Synnax Labs, Inc.
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
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
)

//go:generate stringer -type=Command
type Command uint8

const (
	open Command = iota
	// Data represents a call to Writer.Write.
	Data
	// Commit represents a call to Writer.Commit.
	Commit
	// Error represents a call to Writer.Error.
	Error
)

// Request represents a streaming call to a Writer.
type Request struct {
	// Command is the command to execute on the writer.
	Command Command `json:"command" msgpack:"command"`
	// Config sets the configuration to use when opening the writer. Only used internally
	// when open command is sent.
	Config Config `json:"config" msgpack:"config"`
	// Frame is the telemetry frame. This field is only acknowledged during Data commands.
	Frame core.Frame `json:"frame" msgpack:"keys"`
}

// Response represents a response to a streaming call to a Writer.
type Response struct {
	// Command is the command that was executed on the writer.
	Command Command
	// Ack is the acknowledgement of the command.
	Ack    bool
	SeqNum int
	NodeID dcore.NodeID
	Err    error
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
