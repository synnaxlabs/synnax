package api

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Channel is an API-friendly version of the channel.Channel type. It is simplified for
// use purely as a data container.
type Channel struct {
	Key      string              `json:"key" msgpack:"key"`
	Name     string              `json:"name" msgpack:"name"`
	NodeID   distribution.NodeID `json:"node_id" msgpack:"node_id"`
	Rate     telem.Rate          `json:"rate" msgpack:"rate" validate:"required"`
	DataType telem.DataType      `json:"data_type" msgpack:"data_type" validate:"required"`
	Density  telem.Density       `json:"density" msgpack:"density"`
}

// ChannelService is the central API for all things Channel related.
type ChannelService struct {
	loggingProvider
	validationProvider
	authProvider
	dbProvider
	internal *channel.Service
}

func NewChannelService(p Provider) *ChannelService {
	return &ChannelService{
		internal:           p.Config.Channel,
		validationProvider: p.Validation,
		authProvider:       p.auth,
		loggingProvider:    p.Logging,
		dbProvider:         p.db,
	}
}

// ChannelCreateRequest is a request to create a Channel in the cluster.
type ChannelCreateRequest struct {
	// Channel is a template for the Channel to create.
	Channel Channel `json:"Channel" msgpack:"Channel" validate:"required"`
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

// Create creates a Channel based on the parameters given in the request.
func (s *ChannelService) Create(
	ctx context.Context,
	req ChannelCreateRequest,
) (res ChannelCreateResponse, _ errors.Typed) {
	req = req.applyDefaults()
	if err := s.Validate(req); err.Occurred() {
		return res, err
	}
	return res, s.dbProvider.WithTxn(func(txn gorp.Txn) errors.Typed {
		chs, err := s.internal.NewCreate().
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

// ChannelRetrieveRequest is a request for retrieving information about a Channel
// from the cluster.
type ChannelRetrieveRequest struct {
	// Optional parameter that queries a Channel by its node ID.
	NodeID distribution.NodeID `query:"node_id"`
	// Optional parameter that queries a Channel by its key.
	Keys []string `query:"keys"`
}

type ChannelRetrieveResponse struct {
	Channels []Channel `json:"channels" msgpack:"channels"`
}

// Retrieve retrieves a Channel based on the parameters given in the request. If no
// parameters are specified, retrieves all channels.
func (s *ChannelService) Retrieve(
	ctx context.Context,
	req ChannelRetrieveRequest,
) (ChannelRetrieveResponse, errors.Typed) {
	var resChannels []channel.Channel
	q := s.internal.NewRetrieve().Entries(&resChannels)

	if len(req.Keys) > 0 {
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
