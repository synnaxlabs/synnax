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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the task API service.
type ServiceConfig struct {
	// DB runs the reads of a delete and of a dispatch response.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Task stores and retrieves tasks.
	//
	// [REQUIRED]
	Task *task.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC checks the access of delete requests, and the read access that a dispatch
	// response needs.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Stopper stops the running tasks of a delete.
	//
	// [REQUIRED]
	Stopper *task.Stopper
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the task API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Task = override.Nil(c.Task, other.Task)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Stopper = override.Nil(c.Stopper, other.Stopper)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.task")
	v.NotNil("db", c.DB)
	v.NotNil("task", c.Task)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("stopper", c.Stopper)
	return v.Error()
}

type Service struct {
	db      *gorp.DB
	access  *rbac.Service
	task    *task.Service
	status  *status.Service
	stopper *task.Stopper
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:      cfg.DB,
		task:    cfg.Task,
		status:  cfg.Status,
		access:  cfg.RBAC,
		stopper: cfg.Stopper,
	}, nil
}

type (
	Key  = task.Key
	Task = task.Task
)

type (
	CreateRequest struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
)

// Create creates the tasks. It returns an error wrapping validate.ErrValidation when a
// task with the key of one exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.task.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Tasks); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Tasks: req.Tasks}, nil
}

// DispatchRequest carries an action sequence to apply to a single task. DispatchKey is
// a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[task.Key, task.Action]

// DispatchResponse carries the actions the Core added to a dispatch: a set of the new
// config hash, after a set of the stored config when storing changed it.
type DispatchResponse = actions.DispatchResponse[task.Action]

// Dispatch applies the action sequence to the target task in tx. Subscribers to the
// task action signals receive the sequence after the transaction commits. The response
// holds no actions for a subject that cannot read the task, because they can carry its
// stored config.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (DispatchResponse, error) {
	computed, err := s.task.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
	if err != nil {
		return DispatchResponse{}, err
	}
	// Committed policies decide, as in bind.Write.
	if err = s.access.NewEnforcer(s.db).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []resource.ID{task.ResourceID(req.Key)},
	}); err != nil {
		return DispatchResponse{}, errors.Skip(err, access.ErrDenied)
	}
	return DispatchResponse{Actions: computed}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the tasks that the top level of Where selects
// by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, task.ResourceID)
}

// Retrieve returns the tasks that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.task.NewRetrieve().Entries(&res.Tasks))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, task.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeStatus {
		statuses, err := s.status.Join[task.StatusDetails](
			ctx,
			snap,
			task.ResourceIDsFromTasks(res.Tasks),
		)
		if err != nil {
			return RetrieveResponse{}, err
		}
		for i, st := range statuses {
			res.Tasks[i].Status = st
		}
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

// PrepareDelete stops the running tasks in req and waits for them to stop, for at most
// the timeout of the Stopper in total. A task that does not stop in time can hold its
// channels open until its Driver removes it. PrepareDelete returns access.ErrDenied,
// and stops nothing, when the subject cannot delete a task in req.
func (s *Service) PrepareDelete(
	ctx context.Context,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.access.NewEnforcer(snap).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionDelete,
			Objects: task.ResourceIDs(req.Keys),
		})
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.stopper.Stop(ctx, req.Keys)
}

// Delete deletes the tasks in req. Run PrepareDelete first to stop them.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
	_ struct{},
) (struct{}, error) {
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
		Name string   `json:"name" msgpack:"name"`
		Key  task.Key `json:"key"  msgpack:"key"`
	}
	CopyResponse struct {
		Task task.Task `json:"task" msgpack:"task"`
	}
)

// Sources implements bind.Copy.
func (r CopyRequest) Sources() []resource.ID {
	return []resource.ID{task.ResourceID(r.Key)}
}

// Copy copies a task.
func (s *Service) Copy(
	ctx context.Context,
	tx gorp.Tx,
	req CopyRequest,
) (CopyResponse, error) {
	t, err := s.task.NewWriter(tx).Copy(ctx, req.Key, req.Name)
	if err != nil {
		return CopyResponse{}, err
	}
	return CopyResponse{Task: t}, nil
}
