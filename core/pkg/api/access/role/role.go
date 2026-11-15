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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbacrole "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the role API service.
type ServiceConfig struct {
	// RBAC stores and retrieves roles.
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

// Create creates the roles in the request. It returns a validation error when a role
// with one of their keys exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.internal.Role.NewWriter(tx, allowInternal)
	if err := w.CreateMany(ctx, &req.Roles); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

// DispatchRequest carries an action sequence to apply to a single role. DispatchKey is
// a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[rbacrole.Key, rbacrole.Action]

// Dispatch applies the action sequence to the target role atomically. Subscribers to
// the role action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.Role.NewWriter(tx, allowInternal).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the roles that the top level of Where selects
// by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, rbacrole.ResourceID)
}

// Retrieve returns the roles that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.Role.NewRetrieve().Entries(&res.Roles))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, rbacrole.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DeleteRequest holds the keys of the roles to delete.
type DeleteRequest struct {
	Keys []rbacrole.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the roles in the request and their policies.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	w := s.internal.Role.NewWriter(tx, allowInternal)
	for _, key := range req.Keys {
		if err := w.Delete(ctx, key); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}
