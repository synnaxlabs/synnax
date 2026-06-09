// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *panel.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Panel,
	}, nil
}

type (
	CreateRequest struct {
		// Project, when non-zero, parents each created panel to that project in
		// the ontology (a project panel). When zero, the panels are parented to
		// the creating user instead — a draft, visible only to its creator until
		// promoted to a project. Either way the panel is also parented to the
		// root Panels group.
		Project project.Key   `json:"project" msgpack:"project"`
		Panels  []panel.Panel `json:"panels" msgpack:"panels"`
	}
	CreateResponse = CreateRequest
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: panel.OntologyIDsFromPanels(req.Panels),
	}); err != nil {
		return res, err
	}
	// No project -> draft, parented to the creating user. With a project ->
	// project panel.
	parent := auth.GetSubject(ctx)
	if req.Project != uuid.Nil {
		parent = project.OntologyID(req.Project)
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, p := range req.Panels {
			if err := w.Create(ctx, &p, parent); err != nil {
				return err
			}
			req.Panels[i] = p
		}
		res.Panels = req.Panels
		return nil
	})
}

type RenameRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  panel.Key `json:"key" msgpack:"key"`
}

func (s *Service) Rename(ctx context.Context, req RenameRequest) (res types.Nil, err error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{panel.OntologyID(req.Key)},
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
		Keys       []panel.Key `json:"keys" msgpack:"keys"`
		Limit      int         `json:"limit" msgpack:"limit"`
		Offset     int         `json:"offset" msgpack:"offset"`
	}
	RetrieveResponse struct {
		Panels []panel.Panel `json:"panels" msgpack:"panels"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (res RetrieveResponse, err error) {
	q := s.internal.NewRetrieve().Search(req.SearchTerm)
	if len(req.Keys) > 0 {
		q = q.Where(panel.MatchKeys(req.Keys...))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	err = q.Entries(&res.Panels).Exec(ctx, nil)
	if eErr := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: panel.OntologyIDsFromPanels(res.Panels),
	}); eErr != nil {
		return RetrieveResponse{}, eErr
	}
	return res, err
}

// DispatchRequest carries an action sequence to apply to a single panel.
// DispatchKey is a client-generated identifier for the batch, registered as
// outstanding by the client before the request so it can recognize and skip
// its own broadcast echo.
type DispatchRequest = actions.DispatchRequest[panel.Key, panel.Action]

// Dispatch applies the action sequence to the target panel atomically. The actions
// are reduced server-side via panel.Reduce; on success the resulting Scoped action
// is broadcast on the panel set channel so connected clients can mirror the change.
func (s *Service) Dispatch(ctx context.Context, req DispatchRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{panel.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
	})
}

type DeleteRequest struct {
	Keys []panel.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: panel.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
