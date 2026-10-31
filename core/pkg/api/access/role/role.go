// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbacrole "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the role API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the role API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.access.role")
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service creates, retrieves, deletes, and assigns roles.
type Service struct{ internal *rbac.Service }

// NewService opens a Service. It returns an error if the configuration is invalid.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.RBAC}, nil
}

const allowInternal = false

type (
	// CreateRequest holds the roles to create.
	CreateRequest struct {
		Roles []rbacrole.Role `json:"roles" msgpack:"roles"`
	}
	// CreateResponse holds the created roles.
	CreateResponse struct {
		Roles []rbacrole.Role `json:"roles" msgpack:"roles"`
	}
)

// Create creates the roles in the request.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{{Type: ontology.ResourceTypeRole}},
		Action:  access.ActionCreate,
	}); err != nil {
		return CreateResponse{}, err
	}
	w := s.internal.Role.NewWriter(tx, allowInternal)
	if err := w.CreateMany(ctx, &req.Roles); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

// Retrieve retrieves the roles that the request selects.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.Role.NewRetrieve().Entries(&res.Roles))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rbacrole.OntologyIDsFromRoles(res.Roles),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DeleteRequest holds the keys of the roles to delete.
type DeleteRequest struct {
	Keys []rbacrole.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the roles in the request.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	roleIDs := make([]ontology.ID, len(req.Keys))
	for i, key := range req.Keys {
		roleIDs[i] = rbacrole.OntologyID(key)
	}
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: roleIDs,
		Action:  access.ActionDelete,
	}); err != nil {
		return struct{}{}, err
	}
	w := s.internal.Role.NewWriter(tx, allowInternal)
	for _, key := range req.Keys {
		if err := w.Delete(ctx, key); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

// AssignRequest names a user and the role to assign to them.
type AssignRequest struct {
	User user.Key     `json:"user" msgpack:"user"`
	Role rbacrole.Key `json:"role" msgpack:"role"`
}

// Assign assigns the role in the request to the user in the request.
func (s *Service) Assign(
	ctx context.Context,
	tx gorp.Tx,
	req AssignRequest,
) (struct{}, error) {
	userID := user.OntologyID(req.User)
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{userID},
		Action:  access.ActionUpdate,
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.Role.NewWriter(tx, allowInternal).
		AssignRole(ctx, userID, req.Role)
}

// UnassignRequest names a user and the role to remove from them.
type UnassignRequest struct {
	User user.Key     `json:"user" msgpack:"user"`
	Role rbacrole.Key `json:"role" msgpack:"role"`
}

// Unassign removes the role in the request from the user in the request.
func (s *Service) Unassign(
	ctx context.Context,
	tx gorp.Tx,
	req UnassignRequest,
) (struct{}, error) {
	userID := user.OntologyID(req.User)
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{userID},
		Action:  access.ActionUpdate,
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.Role.NewWriter(tx, allowInternal).
		UnassignRole(ctx, userID, req.Role)
}
