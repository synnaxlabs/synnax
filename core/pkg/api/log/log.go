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
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
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
		Logs      []log.Log     `json:"logs" msgpack:"logs"`
		Workspace workspace.Key `json:"workspace" msgpack:"workspace"`
	}
	CreateResponse struct {
		Logs []log.Log `json:"logs" msgpack:"logs"`
	}
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	var res CreateResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionCreate,
			Objects: []ontology.ID{{Type: ontology.ResourceTypeLog}},
		}); err != nil {
			return err
		}
		for i, l := range req.Logs {
			if err := s.internal.NewWriter(tx).Create(ctx, req.Workspace, &l); err != nil {
				return err
			}
			req.Logs[i] = l
		}
		res.Logs = req.Logs
		return nil
	}); err != nil {
		return CreateResponse{}, err
	}
	return res, nil
}

type RenameRequest struct {
	Name string  `json:"name" msgpack:"name"`
	Key  log.Key `json:"key" msgpack:"key"`
}

func (s *Service) Rename(ctx context.Context, req RenameRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []ontology.ID{log.OntologyID(req.Key)},
		}); err != nil {
			return err
		}
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type SetDataRequest struct {
	Data log.Log `json:"data" msgpack:"data"`
	Key  log.Key `json:"key" msgpack:"key"`
}

func (s *Service) SetData(ctx context.Context, req SetDataRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []ontology.ID{log.OntologyID(req.Key)},
		}); err != nil {
			return err
		}
		return s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
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

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (RetrieveResponse, error) {
	var res RetrieveResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewRetrieve().
			Where(log.MatchKeys(req.Keys...)).Entries(&res.Logs).Exec(ctx, tx); err != nil {
			return err
		}
		return s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionRetrieve,
			Objects: log.OntologyIDsFromLogs(res.Logs),
		})
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []log.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionDelete,
			Objects: log.OntologyIDs(req.Keys),
		}); err != nil {
			return err
		}
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
