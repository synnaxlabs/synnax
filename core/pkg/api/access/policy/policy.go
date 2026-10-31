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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbacpolicy "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
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

// Create creates the policies in the request.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: []ontology.ID{{Type: ontology.ResourceTypePolicy}},
		Action:  access.ActionCreate,
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
	q, err := req.Apply(s.internal.Policy.NewRetrieve().Entries(&res.Policies))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rbacpolicy.OntologyIDsFromPolicies(res.Policies),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DeleteRequest holds the keys of the policies to delete.
type DeleteRequest struct {
	Keys []rbacpolicy.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the policies in the request.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Objects: rbacpolicy.OntologyIDs(req.Keys),
		Action:  access.ActionDelete,
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.Policy.NewWriter(tx, allowInternal).
		Delete(ctx, req.Keys...)
}
