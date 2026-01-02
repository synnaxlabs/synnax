// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
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

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *ranger.Service
	label    *label.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Ranger,
		label:    cfg.Service.Label,
	}
}

type (
	CreateRequest struct {
		Parent ontology.ID `json:"parent" msgpack:"parent"`
		Ranges []Range     `json:"ranges" msgpack:"ranges"`
	}
	CreateResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	ids := rangeAccessOntologyIDs(req.Ranges)
	if !req.Parent.IsZero() {
		ids = append(ids, req.Parent)
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: ids,
	}); err != nil {
		return CreateResponse{}, err
	}
	var res CreateResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		svcRanges := translateRangesToService(req.Ranges)
		if err := s.
			internal.
			NewWriter(tx).
			CreateManyWithParent(ctx, &svcRanges, req.Parent); err != nil {
			return err
		}
		res = CreateResponse{Ranges: translateRangesFromService(svcRanges)}
		return nil
	}); err != nil {
		return CreateResponse{}, err
	}
	return res, nil
}

type (
	RetrieveRequest struct {
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
	RetrieveResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
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
		return RetrieveResponse{}, err
	}
	apiRanges := translateRangesFromService(svcRanges)
	var err error
	if req.IncludeLabels {
		for i, rng := range apiRanges {
			if rng.Labels, err = s.label.RetrieveFor(ctx, rng.OntologyID(), nil); err != nil {
				return RetrieveResponse{}, err
			}
			apiRanges[i] = rng
		}
	}
	if req.IncludeParent {
		for i, rng := range apiRanges {
			parentKey, err := s.internal.RetrieveParentKey(ctx, rng.Key, nil)
			if errors.Is(err, query.NotFound) {
				continue
			}
			if err != nil {
				return RetrieveResponse{}, err
			}
			var parent ranger.Range
			if err = s.internal.NewRetrieve().Entry(&parent).WhereKeys(parentKey).Exec(ctx, nil); err != nil {
				return RetrieveResponse{}, err
			}
			rng.Parent = &Range{Range: parent}
			apiRanges[i] = rng
		}
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rangeAccessOntologyIDs(apiRanges),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Ranges: apiRanges}, nil
}

type RenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *Service) Rename(
	ctx context.Context,
	req RenameRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: ranger.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for _, key := range req.Keys {
			if err := w.Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

