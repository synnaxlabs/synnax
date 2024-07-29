// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/workspace/lineplot"
	"github.com/synnaxlabs/x/gorp"
	"go/types"
)

type LinePlotService struct {
	dbProvider
	accessProvider
	internal *lineplot.Service
}

func NewLinePlotService(p Provider) *LinePlotService {
	return &LinePlotService{
		dbProvider:     p.db,
		internal:       p.Config.LinePlot,
		accessProvider: p.access,
	}
}

type LinePlotCreateRequest struct {
	Workspace uuid.UUID           `json:"workspace" msgpack:"workspace"`
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
}

type LinePlotCreateResponse struct {
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
}

func (s *LinePlotService) Create(ctx context.Context, req LinePlotCreateRequest) (res LinePlotCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: []ontology.ID{lineplot.OntologyID(uuid.Nil)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		for _, lp := range req.LinePlots {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Workspace, &lp); err != nil {
				return err
			}
			res.LinePlots = append(res.LinePlots, lp)
		}
		return err
	})
}

type LinePlotRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *LinePlotService) Rename(ctx context.Context, req LinePlotRenameRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Update,
		Objects: []ontology.ID{lineplot.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type LinePlotSetDataRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Data string    `json:"data" msgpack:"data"`
}

func (s *LinePlotService) SetData(ctx context.Context, req LinePlotSetDataRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: []ontology.ID{lineplot.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
	})
}

type (
	LinePlotRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	LinePlotRetrieveResponse struct {
		LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
	}
)

func (s *LinePlotService) Retrieve(ctx context.Context, req LinePlotRetrieveRequest) (res LinePlotRetrieveResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: lineplot.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	err = s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.LinePlots).Exec(ctx, nil)
	return res, err
}

type LinePlotDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *LinePlotService) Delete(ctx context.Context, req LinePlotDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: lineplot.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
