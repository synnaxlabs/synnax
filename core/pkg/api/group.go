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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
)

type GroupService struct {
	dbProvider
	accessProvider
	internal *group.Service
}

func NewGroupService(p Provider) *GroupService {
	return &GroupService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Distribution.Group,
	}
}

type (
	GroupCreateRequest struct {
		Name   string      `json:"name" msgpack:"name" validate:"required"`
		Key    uuid.UUID   `json:"key" msgpack:"key"`
		Parent ontology.ID `json:"parent" msgpack:"parent"`
	}
	GroupCreateResponse struct {
		Group group.Group `json:"group" msgpack:"group"`
	}
)

func (s *GroupService) Create(
	ctx context.Context,
	req GroupCreateRequest,
) (GroupCreateResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return GroupCreateResponse{}, err
	}
	var res GroupCreateResponse
	if err := s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		g, err := w.CreateWithKey(ctx, req.Key, req.Name, req.Parent)
		if err != nil {
			return err
		}
		res.Group = g
		return nil
	}); err != nil {
		return GroupCreateResponse{}, err
	}
	return res, nil
}

type GroupDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys" validate:"required"`
}

func (s *GroupService) Delete(
	ctx context.Context,
	req GroupDeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: group.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type GroupRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key" validate:"required"`
	Name string    `json:"name" msgpack:"name" validate:"required"`
}

func (s *GroupService) Rename(
	ctx context.Context,
	req GroupRenameRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}
