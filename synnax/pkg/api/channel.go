// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Channel is an API-friendly version of the channel.Channel type. It is simplified for
// use purely as a data container.
type Channel struct {
	Key      string               `json:"key" msgpack:"key"`
	Name     string               `json:"name" msgpack:"name"`
	NodeKey  distribution.NodeKey `json:"node_key" msgpack:"node_key"`
	Rate     telem.Rate           `json:"rate" msgpack:"rate"`
	DataType telem.DataType       `json:"data_type" msgpack:"data_type" validate:"required"`
	Density  telem.Density        `json:"density" msgpack:"density"`
	IsIndex  bool                 `json:"is_index" msgpack:"is_index"`
	Index    string               `json:"index" msgpack:"index"`
}

// ChannelService is the central API for all things Channel related.
type ChannelService struct {
	validationProvider
	authProvider
	dbProvider
	internal channel.Service
}

func NewChannelService(p Provider) *ChannelService {
	return &ChannelService{
		internal:           p.Config.Channel,
		validationProvider: p.Validation,
		authProvider:       p.auth,
		dbProvider:         p.db,
	}
}

// ChannelCreateRequest is a request to create a Channel in the cluster.
type ChannelCreateRequest struct {
	// Channel is a template for the Channel to create.
	Channels []Channel `json:"channels" msgpack:"channels" validate:"required,dive"`
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
	if err := validateChannelNames(req.Channels); err.Occurred() {
		return res, err
	}
	translated, err := translateChannelsBackward(req.Channels)
	if err != nil {
		return res, errors.Parse(err)
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).CreateMany(ctx, &translated)
		res = ChannelCreateResponse{Channels: translateChannelsForward(translated)}
		return errors.MaybeQuery(err)
	})
}

// ChannelRetrieveRequest is a request for retrieving information about a Channel
// from the cluster.
type ChannelRetrieveRequest struct {
	// Optional parameter that queries a Channel by its node Key.
	NodeKey distribution.NodeKey `query:"node_key"`
	// Optional parameter that queries a Channel by its key.
	KeysOrNames []string `query:"keys_or_names"`
}

// ChannelRetrieveResponse is the response for a ChannelRetrieveRequest.
type ChannelRetrieveResponse struct {
	// Channels is a slice of Channels matching the request.
	Channels []Channel `json:"channels" msgpack:"channels"`
	// NotFound is a slice of strings matching the keys or names of Channels that
	// were not found.
	NotFound []string `json:"not_found" msgpack:"not_found"`
}

// Retrieve retrieves a Channel based on the parameters given in the request. If no
// parameters are specified, retrieves all channels.
func (s *ChannelService) Retrieve(
	ctx context.Context,
	req ChannelRetrieveRequest,
) (ChannelRetrieveResponse, errors.Typed) {
	var (
		keys, names = splitKeysAndNames(req.KeysOrNames)
		resChannels []channel.Channel
		notFound    []string
		q           = s.internal.NewRetrieve().Entries(&resChannels)
		hasNames    = len(names) > 0
		hasKeys     = len(keys) > 0
	)

	if hasKeys {
		q = q.WhereKeys(keys...)
	}

	if hasNames {
		q = q.WhereNames(names...)
	}

	if req.NodeKey != 0 {
		q = q.WhereNodeKey(req.NodeKey)
	}

	err := errors.MaybeQuery(q.Exec(ctx, nil))

	if hasKeys {
		notFound, _ = lo.Difference(
			keys.Strings(),
			channel.KeysFromChannels(resChannels).Strings(),
		)
	}

	if hasNames {
		_notFound, _ := lo.Difference(
			names,
			lo.Map(resChannels, func(ch channel.Channel, _ int) string { return ch.Name }),
		)
		notFound = append(notFound, _notFound...)
	}

	return ChannelRetrieveResponse{Channels: translateChannelsForward(resChannels), NotFound: notFound}, err
}

func splitKeysAndNames(keysOrNames []string) (channel.Keys, []string) {
	var (
		keys  []string
		names []string
	)
	for _, keyOrName := range keysOrNames {
		if channel.IsValidStringKey(keyOrName) {
			keys = append(keys, keyOrName)
		} else {
			names = append(names, keyOrName)
		}
	}
	channelKeys, _ := channel.ParseKeys(keys)
	return channelKeys, names
}

func translateChannelsForward(channels []channel.Channel) []Channel {
	translated := make([]Channel, len(channels))
	for i, ch := range channels {
		translated[i] = Channel{
			Key:      ch.Key().String(),
			Name:     ch.Name,
			NodeKey:  ch.NodeKey,
			Rate:     ch.Rate,
			DataType: ch.DataType,
			IsIndex:  ch.IsIndex,
			Index:    ch.Index().String(),
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
			NodeKey:  ch.NodeKey,
			Rate:     ch.Rate,
			DataType: ch.DataType,
			IsIndex:  ch.IsIndex,
		}
		if ch.Key != "" {
			key, err := channel.ParseKey(ch.Key)
			if err != nil {
				return nil, err
			}
			tCH.StorageKey = key.LocalKey()
		}
		if ch.Index != "" {
			index, err := channel.ParseKey(ch.Index)
			if err != nil {
				return nil, err
			}
			tCH.LocalIndex = index.LocalKey()
		}
		if ch.IsIndex {
			tCH.LocalIndex = tCH.StorageKey
		}
		translated[i] = tCH
	}
	return translated, nil
}

func validateChannelNames(channels []Channel) errors.Typed {
	for _, ch := range channels {
		if channel.IsValidStringKey(ch.Name) {
			return errors.Validation(roacherrors.Newf(
				"channel name %s cannot have the same format as a key",
				ch.Name,
			))
		}
	}
	return errors.Nil
}
