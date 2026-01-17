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
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
)

type WorkspaceService struct {
	dbProvider
	accessProvider
	internal *workspace.Service
}

func NewWorkspaceService(p Provider) *WorkspaceService {
	return &WorkspaceService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Service.Workspace,
	}
}

type (
	WorkspaceCreateRequest struct {
		Workspaces []workspace.Workspace `json:"workspaces" msgpack:"workspaces"`
	}
	WorkspaceCreateResponse = WorkspaceCreateRequest
)

func (s *WorkspaceService) Create(ctx context.Context, req WorkspaceCreateRequest) (res WorkspaceCreateResponse, err error) {

	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: workspace.OntologyIDsFromWorkspaces(req.Workspaces),
	}); err != nil {
		return res, err
	}
	userKey, err := user.KeyFromOntologyID(getSubject(ctx))
	if err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, ws := range req.Workspaces {
			ws.Author = userKey
			if err := w.Create(ctx, &ws); err != nil {
				return err
			}
			req.Workspaces[i] = ws
		}
		res.Workspaces = req.Workspaces
		return nil
	})
}

type WorkspaceRenameRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  uuid.UUID `json:"key" msgpack:"key"`
}

func (s *WorkspaceService) Rename(ctx context.Context, req WorkspaceRenameRequest) (res types.Nil, err error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{workspace.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type WorkspaceSetLayoutRequest struct {
	Layout string    `json:"layout" msgpack:"layout"`
	Key    uuid.UUID `json:"key" msgpack:"key"`
}

func (s *WorkspaceService) SetLayout(ctx context.Context, req WorkspaceSetLayoutRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{workspace.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).SetLayout(ctx, req.Key, req.Layout)
	})
}

type (
	WorkspaceRetrieveRequest struct {
		SearchTerm string      `json:"search_term" msgpack:"search_term"`
		Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
		Limit      int         `json:"limit" msgpack:"limit"`
		Offset     int         `json:"offset" msgpack:"offset"`
		Author     uuid.UUID   `json:"author" msgpack:"author"`
	}
	WorkspaceRetrieveResponse struct {
		Workspaces []workspace.Workspace `json:"workspaces" msgpack:"workspaces"`
	}
)

func (s *WorkspaceService) Retrieve(
	ctx context.Context,
	req WorkspaceRetrieveRequest,
) (res WorkspaceRetrieveResponse, err error) {
	q := s.internal.NewRetrieve().Search(req.SearchTerm)
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if req.Author != uuid.Nil {
		q = q.WhereAuthor(req.Author)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	err = q.Entries(&res.Workspaces).Exec(ctx, nil)
	if eErr := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: workspace.OntologyIDsFromWorkspaces(res.Workspaces),
	}); eErr != nil {
		return WorkspaceRetrieveResponse{}, eErr
	}
	return res, err
}

type WorkspaceDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *WorkspaceService) Delete(ctx context.Context, req WorkspaceDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: workspace.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
