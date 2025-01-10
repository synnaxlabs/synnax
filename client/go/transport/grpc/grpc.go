package grpc

import (
	"github.com/synnaxlabs/client/transport"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api/grpc"
	"github.com/synnaxlabs/x/address"
	pgrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func New(targetPrefix address.Address) (t transport.Transport) {
	pool := fgrpc.NewPool(targetPrefix, pgrpc.WithTransportCredentials(insecure.NewCredentials()))
	t.ChannelCreate = grpc.NewChannelCreateClient(pool)
	t.ChannelRetrieve = grpc.NewChannelRetrieveClient(pool)
	t.ChannelDelete = grpc.NewChannelDeleteClient(pool)
	t.FrameStreamer = grpc.NewFrameStreamerClient(pool)
	t.FrameIterator = grpc.NewFrameIteratorClient(pool)
	t.FrameWriter = grpc.NewFrameWriterClient(pool)
	t.AuthLogin = grpc.NewAuthLoginClient(pool)
	return
}
