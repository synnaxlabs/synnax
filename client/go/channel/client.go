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

type CreateOption func(*api.ChannelCreateRequest)

func RetrieveIfNameExists() CreateOption {
	return func(req *api.ChannelCreateRequest) {
		req.RetrieveIfNameExists = true
	}
}

func (c *Client) Create(ctx context.Context, ch *Channel, opts ...CreateOption) error {
	req := api.ChannelCreateRequest{Channels: []Channel{*ch}}
	for _, opt := range opts {
		opt(&req)
	}
	res, err := c.createTransport.Send(ctx, "", req)
	if err != nil {
		return err
	}
	*ch = res.Channels[0]
	return nil
}

func (c *Client) CreateMany(ctx context.Context, chs *[]Channel, opts ...CreateOption) error {
	req := api.ChannelCreateRequest{Channels: *chs}
	for _, opt := range opts {
		opt(&req)
	}
	res, err := c.createTransport.Send(ctx, "", req)
	if err != nil {
		return err
	}
	*chs = res.Channels
	return nil
}

type RetrieveDeleteOption func(*api.ChannelRetrieveRequest)

func WhereName(name ...string) RetrieveDeleteOption {
	return func(req *api.ChannelRetrieveRequest) {
		req.Names = append(req.Names, name...)
	}
}

func WhereKey(key ...Key) RetrieveDeleteOption {
	return func(req *api.ChannelRetrieveRequest) {
		req.Keys = append(req.Keys, key...)
	}
}

func WhereKeys(keys []Key) RetrieveDeleteOption {
	return func(req *api.ChannelRetrieveRequest) {
		req.Keys = append(req.Keys, keys...)
	}
}

func WhereNames(names []string) RetrieveDeleteOption {
	return func(req *api.ChannelRetrieveRequest) {
		req.Names = append(req.Names, names...)
	}
}

var (
	NotFoundError      = errors.Wrapf(query.NotFound, "channel not found")
	MultipleFoundError = errors.Wrapf(query.UniqueViolation, "multiple channels found")
)

func (c *Client) Retrieve(ctx context.Context, opts ...RetrieveDeleteOption) (ch Channel, err error) {
	req := api.ChannelRetrieveRequest{}
	for _, opt := range opts {
		opt(&req)
	}
	res, err := c.retrieveTransport.Send(ctx, "", req)
	if err != nil {
		return ch, err
	}
	if len(res.Channels) == 0 {
		return ch, NotFoundError
	}
	if len(res.Channels) > 1 {
		return ch, MultipleFoundError
	}
	return res.Channels[0], nil
}

func (c *Client) RetrieveMany(ctx context.Context, opts ...RetrieveDeleteOption) (chs []Channel, err error) {
	req := api.ChannelRetrieveRequest{}
	for _, opt := range opts {
		opt(&req)
	}
	res, err := c.retrieveTransport.Send(ctx, "", req)
	if err != nil {
		return chs, err
	}
	return res.Channels, nil
}

func (c *Client) Delete(ctx context.Context, opts ...RetrieveDeleteOption) error {
	rReq := api.ChannelRetrieveRequest{}
	for _, opt := range opts {
		opt(&rReq)
	}
	dReq := api.ChannelDeleteRequest{Keys: rReq.Keys, Names: rReq.Names}
	_, err := c.deleteTransport.Send(ctx, "", dReq)
	return err
}
