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

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Service is the API-layer panel service. It enforces access control and routes
// requests to the distribution panel service. Write handlers execute inside the
// transaction provided by the binding layer (fgorp.CreateWriteUnaryHandler).
type Service struct {
	access   *rbac.Service
	internal *panel.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Panel,
	}, nil
}

type (
	CreateRequest struct {
		// Panels are the panels to create. Each panel carries its own optional
		// Parent; when absent the panel is parented to the creating user instead
		// — a draft, visible only to its creator.
		Panels []panel.Panel `json:"panels" msgpack:"panels"`
	}
	CreateResponse = CreateRequest
)

// Create persists the panels in req and returns them with their assigned keys.
// Each panel is parented to its own Parent, or to the creating user as a draft
// when Parent is absent.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (res CreateResponse, err error) {
	subject := auth.GetSubject(ctx)
	objects := []ontology.ID{{Type: ontology.ResourceTypePanel}}
	for i := range req.Panels {
		// No parent -> draft, parented to the creating user.
		if req.Panels[i].Parent == nil || req.Panels[i].Parent.IsZero() {
			req.Panels[i].Parent = &subject
		} else {
			objects = append(objects, *req.Panels[i].Parent)
		}
	}
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: objects,
	}); err != nil {
		return res, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Panels); err != nil {
		return res, err
	}
	res.Panels = req.Panels
	return res, nil
}

type (
	RetrieveRequest struct {
		SearchTerm          string      `json:"search_term" msgpack:"search_term"`
		Keys                []panel.Key `json:"keys" msgpack:"keys"`
		Limit               int         `json:"limit" msgpack:"limit"`
		Offset              int         `json:"offset" msgpack:"offset"`
		IgnoreNotFoundError bool        `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
	}
	RetrieveResponse struct {
		Panels []panel.Panel `json:"panels,omitzero" msgpack:"panels,omitzero"`
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
	if req.IgnoreNotFoundError && err != nil {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if eErr := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
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
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (res types.Nil, err error) {
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{panel.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.internal.NewWriter(tx).Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

type DeleteRequest struct {
	Keys []panel.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (res types.Nil, err error) {
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: panel.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
