// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the line plot API service.
type ServiceConfig struct {
	// LinePlot stores and retrieves line plots.
	//
	// [REQUIRED]
	LinePlot *lineplot.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the line plot API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.LinePlot = override.Nil(c.LinePlot, other.LinePlot)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.lineplot")
	v.NotNil("line_plot", c.LinePlot)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *lineplot.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.LinePlot,
		access:   cfg.RBAC,
	}, nil
}

type CreateRequest struct {
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
	// DispatchKey is stamped onto the create broadcasts so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

type CreateResponse struct {
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
}

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	enforcer := s.access.NewEnforcer(tx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeLineplot}},
	}); err != nil {
		return CreateResponse{}, err
	}
	var stored []lineplot.LinePlot
	if err := s.internal.NewRetrieve().
		Where(lineplot.MatchKeys(lo.Map(
			req.LinePlots,
			func(e lineplot.LinePlot, _ int) lineplot.Key { return e.Key },
		)...)).
		Entries(&stored).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return CreateResponse{}, err
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.CreateObjects(
			req.LinePlots,
			stored,
			func(e lineplot.LinePlot) *group.Key { return e.Group },
		),
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.LinePlots); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{LinePlots: req.LinePlots}, nil
}

// DispatchRequest carries an action sequence to apply to a single line plot.
// DispatchKey identifies the originating client's batch so cluster broadcasts can be
// deduplicated against the local optimistic update.
type DispatchRequest = actions.DispatchRequest[lineplot.Key, lineplot.Action]

// Dispatch applies the action sequence to the target line plot atomically. Subscribers
// to the line plot action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{lineplot.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.
		Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

type (
	RetrieveRequest struct {
		Keys                []lineplot.Key `json:"keys"                   msgpack:"keys"`
		IgnoreNotFoundError bool           `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
		// Groups keeps only the line plots in any of these groups.
		Groups []group.Key `json:"groups" msgpack:"groups"`
		// Grouped keeps only the line plots in a group when true, and only the line
		// plots in no group when false. Nil applies no filter.
		Grouped *bool `json:"grouped" msgpack:"grouped"`
		// Projects keeps only the line plots owned by any of these projects.
		Projects []project.Key `json:"projects" msgpack:"projects"`
		// Ranges keeps only the line plots owned by any of these ranges.
		Ranges []ranger.Key `json:"ranges" msgpack:"ranges"`
	}
	RetrieveResponse struct {
		LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
	}
)

// Retrieve returns the line plots that match req. A request with no keys, groups, or
// grouped filter matches nothing.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	if len(req.Keys) == 0 && len(req.Groups) == 0 && req.Grouped == nil {
		return RetrieveResponse{LinePlots: []lineplot.LinePlot{}}, nil
	}
	var res RetrieveResponse
	q := s.internal.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.Where(lineplot.MatchKeys(req.Keys...))
	}
	if len(req.Groups) > 0 {
		q = q.Where(lineplot.MatchGroups(req.Groups...))
	}
	if req.Grouped != nil {
		q = q.Where(lineplot.MatchHasGroup(*req.Grouped))
	}
	if len(req.Projects) > 0 {
		q = q.Where(lineplot.MatchProjects(req.Projects...))
	}
	if len(req.Ranges) > 0 {
		q = q.Where(lineplot.MatchRanges(req.Ranges...))
	}
	err := q.Entries(&res.LinePlots).Exec(ctx, nil)
	if req.IgnoreNotFoundError && err != nil {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: lineplot.ResourceIDsFromLinePlots(res.LinePlots),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []lineplot.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: lineplot.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// SetGroupRequest moves line plots into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key     `json:"group" msgpack:"group"`
	Keys  []lineplot.Key `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the line plots with the given keys into a group.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	var plots []lineplot.LinePlot
	if err := s.internal.NewRetrieve().
		Where(lineplot.MatchKeys(req.Keys...)).
		Entries(&plots).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.MoveObjects(
			lineplot.ResourceIDs(req.Keys),
			lo.Map(
				plots,
				func(e lineplot.LinePlot, _ int) *group.Key { return e.Group },
			),
			req.Group,
		),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}

// SetProjectRequest moves line plots into Project.
type SetProjectRequest struct {
	Project project.Key    `json:"project" msgpack:"project"`
	Keys    []lineplot.Key `json:"keys"    msgpack:"keys"`
}

// SetProject moves the line plots with the given keys into a project.
func (s *Service) SetProject(
	ctx context.Context,
	tx gorp.Tx,
	req SetProjectRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: lineplot.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).
		SetProject(ctx, req.Project, req.Keys...)
}
