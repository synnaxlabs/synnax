// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/control"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// State is the control state of a single channel.
type State = control.State

// ServiceConfig is the configuration for the control API service.
type ServiceConfig struct {
	// Control reads the control state of channels.
	//
	// [REQUIRED]
	Control *control.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the control API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Control = override.Nil(c.Control, other.Control)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.control")
	v.NotNil("control", c.Control)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service exposes the cluster's control state to API clients.
type Service struct {
	access   *rbac.Service
	internal *control.Service
}

// NewService creates a new control API service using the provided configuration(s).
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Control, access: cfg.RBAC}, nil
}

// RetrieveRequest is a request to read the control state of a set of channels.
type RetrieveRequest struct {
	// Keys are the channels to read control state for. Leave empty to read the state of
	// every controlled channel in the cluster.
	Keys channel.Keys `json:"keys" msgpack:"keys"`
}

// RetrieveResponse is a response to a RetrieveRequest.
type RetrieveResponse struct {
	// States holds one entry per controlled channel. Channels that no subject controls
	// are absent.
	States []State `json:"states" msgpack:"states"`
}

// Retrieve returns the control state of the requested channels.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	states, err := s.internal.Retrieve(ctx, req.Keys...)
	if err != nil {
		return RetrieveResponse{}, err
	}
	keys := make(channel.Keys, len(states))
	for i, state := range states {
		keys[i] = state.Resource
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: framer.OntologyIDs(keys),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{States: states}, nil
}
