package synnax

import (
	"context"
	"github.com/synnaxlabs/client/auth"
	"github.com/synnaxlabs/client/channel"
	"github.com/synnaxlabs/client/framer"
	"github.com/synnaxlabs/client/transport/grpc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/address"
)

type Synnax struct {
	*framer.Client
	Channels *channel.Client
}

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	Secure   bool
}

func Open(ctx context.Context, cfg Config) *Synnax {
	transport := grpc.New(address.Newf("%s:%s", cfg.Host, cfg.Port))
	s := &Synnax{}
	a := auth.New(transport.AuthLogin, auth.InsecureCredentials{
		Username: cfg.Username,
		Password: password.Raw(cfg.Password),
	})
	transport.Use(a.Middleware())
	s.Channels = channel.NewClient(transport.ChannelCreate, transport.ChannelRetrieve, transport.ChannelDelete)
	s.Client = framer.NewClient(transport.FrameIterator, transport.FrameWriter, transport.FrameStreamer)
	return s
}
