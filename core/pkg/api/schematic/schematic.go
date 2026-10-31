// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the schematic API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Schematic is the service that stores and modifies schematics.
	//
	// [REQUIRED]
	Schematic *schematic.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the schematic API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Schematic = override.Nil(c.Schematic, other.Schematic)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.schematic")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("schematic", c.Schematic)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *schematic.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Schematic, access: cfg.RBAC}, nil
}

type (
	CreateRequest struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
		Project    project.Key           `json:"project"    msgpack:"project"`
	}
	CreateResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeSchematic}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		CreateMany(ctx, req.Project, &req.Schematics); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Schematics: req.Schematics}, nil
}

// DispatchRequest carries an action sequence to apply to a single schematic.
// DispatchKey is a client-generated identifier for the batch, registered as outstanding
// on the originator before the request is sent. The server echoes it verbatim on the
// broadcast frame so the originator can recognize its own echo race-safely.
type DispatchRequest = actions.DispatchRequest[schematic.Key, schematic.Action]

// Dispatch applies the action sequence to the target schematic atomically. Subscribers
// to the schematic action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.
		Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Schematics))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: schematic.OntologyIDsFromSchematics(res.Schematics),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []schematic.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: schematic.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

type (
	CopyRequest struct {
		Name     string        `json:"name"     msgpack:"name"`
		Key      schematic.Key `json:"key"      msgpack:"key"`
		Snapshot bool          `json:"snapshot" msgpack:"snapshot"`
	}
	CopyResponse struct {
		Schematic schematic.Schematic `json:"schematic" msgpack:"schematic"`
	}
)

func (s *Service) Copy(
	ctx context.Context,
	tx gorp.Tx,
	req CopyRequest,
) (CopyResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return CopyResponse{}, err
	}
	var res CopyResponse
	if err := s.internal.NewWriter(tx).
		Copy(ctx, req.Key, req.Name, req.Snapshot, &res.Schematic); err != nil {
		return CopyResponse{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{schematic.OntologyID(res.Schematic.Key)},
	}); err != nil {
		return CopyResponse{}, err
	}
	return res, nil
}
