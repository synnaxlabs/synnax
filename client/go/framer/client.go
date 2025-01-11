package framer

import (
	"context"
	"github.com/synnaxlabs/client/channel"
)

type Client struct {
	iteratorClient IteratorClient
	writerClient   WriterClient
	streamerClient StreamerClient
	channelClient  *channel.Client
}

func NewClient(
	channelClient *channel.Client,
	iteratorClient IteratorClient,
	writerClient WriterClient,
	streamerClient StreamerClient,
) *Client {
	return &Client{
		channelClient:  channelClient,
		iteratorClient: iteratorClient,
		writerClient:   writerClient,
		streamerClient: streamerClient,
	}
}

func (c *Client) OpenIterator(ctx context.Context, cfg IteratorConfig) (*Iterator, error) {
	return openIterator(ctx, c.channelClient, c.iteratorClient, cfg)
}

func (c *Client) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	return openWriter(ctx, c.writerClient, cfg)
}

func (c *Client) OpenStreamer(ctx context.Context, cfg StreamerConfig) (*Streamer, error) {
	return openStreamer(ctx, c.streamerClient, cfg)
}
