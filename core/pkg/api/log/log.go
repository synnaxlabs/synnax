// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the log API service.
type ServiceConfig struct {
	// Log stores and retrieves log visualizations.
	//
	// [REQUIRED]
	Log *log.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the log API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Log = override.Nil(c.Log, other.Log)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.log")
	v.NotNil("log", c.Log)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *log.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Log,
		access:   cfg.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Logs    []log.Log   `json:"logs"    msgpack:"logs"`
		Project project.Key `json:"project" msgpack:"project"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Logs []log.Log `json:"logs" msgpack:"logs"`
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
		Objects: []resource.ID{{Type: resource.TypeLog}},
	}); err != nil {
		return CreateResponse{}, err
	}
	var stored []log.Log
	if err := s.internal.NewRetrieve().
		Where(log.MatchKeys(lo.Map(
			req.Logs,
			func(e log.Log, _ int) log.Key { return e.Key },
		)...)).
		Entries(&stored).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return CreateResponse{}, err
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.CreateObjects(
			req.Logs,
			stored,
			func(e log.Log) *group.Key { return e.Group },
		),
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, req.Project, &req.Logs); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Logs: req.Logs}, nil
}

// DispatchRequest carries an action sequence to apply to a single log. DispatchKey
// identifies the originating client's batch so cluster broadcasts can be deduplicated
// against the local optimistic update.
type DispatchRequest = actions.DispatchRequest[log.Key, log.Action]

// Dispatch applies the action sequence to the target log atomically. Subscribers to the
// log action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{log.ResourceID(req.Key)},
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
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Logs))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: log.ResourceIDsFromLogs(res.Logs),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []log.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: log.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// SetGroupRequest moves logs into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key `json:"group" msgpack:"group"`
	Keys  []log.Key  `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the logs with the given keys into a group.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	var logs []log.Log
	if err := s.internal.NewRetrieve().
		Where(log.MatchKeys(req.Keys...)).
		Entries(&logs).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.MoveObjects(
			log.ResourceIDs(req.Keys),
			lo.Map(logs, func(e log.Log, _ int) *group.Key { return e.Group }),
			req.Group,
		),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}
