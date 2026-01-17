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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/gorp"
)

type ViewService struct {
	dbProvider
	accessProvider
	internal *view.Service
}

// NewViewService creates a new service for managing views.
func NewViewService(p Provider) *ViewService {
	return &ViewService{
		internal:       p.Service.View,
		dbProvider:     p.db,
		accessProvider: p.access,
	}
}

type View = view.View

type ViewCreateRequest struct {
	Views []View `json:"views" msgpack:"views"`
}

type ViewCreateResponse struct {
	Views []View `json:"views" msgpack:"views"`
}

func (s *ViewService) Create(
	ctx context.Context,
	req ViewCreateRequest,
) (ViewCreateResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: view.OntologyIDsFromViews(req.Views),
	}); err != nil {
		return ViewCreateResponse{}, err
	}
	var res ViewCreateResponse
	if err := s.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Views); err != nil {
			return err
		}
		res.Views = req.Views
		return nil
	}); err != nil {
		return ViewCreateResponse{}, err
	}
	return res, nil
}

type ViewRetrieveRequest struct {
	SearchTerm string      `json:"search_term" msgpack:"search_term"`
	Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
	Types      []string    `json:"types" msgpack:"types"`
	Limit      int         `json:"limit" msgpack:"limit"`
	Offset     int         `json:"offset" msgpack:"offset"`
}

type ViewRetrieveResponse struct {
	Views []View `json:"views" msgpack:"views"`
}

func (s *ViewService) Retrieve(
	ctx context.Context,
	req ViewRetrieveRequest,
) (ViewRetrieveResponse, error) {
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
	if len(req.Types) != 0 {
		q = q.WhereTypes(req.Types...)
	}

	var views []view.View
	if err := q.Entries(&views).Exec(ctx, nil); err != nil {
		return ViewRetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: view.OntologyIDsFromViews(views),
	}); err != nil {
		return ViewRetrieveResponse{}, err
	}
	return ViewRetrieveResponse{Views: views}, nil
}

type ViewDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *ViewService) Delete(
	ctx context.Context,
	req ViewDeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: view.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys...)
	})
}
