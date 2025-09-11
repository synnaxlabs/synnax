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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
)

type StatusService struct {
	dbProvider
	accessProvider
	internal *status.Service
}

func NewStatusService(p Provider) *StatusService {
	return &StatusService{
		internal:       p.Service.Status,
		dbProvider:     p.db,
		accessProvider: p.access,
	}
}

type Status = status.Status

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

// Set creates or updates statuses in the cluster.
func (s *StatusService) Set(
	ctx context.Context,
	req StatusSetRequest,
) (res StatusSetResponse, err error) {
	// For status setting, we use Create action for new statuses
	// and Update action for existing ones. Since Set can do both,
	// we'll use Create permission.
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: status.OntologyIDsFromStatuses(req.Statuses),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.internal.NewWriter(tx).SetManyWithParent(ctx, &req.Statuses, req.Parent)
		if err != nil {
			return err
		}
		res.Statuses = req.Statuses
		return nil
	})
}

type StatusRetrieveRequest struct {
	// Keys are the keys of the statuses to retrieve.
	Keys []string `json:"keys" msgpack:"keys"`
	// SearchTerm is used for fuzzy searching statuses.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Limit is the maximum number of statuses to retrieve.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset is the number of statuses to skip.
	Offset int `json:"offset" msgpack:"offset"`
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

	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit != 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset != 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.Keys) != 0 {
		q = q.WhereKeys(req.Keys...)
	}

	if err = q.Entries(&res.Statuses).Exec(ctx, nil); err != nil {
		return StatusRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: status.OntologyIDsFromStatuses(res.Statuses),
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
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: status.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys...)
	})
}
