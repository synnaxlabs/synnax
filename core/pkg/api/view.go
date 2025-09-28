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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/gorp"
)

type ViewService struct {
	dbProvider
	accessProvider
	internal *view.Service
}

func NewViewService(p Provider) *ViewService {
	return &ViewService{
		internal:       p.Service.View,
		dbProvider:     p.db,
		accessProvider: p.access,
	}
}

type View = view.View

type ViewCreateRequest struct {
	Parent ontology.ID `json:"parent" msgpack:"parent"`
	Views  []View      `json:"views" msgpack:"views"`
}

type ViewCreateResponse struct {
	Views []View `json:"views" msgpack:"views"`
}

func (s *ViewService) Create(
	ctx context.Context,
	req ViewCreateRequest,
) (res ViewCreateResponse, err error) {
	objects := view.OntologyIDsFromViews(req.Views)
	if !req.Parent.IsZero() {
		objects = append(objects, req.Parent)
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: objects,
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.internal.NewWriter(tx).CreateManyWithParent(ctx, &req.Views, req.Parent)
		if err != nil {
			return err
		}
		res.Views = req.Views
		return nil
	})
}

type ViewRetrieveRequest struct {
	Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
	SearchTerm string      `json:"search_term" msgpack:"search_term"`
	Limit      int         `json:"limit" msgpack:"limit"`
	Offset     int         `json:"offset" msgpack:"offset"`
}

type ViewRetrieveResponse struct {
	Views []View `json:"views" msgpack:"views"`
}

func (s *ViewService) Retrieve(
	ctx context.Context,
	req ViewRetrieveRequest,
) (res ViewRetrieveResponse, err error) {
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

	if err = q.Entries(&res.Views).Exec(ctx, nil); err != nil {
		return ViewRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: view.OntologyIDsFromViews(res.Views),
	}); err != nil {
		return ViewRetrieveResponse{}, err
	}
	return res, nil
}

type ViewDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *ViewService) Delete(
	ctx context.Context,
	req ViewDeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: view.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys...)
	})
}