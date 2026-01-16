// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
	"github.com/synnaxlabs/x/gorp"
)

type TableService struct {
	dbProvider
	accessProvider
	internal *table.Service
}

func NewTableService(p Provider) *TableService {
	return &TableService{
		dbProvider:     p.db,
		internal:       p.Service.Table,
		accessProvider: p.access,
	}
}

type (
	TableCreateRequest struct {
		Workspace uuid.UUID     `json:"workspace" msgpack:"workspace"`
		Tables    []table.Table `json:"tables" msgpack:"tables"`
	}
	TableCreateResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

func (s *TableService) Create(ctx context.Context, req TableCreateRequest) (res TableCreateResponse, err error) {
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: table.OntologyIDsFromTables(req.Tables),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		for i, table_ := range req.Tables {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Workspace, &table_); err != nil {
				return err
			}
			req.Tables[i] = table_
		}
		res.Tables = req.Tables
		return nil
	})
}

type TableRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *TableService) Rename(ctx context.Context, req TableRenameRequest) (res types.Nil, err error) {
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{table.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type TableSetDataRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Data string    `json:"data" msgpack:"data"`
}

func (s *TableService) SetData(ctx context.Context, req TableSetDataRequest) (res types.Nil, err error) {
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{table.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
	})
}

type (
	TableRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	TableRetrieveResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

func (s *TableService) Retrieve(ctx context.Context, req TableRetrieveRequest) (res TableRetrieveResponse, err error) {
	err = s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.Tables).Exec(ctx, nil)
	if err != nil {
		return TableRetrieveResponse{}, err
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: table.OntologyIDs(req.Keys),
	}); err != nil {
		return TableRetrieveResponse{}, err
	}
	return res, err
}

type TableDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *TableService) Delete(ctx context.Context, req TableDeleteRequest) (res types.Nil, err error) {
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: table.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
