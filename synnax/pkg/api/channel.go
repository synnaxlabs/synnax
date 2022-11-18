package api

import (
	"context"
	"github.com/sirupsen/logrus"
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
	Rate     telem.Rate          `json:"rate" msgpack:"rate"`
	DataType telem.DataType      `json:"data_type" msgpack:"data_type" validate:"required"`
	Density  telem.Density       `json:"density" msgpack:"density"`
	IsIndex  bool                `json:"is_index" msgpack:"is_index"`
	Index    string              `json:"index" msgpack:"index"`
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
	Channels []Channel `json:"channels" msgpack:"channels" validate:"required"`
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
	if err := s.Validate(req); err.Occurred() {
		return res, err
	}
	translated, err := translateChannelsBackward(req.Channels)
	if err != nil {
		return res, errors.Parse(err)
	}
	return res, s.dbProvider.WithTxn(func(txn gorp.Txn) errors.Typed {
		err := s.internal.CreateManyWithTxn(txn, &translated)
		res = ChannelCreateResponse{Channels: translateChannelsForward(translated)}
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
	// Optional parameter that queries a Channel by its name.
	Names []string `query:"names"`
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

	logrus.Info(req)

	if len(req.Keys) > 0 {
		keys, err := channel.ParseKeys(req.Keys)
		if err != nil {
			return ChannelRetrieveResponse{}, errors.Parse(err)
		}
		q = q.WhereKeys(keys...)
	}

	if len(req.Names) > 0 {
		q = q.WhereNames(req.Names...)
	}

	if req.NodeID != 0 {
		q = q.WhereNodeID(req.NodeID)
	}

	err := errors.MaybeQuery(q.Exec(ctx))
	return ChannelRetrieveResponse{Channels: translateChannelsForward(resChannels)}, err
}

func translateChannelsForward(channels []channel.Channel) []Channel {
	translated := make([]Channel, len(channels))
	for i, ch := range channels {
		translated[i] = Channel{
			Key:      ch.Key().String(),
			Name:     ch.Name,
			NodeID:   ch.NodeID,
			Rate:     ch.Rate,
			DataType: ch.DataType,
			IsIndex:  ch.IsIndex,
			Index:    ch.StorageIndex.String(),
			Density:  ch.DataType.Density(),
		}
	}
	return translated
}

func translateChannelsBackward(channels []Channel) ([]channel.Channel, error) {
	translated := make([]channel.Channel, len(channels))
	for i, ch := range channels {
		tCH := channel.Channel{
			Name:     ch.Name,
			NodeID:   ch.NodeID,
			Rate:     ch.Rate,
			DataType: ch.DataType,
			IsIndex:  ch.IsIndex,
		}
		if ch.Key != "" {
			key, err := channel.ParseKey(ch.Key)
			if err != nil {
				return nil, err
			}
			tCH.StorageKey = key.StorageKey()
		}
		if ch.Index != "" {
			index, err := channel.ParseKey(ch.Index)
			if err != nil {
				return nil, err
			}
			tCH.StorageIndex = index.StorageKey()
		}
		translated[i] = tCH
	}
	return translated, nil
}
