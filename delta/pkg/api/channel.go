package api

import (
	"context"
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/telem"
)

// Channel is an API-friendly version of the channel.Channel type. It is simplified for
// use purely as a data container.
type Channel struct {
	Key     string              `json:"key" msgpack:"key"`
	Name    string              `json:"name" msgpack:"name"`
	NodeID  distribution.NodeID `json:"node_id" msgpack:"node_id" validate:"required"`
	Rate    telem.Rate          `json:"data_rate" msgpack:"data_rate" validate:"required"`
	Density telem.Density       `json:"data_type" msgpack:"data_type" validate:"required"`
}

type ChannelService struct {
	CoreProvider
	AuthProvider
	DBProvider
	Internal *channel.Service
}

func NewChannelService(p Provider) *ChannelService {
	return &ChannelService{
		Internal:     p.Config.Channel,
		AuthProvider: p.Auth,
		CoreProvider: p.Core,
		DBProvider:   p.DB,
	}
}

type ChannelCreateRequest struct {
	Channel Channel `json:"channel" msgpack:"channel" validate:"required"`
	Count   int     `json:"count" msgpack:"count"`
}

func (c ChannelCreateRequest) setDefaults() ChannelCreateRequest {
	if c.Count == 0 {
		c.Count = 1
	}
	return c
}

type ChannelCreateResponse struct {
	Channels []Channel `json:"channels" msgpack:"channels"`
}

func (s *ChannelService) Create(
	ctx context.Context,
	req ChannelCreateRequest,
) (res ChannelCreateResponse, _ errors.Typed) {
	req = req.setDefaults()
	if err := s.Validate(req); err.Occurred() {
		return res, err
	}
	return res, s.DBProvider.WithTxn(func(txn gorp.Txn) errors.Typed {
		chs, err := s.Internal.NewCreate().
			WithName(req.Channel.Name).
			WithNodeID(req.Channel.NodeID).
			WithRate(req.Channel.Rate).
			WithDensity(req.Channel.Density).
			WithTxn(txn).
			ExecN(ctx, req.Count)
		res = ChannelCreateResponse{Channels: translateChannels(chs)}
		return errors.MaybeQuery(err)
	})
}

type ChannelRetrieveRequest struct {
	NodeID distribution.NodeID `query:"node_id"`
	Key    []string            `query:"keys"`
}

type ChannelRetrieveResponse struct {
	Channels []Channel `json:"channels" msgpack:"channels"`
}

func (s *ChannelService) Retrieve(
	ctx context.Context,
	req ChannelRetrieveRequest,
) (ChannelRetrieveResponse, errors.Typed) {
	var resChannels []channel.Channel
	q := s.Internal.NewRetrieve().Entries(&resChannels)

	if len(req.Key) != 0 {
		keys, err := channel.ParseKeys(req.Key)
		if err != nil {
			return ChannelRetrieveResponse{}, errors.Parse(err)
		}
		q = q.WhereKeys(keys...)
	}

	if req.NodeID != 0 {
		q = q.WhereNodeID(req.NodeID)
	}

	err := errors.MaybeQuery(q.Exec(ctx))
	return ChannelRetrieveResponse{Channels: translateChannels(resChannels)}, err
}

func translateChannels(channels []channel.Channel) []Channel {
	translated := make([]Channel, len(channels))
	for i, ch := range channels {
		translated[i] = Channel{
			Key:     ch.Key().String(),
			Name:    ch.Name,
			NodeID:  ch.NodeID,
			Rate:    ch.Rate,
			Density: ch.Density,
		}
	}
	return translated
}
