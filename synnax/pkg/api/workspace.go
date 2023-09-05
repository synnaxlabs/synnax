// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/workspace"
	"github.com/synnaxlabs/synnax/pkg/workspace/pid"
	"github.com/synnaxlabs/x/gorp"
	"go/types"
)

type WorkspaceService struct {
	dbProvider
	internal *workspace.Service
}

func NewWorkspaceService(p Provider) *WorkspaceService {
	return &WorkspaceService{
		dbProvider: p.db,
		internal:   p.Config.Workspace,
	}
}

type WorkspaceCreateRequest struct {
	Workspaces []workspace.Workspace `json:"workspace" msgpack:"workspace"`
}

type WorkspaceCreateResponse struct {
	Workspaces []workspace.Workspace `json:"workspace" msgpack:"workspace"`
}

func (s *WorkspaceService) Create(ctx context.Context, req WorkspaceCreateRequest) (res WorkspaceCreateResponse, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		for _, w := range req.Workspaces {
			err := s.internal.NewWriter(tx).Create(ctx, &w)
			if err != nil {
				return errors.MaybeQuery(err)
			}
			res.Workspaces = append(res.Workspaces, w)
		}
		res = WorkspaceCreateResponse{Workspaces: req.Workspaces}
		return errors.MaybeQuery(err)
	})
}

type WorkspaceRetrieveRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

type WorkspaceRetrieveResponse struct {
	Workspaces []workspace.Workspace `json:"workspaces" msgpack:"workspaces"`
}

func (s *WorkspaceService) Retrieve(ctx context.Context, req WorkspaceRetrieveRequest) (res WorkspaceRetrieveResponse, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Workspaces).Exec(ctx, tx)
		return errors.MaybeQuery(err)
	})
}

type WorkspaceDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *WorkspaceService) Delete(ctx context.Context, req WorkspaceDeleteRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
		return errors.MaybeQuery(err)
	})
}

type WorkspacePIDCreateRequest struct {
	PIDs []pid.PID `json:"pid" msgpack:"pid"`
}

type WorkspacePIDCreateResponse struct {
	PIDs []pid.PID `json:"pid" msgpack:"pid"`
}

func (s *WorkspaceService) CreatePID(ctx context.Context, req WorkspacePIDCreateRequest) (res WorkspacePIDCreateResponse, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		for _, pid_ := range req.PIDs {
			err := s.internal.PID.NewWriter(tx).Create(ctx, &pid_)
			if err != nil {
				return errors.MaybeQuery(err)
			}
			res.PIDs = append(res.PIDs, pid_)
		}
		return errors.Nil
	})
}

type WorkspacePIDRetrieveRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

type WorkspacePIDRetrieveResponse struct {
	PIDs []pid.PID `json:"pids" msgpack:"pids"`
}

func (s *WorkspaceService) RetrievePID(ctx context.Context, req WorkspacePIDRetrieveRequest) (res WorkspacePIDRetrieveResponse, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.PID.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.PIDs).Exec(ctx, tx)
		return errors.MaybeQuery(err)
	})
}

type WorkspacePIDDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *WorkspaceService) DeletePID(ctx context.Context, req WorkspacePIDDeleteRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.PID.NewWriter(tx).Delete(ctx, req.Keys...)
		return errors.MaybeQuery(err)
	})
}
