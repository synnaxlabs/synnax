package grpc

import (
	"github.com/arya-analytics/freighter/fgrpc"
)

type API struct {
	Transports []fgrpc.BindableTransport
}
