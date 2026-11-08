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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the log API service.
type ServiceConfig struct {
	// Log stores and retrieves log visualizations.
	//
	// [REQUIRED]
	Log *log.Service
	// RBAC checks the access of retrieve requests.
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
		Logs []log.Log `json:"logs" msgpack:"logs"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Logs []log.Log `json:"logs" msgpack:"logs"`
	}
)

// Create creates the logs, and overwrites each one whose key exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Logs); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Logs: req.Logs}, nil
}

// DispatchRequest carries an action sequence to apply to a single log. DispatchKey
// identifies the originating client's batch so cluster broadcasts can be deduplicated
// against the local optimistic update.
type DispatchRequest = actions.DispatchRequest[log.Key, log.Action]

// Dispatch applies the action sequence to the target log in tx. Subscribers to the
// log action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.
		Dispatch(ctx, tx, req.Key, req.DispatchKey, req.Actions)
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

// Delete deletes the logs with the given keys. A key that matches no log is
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// SetGroupRequest moves logs into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key `json:"group" msgpack:"group"`
	Keys  []log.Key  `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the logs with the given keys into a group. A move into a group under
// another project moves the logs into that project.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}

// SetProjectRequest moves logs into Project.
type SetProjectRequest struct {
	Project project.Key `json:"project" msgpack:"project"`
	Keys    []log.Key   `json:"keys"    msgpack:"keys"`
}

// SetProject moves the logs with the given keys out of their group and into a
// project.
func (s *Service) SetProject(
	ctx context.Context,
	tx gorp.Tx,
	req SetProjectRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		SetProject(ctx, req.Project, req.Keys...)
}
