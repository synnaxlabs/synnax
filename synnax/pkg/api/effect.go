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
		Effects []effect.Effect `json:"effects" msgpack:"effects"`
	}
)

func (s *EffectService) DeleteEffect(ctx context.Context, req EffectDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: effect.OntologyIDsFromEffects(req.Effects),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for _, e := range req.Effects {
			if err := w.Delete(ctx, e.Key); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	EffectRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	EffectRetrieveResponse struct {
		Effects []effect.Effect `json:"effects" msgpack:"effects"`
	}
)

func (s *EffectService) RetrieveEffect(ctx context.Context, req EffectRetrieveRequest) (res EffectRetrieveResponse, err error) {
	err = s.internal.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Effects).Exec(ctx, nil)
	if err != nil {
		return res, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: effect.OntologyIDsFromEffects(res.Effects),
	}); err != nil {
		res.Effects = nil
	}
	return res, nil
}
