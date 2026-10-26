// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the task API service.
type ServiceConfig struct {
	// Task stores and retrieves tasks.
	//
	// [REQUIRED]
	Task *task.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the task API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Task = override.Nil(c.Task, other.Task)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.task")
	v.NotNil("task", c.Task)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access *rbac.Service
	task   *task.Service
	status *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		task:   cfg.Task,
		status: cfg.Status,
		access: cfg.RBAC,
	}, nil
}

type (
	Key  = task.Key
	Task = task.Task
)

type (
	CreateRequest struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
	CreateResponse struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
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
		Objects: []ontology.ID{{Type: ontology.ResourceTypeTask}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.task.NewWriter(tx).CreateMany(ctx, &req.Tasks); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.task.NewRetrieve().Entries(&res.Tasks))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeStatus {
		statuses, err := s.status.Join[task.StatusDetails](
			ctx,
			task.OntologyIDsFromTasks(res.Tasks),
		)
		if err != nil {
			return RetrieveResponse{}, err
		}
		for i, st := range statuses {
			res.Tasks[i].Status = st
		}
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: task.OntologyIDsFromTasks(res.Tasks),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// RenameRequest names the task to rename and the name to give it.
type RenameRequest struct {
	// Name is the new name. It must not be empty.
	Name string `json:"name" msgpack:"name"`
	// Key selects the task to rename.
	Key task.Key `json:"key" msgpack:"key"`
}

// Rename renames the task. It requires update access on it.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{task.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.task.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: task.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	w := s.task.NewWriter(tx)
	for _, k := range req.Keys {
		if err := w.Delete(ctx, k, false); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

type (
	CopyRequest struct {
		Name     string   `json:"name"     msgpack:"name"`
		Key      task.Key `json:"key"      msgpack:"key"`
		Snapshot bool     `json:"snapshot" msgpack:"snapshot"`
	}
	CopyResponse struct {
		Task task.Task `json:"task" msgpack:"task"`
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
		Objects: []ontology.ID{task.OntologyID(req.Key)},
	}); err != nil {
		return CopyResponse{}, err
	}
	t, err := s.task.NewWriter(tx).Copy(ctx, req.Key, req.Name, req.Snapshot)
	if err != nil {
		return CopyResponse{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{t.OntologyID()},
	}); err != nil {
		return CopyResponse{}, err
	}
	return CopyResponse{Task: t}, nil
}
