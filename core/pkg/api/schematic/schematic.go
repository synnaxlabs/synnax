// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *schematic.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
		internal: cfg.Service.Schematic,
	}
}

type (
	CreateRequest struct {
		Workspace  uuid.UUID             `json:"workspace" msgpack:"workspace"`
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
	CreateResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (res CreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: schematic.OntologyIDsFromSchematics(req.Schematics),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		for i, schematic_ := range req.Schematics {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Workspace, &schematic_); err != nil {
				return err
			}
			req.Schematics[i] = schematic_
		}
		res.Schematics = req.Schematics
		return nil
	})
}

type RenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *Service) Rename(ctx context.Context, req RenameRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type SetDataRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Data string    `json:"data" msgpack:"data"`
}

func (s *Service) SetData(ctx context.Context, req SetDataRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
	})
}

type (
	RetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	RetrieveResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (res RetrieveResponse, err error) {
	err = s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.Schematics).Exec(ctx, nil)
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: schematic.OntologyIDs(req.Keys),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, err
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: schematic.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type (
	CopyRequest struct {
		Key      uuid.UUID `json:"key" msgpack:"key"`
		Name     string    `json:"name" msgpack:"name"`
		Snapshot bool      `json:"snapshot" msgpack:"snapshot"`
	}
	CopyResponse struct {
		Schematic schematic.Schematic `json:"schematic" msgpack:"schematic"`
	}
)

func (s *Service) Copy(ctx context.Context, req CopyRequest) (res CopyResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Copy(ctx, req.Key, req.Name, req.Snapshot, &res.Schematic)
	}); err != nil {
		return CopyResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{schematic.OntologyID(res.Schematic.Key)},
	}); err != nil {
		return CopyResponse{}, err
	}
	return res, err
}

type (
	CreateSymbolRequest struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
		Parent  ontology.ID     `json:"parent" msgpack:"parent"`
	}
	CreateSymbolResponse struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

func (s *Service) CreateSymbol(ctx context.Context, req CreateSymbolRequest) (res CreateSymbolResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: symbol.OntologyIDsFromSymbols(req.Symbols),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		writer := s.internal.Symbol.NewWriter(tx)
		for i, sym := range req.Symbols {
			if err = writer.Create(ctx, &sym, req.Parent); err != nil {
				return err
			}
			req.Symbols[i] = sym
		}
		res.Symbols = req.Symbols
		return nil
	})
}

type (
	RetrieveSymbolRequest struct {
		Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
		SearchTerm string      `json:"search_term" msgpack:"search_term"`
	}
	RetrieveSymbolResponse struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

func (s *Service) RetrieveSymbol(ctx context.Context, req RetrieveSymbolRequest) (res RetrieveSymbolResponse, err error) {
	q := s.internal.Symbol.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	err = q.Entries(&res.Symbols).Exec(ctx, nil)
	if err != nil {
		return RetrieveSymbolResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: symbol.OntologyIDsFromSymbols(res.Symbols),
	}); err != nil {
		return RetrieveSymbolResponse{}, err
	}
	return res, err
}

type RenameSymbolRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *Service) RenameSymbol(ctx context.Context, req RenameSymbolRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{symbol.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.Symbol.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type DeleteSymbolRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) DeleteSymbol(ctx context.Context, req DeleteSymbolRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: symbol.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.Symbol.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type RetrieveSymbolGroupRequest struct{}

type RetrieveSymbolGroupResponse struct {
	Group group.Group `json:"group" msgpack:"group"`
}

func (s *Service) RetrieveSymbolGroup(
	ctx context.Context,
	_ RetrieveSymbolGroupRequest,
) (RetrieveSymbolGroupResponse, error) {
	g := s.internal.Symbol.Group()
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{g.OntologyID()},
	}); err != nil {
		return RetrieveSymbolGroupResponse{}, err
	}
	return RetrieveSymbolGroupResponse{Group: g}, nil
}
