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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the table API service.
type ServiceConfig struct {
	// Table stores and retrieves tables.
	//
	// [REQUIRED]
	Table *table.Service
	// RBAC checks the access of retrieve requests.
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

// Create creates the tables, and overwrites each one whose key exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
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

// Dispatch applies the action sequence to the target table in tx. Subscribers to the
// table action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
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

// Delete deletes the tables with the given keys. A key that matches no table is
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// SetGroupRequest moves tables into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key  `json:"group" msgpack:"group"`
	Keys  []table.Key `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the tables with the given keys into a group. A move into a group under
// another project moves the tables into that project.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}

// SetProjectRequest moves tables into Project.
type SetProjectRequest struct {
	Project project.Key `json:"project" msgpack:"project"`
	Keys    []table.Key `json:"keys"    msgpack:"keys"`
}

// SetProject moves the tables with the given keys out of their group and into a
// project.
func (s *Service) SetProject(
	ctx context.Context,
	tx gorp.Tx,
	req SetProjectRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		SetProject(ctx, req.Project, req.Keys...)
}
