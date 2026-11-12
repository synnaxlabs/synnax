// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"
	"encoding/json/v2"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/project/bundle"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the project API service.
type ServiceConfig struct {
	// Project stores and retrieves projects.
	//
	// [REQUIRED]
	Project *project.Service
	// Bundle exports and imports projects as bundles.
	//
	// [REQUIRED]
	Bundle *bundle.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the project API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Project = override.Nil(c.Project, other.Project)
	c.Bundle = override.Nil(c.Bundle, other.Bundle)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.project")
	v.NotNil("project", c.Project)
	v.NotNil("bundle", c.Bundle)
	return v.Error()
}

type Service struct {
	internal *project.Service
	bundle   *bundle.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Project,
		bundle:   cfg.Bundle,
	}, nil
}

type (
	CreateRequest struct {
		Projects []project.Project `json:"projects" msgpack:"projects"`
	}
	CreateResponse = CreateRequest
)

// Create creates the projects. It returns an error wrapping validate.ErrValidation when
// a project with one of their keys exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Projects); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

// DispatchRequest carries an action sequence to apply to a single project. DispatchKey
// is a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[project.Key, project.Action]

// Dispatch applies the action sequence to the target project atomically. Subscribers to
// the project action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the projects that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, project.ResourceID)
}

// Retrieve returns the projects that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Projects))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, project.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type (
	// ExportRequest names the project to export.
	ExportRequest struct {
		// Key identifies the project to export.
		Key project.Key `json:"key" msgpack:"key"`
		// Encoding names the serialization member files are written in. "JSON" is the
		// only supported value.
		Encoding string `json:"encoding" msgpack:"encoding"`
	}
	// ExportResponse holds the bundle's contents keyed by path from the bundle root.
	// The HTTP transport encodes it as a zip archive.
	ExportResponse = zip.Files
)

var _ bind.Named = ExportRequest{}

// Named implements bind.Named.
func (r ExportRequest) Named() []resource.ID {
	return []resource.ID{project.ResourceID(r.Key)}
}

// Export exports the project and the records it owns as a bundle. It requires retrieve
// access on every document, panel, and group the bundle carries.
func (s *Service) Export(
	ctx context.Context,
	snap bind.Snapshot,
	req ExportRequest,
) (ExportResponse, error) {
	encoder, err := imex.ResolveEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}
	files, members, err := s.bundle.Export(ctx, snap, req.Key, encoder)
	if err != nil {
		return nil, err
	}
	if err = snap.Enforce(members...); err != nil {
		return nil, err
	}
	return files, nil
}

type (
	// ImportRequest holds a bundle's contents keyed by path from the bundle root. The
	// HTTP transport decodes it from a zip archive.
	ImportRequest = zip.Files
	// ImportResponse carries the project the import created.
	ImportResponse struct {
		// Project is the created project holding the imported resources.
		Project project.Project `json:"project" msgpack:"project"`
	}
)

// PrepareImport decodes the bundle in req. It returns a validation error for missing
// params or an invalid bundle.
func (s *Service) PrepareImport(
	ctx context.Context,
	req ImportRequest,
) (bundle.Bundle, error) {
	fileName, err := parseImportParams(ctx)
	if err != nil {
		return bundle.Bundle{}, err
	}
	return s.bundle.Decode(ctx, req, fileName)
}

// Import imports the bundle that PrepareImport decoded into a new project.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	_ ImportRequest,
	b bundle.Bundle,
) (ImportResponse, error) {
	p, err := s.bundle.Import(ctx, tx, b)
	if err != nil {
		return ImportResponse{}, err
	}
	return ImportResponse{Project: p}, nil
}

type importParams struct {
	// FileName is the name of the uploaded archive or picked directory, e.g.
	// "Test Stand 12.zip". Its extension-stripped form names the project when the
	// bundle carries no name.
	FileName string `json:"file_name"`
}

// parseImportParams decodes the required "params" request param — a JSON object
// carrying the out-of-band import options. A missing param, malformed JSON, or a
// missing file name returns a validation error scoped to the offending field.
func parseImportParams(ctx context.Context) (string, error) {
	v, ok := freighter.MDFromContext(ctx).Get("params")
	s, isStr := v.(string)
	if !ok || !isStr || s == "" {
		return "", validate.PathedError(validate.ErrRequired, "params")
	}
	var params importParams
	if err := json.Unmarshal([]byte(s), &params); err != nil {
		return "", validate.PathedError(
			errors.Wrapf(validate.ErrValidation, "invalid params: %v", err),
			"params",
		)
	}
	if params.FileName == "" {
		return "", validate.PathedError(validate.ErrRequired, "file_name")
	}
	return params.FileName, nil
}

type DeleteRequest struct {
	Keys []project.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the projects with the given keys, and the records each one holds.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
