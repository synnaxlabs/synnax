package framer

import "context"

type Client struct {
	iteratorClient IteratorClient
	writerClient   WriterClient
	streamerClient StreamerClient
}

func NewClient(
	iteratorClient IteratorClient,
	writerClient WriterClient,
	streamerClient StreamerClient,
) *Client {
	return &Client{
		iteratorClient: iteratorClient,
		writerClient:   writerClient,
		streamerClient: streamerClient,
	}
}

func (c *Client) OpenIterator(ctx context.Context, cfg IteratorConfig) (*Iterator, error) {
	return openIterator(ctx, c.iteratorClient, cfg)
}

func (c *Client) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	return openWriter(ctx, c.writerClient, cfg)
}

func (c *Client) OpenStreamer(ctx context.Context, cfg StreamerConfig) (*Streamer, error) {
	return openStreamer(ctx, c.streamerClient, cfg)
}
