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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Service struct {
	access *rbac.Service
	task   *task.Service
	status *status.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		task:   cfg.Service.Task,
		status: cfg.Service.Status,
		access: cfg.Service.RBAC,
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

type (
	RetrieveRequest struct {
		Internal            *bool      `json:"internal" msgpack:"internal"`
		Snapshot            *bool      `json:"snapshot" msgpack:"snapshot"`
		SearchTerm          string     `json:"search_term" msgpack:"search_term"`
		Keys                []task.Key `json:"keys" msgpack:"keys"`
		Names               []string   `json:"names" msgpack:"names"`
		Types               []string   `json:"types" msgpack:"types"`
		Limit               int        `json:"limit" msgpack:"limit"`
		Offset              int        `json:"offset" msgpack:"offset"`
		Rack                rack.Key   `json:"rack" msgpack:"rack"`
		IncludeStatus       bool       `json:"include_status" msgpack:"include_status"`
		IgnoreNotFoundError bool       `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
	}
	RetrieveResponse struct {
		Tasks []task.Task `json:"tasks,omitzero" msgpack:"tasks,omitzero"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		hasSearch = len(req.SearchTerm) > 0
		hasKeys   = len(req.Keys) > 0
		hasNames  = len(req.Names) > 0
		hasTypes  = len(req.Types) > 0
		hasLimit  = req.Limit > 0
		hasOffset = req.Offset > 0
	)
	q := s.task.NewRetrieve()
	if req.Internal != nil {
		q = q.Where(task.MatchInternal(*req.Internal))
	}
	if req.Snapshot != nil {
		q = q.Where(task.MatchSnapshot(*req.Snapshot))
	}
	if hasNames {
		q = q.Where(task.MatchNames(req.Names...))
	}
	if hasKeys {
		q = q.Where(task.MatchKeys(req.Keys...))
	}
	if hasTypes {
		q = q.Where(task.MatchTypes(req.Types...))
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
		q = q.Where(task.MatchRacks(req.Rack))
	}
	var res RetrieveResponse
	err := q.Entries(&res.Tasks).Exec(ctx, nil)
	if req.IgnoreNotFoundError && err != nil {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if err != nil {
		return RetrieveResponse{}, err
	}

	if req.IncludeStatus {
		statuses := make([]task.Status, 0, len(res.Tasks))
		if err := status.NewRetrieve[task.StatusDetails](s.status).
			Where(status.MatchKeys[task.StatusDetails](ontology.IDsToKeys(task.OntologyIDsFromTasks(res.Tasks))...)).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return RetrieveResponse{}, err
		}
		for i, stat := range statuses {
			res.Tasks[i].Status = &stat
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

type DeleteRequest struct {
	Keys []task.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: task.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	w := s.task.NewWriter(tx)
	for _, k := range req.Keys {
		if err := w.Delete(ctx, k, false); err != nil {
			return types.Nil{}, err
		}
	}
	return types.Nil{}, nil
}

type (
	CopyRequest struct {
		Name     string   `json:"name" msgpack:"name"`
		Key      task.Key `json:"key" msgpack:"key"`
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
		Objects: []ontology.ID{req.Key.OntologyID()},
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
