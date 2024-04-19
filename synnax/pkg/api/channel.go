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
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/ranger"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go/types"
)

// Channel is an API-friendly version of the channel.Channel type. It is simplified for
// use purely as a data container.
type Channel struct {
	Key         channel.Key          `json:"key" msgpack:"key"`
	Name        string               `json:"name" msgpack:"name"`
	Leaseholder distribution.NodeKey `json:"leaseholder" msgpack:"leaseholder"`
	Rate        telem.Rate           `json:"rate" msgpack:"rate"`
	DataType    telem.DataType       `json:"data_type" msgpack:"data_type" validate:"required"`
	Density     telem.Density        `json:"density" msgpack:"density"`
	IsIndex     bool                 `json:"is_index" msgpack:"is_index"`
	Index       channel.Key          `json:"index" msgpack:"index"`
	Alias       string               `json:"alias" msgpack:"alias"`
}

// ChannelService is the central API for all things Channel related.
type ChannelService struct {
	validationProvider
	dbProvider
	internal channel.Service
	ranger   *ranger.Service
}

func NewChannelService(p Provider) *ChannelService {
	return &ChannelService{
		internal:           p.Config.Channel,
		ranger:             p.Config.Ranger,
		validationProvider: p.Validation,
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
) (res ChannelCreateResponse, _ error) {
	if err := s.Validate(req); err != nil {
		return res, err
	}
	translated, err := translateChannelsBackward(req.Channels)
	if err != nil {
		return res, errors.Parse(err)
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.internal.NewWriter(tx).CreateMany(ctx, &translated)
		res.Channels = translateChannelsForward(translated)
		return err
	})
}

// ChannelRetrieveRequest is a request for retrieving information about a Channel
// from the cluster.
type ChannelRetrieveRequest struct {
	// Optional parameter that queries a Channel by its node Name.
	NodeKey distribution.NodeKey `json:"node_key" msgpack:"node_key"`
	// Optional parameter that queries a Channel by its key.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Optional parameter that queries a Channel by its name.
	Names []string `json:"names" msgpack:"names"`
	// Optional search parameters that fuzzy match a Channel's properties.
	Search   string    `json:"search" msgpack:"search"`
	RangeKey uuid.UUID `json:"range_key" msgpack:"range_key"`
	Limit    int       `json:"limit" msgpack:"limit"`
	Offset   int       `json:"offset" msgpack:"offset"`
}

// ChannelRetrieveResponse is the response for a ChannelRetrieveRequest.
type ChannelRetrieveResponse struct {
	// Channels is a slice of Channels matching the request.
	Channels []Channel `json:"channels" msgpack:"channels"`
}

// Retrieve retrieves a Channel based on the parameters given in the request. If no
// parameters are specified, retrieves all channels.
func (s *ChannelService) Retrieve(
	ctx context.Context,
	req ChannelRetrieveRequest,
) (ChannelRetrieveResponse, error) {
	var (
		resChannels   []channel.Channel
		aliasChannels []channel.Channel
		q             = s.internal.NewRetrieve().Entries(&resChannels)
		hasNames      = len(req.Names) > 0
		hasKeys       = len(req.Keys) > 0
		hasSearch     = len(req.Search) > 0
	)

	var resRng ranger.Range
	if req.RangeKey != uuid.Nil {
		err := s.ranger.NewRetrieve().WhereKeys(req.RangeKey).Entry(&resRng).Exec(ctx, nil)
		isNotFound := roacherrors.Is(err, query.NotFound)
		if err != nil && !isNotFound {
			return ChannelRetrieveResponse{}, errors.Auto(err)
		}
		// We can still do a best effort search without the range even if we don't find it.
		if !isNotFound && hasSearch {
			keys, err := resRng.SearchAliases(ctx, req.Search)
			if err != nil {
				return ChannelRetrieveResponse{}, errors.Auto(err)
			}
			aliasChannels = make([]channel.Channel, 0, len(keys))
			err = s.internal.NewRetrieve().WhereKeys(keys...).Entries(&aliasChannels).Exec(ctx, nil)
			if err != nil {
				return ChannelRetrieveResponse{}, errors.Auto(err)
			}
		}
	}

	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}

	if hasNames {
		q = q.WhereNames(req.Names...)
	}

	if hasSearch {
		q = q.Search(req.Search)
	}

	if req.NodeKey != 0 {
		q = q.WhereNodeKey(req.NodeKey)
	}

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}

	err := q.Exec(ctx, nil)

	if len(aliasChannels) > 0 {
		aliasKeys := channel.KeysFromChannels(aliasChannels)
		resChannels = append(aliasChannels, lo.Filter(resChannels, func(ch channel.Channel, i int) bool {
			return !aliasKeys.Contains(ch.Key())
		})...)
	}

	oChannels := translateChannelsForward(resChannels)
	if resRng.Key != uuid.Nil {
		for i, ch := range resChannels {
			al, err := resRng.GetAlias(ctx, ch.Key())
			if err == nil {
				oChannels[i].Alias = al
			}
		}
	}

	return ChannelRetrieveResponse{Channels: oChannels}, err
}

func translateChannelsForward(channels []channel.Channel) []Channel {
	translated := make([]Channel, len(channels))
	for i, ch := range channels {
		translated[i] = Channel{
			Key:         ch.Key(),
			Name:        ch.Name,
			Leaseholder: ch.Leaseholder,
			Rate:        ch.Rate,
			DataType:    ch.DataType,
			IsIndex:     ch.IsIndex,
			Index:       ch.Index(),
			Density:     ch.DataType.Density(),
		}
	}
	return translated
}

func translateChannelsBackward(channels []Channel) ([]channel.Channel, error) {
	translated := make([]channel.Channel, len(channels))
	for i, ch := range channels {
		tCH := channel.Channel{
			Name:        ch.Name,
			Leaseholder: ch.Leaseholder,
			Rate:        ch.Rate,
			DataType:    ch.DataType,
			IsIndex:     ch.IsIndex,
			LocalIndex:  ch.Index.LocalKey(),
		}
		if ch.IsIndex {
			tCH.LocalIndex = tCH.LocalKey
		}
		translated[i] = tCH
	}
	return translated, nil
}

type ChannelDeleteRequest struct {
	Keys  channel.Keys `json:"keys" msgpack:"keys" validate:"required"`
	Names []string     `json:"names" msgpack:"names" validate:"required"`
}

func (s *ChannelService) Delete(
	ctx context.Context,
	req ChannelDeleteRequest,
) (types.Nil, error) {
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		c := errutil.NewCatch(errutil.WithAggregation())
		w := s.internal.NewWriter(tx)
		if len(req.Keys) > 0 {
			c.Exec(func() error { return w.DeleteMany(ctx, req.Keys) })
		}
		if len(req.Names) > 0 {
			c.Exec(func() error { return w.DeleteManyByNames(ctx, req.Names) })
		}
		return c.Error()
	})
}
