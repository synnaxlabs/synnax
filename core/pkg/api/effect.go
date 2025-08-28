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
	"github.com/synnaxlabs/synnax/pkg/service/effect"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/gorp"
)

type EffectService struct {
	dbProvider
	accessProvider
	internal *effect.Service
}

func NewEffectService(p Provider) *EffectService {
	return &EffectService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Service.Effect,
	}
}

type (
	EffectCreateRequest struct {
		Effects []effect.Effect `json:"effects" msgpack:"effects"`
	}
	EffectCreateResponse struct {
		Effects []effect.Effect `json:"effects" msgpack:"effects"`
	}
)

func (s *EffectService) CreateEffect(ctx context.Context, req EffectCreateRequest) (res EffectCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: effect.OntologyIDsFromEffects(req.Effects),
	}); err != nil {
		return res, err
	}

	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for _, e := range req.Effects {
			if err := w.Create(ctx, &e); err != nil {
				return err
			}
		}
		res.Effects = req.Effects
		return nil
	})
}

type (
	EffectDeleteRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
)

func (s *EffectService) DeleteEffect(ctx context.Context, req EffectDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: effect.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for _, k := range req.Keys {
			if err = w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	EffectRetrieveRequest struct {
		Keys          []uuid.UUID `json:"keys" msgpack:"keys"`
		Term          string      `json:"term" msgpack:"term"`
		Limit         int         `json:"limit" msgpack:"limit"`
		Offset        int         `json:"offset" msgpack:"offset"`
		IncludeStatus bool        `json:"include_status" msgpack:"include_status"`
		IncludeLabels bool        `json:"include_labels" msgpack:"include_labels"`
	}
	EffectRetrieveResponse struct {
		Effects []effect.Effect `json:"effects" msgpack:"effects"`
	}
)

func (s *EffectService) RetrieveEffect(ctx context.Context, req EffectRetrieveRequest) (res EffectRetrieveResponse, err error) {
	var (
		resEffects []effect.Effect
		q          = s.internal.NewRetrieve().Entries(&resEffects)
		hasKeys    = len(req.Keys) > 0
		hasSearch  = req.Term != ""
	)

	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasSearch {
		q = q.Search(req.Term)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}

	if err = q.Exec(ctx, nil); err != nil {
		return res, err
	}

	if req.IncludeLabels {
		for i, eff := range resEffects {
			if eff.Labels, err = eff.RetrieveLabels(ctx); err != nil {
				return res, err
			}
			resEffects[i] = eff
		}
	}

	if req.IncludeStatus {
		for i, eff := range resEffects {
			if status, exists := s.internal.GetStatus(ctx, eff.Key); exists {
				eff.Status = &status
			}
			resEffects[i] = eff
		}
	}

	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: effect.OntologyIDsFromEffects(resEffects),
	}); err != nil {
		return res, err
	}

	return EffectRetrieveResponse{Effects: resEffects}, nil
}

type (
	EffectValidateRequest struct {
		Graph spec.Graph `json:"graph" msgpack:"graph"`
	}
	EffectValidateResponse struct {
		Error error `json:"error" msgpack:"error"`
	}
)

func (s *EffectService) Validate(ctx context.Context, req EffectValidateRequest) (res EffectValidateResponse, err error) {
	res.Error = s.internal.Validate(ctx, req.Graph)
	return res, nil
}
