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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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

// CreatePolicy creates or overwrites the policies in the request. It requires create
// access on policies, and update access on the role of each policy and on the stored
// role of each policy it overwrites.
func (s *Service) CreatePolicy(
	ctx context.Context,
	tx gorp.Tx,
	req CreatePolicyRequest,
) (CreatePolicyResponse, error) {
	enforcer := s.internal.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Objects: []resource.ID{{Type: resource.TypePolicy}},
		Action:  access.ActionCreate,
	}); err != nil {
		return CreatePolicyRequest{}, err
	}
	keys := make([]policy.Key, 0, len(req.Policies))
	roles := make([]role.Key, 0, len(req.Policies))
	for _, p := range req.Policies {
		if p.Key != uuid.Nil() {
			keys = append(keys, p.Key)
		}
		roles = append(roles, p.Role)
	}
	stored, err := s.retrievePolicies(ctx, tx, keys)
	if err != nil {
		return CreatePolicyRequest{}, err
	}
	for _, p := range stored {
		roles = append(roles, p.Role)
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Objects: role.ResourceIDs(lo.Uniq(roles)),
		Action:  access.ActionUpdate,
	}); err != nil {
		return CreatePolicyRequest{}, err
	}
	w := s.internal.Policy.NewWriter(tx, allowInternal)
	if err := w.CreateMany(ctx, &req.Policies); err != nil {
		return CreatePolicyRequest{}, err
	}
	return CreatePolicyResponse(req), nil
}

// retrievePolicies returns the stored policies with the given keys, skipping keys that
// no policy holds.
func (s *Service) retrievePolicies(
	ctx context.Context,
	tx gorp.Tx,
	keys []policy.Key,
) ([]policy.Policy, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	var stored []policy.Policy
	err := s.internal.Policy.NewRetrieve().
		Where(policy.MatchKeys(keys...)).
		Entries(&stored).
		Exec(ctx, tx)
	return stored, errors.Skip(err, query.ErrNotFound)
}

type RetrievePolicyRequest struct {
	Internal *bool `json:"internal" msgpack:"internal"`
	// Subjects keeps only the policies of the roles that these subjects hold.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
	// Roles keeps only the policies of these roles.
	Roles  []role.Key   `json:"roles"  msgpack:"roles"`
	Keys   []policy.Key `json:"keys"   msgpack:"keys"`
	Limit  int          `json:"limit"  msgpack:"limit"`
	Offset int          `json:"offset" msgpack:"offset"`
}

type RetrievePolicyResponse struct {
	Policies []policy.Policy `json:"policies" msgpack:"policies"`
}

func (s *Service) RetrievePolicy(
	ctx context.Context,
	req RetrievePolicyRequest,
) (RetrievePolicyResponse, error) {
	q := s.internal.Policy.NewRetrieve()
	if len(req.Subjects) > 0 {
		roles, err := s.internal.RolesOf(ctx, nil, req.Subjects...)
		if err != nil {
			return RetrievePolicyResponse{}, err
		}
		if len(roles) == 0 {
			return RetrievePolicyResponse{Policies: []policy.Policy{}}, nil
		}
		q = q.Where(policy.MatchRoles(roles...))
	}
	if len(req.Roles) > 0 {
		q = q.Where(policy.MatchRoles(req.Roles...))
	}
	if len(req.Keys) > 0 {
		q = q.Where(policy.MatchKeys(req.Keys...))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if req.Internal != nil {
		q = q.Where(policy.MatchInternal(*req.Internal))
	}
	var res RetrievePolicyResponse
	if err := q.Entries(&res.Policies).Exec(ctx, nil); err != nil {
		return RetrievePolicyResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: policy.ResourceIDsFromPolicies(res.Policies),
	}); err != nil {
		return RetrievePolicyResponse{}, err
	}
	return res, nil
}

type DeletePolicyRequest struct {
	Keys []policy.Key `json:"keys" msgpack:"keys"`
}

// DeletePolicy deletes the policies with the keys in the request. It requires delete
// access on the policies and update access on their roles.
func (s *Service) DeletePolicy(
	ctx context.Context,
	tx gorp.Tx,
	req DeletePolicyRequest,
) (struct{}, error) {
	enforcer := s.internal.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Objects: policy.ResourceIDs(req.Keys),
		Action:  access.ActionDelete,
	}); err != nil {
		return struct{}{}, err
	}
	stored, err := s.retrievePolicies(ctx, tx, req.Keys)
	if err != nil {
		return struct{}{}, err
	}
	roles := lo.Uniq(lo.Map(stored, func(p policy.Policy, _ int) role.Key {
		return p.Role
	}))
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Objects: role.ResourceIDs(roles),
		Action:  access.ActionUpdate,
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
		Objects: []resource.ID{{Type: resource.TypeRole}},
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
	RetrieveRoleRequest struct {
		Internal   *bool      `json:"internal"    msgpack:"internal"`
		Keys       []role.Key `json:"keys"        msgpack:"keys"`
		SearchTerm string     `json:"search_term" msgpack:"search_term"`
		Limit      int        `json:"limit"       msgpack:"limit"`
		Offset     int        `json:"offset"      msgpack:"offset"`
	}
	RetrieveRoleResponse struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
)

func (s *Service) RetrieveRole(
	ctx context.Context,
	req RetrieveRoleRequest,
) (RetrieveRoleResponse, error) {
	q := s.internal.Role.NewRetrieve()
	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	if len(req.Keys) > 0 {
		q = q.Where(role.MatchKeys(req.Keys...))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if req.Internal != nil {
		q = q.Where(role.MatchInternal(*req.Internal))
	}
	var res RetrieveRoleResponse
	if err := q.Entries(&res.Roles).Exec(ctx, nil); err != nil {
		return RetrieveRoleResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: role.ResourceIDsFromRoles(res.Roles),
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
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: role.ResourceIDs(req.Keys),
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
