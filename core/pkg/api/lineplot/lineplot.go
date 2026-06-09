// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *lineplot.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		internal: cfg.Service.LinePlot,
		access:   cfg.Service.RBAC,
	}, nil
}

type CreateRequest struct {
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
	Workspace workspace.Key       `json:"workspace" msgpack:"workspace"`
}

type CreateResponse struct {
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: lineplot.OntologyIDsFromLinePlots(req.LinePlots),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for i, lp := range req.LinePlots {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Workspace, &lp); err != nil {
				return err
			}
			req.LinePlots[i] = lp
		}
		res.LinePlots = req.LinePlots
		return nil
	})
}

// DispatchRequest carries an action sequence to apply to a single line plot.
// DispatchKey identifies the originating client's batch so cluster broadcasts
// can be deduplicated against the local optimistic update.
type DispatchRequest = actions.DispatchRequest[lineplot.Key, lineplot.Action]

// Dispatch applies the action sequence to the target line plot atomically.
// Subscribers to the line plot action signals receive the sequence after the
// transaction commits.
func (s *Service) Dispatch(ctx context.Context, req DispatchRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{lineplot.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
	})
}

type (
	RetrieveRequest struct {
		Keys []lineplot.Key `json:"keys" msgpack:"keys"`
	}
	RetrieveResponse struct {
		LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
	}
)

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: lineplot.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	if err = s.internal.NewRetrieve().Where(lineplot.MatchKeys(req.Keys...)).Entries(&res.LinePlots).
		Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []lineplot.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: lineplot.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
