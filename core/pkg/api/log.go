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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/x/gorp"
)

type LogService struct {
	dbProvider
	accessProvider
	internal *log.Service
}

func NewLogService(p Provider) *LogService {
	return &LogService{
		dbProvider:     p.db,
		internal:       p.Service.Log,
		accessProvider: p.access,
	}
}

type (
	LogCreateRequest struct {
		Workspace uuid.UUID `json:"workspace" msgpack:"workspace"`
		Logs      []log.Log `json:"logs" msgpack:"logs"`
	}
	LogCreateResponse struct {
		Logs []log.Log `json:"logs" msgpack:"logs"`
	}
)

func (s *LogService) Create(ctx context.Context, req LogCreateRequest) (res LogCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: log.OntologyIDsFromLogs(req.Logs),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		for i, log_ := range req.Logs {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Workspace, &log_); err != nil {
				return err
			}
			req.Logs[i] = log_
		}
		res.Logs = req.Logs
		return nil
	})
}

type LogRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *LogService) Rename(ctx context.Context, req LogRenameRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{log.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type LogSetDataRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Data string    `json:"data" msgpack:"data"`
}

func (s *LogService) SetData(ctx context.Context, req LogSetDataRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{log.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
	})
}

type (
	LogRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	LogRetrieveResponse struct {
		Logs []log.Log `json:"logs" msgpack:"logs"`
	}
)

func (s *LogService) Retrieve(ctx context.Context, req LogRetrieveRequest) (res LogRetrieveResponse, err error) {
	err = s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.Logs).Exec(ctx, nil)
	if err != nil {
		return LogRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: log.OntologyIDs(req.Keys),
	}); err != nil {
		return LogRetrieveResponse{}, err
	}
	return res, err
}

type LogDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *LogService) Delete(ctx context.Context, req LogDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: log.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
