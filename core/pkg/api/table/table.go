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
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the table API service.
type ServiceConfig struct {
	// Table stores and retrieves tables.
	//
	// [REQUIRED]
	Table *table.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the table API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Table = override.Nil(c.Table, other.Table)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.table")
	v.NotNil("table", c.Table)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *table.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Table,
		access:   cfg.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	enforcer := s.access.NewEnforcer(tx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeTable}},
	}); err != nil {
		return CreateResponse{}, err
	}
	var stored []table.Table
	if err := s.internal.NewRetrieve().
		Where(table.MatchKeys(lo.Map(
			req.Tables,
			func(e table.Table, _ int) table.Key { return e.Key },
		)...)).
		Entries(&stored).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return CreateResponse{}, err
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: slices.Concat(
			group.CreateObjects(
				req.Tables,
				stored,
				func(e table.Table) *group.Key { return e.Group },
			),
			project.CreateObjects(
				req.Tables,
				stored,
				func(e table.Table) *resource.ID {
					return project.OwnerID(e.Project, e.Range)
				},
			),
		),
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Tables); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Tables: req.Tables}, nil
}

// DispatchRequest carries an action sequence to apply to a single table. DispatchKey is
// a client-generated identifier for the batch, registered as outstanding on the
// originator before the request is sent. The server echoes it verbatim on the broadcast
// frame so the originator can recognize its own echo race-safely.
type DispatchRequest = actions.DispatchRequest[table.Key, table.Action]

// Dispatch applies the action sequence to the target table atomically. Subscribers to
// the table action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{table.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.
		Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Tables))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: table.ResourceIDsFromTables(res.Tables),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []table.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: table.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// SetGroupRequest moves tables into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key  `json:"group" msgpack:"group"`
	Keys  []table.Key `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the tables with the given keys into a group. A move into a group under
// another project also requires update access on the stored owner and on that project.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	var tables []table.Table
	if err := s.internal.NewRetrieve().
		Where(table.MatchKeys(req.Keys...)).
		Entries(&tables).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.MoveObjects(
			table.ResourceIDs(req.Keys),
			lo.Map(tables, func(e table.Table, _ int) *group.Key { return e.Group }),
			req.Group,
		),
	}); err != nil {
		return struct{}{}, err
	}
	projects, err := project.SetGroupObjects(
		ctx,
		tx,
		req.Group,
		resource.TypeTable,
		lo.Map(tables, func(e table.Table, _ int) *resource.ID {
			return project.OwnerID(e.Project, e.Range)
		}),
	)
	if err != nil {
		return struct{}{}, err
	}
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: projects,
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}

// SetProjectRequest moves tables into Project.
type SetProjectRequest struct {
	Project project.Key `json:"project" msgpack:"project"`
	Keys    []table.Key `json:"keys"    msgpack:"keys"`
}

// SetProject moves the tables with the given keys out of their group and into a
// project. It requires update access on the tables, their owners, and the project.
func (s *Service) SetProject(
	ctx context.Context,
	tx gorp.Tx,
	req SetProjectRequest,
) (struct{}, error) {
	var stored []table.Table
	if err := s.internal.NewRetrieve().
		Where(table.MatchKeys(req.Keys...)).
		Entries(&stored).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: project.MoveObjects(
			table.ResourceIDs(req.Keys),
			lo.Map(stored, func(e table.Table, _ int) *resource.ID {
				return project.OwnerID(e.Project, e.Range)
			}),
			new(project.ResourceID(req.Project)),
		),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).
		SetProject(ctx, req.Project, req.Keys...)
}
