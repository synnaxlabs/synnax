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

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	license "github.com/synnaxlabs/synnax/pkg/service/channel/license"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xconfig "github.com/synnaxlabs/x/config"
)

type Service struct {
	access   *rbac.Service
	internal *license.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.Service.RBAC,
		internal: cfg.Service.License,
	}, nil
}

type (
	RetrieveRequest  = struct{}
	RetrieveResponse = license.Info
	ApplyRequest     struct {
		Token string `json:"token" msgpack:"token"`
	}
	ApplyResponse = license.Info
)

var objectID = ontology.ID{Type: ontology.ResourceTypeLicense}

// Retrieve returns the state of the Core's license, this machine's fingerprint,
// and the license when one applies.
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

// Apply accepts a token for this Core and returns the resulting state.
func (s *Service) Apply(
	ctx context.Context,
	req ApplyRequest,
) (ApplyResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{objectID},
	}); err != nil {
		return ApplyResponse{}, err
	}
	return s.internal.Apply(ctx, req.Token)
}
