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
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *log.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		internal: cfg.Service.Log,
		access:   cfg.Service.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Logs    []log.Log   `json:"logs" msgpack:"logs"`
		Project project.Key `json:"project" msgpack:"project"`
	}
	CreateResponse struct {
		Logs []log.Log `json:"logs" msgpack:"logs"`
	}
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: log.OntologyIDsFromLogs(req.Logs),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for i, l := range req.Logs {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Project, &l); err != nil {
				return err
			}
			req.Logs[i] = l
		}
		res.Logs = req.Logs
		return nil
	})
}

// DispatchRequest carries an action sequence to apply to a single log.
// DispatchKey identifies the originating client's batch so cluster broadcasts
// can be deduplicated against the local optimistic update.
type DispatchRequest = actions.DispatchRequest[log.Key, log.Action]

// Dispatch applies the action sequence to the target log atomically. Subscribers
// to the log action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(ctx context.Context, req DispatchRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{log.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
	})
}

type (
	RetrieveRequest struct {
		Keys []log.Key `json:"keys" msgpack:"keys"`
	}
	RetrieveResponse struct {
		Logs []log.Log `json:"logs" msgpack:"logs"`
	}
)

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, err error) {
	err = s.internal.NewRetrieve().
		Where(log.MatchKeys(req.Keys...)).Entries(&res.Logs).Exec(ctx, nil)
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: log.OntologyIDs(req.Keys),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, err
}

type DeleteRequest struct {
	Keys []log.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: log.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
