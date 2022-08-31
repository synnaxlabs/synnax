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
	Key      string              `json:"key" msgpack:"key"`
	Name     string              `json:"name" msgpack:"name"`
	NodeID   distribution.NodeID `json:"node_id" msgpack:"node_id" validate:"required"`
	Rate     telem.Rate          `json:"rate" msgpack:"rate" validate:"required"`
	DataType telem.DataType      `json:"data_type" msgpack:"data_type" validate:"required"`
	Density  telem.Density       `json:"density" msgpack:"density"`
}

// ChannelService is the central API for all things channel related.
type ChannelService struct {
	LoggingProvider
	ValidationProvider
	AuthProvider
	DBProvider
	Internal *channel.Service
}

func NewChannelService(p Provider) *ChannelService {
	return &ChannelService{
		Internal:           p.Config.Channel,
		ValidationProvider: p.Validation,
		AuthProvider:       p.Auth,
		LoggingProvider:    p.Logging,
		DBProvider:         p.DB,
	}
}

// ChannelCreateRequest is a request to create a channel in the cluster.
type ChannelCreateRequest struct {
	// Channel is a template for the channel to create.
	Channel Channel `json:"channel" msgpack:"channel" validate:"required"`
	// Count is the number of channels to create using the template.
	Count int `json:"count" msgpack:"count"`
}

func (c ChannelCreateRequest) applyDefaults() ChannelCreateRequest {
	if c.Count == 0 {
		c.Count = 1
	}
	return c
}

// ChannelCreateResponse is the response returned after a set of channels have
// successfully been created in the cluster.
type ChannelCreateResponse struct {
	Channels []Channel `json:"channels" msgpack:"channels"`
}

// Create creates a channel based on the parameters given in the request.
func (s *ChannelService) Create(
	ctx context.Context,
	req ChannelCreateRequest,
) (res ChannelCreateResponse, _ errors.Typed) {
	req = req.applyDefaults()
	if err := s.Validate(req); err.Occurred() {
		return res, err
	}
	return res, s.DBProvider.WithTxn(func(txn gorp.Txn) errors.Typed {
		chs, err := s.Internal.NewCreate().
			WithName(req.Channel.Name).
			WithNodeID(req.Channel.NodeID).
			WithRate(req.Channel.Rate).
			WithDataType(req.Channel.DataType).
			WithTxn(txn).
			ExecN(ctx, req.Count)
		res = ChannelCreateResponse{Channels: translateChannels(chs)}
		return errors.MaybeQuery(err)
	})
}

// ChannelRetrieveRequest is a request for retrieving information about a channel
// from the cluster.
type ChannelRetrieveRequest struct {
	// Optional parameter that queries a channel by its node ID.
	NodeID distribution.NodeID `query:"node_id"`
	// Optional parameter that queries a channel by its key.
	Keys []string `query:"keys"`
}

type ChannelRetrieveResponse struct {
	Channels []Channel `json:"channels" msgpack:"channels"`
}

// Retrieve retrieves a channel based on the parameters given in the request. If no
// parameters are specified, retrieves all channels.
func (s *ChannelService) Retrieve(
	ctx context.Context,
	req ChannelRetrieveRequest,
) (ChannelRetrieveResponse, errors.Typed) {
	var resChannels []channel.Channel
	q := s.Internal.NewRetrieve().Entries(&resChannels)

	if len(req.Keys) != 0 {
		keys, err := channel.ParseKeys(req.Keys)
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
			Key:      ch.Key().String(),
			Name:     ch.Name,
			NodeID:   ch.NodeID,
			Rate:     ch.Rate,
			DataType: ch.DataType,
			Density:  ch.Density,
		}
	}
	return translated
}
