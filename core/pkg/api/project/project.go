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
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the project API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Project stores and retrieves projects.
	//
	// [REQUIRED]
	Project *project.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the project API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Project = override.Nil(c.Project, other.Project)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.project")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("project", c.Project)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *project.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.RBAC,
		internal: cfg.Project,
	}, nil
}

type (
	CreateRequest struct {
		Projects []project.Project `json:"projects" msgpack:"projects"`
	}
	CreateResponse = CreateRequest
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeProject}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Projects); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

type RenameRequest struct {
	Name string      `json:"name" msgpack:"name"`
	Key  project.Key `json:"key"  msgpack:"key"`
}

func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{project.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Projects))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: project.OntologyIDsFromProjects(res.Projects),
	}); err != nil {
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

// Export exports the project and its ontology descendants as a bundle. It requires
// retrieve access on the project, which it enforces before it reads a member, and on
// every document, panel, and group the bundle carries.
func (s *Service) Export(
	ctx context.Context,
	req ExportRequest,
) (ExportResponse, error) {
	encoder, err := imex.ResolveEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}
	var (
		enforcer = s.access.NewEnforcer(nil)
		subject  = auth.GetSubject(ctx)
	)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{project.OntologyID(req.Key)},
	}); err != nil {
		return nil, err
	}
	files, members, err := s.internal.Export(ctx, req.Key, encoder)
	if err != nil {
		return nil, err
	}
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionRetrieve,
		Objects: members,
	}); err != nil {
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

// Import imports a project bundle in a single transaction. It requires create access on
// the project type and on every resource kind the bundle carries, all enforced before
// any import work runs.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	req ImportRequest,
) (ImportResponse, error) {
	fileName, err := parseImportParams(ctx)
	if err != nil {
		return ImportResponse{}, err
	}
	objects, err := s.internal.ImportObjects(ctx, req)
	if err != nil {
		return ImportResponse{}, err
	}
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: objects,
	}); err != nil {
		return ImportResponse{}, err
	}
	p, err := s.internal.Import(ctx, tx, req, fileName)
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

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: project.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
