// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *table.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Table,
	}
}

type (
	CreateRequest struct {
		Workspace uuid.UUID     `json:"workspace" msgpack:"workspace"`
		Tables    []table.Table `json:"tables" msgpack:"tables"`
	}
	CreateResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: table.OntologyIDsFromTables(req.Tables),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for i, t := range req.Tables {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Workspace, &t); err != nil {
				return err
			}
			req.Tables[i] = t
		}
		res.Tables = req.Tables
		return nil
	})
}

type RenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *Service) Rename(ctx context.Context, req RenameRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{table.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type SetDataRequest struct {
	Key  uuid.UUID      `json:"key" msgpack:"key"`
	Data map[string]any `json:"data" msgpack:"data"`
}

func (s *Service) SetData(ctx context.Context, req SetDataRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{table.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
	})
}

type (
	RetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	RetrieveResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, err error) {
	err = s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.Tables).Exec(ctx, nil)
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: table.OntologyIDs(req.Keys),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, err
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: table.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
