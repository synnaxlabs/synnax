package writer

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
)

type Command uint8

const (
	Data Command = iota + 1
	Commit
	Error
)

type Request struct {
	// Command is the command to execute on the writer.
	Command Command
	// Keys should only be set when opening the writer.
	Keys channel.Keys `json:"openKeys"`
	// Segments is the set of segments to write. It is only relevant for calls to Data.
	Segments []core.Segment `json:"segments"`
}

type Response struct {
	Command Command
	Ack     bool
	SeqNum  int
	Err     error
}

type (
	ServerStream    = freighter.ServerStream[Request, Response]
	ClientStream    = freighter.ClientStream[Request, Response]
	TransportServer = freighter.StreamServer[Request, Response]
	TransportClient = freighter.StreamClient[Request, Response]
)
