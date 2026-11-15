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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbacpolicy "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the policy API service.
type ServiceConfig struct {
	// RBAC stores and retrieves policies.
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

// Create creates the policies in the request. It returns a validation error when a
// policy with one of their keys exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.internal.Policy.NewWriter(tx, allowInternal)
	if err := w.CreateMany(ctx, &req.Policies); err != nil {
		return CreateRequest{}, err
	}
	return CreateResponse(req), nil
}

// DispatchRequest carries an action sequence to apply to a single policy. DispatchKey
// is a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[rbacpolicy.Key, rbacpolicy.Action]

// Dispatch applies the action sequence to the target policy atomically. Subscribers to
// the policy action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.Policy.NewWriter(tx, allowInternal).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the policies that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, rbacpolicy.ResourceID)
}

// Retrieve returns the policies that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewPolicyRetrieve().Entries(&res.Policies))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, rbacpolicy.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DeleteRequest holds the keys of the policies to delete.
type DeleteRequest struct {
	Keys []rbacpolicy.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the policies with the keys in the request.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.Policy.NewWriter(tx, allowInternal).
		Delete(ctx, req.Keys...)
}
