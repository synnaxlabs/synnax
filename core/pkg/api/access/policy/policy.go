// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbacpolicy "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the policy API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the policy API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.access.policy")
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service creates, retrieves, and deletes access policies.
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
	// CreateRequest holds the policies to create.
	CreateRequest struct {
		Policies []rbacpolicy.Policy `json:"policies" msgpack:"policies"`
	}
	// CreateResponse holds the created policies.
	CreateResponse = CreateRequest
)

// Create creates or overwrites the policies in the request. It requires create access
// on policies, and update access on the role of each policy and on the stored role of
// each policy it overwrites.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	enforcer := s.internal.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Objects: []resource.ID{{Type: resource.TypePolicy}},
		Action:  access.ActionCreate,
	}); err != nil {
		return CreateRequest{}, err
	}
	keys := make([]rbacpolicy.Key, 0, len(req.Policies))
	roles := make([]role.Key, 0, len(req.Policies))
	for _, p := range req.Policies {
		if p.Key != uuid.Nil() {
			keys = append(keys, p.Key)
		}
		roles = append(roles, p.Role)
	}
	stored, err := s.retrieve(ctx, tx, keys)
	if err != nil {
		return CreateRequest{}, err
	}
	for _, p := range stored {
		roles = append(roles, p.Role)
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Objects: role.ResourceIDs(lo.Uniq(roles)),
		Action:  access.ActionUpdate,
	}); err != nil {
		return CreateRequest{}, err
	}
	w := s.internal.Policy.NewWriter(tx, allowInternal)
	if err := w.CreateMany(ctx, &req.Policies); err != nil {
		return CreateRequest{}, err
	}
	return CreateResponse(req), nil
}

// Retrieve retrieves the policies that the request selects.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewPolicyRetrieve().Entries(&res.Policies))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rbacpolicy.ResourceIDsFromPolicies(res.Policies),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DeleteRequest holds the keys of the policies to delete.
type DeleteRequest struct {
	Keys []rbacpolicy.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the policies with the keys in the request. It requires delete access
// on the policies and update access on their roles.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	enforcer := s.internal.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Objects: rbacpolicy.ResourceIDs(req.Keys),
		Action:  access.ActionDelete,
	}); err != nil {
		return struct{}{}, err
	}
	stored, err := s.retrieve(ctx, tx, req.Keys)
	if err != nil {
		return struct{}{}, err
	}
	roles := lo.Uniq(lo.Map(stored, func(p rbacpolicy.Policy, _ int) role.Key {
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

// retrieve returns the stored policies with the given keys, skipping keys that no
// policy holds.
func (s *Service) retrieve(
	ctx context.Context,
	tx gorp.Tx,
	keys []rbacpolicy.Key,
) ([]rbacpolicy.Policy, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	var stored []rbacpolicy.Policy
	err := s.internal.Policy.NewRetrieve().
		Where(rbacpolicy.MatchKeys(keys...)).
		Entries(&stored).
		Exec(ctx, tx)
	return stored, errors.Skip(err, query.ErrNotFound)
}
