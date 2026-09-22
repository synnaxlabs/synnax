// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package verification exposes the Core's grant over the API and gates every other
// endpoint on it.
package verification

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcverification "github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xconfig "github.com/synnaxlabs/x/config"
)

type Service struct {
	access   *rbac.Service
	internal *svcverification.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Verification,
	}, nil
}

type (
	RetrieveRequest  = types.Nil
	RetrieveResponse = svcverification.Info
	ActivateRequest  struct {
		Token string `json:"token" msgpack:"token"`
	}
	ActivateResponse = svcverification.Info
)

var objectID = ontology.ID{Type: ontology.ResourceTypeVerification}

// Retrieve returns the state of the Core's grant, the host hashes, and the grant when
// one applies.
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

// Activate accepts a token for this Core and returns the resulting state.
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
	return s.internal.Activate(ctx, req.Token)
}
