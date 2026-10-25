// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package license exposes the Core's license over the API and gates every other
// endpoint on it.
package license

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the license API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// License holds the Core's license.
	//
	// [REQUIRED]
	License *license.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the license API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.License = override.Nil(c.License, other.License)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.license")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("license", c.License)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *license.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.RBAC,
		internal: cfg.License,
	}, nil
}

type (
	RetrieveRequest  = struct{}
	RetrieveResponse = license.Info
	ActivateRequest  struct {
		Key string `json:"key" msgpack:"key"`
	}
	ActivateResponse  = license.Info
	DeactivateRequest struct {
		Jti uuid.UUID `json:"jti" msgpack:"jti"`
	}
	DeactivateResponse = license.Info
)

// objectID names the license as a builtin singleton, so the built-in roles' builtin
// grants control access to it.
var objectID = ontology.ID{Type: ontology.ResourceTypeBuiltin, Key: license.OntologyKey}

// Retrieve returns the state of the Core's license, this machine's fingerprint, and the
// license when one applies.
func (s *Service) Retrieve(
	ctx context.Context,
	_ RetrieveRequest,
) (RetrieveResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{objectID},
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return s.internal.Retrieve(), nil
}

// Activate accepts a license key for this Core and returns the resulting state.
func (s *Service) Activate(
	ctx context.Context,
	req ActivateRequest,
) (ActivateResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{objectID},
	}); err != nil {
		return ActivateResponse{}, err
	}
	return s.internal.Activate(ctx, req.Key)
}

// Deactivate removes the license key with the given ID from this Core and returns the
// resulting state.
func (s *Service) Deactivate(
	ctx context.Context,
	req DeactivateRequest,
) (DeactivateResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{objectID},
	}); err != nil {
		return DeactivateResponse{}, err
	}
	return s.internal.Deactivate(ctx, req.Jti)
}
