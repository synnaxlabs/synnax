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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *status.Service
	label    *label.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Status,
		label:    cfg.Service.Label,
	}
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

func statusAccessOntologyIDs(statuses []status.Status[any]) []ontology.ID {
	ids := make([]ontology.ID, 0, len(statuses))
	for _, s := range statuses {
		ids = append(ids, status.OntologyID(s.Key))
		ids = append(ids, label.OntologyIDsFromLabels(s.Labels)...)
	}
	return ids
}

// Set creates or updates statuses in the cluster.
func (s *Service) Set(
	ctx context.Context,
	req SetRequest,
) (res SetResponse, err error) {
	ids := statusAccessOntologyIDs(req.Statuses)
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: ids,
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err = s.internal.NewWriter(tx).SetManyWithParent(
			ctx,
			&req.Statuses,
			req.Parent,
		); err != nil {
			return err
		}
		res.Statuses = req.Statuses
		return nil
	})
}

type RetrieveRequest struct {
	// Keys are the keys of the statuses to retrieve.
	Keys []string `json:"keys" msgpack:"keys"`
	// SearchTerm is used for fuzzy searching statuses.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Limit is the maximum number of statuses to retrieve.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset is the number of statuses to skip.
	Offset int `json:"offset" msgpack:"offset"`
	// IncludeLabels sets whether to fetch labels for the retrieved statuses.
	IncludeLabels bool `json:"include_labels" msgpack:"include_labels"`
	// HasLabels retrieves statuses that are labeled by one or more labels with the
	// given keys.
	HasLabels []uuid.UUID `json:"has_labels" msgpack:"has_labels"`
}

type RetrieveResponse struct {
	// Statuses are the statuses that were retrieved.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (res RetrieveResponse, err error) {
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
		return RetrieveResponse{}, err
	}
	res.Statuses = resStatuses
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
	if err = s.access.Enforce(ctx, access.Request{
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
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: status.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys...)
	})
}
