// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Service is the central service for all things Channel related.
type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *channel.Service
	ranger   *ranger.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		access:   cfg.Service.RBAC,
		internal: cfg.Distribution.Channel,
		ranger:   cfg.Service.Ranger,
		db:       cfg.Distribution.DB,
	}
}

// CreateRequest is a request to create a Channel in the cluster.
type CreateRequest struct {
	// Channel is a template for the Channel to create.
	Channels             []Channel `json:"channels" msgpack:"channels"`
	RetrieveIfNameExists bool      `json:"retrieve_if_name_exists" msgpack:"retrieve_if_name_exists"`
}

// CreateResponse is the response returned after a set of channels have
// successfully been created in the cluster.
type CreateResponse struct {
	Channels []Channel `json:"channels" msgpack:"channels"`
}

// Create creates a Channel based on the parameters given in the request.
func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	translated, err := translateChannelsBackward(req.Channels)
	if err != nil {
		return CreateResponse{}, err
	}
	for i := range translated {
		translated[i].Internal = false
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: channel.OntologyIDsFromChannels(translated),
	}); err != nil {
		return CreateResponse{}, err
	}
	var res CreateResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		opts := []channel.CreateOption{}
		if req.RetrieveIfNameExists {
			opts = append(opts, channel.RetrieveIfNameExists())
		}
		if err := w.CreateMany(ctx, &translated, opts...); err != nil {
			return err
		}
		res.Channels = translateChannelsForward(translated)
		return nil
	}); err != nil {
		return CreateResponse{}, err
	}
	return res, nil
}

// RetrieveRequest is a request for retrieving information about a Channel
// from the cluster.
type RetrieveRequest struct {
	// Optional parameter that queries a Channel by its node Name.
	NodeKey cluster.NodeKey `json:"node_key" msgpack:"node_key"`
	// Optional parameter that queries a Channel by its key.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Optional parameter that queries a Channel by its name.
	Names []string `json:"names" msgpack:"names"`
	// Optional search parameters that fuzzy match a Channel's properties.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// RangeKey is used for fetching aliases.
	RangeKey uuid.UUID `json:"range_key" msgpack:"range_key"`
	// Limit limits the number of results returned.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset offsets the results returned.
	Offset int `json:"offset" msgpack:"offset"`
	// DataTypes filters for channels whose DataType attribute matches the provided data types.
	DataTypes []telem.DataType `json:"data_types" msgpack:"data_types"`
	// NotDataTypes filters for channels whose DataType attribute does not match the provided data types.
	NotDataTypes []telem.DataType `json:"not_data_types" msgpack:"not_data_types"`
	// Virtual filters for channels that are virtual if true, or are not virtual if false.
	Virtual *bool `json:"virtual" msgpack:"virtual"`
	// IsIndex filters for channels that are indexes if true, or are not indexes if false.
	IsIndex *bool `json:"is_index" msgpack:"is_index"`
	// Internal filters for channels that are internal if true, or are not internal if false.
	Internal *bool `json:"internal" msgpack:"internal"`
}

// RetrieveResponse is the response for a RetrieveRequest.
type RetrieveResponse struct {
	// Channels is a slice of Channels matching the request.
	Channels []Channel `json:"channels" msgpack:"channels"`
}

// Retrieve retrieves a Channel based on the parameters given in the request. If no
// parameters are specified, retrieves all channels.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		resChannels     []channel.Channel
		aliasChannels   []channel.Channel
		q               = s.internal.NewRetrieve().Entries(&resChannels)
		hasNames        = len(req.Names) > 0
		hasKeys         = len(req.Keys) > 0
		hasSearch       = len(req.SearchTerm) > 0
		hasDataTypes    = len(req.DataTypes) > 0
		hasNotDataTypes = len(req.NotDataTypes) > 0
	)

	var resRng ranger.Range
	if req.RangeKey != uuid.Nil {
		err := s.ranger.NewRetrieve().WhereKeys(req.RangeKey).Entry(&resRng).Exec(ctx, nil)
		isNotFound := errors.Is(err, query.NotFound)
		if err != nil && !isNotFound {
			return RetrieveResponse{}, err
		}
		// We can still do a best effort search without the range even if we don't find it.
		if !isNotFound && hasSearch {
			//keys, err := resRng.SearchAliases(ctx, req.SearchTerm)
			//if err != nil {
			//	return RetrieveResponse{}, err
			//}
			//aliasChannels = make([]channel.Channel, 0, len(keys))
			//err = s.internal.NewRetrieve().WhereKeys(keys...).Entries(&aliasChannels).Exec(ctx, nil)
			//if err != nil {
			//	return RetrieveResponse{}, err
			//}
		}
	}
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if req.NodeKey != 0 {
		q = q.WhereNodeKey(req.NodeKey)
	}
	if hasDataTypes {
		q = q.WhereDataTypes(req.DataTypes...)
	}
	if hasNotDataTypes {
		q = q.WhereNotDataTypes(req.NotDataTypes...)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if req.Virtual != nil {
		q = q.WhereVirtual(*req.Virtual)
	}
	if req.IsIndex != nil {
		q = q.WhereIsIndex(*req.IsIndex)
	}
	if req.Internal != nil {
		q = q.WhereInternal(*req.Internal)
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if len(aliasChannels) > 0 {
		aliasKeys := channel.KeysFromChannels(aliasChannels)
		resChannels = append(aliasChannels, lo.Filter(resChannels, func(ch channel.Channel, _ int) bool {
			return !aliasKeys.Contains(ch.Key())
		})...)
	}
	oChannels := translateChannelsForward(resChannels)
	if resRng.Key != uuid.Nil {
		//for i, ch := range resChannels {
		//al, err := resRng.RetrieveAlias(ctx, ch.Key())
		//if err == nil {
		//	oChannels[i].Alias = al
		//}
		//}
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: channel.OntologyIDsFromChannels(resChannels),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Channels: oChannels}, nil
}

func translateChannelsForward(channels []channel.Channel) []Channel {
	translated := make([]Channel, len(channels))
	for i, ch := range channels {
		translated[i] = Channel{
			Key:         ch.Key(),
			Name:        ch.Name,
			Leaseholder: uint32(ch.Leaseholder),
			DataType:    string(ch.DataType),
			IsIndex:     ch.IsIndex,
			Index:       ch.Index(),
			Density:     int64(ch.DataType.Density()),
			Virtual:     ch.Virtual,
			Internal:    ch.Internal,
			Expression:  ch.Expression,
			Operations:  ch.Operations,
		}
	}
	return translated
}

// translateChannelsBackward translates a slice of api channel structs to a slice of
// internal channel structs.
func translateChannelsBackward(channels []Channel) ([]channel.Channel, error) {
	translated := make([]channel.Channel, len(channels))
	for i, ch := range channels {
		tCh := channel.Channel{
			Name:        ch.Name,
			Leaseholder: cluster.NodeKey(ch.Leaseholder),
			DataType:    telem.DataType(ch.DataType),
			IsIndex:     ch.IsIndex,
			LocalIndex:  channel.Key(ch.Index).LocalKey(),
			LocalKey:    channel.Key(ch.Key).LocalKey(),
			Virtual:     ch.Virtual,
			Internal:    ch.Internal,
			Expression:  ch.Expression,
			Operations:  ch.Operations,
		}
		if ch.IsIndex {
			tCh.LocalIndex = tCh.LocalKey
		}

		translated[i] = tCh
	}
	return translated, nil
}

type DeleteRequest struct {
	Keys  channel.Keys `json:"keys" msgpack:"keys" validate:"required"`
	Names []string     `json:"names" msgpack:"names" validate:"required"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		c := errors.NewCatcher(errors.WithAggregation())
		w := s.internal.NewWriter(tx)
		if len(req.Keys) > 0 {
			c.Exec(func() error {
				if err := s.access.Enforce(ctx, access.Request{
					Subject: auth.GetSubject(ctx),
					Action:  access.ActionDelete,
					Objects: req.Keys.OntologyIDs(),
				}); err != nil {
					return err
				}
				return w.DeleteMany(ctx, req.Keys, false)
			})
		}
		if len(req.Names) > 0 {
			c.Exec(func() error {
				res := make([]channel.Channel, 0, len(req.Names))
				err := s.internal.NewRetrieve().WhereNames(req.Names...).Entries(&res).Exec(ctx, tx)
				if err != nil {
					return err
				}
				if err = s.access.Enforce(ctx, access.Request{
					Subject: auth.GetSubject(ctx),
					Action:  access.ActionDelete,
					Objects: channel.OntologyIDsFromChannels(res),
				}); err != nil {
					return err
				}
				return w.DeleteManyByNames(ctx, req.Names, false)
			})
		}
		return c.Error()
	})
}

type RenameRequest struct {
	Keys  channel.Keys `json:"keys" msgpack:"keys" validate:"required"`
	Names []string     `json:"names" msgpack:"names" validate:"required"`
}

func (s *Service) Rename(
	ctx context.Context,
	req RenameRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: req.Keys.OntologyIDs(),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).RenameMany(ctx, req.Keys, req.Names, false)
	})
}

type RetrieveGroupRequest struct{}

type RetrieveGroupResponse struct {
	Group group.Group `json:"group" msgpack:"group"`
}

func (s *Service) RetrieveGroup(
	ctx context.Context,
	_ RetrieveGroupRequest,
) (RetrieveGroupResponse, error) {
	g := s.internal.Group()
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{g.OntologyID()},
	}); err != nil {
		return RetrieveGroupResponse{}, err
	}
	return RetrieveGroupResponse{Group: g}, nil
}
