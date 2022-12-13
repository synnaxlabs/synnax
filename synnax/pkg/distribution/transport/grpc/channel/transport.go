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

// Transport is a grpc backed implementation of the channel.Transport interface.
type Transport struct {
	client *client
	server *server
}

// CreateClient implements the channel.Transport interface.
func (t Transport) CreateClient() channel.CreateTransportClient { return t.client }

// CreateServer implements the channel.Transport interface.
func (t Transport) CreateServer() channel.CreateTransportServer { return t.server }

// BindTo implements the fgrpc.BindableTransport interface.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) { t.server.BindTo(reg) }

var (
	_ channel.CreateTransportClient  = (*client)(nil)
	_ channel.CreateTransportServer  = (*server)(nil)
	_ channelv1.ChannelServiceServer = (*server)(nil)
	_ channel.Transport              = (*Transport)(nil)
	_ fgrpc.BindableTransport        = (*Transport)(nil)
)

// New creates a new grpc Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	c := &client{
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
	}
	s := &server{
		RequestTranslator:  createMessageTranslator{},
		ResponseTranslator: createMessageTranslator{},
		ServiceDesc:        &channelv1.ChannelService_ServiceDesc,
	}
	return Transport{c, s}
}
