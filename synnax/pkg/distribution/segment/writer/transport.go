package writer

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
)

type Request struct {
	OpenKeys channel.Keys   `json:"openKeys"`
	Segments []core.Segment `json:"segments"`
}

type Response struct {
	Err error
}

type (
	ServerStream = freighter.ServerStream[Request, Response]
	ClientStream    = freighter.ClientStream[Request, Response]
	TransportServer = freighter.StreamServer[Request, Response]
	TransportClient = freighter.StreamClient[Request, Response]
)
