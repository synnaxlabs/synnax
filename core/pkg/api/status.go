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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
)

type StatusService struct {
	dbProvider
	accessProvider
	internal *status.Service
	label    *label.Service
}

func NewStatusService(p Provider) *StatusService {
	return &StatusService{
		internal:       p.Service.Status,
		label:          p.Service.Label,
		dbProvider:     p.db,
		accessProvider: p.access,
	}
}

type Status struct {
	status.Status[any]
	Labels []label.Label
}

func translateStatusesToService(statuses []Status) []status.Status[any] {
	return lo.Map(statuses, func(s Status, _ int) status.Status[any] {
		return s.Status
	})
}

func translateStatusesFromService(statuses []status.Status[any]) []Status {
	return lo.Map(statuses, func(s status.Status[any], _ int) Status {
		return Status{Status: s}
	})

}

// StatusSetRequest is a request to set (create or update) statuses in the cluster.
type StatusSetRequest struct {
	// Parent is the parent ontology ID for the statuses.
	Parent ontology.ID `json:"parent" msgpack:"parent"`
	// Statuses are the statuses to set.
	Statuses []Status `json:"statuses" msgpack:"statuses"`
}

// StatusSetResponse is a response to a StatusSetRequest.
type StatusSetResponse struct {
	// Statuses are the statuses that were set.
	Statuses []Status `json:"statuses" msgpack:"statuses"`
}

func statusAccessOntologyIDs(statuses []Status) []ontology.ID {
	ids := make([]ontology.ID, 0, len(statuses))
	for _, s := range statuses {
		ids = append(ids, s.OntologyID())
		ids = append(ids, label.OntologyIDsFromLabels(s.Labels)...)
	}
	return ids
}

// Set creates or updates statuses in the cluster.
func (s *StatusService) Set(
	ctx context.Context,
	req StatusSetRequest,
) (res StatusSetResponse, err error) {
	ids := statusAccessOntologyIDs(req.Statuses)
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: ids,
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		translated := translateStatusesToService(req.Statuses)
		if err = s.internal.NewWriter(tx).SetManyWithParent(
			ctx,
			&translated,
			req.Parent,
		); err != nil {
			return err
		}
		res.Statuses = translateStatusesFromService(translated)
		return nil
	})
}

type StatusRetrieveRequest struct {
	// SearchTerm is used for fuzzy searching statuses.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Keys are the keys of the statuses to retrieve.
	Keys []string `json:"keys" msgpack:"keys"`
	// HasLabels retrieves statuses that are labeled by one or more labels with the
	// given keys.
	HasLabels []uuid.UUID `json:"has_labels" msgpack:"has_labels"`
	// Limit is the maximum number of statuses to retrieve.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset is the number of statuses to skip.
	Offset int `json:"offset" msgpack:"offset"`
	// IncludeLabels sets whether to fetch labels for the retrieved statuses.
	IncludeLabels bool `json:"include_labels" msgpack:"include_labels"`
}

type StatusRetrieveResponse struct {
	// Statuses are the statuses that were retrieved.
	Statuses []Status `json:"statuses" msgpack:"statuses"`
}

func (s *StatusService) Retrieve(
	ctx context.Context,
	req StatusRetrieveRequest,
) (res StatusRetrieveResponse, err error) {
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
		q = q.WhereHasLabels(req.HasLabels...)
	}
	if len(req.Keys) != 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if err = q.Entries(&resStatuses).Exec(ctx, nil); err != nil {
		return StatusRetrieveResponse{}, err
	}
	res.Statuses = translateStatusesFromService(resStatuses)
	ids := statusAccessOntologyIDs(res.Statuses)
	if req.IncludeLabels {
		for i, stat := range res.Statuses {
			labels, err := s.label.RetrieveFor(ctx, stat.OntologyID(), nil)
			if err != nil {
				return StatusRetrieveResponse{}, err
			}
			res.Statuses[i].Labels = labels
		}
	}

	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ids,
	}); err != nil {
		return StatusRetrieveResponse{}, err
	}
	return res, nil
}

type StatusDeleteRequest struct {
	// Keys are the keys of the statuses to delete.
	Keys []string `json:"keys" msgpack:"keys"`
}

func (s *StatusService) Delete(
	ctx context.Context,
	req StatusDeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: status.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys...)
	})
}
