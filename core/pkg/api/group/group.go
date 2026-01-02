// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *group.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
		internal: cfg.Distribution.Group,
	}
}

type (
	CreateRequest struct {
		Name   string      `json:"name" msgpack:"name" validate:"required"`
		Key    uuid.UUID   `json:"key" msgpack:"key"`
		Parent ontology.ID `json:"parent" msgpack:"parent"`
	}
	CreateResponse struct {
		Group group.Group `json:"group" msgpack:"group"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return CreateResponse{}, err
	}
	var res CreateResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		g, err := w.CreateWithKey(ctx, req.Key, req.Name, req.Parent)
		if err != nil {
			return err
		}
		res.Group = g
		return nil
	}); err != nil {
		return CreateResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys" validate:"required"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: group.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type RenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key" validate:"required"`
	Name string    `json:"name" msgpack:"name" validate:"required"`
}

func (s *Service) Rename(
	ctx context.Context,
	req RenameRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}
