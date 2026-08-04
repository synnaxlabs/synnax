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

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Service implements the ImEx API.
type Service struct {
	access   *rbac.Service
	internal *imex.Service
}

// NewService creates a new ImEx API service.
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
	// ImportRequest is the request type for the Import endpoint.
	ImportRequest = imex.Envelope
	// ImportResponse is the response type for the Import endpoint.
	ImportResponse = ontology.ID
)

// Import imports a resource from an envelope.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	req ImportRequest,
) (ImportResponse, error) {
	// Typeless envelopes (legacy Console state files) must be resolved to a concrete
	// registration type before access control can name the resource being created.
	typ, err := s.internal.ResolveType(ctx, req)
	if err != nil {
		return ImportResponse{}, err
	}
	req.Type = typ
	resourceType, err := s.internal.ImporterType(typ)
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
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{opts.Parent},
	}); err != nil {
		return ImportResponse{}, err
	}
	id, err := s.internal.Import(ctx, tx, req, opts)
	if err != nil {
		return ImportResponse{}, err
	}
	return id, nil
}

type importParams struct {
	// FileName is the name of the file the envelope was read from, e.g. "table.json".
	FileName string `json:"file_name"`
	// Parent carries the compact "type:key" string form clients send.
	Parent string `json:"parent"`
}

// parseImportOptions decodes the required "params" request param — a JSON object
// carrying the out-of-band import options. A missing param, malformed JSON, or a
// missing or invalid field returns a validation error scoped to the offending field.
func parseImportOptions(ctx context.Context) (imex.ImportOptions, error) {
	v, ok := freighter.MDFromContext(ctx).Get("params")
	s, isStr := v.(string)
	if !ok || !isStr || s == "" {
		return imex.ImportOptions{}, validate.PathedError(validate.ErrRequired, "params")
	}
	var params importParams
	if err := json.Unmarshal([]byte(s), &params); err != nil {
		return imex.ImportOptions{}, validate.PathedError(
			errors.Wrapf(validate.ErrValidation, "invalid params: %v", err),
			"params",
		)
	}
	if params.FileName == "" {
		return imex.ImportOptions{}, validate.PathedError(
			validate.ErrRequired, "file_name",
		)
	}
	if params.Parent == "" {
		return imex.ImportOptions{}, validate.PathedError(validate.ErrRequired, "parent")
	}
	parent, err := ontology.ParseID(params.Parent)
	if err != nil {
		return imex.ImportOptions{}, validate.PathedError(err, "parent")
	}
	if parent.Key == "" {
		return imex.ImportOptions{}, validate.PathedError(
			errors.Wrap(validate.ErrValidation, "must carry a non-empty key"),
			"parent",
		)
	}
	return imex.ImportOptions{FileName: params.FileName, Parent: parent}, nil
}

type (
	// ExportRequest is the request type for the Export endpoint.
	ExportRequest = ontology.ID
	// ExportResponse is the response type for the Export endpoint.
	ExportResponse = imex.Envelope
)

// Export exports a resource to an envelope.
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
