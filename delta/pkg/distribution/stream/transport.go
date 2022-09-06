package stream

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/freighter"
	"go/types"
)

type WriteRequest struct {
	Samples []Sample
}

type ReadRequest struct {
	Keys []channel.Key
}

type ReadResponse struct {
	Samples []Sample
}

type (
	WriteClientStream = freighter.ClientStream[WriteRequest, types.Nil]
	WriteServerStream = freighter.ServerStream[WriteRequest, types.Nil]
	WriteTransport    = freighter.StreamTransport[WriteRequest, types.Nil]
	ReadClientStream  = freighter.ClientStream[ReadRequest, ReadResponse]
	ReadServerStream  = freighter.ServerStream[ReadRequest, ReadResponse]
	ReadTransport     = freighter.StreamTransport[ReadRequest, ReadResponse]
)

type Transport interface {
	Writer() WriteTransport
	Reader() ReadTransport
}
