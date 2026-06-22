// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

type (
	Key   = ranger.Key
	Range = ranger.Range
)

func rangeAccessOntologyIDs(ranges []Range) []ontology.ID {
	ids := make([]ontology.ID, 0, len(ranges))
	for _, r := range ranges {
		ids = append(ids, r.OntologyID())
		ids = append(ids, label.OntologyIDsFromLabels(r.Labels)...)
		if r.Parent == nil {
			continue
		}
		ids = append(ids, r.Parent.OntologyID())
	}
	return ids
}

type Service struct {
	access   *rbac.Service
	internal *ranger.Service
	label    *label.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Ranger,
		label:    cfg.Service.Label,
	}, nil
}

type (
	CreateRequest struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
	CreateResponse struct {
		Ranges []Range `json:"ranges" msgpack:"ranges"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: rangeAccessOntologyIDs(req.Ranges),
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Ranges); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

type (
	RetrieveRequest struct {
		Keys          []ranger.Key    `json:"keys" msgpack:"keys"`
		Names         []string        `json:"names" msgpack:"names"`
		SearchTerm    string          `json:"search_term" msgpack:"search_term"`
		HasLabels     []label.Key     `json:"has_labels" msgpack:"has_labels"`
		OverlapsWith  telem.TimeRange `json:"overlaps_with" msgpack:"overlaps_with"`
		Limit         int             `json:"limit" msgpack:"limit"`
		Offset        int             `json:"offset" msgpack:"offset"`
		IncludeLabels bool            `json:"include_labels" msgpack:"include_labels"`
		IncludeParent bool            `json:"include_parent" msgpack:"include_parent"`
	}
	RetrieveResponse struct {
		Ranges []Range `json:"ranges,omitzero" msgpack:"ranges,omitzero"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		ranges          []Range
		q               = s.internal.NewRetrieve().Entries(&ranges)
		hasNames        = len(req.Names) > 0
		hasKeys         = len(req.Keys) > 0
		hasSearch       = req.SearchTerm != ""
		hasOverlapsWith = !req.OverlapsWith.IsZero()
		hasLabels       = len(req.HasLabels) > 0
	)
	if hasOverlapsWith {
		q = q.Where(ranger.MatchOverlap(req.OverlapsWith))
	}
	if hasNames {
		q = q.Where(ranger.MatchNames(req.Names...))
	}
	if hasKeys {
		q = q.Where(ranger.MatchKeys(req.Keys...))
	}
	if hasLabels {
		q = q.Where(ranger.MatchLabels(req.HasLabels...))
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
	var err error
	if req.IncludeLabels {
		for i, r := range ranges {
			if ranges[i].Labels, err = s.label.
				RetrieveFor(ctx, r.OntologyID(), nil); err != nil {
				return RetrieveResponse{}, err
			}
		}
	}
	if req.IncludeParent {
		for i, r := range ranges {
			parentKey, err := s.internal.RetrieveParentKey(ctx, r.Key, nil)
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			if err != nil {
				return RetrieveResponse{}, err
			}
			ranges[i].Parent = &ranger.Range{}
			if err = s.
				internal.
				NewRetrieve().
				Entry(ranges[i].Parent).
				Where(ranger.MatchKeys(parentKey)).
				Exec(ctx, nil); err != nil {
				return RetrieveResponse{}, err
			}
		}
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rangeAccessOntologyIDs(ranges),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Ranges: ranges}, nil
}

type RenameRequest struct {
	Name string     `json:"name" msgpack:"name"`
	Key  ranger.Key `json:"key" msgpack:"key"`
}

func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	Keys []ranger.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: ranger.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
