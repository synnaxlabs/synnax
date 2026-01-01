// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
)

type TaskService struct {
	dbProvider
	accessProvider
	task   *task.Service
	status statusProvider
}

func NewTaskService(p Provider) *TaskService {
	return &TaskService{
		dbProvider:     p.db,
		task:           p.Service.Task,
		status:         p.status,
		accessProvider: p.access,
	}
}

type (
	Task = task.Task
)

type (
	TaskCreateRequest struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
	TaskCreateResponse struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
)

func (svc *TaskService) Create(
	ctx context.Context,
	req TaskCreateRequest,
) (TaskCreateResponse, error) {
	var res TaskCreateResponse
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: task.OntologyIDsFromTasks(req.Tasks),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.task.NewWriter(tx)
		for i, m := range req.Tasks {
			if err := w.Create(ctx, &m); err != nil {
				return err
			}
			req.Tasks[i] = m
		}
		res.Tasks = req.Tasks
		return nil
	})
}

type (
	TaskRetrieveRequest struct {
		Rack          rack.Key   `json:"rack" msgpack:"rack"`
		Keys          []task.Key `json:"keys" msgpack:"keys"`
		Names         []string   `json:"names" msgpack:"names"`
		Types         []string   `json:"types" msgpack:"types"`
		IncludeStatus bool       `json:"include_status" msgpack:"include_status"`
		Internal      *bool      `json:"internal" msgpack:"internal"`
		Snapshot      *bool      `json:"snapshot" msgpack:"snapshot"`
		SearchTerm    string     `json:"search_term" msgpack:"search_term"`
		Limit         int        `json:"limit" msgpack:"limit"`
		Offset        int        `json:"offset" msgpack:"offset"`
	}
	TaskRetrieveResponse struct {
		Tasks []task.Task `json:"tasks" msgpack:"tasks"`
	}
)

func (svc *TaskService) Retrieve(
	ctx context.Context,
	req TaskRetrieveRequest,
) (TaskRetrieveResponse, error) {
	var (
		res       TaskRetrieveResponse
		hasSearch = len(req.SearchTerm) > 0
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasTypes  = len(req.Types) > 0
		hasLimit  = req.Limit > 0
		hasOffset = req.Offset > 0
	)
	q := svc.task.NewRetrieve()
	if req.Internal != nil {
		q = q.WhereInternal(*req.Internal, gorp.Required())
	}
	if req.Snapshot != nil {
		q = q.WhereSnapshot(*req.Snapshot, gorp.Required())
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasTypes {
		q = q.WhereTypes(req.Types...)
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if !req.Rack.IsZero() {
		q = q.WhereRacks(req.Rack)
	}
	err := q.Entries(&res.Tasks).Exec(ctx, nil)
	if err != nil {
		return res, err
	}

	if req.IncludeStatus {
		statuses := make([]task.Status, 0, len(res.Tasks))
		if err = status.NewRetrieve[task.StatusDetails](svc.status.status).
			WhereKeys(ontology.IDsToString(task.OntologyIDsFromTasks(res.Tasks))...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return res, err
		}
		for i, stat := range statuses {
			res.Tasks[i].Status = &stat
		}
	}
	if err = svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: task.OntologyIDsFromTasks(res.Tasks),
	}); err != nil {
		return TaskRetrieveResponse{}, err
	}
	return res, nil
}

type TaskDeleteRequest struct {
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

func (svc *TaskService) Delete(
	ctx context.Context,
	req TaskDeleteRequest,
) (types.Nil, error) {
	var res types.Nil
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: task.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.task.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k, false); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	TaskCopyRequest struct {
		Key      task.Key `json:"key" msgpack:"key"`
		Name     string   `json:"name" msgpack:"name"`
		Snapshot bool     `json:"snapshot" msgpack:"snapshot"`
	}
	TaskCopyResponse struct {
		Task task.Task `json:"task" msgpack:"task"`
	}
)

func (svc *TaskService) Copy(
	ctx context.Context,
	req TaskCopyRequest,
) (TaskCopyResponse, error) {
	var res TaskCopyResponse
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{task.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	err := svc.WithTx(ctx, func(tx gorp.Tx) (err error) {
		res.Task, err = svc.task.NewWriter(tx).Copy(
			ctx,
			req.Key,
			req.Name,
			req.Snapshot,
		)
		return err
	})
	if err != nil {
		return TaskCopyResponse{}, err
	}
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{task.OntologyID(res.Task.Key)},
	}); err != nil {
		return TaskCopyResponse{}, err
	}
	return res, nil
}
