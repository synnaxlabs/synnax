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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the panel API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Panel stores and retrieves panels.
	//
	// [REQUIRED]
	Panel *panel.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the panel API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Panel = override.Nil(c.Panel, other.Panel)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.panel")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("panel", c.Panel)
	return v.Error()
}

// Service is the API-layer panel service. It enforces access control and routes
// requests to the distribution panel service. Write handlers execute inside the
// transaction provided by the binding layer (fgorp.CreateWriteUnaryHandler).
type Service struct {
	access   *rbac.Service
	internal *panel.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.RBAC,
		internal: cfg.Panel,
	}, nil
}

type (
	CreateRequest struct {
		// Panels are the panels to create. Each panel carries the Parent resource
		// it attaches to in the ontology; a panel without a Parent is rejected.
		Panels []panel.Panel `json:"panels" msgpack:"panels"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Panels []panel.Panel `json:"panels" msgpack:"panels"`
	}
)

// Create persists the panels in req and returns them with their assigned keys,
// parenting each to its Parent. It returns a validation error when any panel
// has no Parent.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (res CreateResponse, err error) {
	objects := []resource.ID{{Type: resource.TypePanel}}
	for i := range req.Panels {
		if req.Panels[i].Parent == nil || req.Panels[i].Parent.IsZero() {
			return res, errors.Wrapf(
				validate.ErrValidation,
				"panel %q has no parent",
				req.Panels[i].Name,
			)
		}
		objects = append(objects, *req.Panels[i].Parent)
	}
	enforcer := s.access.NewEnforcer(tx)
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: objects,
	}); err != nil {
		return res, err
	}
	var stored []panel.Panel
	if err = s.internal.NewRetrieve().
		Where(panel.MatchKeys(lo.Map(
			req.Panels,
			func(p panel.Panel, _ int) panel.Key { return p.Key },
		)...)).
		Entries(&stored).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return res, err
	}
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.CreateObjects(
			req.Panels,
			stored,
			func(p panel.Panel) *group.Key { return p.Group },
		),
	}); err != nil {
		return res, err
	}
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Panels); err != nil {
		return res, err
	}
	res.Panels = req.Panels
	return res, nil
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Panels))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: panel.ResourceIDsFromPanels(res.Panels),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
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
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{panel.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.
		Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

type DeleteRequest struct {
	Keys []panel.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: panel.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// SetGroupRequest moves panels into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key  `json:"group" msgpack:"group"`
	Keys  []panel.Key `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the panels with the given keys into a group.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	var panels []panel.Panel
	if err := s.internal.NewRetrieve().
		Where(panel.MatchKeys(req.Keys...)).
		Entries(&panels).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.MoveObjects(
			panel.ResourceIDs(req.Keys),
			lo.Map(panels, func(e panel.Panel, _ int) *group.Key { return e.Group }),
			req.Group,
		),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}
