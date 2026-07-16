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
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct{ internal *rbac.Service }

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Service.RBAC}, nil
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

type RetrievePolicyRequest struct {
	Internal *bool         `json:"internal" msgpack:"internal"`
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
	Keys     []policy.Key  `json:"keys" msgpack:"keys"`
	Limit    int           `json:"limit" msgpack:"limit"`
	Offset   int           `json:"offset" msgpack:"offset"`
}

type RetrievePolicyResponse struct {
	Policies []policy.Policy `json:"policies,omitzero" msgpack:"policies,omitzero"`
}

func (s *Service) RetrievePolicy(
	ctx context.Context,
	req RetrievePolicyRequest,
) (RetrievePolicyResponse, error) {
	q := s.internal.Policy.NewRetrieve()
	if len(req.Subjects) > 0 {
		subjectKeys, err := s.internal.Policy.ResolveSubjects(ctx, nil, req.Subjects...)
		if err != nil {
			return RetrievePolicyResponse{}, err
		}
		if len(req.Keys) > 0 {
			subjectKeys = lo.Intersect(subjectKeys, req.Keys)
		}
		q = q.Where(policy.MatchKeys(subjectKeys...))
	} else if len(req.Keys) > 0 {
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
) (types.Nil, error) {
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: policy.OntologyIDs(req.Keys),
		Action:  access.ActionDelete,
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.Policy.NewWriter(tx, allowInternal).
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
	RetrieveRoleRequest struct {
		Internal *bool      `json:"internal" msgpack:"internal"`
		Keys     []role.Key `json:"keys" msgpack:"keys"`
		Limit    int        `json:"limit" msgpack:"limit"`
		Offset   int        `json:"offset" msgpack:"offset"`
	}
	RetrieveRoleResponse struct {
		Roles []role.Role `json:"roles,omitzero" msgpack:"roles,omitzero"`
	}
)

func (s *Service) RetrieveRole(
	ctx context.Context,
	req RetrieveRoleRequest,
) (RetrieveRoleResponse, error) {
	q := s.internal.Role.NewRetrieve()
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
) (types.Nil, error) {
	roleIDs := make([]ontology.ID, len(req.Keys))
	for i, key := range req.Keys {
		roleIDs[i] = role.OntologyID(key)
	}
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: roleIDs,
		Action:  access.ActionDelete,
	}); err != nil {
		return types.Nil{}, err
	}
	w := s.internal.Role.NewWriter(tx, allowInternal)
	for _, key := range req.Keys {
		if err := w.Delete(ctx, key); err != nil {
			return types.Nil{}, err
		}
	}
	return types.Nil{}, nil
}

type AssignRoleRequest struct {
	User user.Key `json:"user" msgpack:"user"`
	Role role.Key `json:"role" msgpack:"role"`
}

func (s *Service) AssignRole(
	ctx context.Context,
	tx gorp.Tx,
	req AssignRoleRequest,
) (types.Nil, error) {
	userID := user.OntologyID(req.User)
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{userID},
		Action:  access.ActionUpdate,
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.Role.NewWriter(tx, allowInternal).AssignRole(ctx, userID, req.Role)
}

type UnassignRoleRequest struct {
	User user.Key `json:"user" msgpack:"user"`
	Role role.Key `json:"role" msgpack:"role"`
}

func (s *Service) UnassignRole(
	ctx context.Context,
	tx gorp.Tx,
	req UnassignRoleRequest,
) (types.Nil, error) {
	userID := user.OntologyID(req.User)
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{userID},
		Action:  access.ActionUpdate,
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.Role.NewWriter(tx, allowInternal).
		UnassignRole(ctx, userID, req.Role)
}
