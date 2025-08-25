// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/gorp"
)

type (
	Range struct {
		ranger.Range
		Labels []label.Label `json:"labels" msgpack:"labels"`
		Parent *ranger.Range `json:"parent" msgpack:"parent"`
	}
	RangeKVPair = ranger.KVPair
)

func translateRangesToService(ranges []Range) []ranger.Range {
	return lo.Map(ranges, func(item Range, index int) ranger.Range {
		return item.Range
	})
}

func translateRangesFromService(ranges []ranger.Range) []Range {
	return lo.Map(ranges, func(item ranger.Range, index int) Range {
		return Range{Range: item}
	})
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

func (s *RangeService) Create(ctx context.Context, req RangeCreateRequest) (res RangeCreateResponse, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: []ontology.ID{{Type: ranger.OntologyType}},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		svcRanges := translateRangesToService(req.Ranges)
		err := s.internal.NewWriter(tx).CreateManyWithParent(ctx, &svcRanges, req.Parent)
		if err != nil {
			return err
		}
		res = RangeCreateResponse{Ranges: translateRangesFromService(svcRanges)}
		return nil
	})
}

type (
	RangeRetrieveRequest struct {
		Keys          []uuid.UUID     `json:"keys" msgpack:"keys"`
		Names         []string        `json:"names" msgpack:"names"`
		SearchTerm    string          `json:"search_term" msgpack:"search_term"`
		OverlapsWith  telem.TimeRange `json:"overlaps_with" msgpack:"overlaps_with"`
		HasLabels     []uuid.UUID     `json:"has_labels" msgpack:"has_labels"`
		Limit         int             `json:"limit" msgpack:"limit"`
		Offset        int             `json:"offset" msgpack:"offset"`
		IncludeLabels bool            `json:"include_labels" msgpack:"include_labels"`
		IncludeParent bool            `json:"include_parent" msgpack:"include_parent"`
	}
	RangeRetrieveResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *RangeService) Retrieve(ctx context.Context, req RangeRetrieveRequest) (res RangeRetrieveResponse, _ error) {
	var (
		svcRanges       []ranger.Range
		q               = s.internal.NewRetrieve().Entries(&svcRanges)
		hasNames        = len(req.Names) > 0
		hasKeys         = len(req.Keys) > 0
		hasSearch       = req.SearchTerm != ""
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
		q = q.Search(req.SearchTerm)
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
	apiRanges := translateRangesFromService(svcRanges)
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
			if errors.Is(err, query.NotFound) {
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
		Action:  access.Retrieve,
		Objects: ranger.OntologyIDsFromRanges(svcRanges),
	}); err != nil {
		return RangeRetrieveResponse{}, err
	}
	return RangeRetrieveResponse{Ranges: apiRanges}, err
}

type RangeRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *RangeService) Rename(ctx context.Context, req RangeRenameRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Update,
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
		return RangeKVGetResponse{}, err
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
	Range uuid.UUID       `json:"range" msgpack:"range"`
	Pairs []ranger.KVPair `json:"pairs" msgpack:"pairs"`
}

func (s *RangeService) KVSet(ctx context.Context, req RangeKVSetRequest) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Update,
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
		Action:  access.Update,
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
	keys := make([]channel.Key, 0, len(req.Aliases))
	for k := range req.Aliases {
		keys = append(keys, k)
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: ranger.AliasOntologyIDs(req.Range, keys),
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

	keys := make([]channel.Key, 0, len(aliases))
	for a := range aliases {
		keys = append(keys, aliases[a])
	}

	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: ranger.AliasOntologyIDs(req.Range, keys),
	}); err != nil {
		return res, err
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
		Objects: ranger.AliasOntologyIDs(req.Range, req.Channels),
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
	var r ranger.Range
	if err := s.internal.NewRetrieve().Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return res, err
	}
	aliases, err := r.RetrieveAliases(ctx)
	if err != nil {
		return res, err
	}

	keys := make([]channel.Key, 0, len(aliases))
	for k := range aliases {
		keys = append(keys, k)
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: ranger.AliasOntologyIDs(req.Range, keys),
	}); err != nil {
		return res, err
	}

	return RangeAliasListResponse{Aliases: aliases}, err
}

type (
	RangeAliasRetrieveRequest struct {
		Range    uuid.UUID     `json:"range" msgpack:"range"`
		Channels []channel.Key `json:"channels" msgpack:"channels"`
	}
	RangeAliasRetrieveResponse struct {
		Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	}
)

func (s *RangeService) AliasRetrieve(ctx context.Context, req RangeAliasRetrieveRequest) (res RangeAliasRetrieveResponse, _ error) {
	var r ranger.Range
	if err := s.internal.NewRetrieve().Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil); err != nil {
		return res, err
	}

	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: ranger.AliasOntologyIDs(req.Range, req.Channels),
	}); err != nil {
		return res, err
	}

	aliases := make(map[channel.Key]string)
	for _, ch := range req.Channels {
		alias, err := r.RetrieveAlias(ctx, ch)
		if err != nil && !errors.Is(err, query.NotFound) {
			return res, err
		}
		if alias != "" {
			aliases[ch] = alias
		}
	}

	return RangeAliasRetrieveResponse{Aliases: aliases}, nil
}
