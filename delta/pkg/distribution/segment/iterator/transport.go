package iterator

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/telem"
)

//go:generate stringer -type=Command
type Command uint8

const (
	Open Command = iota
	Next
	Prev
	First
	Last
	NextSpan
	PrevSpan
	NextRange
	Valid
	Error
	SeekFirst
	SeekLast
	SeekLT
	SeekGE
)

// Request is a request to a remote iterator.
type Request struct {
	Command Command
	Span    telem.TimeSpan
	Range   telem.TimeRange
	Stamp   telem.TimeStamp
	Keys    channel.Keys
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
	// NodeID is the node ID where the remote iterator lives.
	NodeID distribcore.NodeID
	// Ack is only relevant for variant AckResponse. Is true if the iterator successfully
	// executed the request.
	Ack bool
	// Counter
	Counter int
	// Command is only relevant for variant AckResponse. It is  the command that was executed
	// on the iterator.
	Command Command
	// Err is only relevant for variant AckResponse. It is an error returned during a call to
	// Iterator.Error
	Error error
	// Segments is only relevant for DataResponse. It is the data returned by the iterator.
	Segments []core.Segment
}

func newAck(host distribcore.NodeID, cmd Command, ok bool) Response {
	return Response{Variant: AckResponse, Ack: ok, Command: cmd, NodeID: host}
}

type (
	Server    = freighter.ServerStream[Request, Response]
	Client    = freighter.ClientStream[Request, Response]
	Transport = freighter.StreamTransport[Request, Response]
)
