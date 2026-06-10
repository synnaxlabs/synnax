// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *table.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		internal: cfg.Service.Table,
		access:   cfg.Service.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Tables  []table.Table `json:"tables" msgpack:"tables"`
		Project project.Key   `json:"project" msgpack:"project"`
	}
	CreateResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: table.OntologyIDsFromTables(req.Tables),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for i, t := range req.Tables {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Project, &t); err != nil {
				return err
			}
			req.Tables[i] = t
		}
		res.Tables = req.Tables
		return nil
	})
}

// DispatchRequest carries an action sequence to apply to a single table.
// DispatchKey is a client-generated identifier for the batch, registered as
// outstanding on the originator before the request is sent. The server echoes
// it verbatim on the broadcast frame so the originator can recognize its own
// echo race-safely.
type DispatchRequest = actions.DispatchRequest[table.Key, table.Action]

// Dispatch applies the action sequence to the target table atomically.
// Subscribers to the table action signals receive the sequence after the
// transaction commits.
func (s *Service) Dispatch(ctx context.Context, req DispatchRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{table.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
	})
}

type (
	RetrieveRequest struct {
		Keys []table.Key `json:"keys" msgpack:"keys"`
	}
	RetrieveResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, err error) {
	err = s.internal.NewRetrieve().
		Where(table.MatchKeys(req.Keys...)).Entries(&res.Tables).Exec(ctx, nil)
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: table.OntologyIDs(req.Keys),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, err
}

type DeleteRequest struct {
	Keys []table.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: table.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
