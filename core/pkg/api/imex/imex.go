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
	"encoding/json"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
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
	opts, err := parseImportOptions(ctx)
	if err != nil {
		return ImportResponse{}, err
	}
	enforcer := s.access.NewEnforcer(tx)
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: resourceType, Key: ""}},
	}); err != nil {
		return ImportResponse{}, err
	}
	if opts.Project != uuid.Nil {
		if err = enforcer.Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []ontology.ID{project.OntologyID(opts.Project)},
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

// importParams mirrors the JSON object clients send on the "params" request param.
type importParams struct {
	// FileName is the name of the file the envelope was read from.
	FileName string `json:"file_name"`
	// Project is the key of the project to create the imported resource under.
	Project string `json:"project"`
}

// parseImportOptions decodes the optional "params" request param — a JSON object
// carrying the out-of-band import options. An absent or empty param yields zero
// options; malformed JSON or an invalid project key returns a validation error scoped
// to the offending field.
func parseImportOptions(ctx context.Context) (imex.ImportOptions, error) {
	var opts imex.ImportOptions
	v, ok := freighter.MDFromContext(ctx).Params.Get("params")
	if !ok {
		return opts, nil
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return opts, nil
	}
	var params importParams
	if err := json.Unmarshal([]byte(s), &params); err != nil {
		return imex.ImportOptions{}, validate.PathedError(
			errors.Wrap(validate.ErrValidation, "params must be a valid JSON object"),
			"params",
		)
	}
	opts.FileName = params.FileName
	if params.Project != "" {
		key, err := uuid.Parse(params.Project)
		if err != nil {
			return imex.ImportOptions{}, validate.PathedError(
				errors.Wrapf(
					validate.ErrValidation, "invalid project key %q", params.Project,
				),
				"project",
			)
		}
		opts.Project = key
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
