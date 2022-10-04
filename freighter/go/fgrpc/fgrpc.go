package fgrpc

import (
	"github.com/synnaxlabs/freighter"
	"google.golang.org/grpc"
)

// BindableTransport is a transport that can be bound to a gRPC service
// registrar.
type BindableTransport interface {
	// BindTo binds the transport to the given gRPC service registrar.
	BindTo(reg grpc.ServiceRegistrar)
}

var reporter = freighter.Reporter{
	Protocol:  "grpc",
	Encodings: []string{"protobuf"},
}
