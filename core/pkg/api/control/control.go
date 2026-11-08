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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/control"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the control API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Control = override.Nil(c.Control, other.Control)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.control")
	v.NotNil("control", c.Control)
	return v.Error()
}

// Service exposes the cluster's control state to API clients.
type Service struct {
	internal *control.Service
}

// NewService creates a new control API service using the provided configuration(s).
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Control}, nil
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

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It names the data of each channel in Keys.
func (r RetrieveRequest) Named() []resource.ID { return framer.ResourceIDs(r.Keys) }

// Retrieve returns the control state of the requested channels. It leaves out each
// channel whose data the subject cannot retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	states, err := s.internal.Retrieve(ctx, req.Keys...)
	if err != nil {
		return RetrieveResponse{}, err
	}
	ids := framer.ResourceIDs(lo.Map(states, func(st State, _ int) channel.Key {
		return st.Resource
	}))
	return RetrieveResponse{States: lo.Filter(states, func(_ State, i int) bool {
		return snap.Allowed(ids[i])
	})}, nil
}
