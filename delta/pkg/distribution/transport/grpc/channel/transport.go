package channel

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/transport/grpc/gen/proto/go/channel/v1"
	"github.com/arya-analytics/freighter/fgrpc"
	"google.golang.org/grpc"
)

type transport = fgrpc.UnaryTransport[
	channel.CreateMessage,
	*channelv1.CreateMessage,
	channel.CreateMessage,
	*channelv1.CreateMessage,
]

var (
	_ channel.CreateTransport        = (*transport)(nil)
	_ channelv1.ChannelServiceServer = (*transport)(nil)
)

func New(pool *fgrpc.Pool) *transport {
	return &transport{
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
		ServiceDesc: &channelv1.ChannelService_ServiceDesc,
	}
}
