// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/validate"
)

type Service struct {
	access   *rbac.Service
	internal *status.Service
	label    *label.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Service.Status,
		label:    cfg.Service.Label,
		access:   cfg.Service.RBAC,
	}, nil
}

func statusAccessOntologyIDs(statuses []status.Status[any]) []ontology.ID {
	ids := make([]ontology.ID, 0, len(statuses))
	for _, s := range statuses {
		ids = append(ids, status.OntologyID(s.Key))
		ids = append(ids, label.OntologyIDsFromLabels(s.Labels)...)
	}
	return ids
}

// SetRequest is a request to set (create or update) statuses in the cluster.
type SetRequest struct {
	// Parent is the parent ontology ID for the statuses.
	Parent ontology.ID `json:"parent" msgpack:"parent"`
	// Statuses are the statuses to set.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

// SetResponse is a response to a SetRequest.
type SetResponse struct {
	// Statuses are the statuses that were set.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

// Set creates or updates statuses in the cluster.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (SetResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: statusAccessOntologyIDs(req.Statuses),
	}); err != nil {
		return SetResponse{}, err
	}
	if err := s.internal.NewWriter(tx).SetManyWithParent(
		ctx,
		&req.Statuses,
		req.Parent,
	); err != nil {
		return SetResponse{}, err
	}
	return SetResponse{Statuses: req.Statuses}, nil
}

// SetByKeyOrNameRequest is a request to upsert a status by key or by name.
type SetByKeyOrNameRequest struct {
	// KeyOrName is either an existing status key or a status name to match.
	KeyOrName string `json:"key_or_name" msgpack:"key_or_name"`
	// Message is the new status message.
	Message string `json:"message" msgpack:"message"`
	// Variant is the new status variant.
	Variant xstatus.Variant `json:"variant" msgpack:"variant"`
}

// SetByKeyOrNameResponse is a response to a SetByKeyOrNameRequest.
type SetByKeyOrNameResponse struct {
	// Key is the key of the upserted status.
	Key string `json:"key" msgpack:"key"`
	// MultipleMatches reports whether multiple statuses matched by name.
	MultipleMatches bool `json:"multiple_matches" msgpack:"multiple_matches"`
}

// SetByKeyOrName upserts a status by key or by name.
func (s *Service) SetByKeyOrName(
	ctx context.Context,
	tx gorp.Tx,
	req SetByKeyOrNameRequest,
) (SetByKeyOrNameResponse, error) {
	if !req.Variant.IsValid() {
		return SetByKeyOrNameResponse{}, errors.Wrap(validate.ErrValidation, "invalid status variant")
	}
	matches, err := s.internal.ResolveKeyOrName(ctx, tx, req.KeyOrName)
	if err != nil {
		return SetByKeyOrNameResponse{}, err
	}
	st := status.SetTarget(matches, req.KeyOrName, req.Message, string(req.Variant))
	action := access.ActionUpdate
	if len(matches) == 0 {
		action = access.ActionCreate
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  action,
		Objects: []ontology.ID{status.OntologyID(st.Key)},
	}); err != nil {
		return SetByKeyOrNameResponse{}, err
	}
	if err := s.internal.NewWriter(tx).Set(ctx, &st); err != nil {
		return SetByKeyOrNameResponse{}, err
	}
	return SetByKeyOrNameResponse{
		Key:             st.Key,
		MultipleMatches: len(matches) > 1,
	}, nil
}

type RetrieveRequest struct {
	// SearchTerm is used for fuzzy searching statuses.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Keys are the keys of the statuses to retrieve.
	Keys []string `json:"keys" msgpack:"keys"`
	// HasLabels retrieves statuses that are labeled by one or more labels with the
	// given keys.
	HasLabels []label.Key `json:"has_labels" msgpack:"has_labels"`
	// Variants filters for statuses with the given variants.
	Variants []xstatus.Variant `json:"variants" msgpack:"variants"`
	// Limit is the maximum number of statuses to retrieve.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset is the number of statuses to skip.
	Offset int `json:"offset" msgpack:"offset"`
	// IncludeLabels sets whether to fetch labels for the retrieved statuses.
	IncludeLabels bool `json:"include_labels" msgpack:"include_labels"`
}

type RetrieveResponse struct {
	// Statuses are the statuses that were retrieved.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve()
	resStatuses := make([]status.Status[any], 0, len(req.Keys))

	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.HasLabels) > 0 {
		q = q.Where(status.MatchLabels[any](req.HasLabels...))
	}
	if len(req.Variants) > 0 {
		q = q.Where(status.MatchVariants[any](req.Variants...))
	}
	if len(req.Keys) != 0 {
		q = q.Where(status.MatchKeys[any](req.Keys...))
	}
	if err := q.Entries(&resStatuses).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	res := RetrieveResponse{Statuses: resStatuses}
	ids := statusAccessOntologyIDs(res.Statuses)
	if req.IncludeLabels {
		for i, stat := range res.Statuses {
			labels, err := s.label.RetrieveFor(ctx, status.OntologyID(stat.Key), nil)
			if err != nil {
				return RetrieveResponse{}, err
			}
			res.Statuses[i].Labels = labels
		}
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ids,
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	// Keys are the keys of the statuses to delete.
	Keys []string `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: status.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
