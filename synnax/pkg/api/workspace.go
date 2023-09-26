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
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/synnax/pkg/workspace"
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
	Workspaces []workspace.Workspace `json:"workspaces" msgpack:"workspaces"`
}

type WorkspaceCreateResponse struct {
	Workspaces []workspace.Workspace `json:"workspaces" msgpack:"workspaces"`
}

func (s *WorkspaceService) Create(ctx context.Context, req WorkspaceCreateRequest) (res WorkspaceCreateResponse, err errors.Typed) {
	userKey, err_ := user.FromOntologyID(getSubject(ctx))
	if err_ != nil {
		return res, errors.Unexpected(err_)
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		for _, w := range req.Workspaces {
			w.Author = userKey
			err := s.internal.NewWriter(tx).Create(ctx, &w)
			if err != nil {
				return errors.MaybeQuery(err)
			}
			res.Workspaces = append(res.Workspaces, w)
		}
		return errors.Nil
	})
}

type WorkspaceRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *WorkspaceService) Rename(ctx context.Context, req WorkspaceRenameRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
		return errors.MaybeQuery(err)
	})
}

type WorkspaceSetLayoutRequest struct {
	Key    uuid.UUID `json:"key" msgpack:"key"`
	Layout string    `json:"layout" msgpack:"layout"`
}

func (s *WorkspaceService) SetLayout(ctx context.Context, req WorkspaceSetLayoutRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).SetLayout(ctx, req.Key, req.Layout)
		return errors.MaybeQuery(err)
	})
}

type WorkspaceRetrieveRequest struct {
	Keys   []uuid.UUID `json:"keys" msgpack:"keys"`
	Search string      `json:"search" msgpack:"search"`
	Author uuid.UUID   `json:"author" msgpack:"author"`
}

type WorkspaceRetrieveResponse struct {
	Workspaces []workspace.Workspace `json:"workspaces" msgpack:"workspaces"`
}

func (s *WorkspaceService) Retrieve(
	ctx context.Context,
	req WorkspaceRetrieveRequest,
) (res WorkspaceRetrieveResponse, err errors.Typed) {
	q := s.internal.NewRetrieve().Search(req.Search)
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if req.Author != uuid.Nil {
		q = q.WhereAuthor(req.Author)
	}
	err = errors.MaybeQuery(q.Entries(&res.Workspaces).Exec(ctx, nil))
	return res, err
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
