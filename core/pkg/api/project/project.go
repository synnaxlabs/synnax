// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *project.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Project,
	}, nil
}

type (
	CreateRequest struct {
		Projects []project.Project `json:"projects" msgpack:"projects"`
	}
	CreateResponse = CreateRequest
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: project.OntologyIDsFromProjects(req.Projects),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, p := range req.Projects {
			if err := w.Create(ctx, &p); err != nil {
				return err
			}
			req.Projects[i] = p
		}
		res.Projects = req.Projects
		return nil
	})
}

type RenameRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  uuid.UUID `json:"key" msgpack:"key"`
}

func (s *Service) Rename(ctx context.Context, req RenameRequest) (res types.Nil, err error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{project.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type (
	RetrieveRequest struct {
		SearchTerm string      `json:"search_term" msgpack:"search_term"`
		Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
		Limit      int         `json:"limit" msgpack:"limit"`
		Offset     int         `json:"offset" msgpack:"offset"`
	}
	RetrieveResponse struct {
		Projects []project.Project `json:"projects" msgpack:"projects"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (res RetrieveResponse, err error) {
	q := s.internal.NewRetrieve().Search(req.SearchTerm)
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	err = q.Entries(&res.Projects).Exec(ctx, nil)
	if eErr := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: project.OntologyIDsFromProjects(res.Projects),
	}); eErr != nil {
		return RetrieveResponse{}, eErr
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
		Objects: project.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
