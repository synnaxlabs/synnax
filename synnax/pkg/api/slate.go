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
	"github.com/synnaxlabs/synnax/pkg/service/slate"
	"github.com/synnaxlabs/x/gorp"
)

type SlateService struct {
	dbProvider
	accessProvider
	internal *slate.Service
}

func NewSlateService(p Provider) *SlateService {
	return &SlateService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Service.Slate,
	}
}

type (
	SlateCreateRequest struct {
		Slates []slate.Slate `json:"slates" msgpack:"slates"`
	}
	SlateCreateResponse = SlateCreateRequest
)

func (s *SlateService) Create(ctx context.Context, req SlateCreateRequest) (res SlateCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: slate.OntologyIDsFromSlates(req.Slates),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, slate_ := range req.Slates {
			if err = w.Create(ctx, &slate_); err != nil {
				return err
			}
			req.Slates[i] = slate_
		}
		res.Slates = req.Slates
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
		Objects: slate.OntologyIDs(req.Keys),
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
		Slates []slate.Slate `json:"slates" msgpack:"slates"`
	}
)

func (s *SlateService) Retrieve(ctx context.Context, req SlateRetrieveRequest) (res SlateRetrieveResponse, err error) {
	if err = s.internal.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Slates).Exec(ctx, nil); err != nil {
		return SlateRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: slate.OntologyIDsFromSlates(res.Slates),
	}); err != nil {
		return SlateRetrieveResponse{}, err
	}
	return res, err
}
