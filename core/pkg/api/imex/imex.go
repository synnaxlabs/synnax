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

	"github.com/synnaxlabs/freighter"
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

// The out-of-band import settings arrive as freighterctx-prefixed HTTP query
// parameters (e.g. freighterctxfile_name) — the request body is the file's raw bytes,
// so there is nowhere in-band to carry them. The transport strips the prefix and
// exposes the values through freighter's request params under the names below.
const (
	// fileNameParam carries the name of the file the envelope was read from, used as
	// the envelope name when the body has no `name` field.
	fileNameParam = "file_name"
	// parentParam carries the string form of the ontology ID to parent the imported
	// resource under.
	parentParam = "parent"
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
	opts, err := parseImportOptions(ctx)
	if err != nil {
		return ImportResponse{}, err
	}
	objects := []ontology.ID{{Type: resourceType, Key: ""}}
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: objects,
	}); err != nil {
		return ImportResponse{}, err
	}
	if !opts.Parent.IsZero() {
		if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []ontology.ID{opts.Parent},
		}); err != nil {
			return ImportResponse{}, err
		}
	}
	id, err := s.internal.Import(ctx, tx, req, opts)
	if err != nil {
		return ImportResponse{}, err
	}
	return id, nil
}

// parseImportOptions extracts the optional file_name and parent query parameters from
// the request's freighter params. An empty or absent parent yields a zero Parent;
// a malformed parent ID returns a validation error.
func parseImportOptions(ctx context.Context) (imex.ImportOptions, error) {
	var (
		opts   imex.ImportOptions
		params = freighter.MDFromContext(ctx).Params
	)
	if v, ok := params.Get(fileNameParam); ok {
		if s, ok := v.(string); ok {
			opts.FileName = s
		}
	}
	if v, ok := params.Get(parentParam); ok {
		if s, ok := v.(string); ok && s != "" {
			parent, err := ontology.ParseID(s)
			if err != nil {
				return imex.ImportOptions{}, err
			}
			opts.Parent = parent
		}
	}
	return opts, nil
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
