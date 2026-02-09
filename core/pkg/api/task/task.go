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
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	svctask "github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db     *gorp.DB
	access *rbac.Service
	task   *svctask.Service
	status *svcstatus.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		db:     cfg.Distribution.DB,
		task:   cfg.Service.Task,
		status: cfg.Service.Status,
		access: cfg.Service.RBAC,
	}
}

type (
	Task = svctask.Task
)

type (
	CreateRequest struct {
		Tasks []svctask.Task `json:"tasks" msgpack:"tasks"`
	}
	CreateResponse struct {
		Tasks []svctask.Task `json:"tasks" msgpack:"tasks"`
	}
)

func (svc *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	var res CreateResponse
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: svctask.OntologyIDsFromTasks(req.Tasks),
	}); err != nil {
		return res, err
	}
	return res, svc.db.WithTx(ctx, func(tx gorp.Tx) error {
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
	RetrieveRequest struct {
		Internal      *bool         `json:"internal" msgpack:"internal"`
		Snapshot      *bool         `json:"snapshot" msgpack:"snapshot"`
		SearchTerm    string        `json:"search_term" msgpack:"search_term"`
		Keys          []svctask.Key `json:"keys" msgpack:"keys"`
		Names         []string      `json:"names" msgpack:"names"`
		Types         []string      `json:"types" msgpack:"types"`
		Limit         int           `json:"limit" msgpack:"limit"`
		Offset        int           `json:"offset" msgpack:"offset"`
		Rack          rack.Key      `json:"rack" msgpack:"rack"`
		IncludeStatus bool          `json:"include_status" msgpack:"include_status"`
	}
	RetrieveResponse struct {
		Tasks []svctask.Task `json:"tasks" msgpack:"tasks"`
	}
)

func (svc *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		res       RetrieveResponse
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
		statuses := make([]svctask.Status, 0, len(res.Tasks))
		if err = svcstatus.NewRetrieve[svctask.StatusDetails](svc.status).
			WhereKeys(ontology.IDsToKeys(svctask.OntologyIDsFromTasks(res.Tasks))...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return res, err
		}
		for i, stat := range statuses {
			res.Tasks[i].Status = &stat
		}
	}
	if err = svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: svctask.OntologyIDsFromTasks(res.Tasks),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []svctask.Key `json:"keys" msgpack:"keys"`
}

func (svc *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	var res types.Nil
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: svctask.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, svc.db.WithTx(ctx, func(tx gorp.Tx) error {
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
	CopyRequest struct {
		Name     string      `json:"name" msgpack:"name"`
		Key      svctask.Key `json:"key" msgpack:"key"`
		Snapshot bool        `json:"snapshot" msgpack:"snapshot"`
	}
	CopyResponse struct {
		Task svctask.Task `json:"task" msgpack:"task"`
	}
)

func (svc *Service) Copy(
	ctx context.Context,
	req CopyRequest,
) (CopyResponse, error) {
	var res CopyResponse
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{svctask.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	err := svc.db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		res.Task, err = svc.task.NewWriter(tx).Copy(
			ctx,
			req.Key,
			req.Name,
			req.Snapshot,
		)
		return err
	})
	if err != nil {
		return CopyResponse{}, err
	}
	if err := svc.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{svctask.OntologyID(res.Task.Key)},
	}); err != nil {
		return CopyResponse{}, err
	}
	return res, nil
}
