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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcimex "github.com/synnaxlabs/synnax/pkg/service/imex"
	xconfig "github.com/synnaxlabs/x/config"
)

type Service struct {
	access   *rbac.Service
	internal *svcimex.Service
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
	ImportRequest struct {
		Parent    ontology.ID        `json:"parent" msgpack:"parent"`
		Resources []svcimex.Envelope `json:"resources" msgpack:"resources"`
	}
	ImportResponse struct{}
)

func (s *Service) Import(
	ctx context.Context,
	req ImportRequest,
) (ImportResponse, error) {
	objects := make([]ontology.ID, len(req.Resources))
	for i, env := range req.Resources {
		objects[i] = ontology.ID{
			Type: ontology.ResourceType(env.Type),
			Key:  env.Key,
		}
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: objects,
	}); err != nil {
		return ImportResponse{}, err
	}
	return ImportResponse{}, s.internal.Import(ctx, req.Parent, req.Resources)
}

type (
	ExportRequest struct {
		Resources []ontology.ID `json:"resources" msgpack:"resources"`
	}
	ExportResponse struct {
		Resources []svcimex.Envelope `json:"resources" msgpack:"resources"`
	}
)

func (s *Service) Export(
	ctx context.Context,
	req ExportRequest,
) (ExportResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: req.Resources,
	}); err != nil {
		return ExportResponse{}, err
	}
	envs, err := s.internal.Export(ctx, req.Resources)
	if err != nil {
		return ExportResponse{}, err
	}
	return ExportResponse{Resources: envs}, nil
}
