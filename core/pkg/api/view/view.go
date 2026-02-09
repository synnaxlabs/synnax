// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *view.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		internal: cfg.Service.View,
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
	}
}

type View = view.View

type CreateRequest struct {
	Views []View `json:"views" msgpack:"views"`
}

type CreateResponse struct {
	Views []View `json:"views" msgpack:"views"`
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: view.OntologyIDsFromViews(req.Views),
	}); err != nil {
		return CreateResponse{}, err
	}
	var res CreateResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Views); err != nil {
			return err
		}
		res.Views = req.Views
		return nil
	}); err != nil {
		return CreateResponse{}, err
	}
	return res, nil
}

type RetrieveRequest struct {
	SearchTerm string      `json:"search_term" msgpack:"search_term"`
	Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
	Types      []string    `json:"types" msgpack:"types"`
	Limit      int         `json:"limit" msgpack:"limit"`
	Offset     int         `json:"offset" msgpack:"offset"`
}

type RetrieveResponse struct {
	Views []View `json:"views" msgpack:"views"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
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
		return RetrieveResponse{}, err
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: view.OntologyIDsFromViews(views),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Views: views}, nil
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
		Objects: view.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys...)
	})
}
