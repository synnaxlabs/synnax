package writer

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/freighter"
)

type Request struct {
	OpenKeys channel.Keys   `json:"openKeys"`
	Segments []core.Segment `json:"segments"`
}

type Response struct {
	Error error
}

type (
	Server    = freighter.ServerStream[Request, Response]
	Client    = freighter.ClientStream[Request, Response]
	Transport = freighter.StreamTransport[Request, Response]
)
