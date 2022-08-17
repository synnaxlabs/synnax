package fgrpc

import (
	"github.com/arya-analytics/freighter"
	"google.golang.org/grpc"
)

type BindableTransport interface {
	BindTo(reg grpc.ServiceRegistrar)
}

var digest = freighter.Digest{
	Protocol:  "grpc",
	Encodings: []string{"protobuf"},
}
