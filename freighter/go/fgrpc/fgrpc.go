package fgrpc

import (
	"github.com/synnaxlabs/freighter"
	"google.golang.org/grpc"
)

type BindableTransport interface {
	BindTo(reg grpc.ServiceRegistrar)
}

var reporter = freighter.Reporter{
	Protocol:  "grpc",
	Encodings: []string{"protobuf"},
}
