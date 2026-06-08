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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *imex.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		internal: cfg.Service.ImEx,
		access:   cfg.Service.RBAC,
	}, nil
}

type (
	ImportRequest  = imex.Envelope
	ImportResponse struct {
		Key string `json:"key" msgpack:"key"`
	}
)

func (s *Service) Import(
	ctx context.Context,
	req ImportRequest,
) (ImportResponse, error) {
	resourceType, err := s.internal.ImporterType(req.Type)
	if err != nil {
		return ImportResponse{}, err
	}
	var key string
	if err = s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err = s.access.Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionCreate,
			Objects: []ontology.ID{{Type: resourceType, Key: ""}},
		}); err != nil {
			return err
		}
		if key, err = s.internal.Import(ctx, tx, req); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return ImportResponse{}, err
	}
	return ImportResponse{Key: key}, nil
}

type (
	ExportRequest  = ontology.ID
	ExportResponse = imex.Envelope
)

func (s *Service) Export(
	ctx context.Context,
	req ExportRequest,
) (ExportResponse, error) {
	var env imex.Envelope
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.access.Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionRetrieve,
			Objects: []ontology.ID{req},
		})
		if err != nil {
			return err
		}
		env, err = s.internal.Export(ctx, tx, req)
		return err
	}); err != nil {
		return ExportResponse{}, err
	}
	return env, nil
}
