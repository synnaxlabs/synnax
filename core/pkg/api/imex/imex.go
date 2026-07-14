// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	access   *rbac.Service
	internal *imex.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Service.ImEx,
		access:   cfg.Service.RBAC,
	}, nil
}

type (
	ImportRequest  = imex.Envelope
	ImportResponse = ontology.ID
)

func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	req ImportRequest,
) (ImportResponse, error) {
	resourceType, err := s.internal.ImporterType(req.Type)
	if err != nil {
		return ImportResponse{}, err
	}
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: resourceType, Key: ""}},
	}); err != nil {
		return ImportResponse{}, err
	}
	id, err := s.internal.Import(ctx, tx, req)
	if err != nil {
		return ImportResponse{}, err
	}
	return id, nil
}

type (
	ExportRequest  = ontology.ID
	ExportResponse = imex.Envelope
)

func (s *Service) Export(
	ctx context.Context,
	req ExportRequest,
) (ExportResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{req},
	}); err != nil {
		return ExportResponse{}, err
	}
	env, err := s.internal.Export(ctx, req)
	if err != nil {
		return ExportResponse{}, err
	}
	return env, nil
}
