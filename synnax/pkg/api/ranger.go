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
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/ranger"
	"github.com/synnaxlabs/x/gorp"
)

type (
	Range       = ranger.Range
	RangeKVPair = ranger.KVPair
)

type RangeService struct {
	dbProvider
	accessProvider
	internal *ranger.Service
}

func NewRangeService(p Provider) *RangeService {
	return &RangeService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Config.Ranger,
	}
}

type (
	RangeCreateRequest struct {
		Parent ontology.ID `json:"parent" msgpack:"parent"`
		Ranges []Range     `json:"ranges" msgpack:"ranges"`
	}
	RangeCreateResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *RangeService) Create(ctx context.Context, req RangeCreateRequest) (res RangeCreateResponse, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: []ontology.ID{{Type: ranger.OntologyType}},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.internal.NewWriter(tx).CreateManyWithParent(ctx, &req.Ranges, req.Parent)
		res = RangeCreateResponse{Ranges: req.Ranges}
		return err
	})
}

type (
	RangeRetrieveRequest struct {
		Keys         []uuid.UUID     `json:"keys" msgpack:"keys"`
		Names        []string        `json:"names" msgpack:"names"`
		Term         string          `json:"term" msgpack:"term"`
		OverlapsWith telem.TimeRange `json:"overlaps_with" msgpack:"overlaps_with"`
		Limit        int             `json:"limit" msgpack:"limit"`
		Offset       int             `json:"offset" msgpack:"offset"`
	}
	RangeRetrieveResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *RangeService) Retrieve(ctx context.Context, req RangeRetrieveRequest) (res RangeRetrieveResponse, _ error) {
	var (
		resRanges       []ranger.Range
		q               = s.internal.NewRetrieve().Entries(&resRanges)
		hasNames        = len(req.Names) > 0
		hasKeys         = len(req.Keys) > 0
		hasSearch       = req.Term != ""
		hasOverlapsWith = !req.OverlapsWith.IsZero()
	)
	if hasOverlapsWith {
		q = q.WhereOverlapsWith(req.OverlapsWith)
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasSearch {
		q = q.Search(req.Term)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	err := q.Exec(ctx, nil)
	if err != nil {
		return RangeRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: ranger.OntologyIDsFromRanges(resRanges),
	}); err != nil {
		return RangeRetrieveResponse{}, err
	}
	return RangeRetrieveResponse{Ranges: resRanges}, err
}

type RangeRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *RangeService) Rename(ctx context.Context, req RangeRenameRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Rename,
		Objects: []ontology.ID{ranger.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type RangeDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *RangeService) Delete(ctx context.Context, req RangeDeleteRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: ranger.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		for _, key := range req.Keys {
			if err := s.internal.NewWriter(tx).Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	RangeKVGetRequest struct {
		Range uuid.UUID `json:"range" msgpack:"range"`
		Keys  []string  `json:"keys" msgpack:"keys"`
	}
	RangeKVGetResponse struct {
		Pairs []ranger.KVPair `json:"pairs" msgpack:"pairs"`
	}
)

func (s *RangeService) KVGet(ctx context.Context, req RangeKVGetRequest) (res RangeKVGetResponse, err error) {
	var r ranger.Range
	if err = s.internal.NewRetrieve().Entry(&r).WhereKeys(req.Range).Exec(ctx, nil); err != nil {
		return
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return RangeKVGetResponse{}, err
	}
	if len(req.Keys) == 0 {
		res.Pairs, err = r.ListKV()
		return
	}
	res.Pairs, err = r.GetManyKV(ctx, req.Keys)
	return
}

type RangeKVSetRequest struct {
	Range uuid.UUID       `json:"range" msgpack:"range"`
	Pairs []ranger.KVPair `json:"pairs" msgpack:"pairs"`
}

func (s *RangeService) KVSet(ctx context.Context, req RangeKVSetRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx); err != nil {
			return err
		}
		rng = rng.UseTx(tx)
		return rng.SetManyKV(ctx, req.Pairs)
	})
}

type RangeKVDeleteRequest struct {
	Range uuid.UUID `json:"range" msgpack:"range"`
	Keys  []string  `json:"keys" msgpack:"keys"`
}

func (s *RangeService) KVDelete(ctx context.Context, req RangeKVDeleteRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx); err != nil {
			return err
		}
		rng = rng.UseTx(tx)
		for _, key := range req.Keys {
			if err := rng.DeleteKV(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

type RangeAliasSetRequest struct {
	Range   uuid.UUID              `json:"range" msgpack:"range"`
	Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
}

func (s *RangeService) AliasSet(ctx context.Context, req RangeAliasSetRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx); err != nil {
			return err
		}
		rng = rng.UseTx(tx)
		for k, v := range req.Aliases {
			if err := rng.SetAlias(ctx, k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	RangeAliasResolveRequest struct {
		Range   uuid.UUID `json:"range" msgpack:"range"`
		Aliases []string  `json:"aliases" msgpack:"aliases"`
	}
	RangeAliasResolveResponse struct {
		Aliases map[string]channel.Key `json:"aliases" msgpack:"aliases"`
	}
)

func (s *RangeService) AliasResolve(ctx context.Context, req RangeAliasResolveRequest) (res RangeAliasResolveResponse, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return res, err
	}

	var r ranger.Range
	if err := s.internal.NewRetrieve().Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return res, err
	}
	aliases := make(map[string]channel.Key, len(req.Aliases))
	for _, alias := range req.Aliases {
		ch, err := r.ResolveAlias(ctx, alias)
		if err != nil && !errors.Is(err, query.NotFound) {
			return res, err
		}
		if ch != 0 {
			aliases[alias] = ch
		}
	}
	return RangeAliasResolveResponse{Aliases: aliases}, nil
}

type RangeAliasDeleteRequest struct {
	Range    uuid.UUID     `json:"range" msgpack:"range"`
	Channels []channel.Key `json:"channels" msgpack:"channels"`
}

func (s *RangeService) AliasDelete(ctx context.Context, req RangeAliasDeleteRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx); err != nil {
			return err
		}
		rng = rng.UseTx(tx)
		for _, alias := range req.Channels {
			if err := rng.DeleteAlias(ctx, alias); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	RangeAliasListRequest struct {
		Range uuid.UUID `json:"range" msgpack:"range"`
	}
	RangeAliasListResponse struct {
		Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	}
)

func (s *RangeService) AliasList(ctx context.Context, req RangeAliasListRequest) (res RangeAliasListResponse, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return res, err
	}
	var r ranger.Range
	if err := s.internal.NewRetrieve().Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return res, err
	}
	aliases, err := r.ListAliases(ctx)
	return RangeAliasListResponse{Aliases: aliases}, err
}
