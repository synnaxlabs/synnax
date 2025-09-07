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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/gorp"
)

type SlateService struct {
	dbProvider
	accessProvider
	internal *arc.Service
}

func NewSlateService(p Provider) *SlateService {
	return &SlateService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Service.arc,
	}
}

type (
	SlateCreateRequest struct {
		Arcs []arc.arc `json:"arcs" msgpack:"arcs"`
	}
	SlateCreateResponse = SlateCreateRequest
)

func (s *SlateService) Create(ctx context.Context, req SlateCreateRequest) (res SlateCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: arc.OntologyIDsFromSlates(req.Arcs),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, slate_ := range req.Arcs {
			if err = w.Create(ctx, &slate_); err != nil {
				return err
			}
			req.Arcs[i] = slate_
		}
		res.Arcs = req.Arcs
		return nil
	})
}

type SlateDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *SlateService) Delete(ctx context.Context, req SlateDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: arc.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type (
	SlateRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	SlateRetrieveResponse struct {
		Arcs []arc.arc `json:"arcs" msgpack:"arcs"`
	}
)

func (s *SlateService) Retrieve(ctx context.Context, req SlateRetrieveRequest) (res SlateRetrieveResponse, err error) {
	if err = s.internal.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Arcs).Exec(ctx, nil); err != nil {
		return SlateRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: arc.OntologyIDsFromSlates(res.Arcs),
	}); err != nil {
		return SlateRetrieveResponse{}, err
	}
	return res, err
}
