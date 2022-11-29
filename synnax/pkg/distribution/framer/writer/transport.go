package writer

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/telem"
)

type Command uint8

const (
	Open Command = iota
	Data
	Commit
	Error
)

type Request struct {
	// Command is the command to execute on the writer.
	Command Command `json:"command" msgpack:"command"`
	// Start sets the starting timestamp when opening the writer.
	Start telem.TimeStamp `json:"start" msgpack:"start"`
	// Keys sets the keys the writer will write to.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Frame is the telemetry frame
	Frame core.Frame `json:"frame" msgpack:"keys"`
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

type Transport interface {
	Server() TransportServer
	Client() TransportClient
}
