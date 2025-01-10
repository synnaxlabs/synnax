package channel

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"go/types"
)

type (
	Key             = api.ChannelKey
	Channel         = api.Channel
	CreateClient    = freighter.UnaryClient[api.ChannelCreateRequest, api.ChannelCreateResponse]
	RetrieveClient  = freighter.UnaryClient[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse]
	DeleteClient    = freighter.UnaryClient[api.ChannelDeleteRequest, types.Nil]
	RetrieveRequest = api.ChannelRetrieveRequest
	DeleteRequest   = api.ChannelDeleteRequest
)

type Client struct {
	createTransport   CreateClient
	retrieveTransport RetrieveClient
	deleteTransport   DeleteClient
}

func NewClient(
	createClient CreateClient,
	retrieveClient RetrieveClient,
	deleteClient DeleteClient,
) *Client {
	return &Client{
		createTransport:   createClient,
		retrieveTransport: retrieveClient,
		deleteTransport:   deleteClient,
	}
}

func (c *Client) CreateOne(ctx context.Context, ch *Channel) error {
	req := api.ChannelCreateRequest{Channels: []Channel{*ch}}
	res, err := c.createTransport.Send(ctx, "", req)
	if err != nil {
		return err
	}
	*ch = res.Channels[0]
	return nil
}

func (c *Client) CreateMany(ctx context.Context, chs *[]Channel) error {
	req := api.ChannelCreateRequest{Channels: *chs}
	res, err := c.createTransport.Send(ctx, "", req)
	if err != nil {
		return err
	}
	*chs = res.Channels
	return nil
}

func (c *Client) RetrieveOne(ctx context.Context, req RetrieveRequest) (ch Channel, err error) {
	res, err := c.retrieveTransport.Send(ctx, "", req)
	if err != nil {
		return ch, err
	}
	if len(res.Channels) == 0 {
		return ch, errors.Wrapf(query.NotFound, "channel not found")
	}
	return res.Channels[0], nil
}

func (c *Client) RetrieveMany(ctx context.Context, req RetrieveRequest) (chs []Channel, err error) {
	res, err := c.retrieveTransport.Send(ctx, "", req)
	if err != nil {
		return chs, err
	}
	return res.Channels, nil
}

func (c *Client) Delete(ctx context.Context, req api.ChannelDeleteRequest) error {
	_, err := c.deleteTransport.Send(ctx, "", req)
	return err
}
