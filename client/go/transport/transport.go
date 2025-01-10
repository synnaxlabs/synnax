package transport

import (
	"github.com/synnaxlabs/client/auth"
	"github.com/synnaxlabs/client/channel"
	"github.com/synnaxlabs/client/framer"
	"github.com/synnaxlabs/freighter"
)

type Transport struct {
	FrameIterator   framer.IteratorClient
	FrameWriter     framer.WriterClient
	FrameStreamer   framer.StreamerClient
	ChannelCreate   channel.CreateClient
	ChannelRetrieve channel.RetrieveClient
	ChannelDelete   channel.DeleteClient
	AuthLogin       auth.LoginClient
}

func (t *Transport) Use(mw ...freighter.Middleware) {
	t.FrameIterator.Use(mw...)
	t.FrameWriter.Use(mw...)
	t.FrameStreamer.Use(mw...)
	t.ChannelCreate.Use(mw...)
	t.ChannelRetrieve.Use(mw...)
	t.ChannelDelete.Use(mw...)
}
