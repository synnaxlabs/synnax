// Copyright 2026 Synnax Labs, Inc.
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
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

type (
	Range struct {
		Parent *ranger.Range `json:"parent" msgpack:"parent"`
		Labels []label.Label `json:"labels" msgpack:"labels"`
		ranger.Range
	}
	RangeKVPair = ranger.KVPair
)

func translateRangesToService(ranges []Range) []ranger.Range {
	return lo.Map(ranges, func(r Range, _ int) ranger.Range { return r.Range })
}

func translateRangesFromService(ranges []ranger.Range) []Range {
	return lo.Map(ranges, func(r ranger.Range, _ int) Range { return Range{Range: r} })
}

func rangeAccessOntologyIDs(ranges []Range) []ontology.ID {
	ids := make([]ontology.ID, 0, len(ranges))
	for _, r := range ranges {
		ids = append(ids, r.OntologyID())
		if r.Parent != nil {
			ids = append(ids, r.Parent.OntologyID())
		}
		labels := label.OntologyIDsFromLabels(r.Labels)
		ids = append(ids, labels...)
	}
	return ids
}

type RangeService struct {
	dbProvider
	accessProvider
	internal *ranger.Service
}

func NewRangeService(p Provider) *RangeService {
	return &RangeService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Service.Ranger,
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

func (s *RangeService) Create(
	ctx context.Context,
	req RangeCreateRequest,
) (RangeCreateResponse, error) {
	ids := rangeAccessOntologyIDs(req.Ranges)
	if !req.Parent.IsZero() {
		ids = append(ids, req.Parent)
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: ids,
	}); err != nil {
		return RangeCreateResponse{}, err
	}
	var res RangeCreateResponse
	if err := s.WithTx(ctx, func(tx gorp.Tx) error {
		svcRanges := translateRangesToService(req.Ranges)
		if err := s.
			internal.
			NewWriter(tx).
			CreateManyWithParent(ctx, &svcRanges, req.Parent); err != nil {
			return err
		}
		res = RangeCreateResponse{Ranges: translateRangesFromService(svcRanges)}
		return nil
	}); err != nil {
		return RangeCreateResponse{}, err
	}
	return res, nil
}

type (
	RangeRetrieveRequest struct {
		SearchTerm    string          `json:"search_term" msgpack:"search_term"`
		Keys          []uuid.UUID     `json:"keys" msgpack:"keys"`
		Names         []string        `json:"names" msgpack:"names"`
		HasLabels     []uuid.UUID     `json:"has_labels" msgpack:"has_labels"`
		OverlapsWith  telem.TimeRange `json:"overlaps_with" msgpack:"overlaps_with"`
		Limit         int             `json:"limit" msgpack:"limit"`
		Offset        int             `json:"offset" msgpack:"offset"`
		IncludeLabels bool            `json:"include_labels" msgpack:"include_labels"`
		IncludeParent bool            `json:"include_parent" msgpack:"include_parent"`
	}
	RangeRetrieveResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *RangeService) Retrieve(
	ctx context.Context,
	req RangeRetrieveRequest,
) (RangeRetrieveResponse, error) {
	var (
		svcRanges       []ranger.Range
		q               = s.internal.NewRetrieve().Entries(&svcRanges)
		hasNames        = len(req.Names) > 0
		hasKeys         = len(req.Keys) > 0
		hasSearch       = req.SearchTerm != ""
		hasOverlapsWith = !req.OverlapsWith.IsZero()
		hasLabels       = len(req.HasLabels) > 0
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
	if hasLabels {
		q = q.WhereHasLabels(req.HasLabels...)
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RangeRetrieveResponse{}, err
	}
	apiRanges := translateRangesFromService(svcRanges)
	var err error
	if req.IncludeLabels {
		for i, rng := range apiRanges {
			if rng.Labels, err = rng.RetrieveLabels(ctx); err != nil {
				return RangeRetrieveResponse{}, err
			}
			apiRanges[i] = rng
		}
	}
	if req.IncludeParent {
		for i, rng := range apiRanges {
			parent, err := rng.RetrieveParent(ctx)
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			if err != nil {
				return RangeRetrieveResponse{}, err
			}
			rng.Parent = &parent
			apiRanges[i] = rng
		}
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rangeAccessOntologyIDs(apiRanges),
	}); err != nil {
		return RangeRetrieveResponse{}, err
	}
	return RangeRetrieveResponse{Ranges: apiRanges}, nil
}

type RangeRenameRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  uuid.UUID `json:"key" msgpack:"key"`
}

func (s *RangeService) Rename(
	ctx context.Context,
	req RangeRenameRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type RangeDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *RangeService) Delete(
	ctx context.Context,
	req RangeDeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: ranger.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for _, key := range req.Keys {
			if err := w.Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	RangeKVGetRequest struct {
		Keys  []string  `json:"keys" msgpack:"keys"`
		Range uuid.UUID `json:"range" msgpack:"range"`
	}
	RangeKVGetResponse struct {
		Pairs []ranger.KVPair `json:"pairs" msgpack:"pairs"`
	}
)

func (s *RangeService) KVGet(
	ctx context.Context,
	req RangeKVGetRequest,
) (RangeKVGetResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return RangeKVGetResponse{}, err
	}
	var r ranger.Range
	if err := s.
		internal.
		NewRetrieve().
		Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return RangeKVGetResponse{}, err
	}

	var (
		res RangeKVGetResponse
		err error
	)
	if len(req.Keys) == 0 {
		res.Pairs, err = r.ListKV(ctx)
		if err != nil {
			return RangeKVGetResponse{}, err
		}
		return res, nil
	}
	res.Pairs, err = r.GetManyKV(ctx, req.Keys)
	if err != nil {
		return RangeKVGetResponse{}, err
	}
	return res, nil
}

type RangeKVSetRequest struct {
	Pairs []ranger.KVPair `json:"pairs" msgpack:"pairs"`
	Range uuid.UUID       `json:"range" msgpack:"range"`
}

func (s *RangeService) KVSet(
	ctx context.Context,
	req RangeKVSetRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.
			internal.
			NewRetrieve().
			Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx); err != nil {
			return err
		}
		rng = rng.UseTx(tx)
		return rng.SetManyKV(ctx, req.Pairs)
	})
}

type RangeKVDeleteRequest struct {
	Keys  []string  `json:"keys" msgpack:"keys"`
	Range uuid.UUID `json:"range" msgpack:"range"`
}

func (s *RangeService) KVDelete(
	ctx context.Context,
	req RangeKVDeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.
			internal.
			NewRetrieve().
			Entry(&rng).
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
	Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	Range   uuid.UUID              `json:"range" msgpack:"range"`
}

func (s *RangeService) AliasSet(
	ctx context.Context,
	req RangeAliasSetRequest,
) (types.Nil, error) {
	keys := lo.Keys(req.Aliases)
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: ranger.AliasOntologyIDs(req.Range, keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.
			internal.
			NewRetrieve().
			Entry(&rng).
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
		Aliases []string  `json:"aliases" msgpack:"aliases"`
		Range   uuid.UUID `json:"range" msgpack:"range"`
	}
	RangeAliasResolveResponse struct {
		Aliases map[string]channel.Key `json:"aliases" msgpack:"aliases"`
	}
)

func (s *RangeService) AliasResolve(
	ctx context.Context,
	req RangeAliasResolveRequest,
) (RangeAliasResolveResponse, error) {
	var r ranger.Range
	if err := s.
		internal.
		NewRetrieve().
		Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return RangeAliasResolveResponse{}, err
	}
	aliases := make(map[string]channel.Key, len(req.Aliases))

	for _, alias := range req.Aliases {
		ch, err := r.ResolveAlias(ctx, alias)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return RangeAliasResolveResponse{}, err
		}
		if ch != 0 {
			aliases[alias] = ch
		}
	}

	keys := lo.Values(aliases)

	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ranger.AliasOntologyIDs(req.Range, keys),
	}); err != nil {
		return RangeAliasResolveResponse{}, err
	}

	return RangeAliasResolveResponse{Aliases: aliases}, nil
}

type RangeAliasDeleteRequest struct {
	Channels []channel.Key `json:"channels" msgpack:"channels"`
	Range    uuid.UUID     `json:"range" msgpack:"range"`
}

func (s *RangeService) AliasDelete(
	ctx context.Context,
	req RangeAliasDeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: ranger.AliasOntologyIDs(req.Range, req.Channels),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		var rng ranger.Range
		if err := s.
			internal.
			NewRetrieve().
			Entry(&rng).
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

func (s *RangeService) AliasList(
	ctx context.Context,
	req RangeAliasListRequest,
) (RangeAliasListResponse, error) {
	var r ranger.Range
	if err := s.
		internal.
		NewRetrieve().
		Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return RangeAliasListResponse{}, err
	}
	aliases, err := r.RetrieveAliases(ctx)
	if err != nil {
		return RangeAliasListResponse{}, err
	}
	keys := lo.Keys(aliases)
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ranger.AliasOntologyIDs(req.Range, keys),
	}); err != nil {
		return RangeAliasListResponse{}, err
	}

	return RangeAliasListResponse{Aliases: aliases}, nil
}

type (
	RangeAliasRetrieveRequest struct {
		Channels []channel.Key `json:"channels" msgpack:"channels"`
		Range    uuid.UUID     `json:"range" msgpack:"range"`
	}
	RangeAliasRetrieveResponse struct {
		Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	}
)

func (s *RangeService) AliasRetrieve(
	ctx context.Context,
	req RangeAliasRetrieveRequest,
) (RangeAliasRetrieveResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ranger.AliasOntologyIDs(req.Range, req.Channels),
	}); err != nil {
		return RangeAliasRetrieveResponse{}, err
	}
	var r ranger.Range
	if err := s.
		internal.
		NewRetrieve().
		Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return RangeAliasRetrieveResponse{}, err
	}

	aliases := make(map[channel.Key]string)
	for _, ch := range req.Channels {
		alias, err := r.RetrieveAlias(ctx, ch)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return RangeAliasRetrieveResponse{}, err
		}
		if alias != "" {
			aliases[ch] = alias
		}
	}

	return RangeAliasRetrieveResponse{Aliases: aliases}, nil
}
