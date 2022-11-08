package iterator

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/telem"
)

//go:generate stringer -type=Command
type Command uint8

const AutoSpan = storage.AutoSpan

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

// Request is a request to an iterator.
type Request struct {
	// Command is the command to execute on the iterator.
	Command Command
	// Stamp should be set during calls to SeekLE and SeekGE.
	Stamp telem.TimeStamp
	// Span should be set during calls to Next and Prev.
	Span telem.TimeSpan
	// Bounds should be set during calls to SetBounds.
	Bounds telem.TimeRange
	// Keys should only be set when opening the iterator.
	Keys channel.Keys
}

type ResponseVariant uint8

const (
	// AckResponse is a response that indicates that an iteration request was acknowledged.
	AckResponse ResponseVariant = iota + 1
	// DataResponse is a response that indicates that an iteration request returned data.
	DataResponse
)

// Response is a response from a remote iterator.
type Response struct {
	// Variant is the type of response returned.
	Variant ResponseVariant
	// Command is non-zero when the
	Command Command
	// Segments is only relevant for DataResponse. It is the data returned by the iterator.
	Segments []core.Segment
	// NodeID is the node ID where the remote iterator lives.
	NodeID distribcore.NodeID
	// Ack is only relevant for variant AckResponse. Is true if the iterator successfully
	// executed the request.
	Ack bool
	// SeqNum
	SeqNum int
	// Err is only relevant for variant AckResponse. It is an error returned during a call to
	// Iterator.Error
	Err error
}

func newAck(host distribcore.NodeID, cmd Command, ok bool) Response {
	return Response{Variant: AckResponse, Ack: ok, Command: cmd, NodeID: host}
}

type (
	ServerStream    = freighter.ServerStream[Request, Response]
	ClientStream    = freighter.ClientStream[Request, Response]
	TransportServer = freighter.StreamServer[Request, Response]
	TransportClient = freighter.StreamClient[Request, Response]
)
