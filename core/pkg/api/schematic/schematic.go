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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
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
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
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
	enforcer := s.access.NewEnforcer(tx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeSchematic}},
	}); err != nil {
		return CreateResponse{}, err
	}
	var stored []schematic.Schematic
	if err := s.internal.NewRetrieve().
		Where(schematic.MatchKeys(lo.Map(
			req.Schematics,
			func(e schematic.Schematic, _ int) schematic.Key { return e.Key },
		)...)).
		Entries(&stored).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return CreateResponse{}, err
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.CreateObjects(
			req.Schematics,
			stored,
			func(e schematic.Schematic) *group.Key { return e.Group },
		),
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Schematics); err != nil {
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
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{schematic.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.
		Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

type (
	RetrieveRequest struct {
		Keys                []schematic.Key `json:"keys"                   msgpack:"keys"`
		IgnoreNotFoundError bool            `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
		// Groups keeps only the schematics in any of these groups.
		Groups []group.Key `json:"groups" msgpack:"groups"`
		// Grouped keeps only the schematics in a group when true, and only the
		// schematics in no group when false. Nil applies no filter.
		Grouped *bool `json:"grouped" msgpack:"grouped"`
		// Projects keeps only the schematics owned by any of these projects.
		Projects []project.Key `json:"projects" msgpack:"projects"`
		// Ranges keeps only the schematics owned by any of these ranges.
		Ranges []ranger.Key `json:"ranges" msgpack:"ranges"`
	}
	RetrieveResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

// Retrieve returns the schematics that match req. A request with no keys, groups, or
// grouped filter matches nothing.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	if len(req.Keys) == 0 && len(req.Groups) == 0 && req.Grouped == nil {
		return RetrieveResponse{Schematics: []schematic.Schematic{}}, nil
	}
	var res RetrieveResponse
	q := s.internal.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.Where(schematic.MatchKeys(req.Keys...))
	}
	if len(req.Groups) > 0 {
		q = q.Where(schematic.MatchGroups(req.Groups...))
	}
	if req.Grouped != nil {
		q = q.Where(schematic.MatchHasGroup(*req.Grouped))
	}
	if len(req.Projects) > 0 {
		q = q.Where(schematic.MatchProjects(req.Projects...))
	}
	if len(req.Ranges) > 0 {
		q = q.Where(schematic.MatchRanges(req.Ranges...))
	}
	err := q.Entries(&res.Schematics).Exec(ctx, nil)
	if req.IgnoreNotFoundError && err != nil {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: schematic.ResourceIDsFromSchematics(res.Schematics),
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
		Objects: schematic.ResourceIDs(req.Keys),
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
		// Range, when set, makes the copy belong to the range instead of the
		// original's project.
		Range *ranger.Key `json:"range" msgpack:"range"`
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
		Objects: []resource.ID{schematic.ResourceID(req.Key)},
	}); err != nil {
		return CopyResponse{}, err
	}
	var res CopyResponse
	if err := s.internal.NewWriter(tx).
		Copy(ctx, req.Key, req.Name, req.Snapshot, req.Range, &res.Schematic); err != nil {
		return CopyResponse{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []resource.ID{schematic.ResourceID(res.Schematic.Key)},
	}); err != nil {
		return CopyResponse{}, err
	}
	return res, nil
}

// SetGroupRequest moves schematics into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key      `json:"group" msgpack:"group"`
	Keys  []schematic.Key `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the schematics with the given keys into a group.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	var schematics []schematic.Schematic
	if err := s.internal.NewRetrieve().
		Where(schematic.MatchKeys(req.Keys...)).
		Entries(&schematics).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.MoveObjects(
			schematic.ResourceIDs(req.Keys),
			lo.Map(
				schematics,
				func(e schematic.Schematic, _ int) *group.Key { return e.Group },
			),
			req.Group,
		),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}

// SetProjectRequest moves schematics into Project.
type SetProjectRequest struct {
	Project project.Key     `json:"project" msgpack:"project"`
	Keys    []schematic.Key `json:"keys"    msgpack:"keys"`
}

// SetProject moves the schematics with the given keys into a project.
func (s *Service) SetProject(
	ctx context.Context,
	tx gorp.Tx,
	req SetProjectRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: schematic.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).
		SetProject(ctx, req.Project, req.Keys...)
}
