package writer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/arya-analytics/freighter"
)

type Request struct {
	OpenKeys channel.Keys   `json:"openKeys"`
	Segments []core.Segment `json:"segments"`
}

type Response struct {
	Err error
}

type (
	Server    = freighter.ServerStream[Request, Response]
	Client    = freighter.ClientStream[Request, Response]
	Transport = freighter.StreamTransport[Request, Response]
)
