package channel

import (
	"context"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/channel/v1"
	"google.golang.org/grpc"
)

type client = fgrpc.UnaryClient[
	channel.CreateMessage,
	*channelv1.CreateMessage,
	channel.CreateMessage,
	*channelv1.CreateMessage,
]
type server = fgrpc.UnaryServer[
	channel.CreateMessage,
	*channelv1.CreateMessage,
	channel.CreateMessage,
	*channelv1.CreateMessage,
]

var (
	_ channel.CreateTransportClient  = (*client)(nil)
	_ channel.CreateTransportServer  = (*server)(nil)
	_ channelv1.ChannelServiceServer = (*server)(nil)
)

func New(pool *fgrpc.Pool) (*client, *server) {
	return &client{
			Pool:               pool,
			RequestTranslator:  createMessageTranslator{},
			ResponseTranslator: createMessageTranslator{},
			Client: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *channelv1.CreateMessage,
			) (*channelv1.CreateMessage, error) {
				return channelv1.NewChannelServiceClient(conn).Exec(ctx, req)
			},
		}, &server{
			RequestTranslator:  createMessageTranslator{},
			ResponseTranslator: createMessageTranslator{},
			ServiceDesc:        &channelv1.ChannelService_ServiceDesc,
		}
}
