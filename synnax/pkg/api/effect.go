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
	"github.com/synnaxlabs/synnax/pkg/service/effect/action"
	"github.com/synnaxlabs/synnax/pkg/service/effect/condition"
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
		internal:       p.Config.Effect,
	}
}

type (
	ConditionCreateRequest struct {
		Conditions []condition.Condition `json:"conditions" msgpack:"conditions"`
	}
	ConditionCreateResponse struct {
		Conditions []condition.Condition `json:"conditions" msgpack:"conditions"`
	}
)

func (s *EffectService) CreateCondition(ctx context.Context, req ConditionCreateRequest) (res ConditionCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: condition.OntologyIDsFromConditions(req.Conditions),
	}); err != nil {
		return res, err
	}

	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.Conditions.NewWriter(tx)
		for _, c := range req.Conditions {
			if err := w.Create(ctx, &c); err != nil {
				return err
			}
		}
		res.Conditions = req.Conditions
		return nil
	})
}

type (
	ConditionDeleteRequest struct {
		Conditions []condition.Condition `json:"conditions" msgpack:"conditions"`
	}
)

func (s *EffectService) DeleteCondition(ctx context.Context, req ConditionDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: condition.OntologyIDsFromConditions(req.Conditions),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.Conditions.NewWriter(tx)
		for _, c := range req.Conditions {
			if err := w.Delete(ctx, c.Key); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	ConditionRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	ConditionRetrieveResponse struct {
		Conditions []condition.Condition `json:"conditions" msgpack:"conditions"`
	}
)

func (s *EffectService) RetrieveCondition(ctx context.Context, req ConditionRetrieveRequest) (res ConditionRetrieveResponse, err error) {
	err = s.internal.Conditions.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Conditions).Exec(ctx, nil)
	if err != nil {
		return res, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: condition.OntologyIDsFromConditions(res.Conditions),
	}); err != nil {
		res.Conditions = nil
	}
	return res, err
}

type (
	ActionCreateRequest struct {
		Actions []action.Action `json:"actions" msgpack:"actions"`
	}
	ActionCreateResponse struct {
		Actions []action.Action `json:"actions" msgpack:"actions"`
	}
)

func (s *EffectService) CreateAction(ctx context.Context, req ActionCreateRequest) (res ActionCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: action.OntologyIDsFromActions(req.Actions),
	}); err != nil {
		return res, err
	}

	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.Actions.NewWriter(tx)
		for _, a := range req.Actions {
			if err := w.Create(ctx, &a); err != nil {
				return err
			}
		}
		res.Actions = req.Actions
		return nil
	})
}

type (
	ActionDeleteRequest struct {
		Actions []action.Action `json:"actions" msgpack:"actions"`
	}
)

func (s *EffectService) DeleteAction(ctx context.Context, req ActionDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: action.OntologyIDsFromActions(req.Actions),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.Actions.NewWriter(tx)
		for _, a := range req.Actions {
			if err := w.Delete(ctx, a.Key); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	ActionRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	ActionRetrieveResponse struct {
		Actions []action.Action `json:"actions" msgpack:"actions"`
	}
)

func (s *EffectService) RetrieveAction(ctx context.Context, req ActionRetrieveRequest) (res ActionRetrieveResponse, err error) {
	err = s.internal.Actions.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Actions).Exec(ctx, nil)
	if err != nil {
		return res, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: action.OntologyIDsFromActions(res.Actions),
	}); err != nil {
		res.Actions = nil
	}
	return res, nil
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
