package grpc

import (
	"github.com/synnaxlabs/freighter/fgrpc"
)

type API struct {
	Transports []fgrpc.BindableTransport
}
