// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the access API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the access API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.access")
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct{ internal *rbac.Service }

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.RBAC}, nil
}

const allowInternal = false

type (
	CreatePolicyRequest struct {
		Policies []policy.Policy `json:"policies" msgpack:"policies"`
	}
	CreatePolicyResponse = CreatePolicyRequest
)

func (s *Service) CreatePolicy(
	ctx context.Context,
	tx gorp.Tx,
	req CreatePolicyRequest,
) (CreatePolicyResponse, error) {
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{{Type: ontology.ResourceTypePolicy}},
		Action:  access.ActionCreate,
	}); err != nil {
		return CreatePolicyRequest{}, err
	}
	w := s.internal.Policy.NewWriter(tx, allowInternal)
	if err := w.CreateMany(ctx, &req.Policies); err != nil {
		return CreatePolicyRequest{}, err
	}
	return CreatePolicyResponse(req), nil
}

type (
	RetrievePolicyRequest  = policy.RetrieveRequest
	RetrievePolicyResponse = policy.RetrieveResponse
)

func (s *Service) RetrievePolicy(
	ctx context.Context,
	req RetrievePolicyRequest,
) (RetrievePolicyResponse, error) {
	var res RetrievePolicyResponse
	q := s.internal.Policy.NewRetrieve().Entries(&res.Policies)
	if req.Where != nil {
		f, err := policy.MatchWhere(*req.Where)
		if err != nil {
			return RetrievePolicyResponse{}, validate.PathedError(err, "where")
		}
		q = q.Where(f)
	}
	if term := lo.FromPtr(req.SearchTerm); term != "" {
		q = q.Search(term)
	}
	if req.Limit != nil {
		q = q.Limit(int(*req.Limit))
	}
	if req.Offset != nil {
		q = q.Offset(int(*req.Offset))
	}
	// A filter is a predicate, so a key that does not exist is a non-match.
	if err := errors.Skip(q.Exec(ctx, nil), query.ErrNotFound); err != nil {
		return RetrievePolicyResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: policy.OntologyIDsFromPolicies(res.Policies),
	}); err != nil {
		return RetrievePolicyResponse{}, err
	}
	return res, nil
}

type DeletePolicyRequest struct {
	Keys []policy.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) DeletePolicy(
	ctx context.Context,
	tx gorp.Tx,
	req DeletePolicyRequest,
) (struct{}, error) {
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: policy.OntologyIDs(req.Keys),
		Action:  access.ActionDelete,
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.Policy.NewWriter(tx, allowInternal).
		Delete(ctx, req.Keys...)
}

type (
	CreateRoleRequest struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
	CreateRoleResponse struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
)

func (s *Service) CreateRole(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRoleRequest,
) (CreateRoleResponse, error) {
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{{Type: ontology.ResourceTypeRole}},
		Action:  access.ActionCreate,
	}); err != nil {
		return CreateRoleResponse{}, err
	}
	w := s.internal.Role.NewWriter(tx, allowInternal)
	if err := w.CreateMany(ctx, &req.Roles); err != nil {
		return CreateRoleResponse{}, err
	}
	return CreateRoleResponse(req), nil
}

type (
	RetrieveRoleRequest  = role.RetrieveRequest
	RetrieveRoleResponse = role.RetrieveResponse
)

func (s *Service) RetrieveRole(
	ctx context.Context,
	req RetrieveRoleRequest,
) (RetrieveRoleResponse, error) {
	var res RetrieveRoleResponse
	q := s.internal.Role.NewRetrieve().Entries(&res.Roles)
	if req.Where != nil {
		f, err := role.MatchWhere(*req.Where)
		if err != nil {
			return RetrieveRoleResponse{}, validate.PathedError(err, "where")
		}
		q = q.Where(f)
	}
	if term := lo.FromPtr(req.SearchTerm); term != "" {
		q = q.Search(term)
	}
	if req.Limit != nil {
		q = q.Limit(int(*req.Limit))
	}
	if req.Offset != nil {
		q = q.Offset(int(*req.Offset))
	}
	// A filter is a predicate, so a key that does not exist is a non-match.
	if err := errors.Skip(q.Exec(ctx, nil), query.ErrNotFound); err != nil {
		return RetrieveRoleResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: role.OntologyIDsFromRoles(res.Roles),
	}); err != nil {
		return RetrieveRoleResponse{}, err
	}
	return res, nil
}

type DeleteRoleRequest struct {
	Keys []role.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) DeleteRole(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRoleRequest,
) (struct{}, error) {
	roleIDs := make([]ontology.ID, len(req.Keys))
	for i, key := range req.Keys {
		roleIDs[i] = role.OntologyID(key)
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

type AssignRoleRequest struct {
	User user.Key `json:"user" msgpack:"user"`
	Role role.Key `json:"role" msgpack:"role"`
}

func (s *Service) AssignRole(
	ctx context.Context,
	tx gorp.Tx,
	req AssignRoleRequest,
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

type UnassignRoleRequest struct {
	User user.Key `json:"user" msgpack:"user"`
	Role role.Key `json:"role" msgpack:"role"`
}

func (s *Service) UnassignRole(
	ctx context.Context,
	tx gorp.Tx,
	req UnassignRoleRequest,
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
