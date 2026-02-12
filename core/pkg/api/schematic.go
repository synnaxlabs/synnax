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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/gorp"
)

type SchematicService struct {
	dbProvider
	accessProvider
	internal *schematic.Service
}

func NewSchematicService(p Provider) *SchematicService {
	return &SchematicService{
		dbProvider:     p.db,
		internal:       p.Service.Schematic,
		accessProvider: p.access,
	}
}

type (
	SchematicCreateRequest struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
		Workspace  uuid.UUID             `json:"workspace" msgpack:"workspace"`
	}
	SchematicCreateResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

func (s *SchematicService) Create(ctx context.Context, req SchematicCreateRequest) (res SchematicCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: schematic.OntologyIDsFromSchematics(req.Schematics),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		for i, schematic := range req.Schematics {
			if err = s.internal.NewWriter(tx).Create(ctx, req.Workspace, &schematic); err != nil {
				return err
			}
			req.Schematics[i] = schematic
		}
		res.Schematics = req.Schematics
		return nil
	})
}

type SchematicRenameRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  uuid.UUID `json:"key" msgpack:"key"`
}

func (s *SchematicService) Rename(ctx context.Context, req SchematicRenameRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type SchematicSetDataRequest struct {
	Data string    `json:"data" msgpack:"data"`
	Key  uuid.UUID `json:"key" msgpack:"key"`
}

func (s *SchematicService) SetData(ctx context.Context, req SchematicSetDataRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
	})
}

type (
	SchematicRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	SchematicRetrieveResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

func (s *SchematicService) Retrieve(ctx context.Context, req SchematicRetrieveRequest) (res SchematicRetrieveResponse, err error) {
	err = s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.Schematics).Exec(ctx, nil)
	if err != nil {
		return SchematicRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: schematic.OntologyIDs(req.Keys),
	}); err != nil {
		return SchematicRetrieveResponse{}, err
	}
	return res, err
}

type SchematicDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *SchematicService) Delete(ctx context.Context, req SchematicDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: schematic.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type (
	SchematicCopyRequest struct {
		Name     string    `json:"name" msgpack:"name"`
		Key      uuid.UUID `json:"key" msgpack:"key"`
		Snapshot bool      `json:"snapshot" msgpack:"snapshot"`
	}
	SchematicCopyResponse struct {
		Schematic schematic.Schematic `json:"schematic" msgpack:"schematic"`
	}
)

func (s *SchematicService) Copy(ctx context.Context, req SchematicCopyRequest) (res SchematicCopyResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{schematic.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	if err := s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Copy(ctx, req.Key, req.Name, req.Snapshot, &res.Schematic)
	}); err != nil {
		return SchematicCopyResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{schematic.OntologyID(res.Schematic.Key)},
	}); err != nil {
		return SchematicCopyResponse{}, err
	}
	return res, err
}

type (
	SchematicCreateSymbolRequest struct {
		Parent  ontology.ID     `json:"parent" msgpack:"parent"`
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
	SchematicCreateSymbolResponse struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

func (s *SchematicService) CreateSymbol(ctx context.Context, req SchematicCreateSymbolRequest) (res SchematicCreateSymbolResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: symbol.OntologyIDsFromSymbols(req.Symbols),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
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
	SchematicRetrieveSymbolRequest struct {
		SearchTerm string      `json:"search_term" msgpack:"search_term"`
		Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	SchematicRetrieveSymbolResponse struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

func (s *SchematicService) RetrieveSymbol(ctx context.Context, req SchematicRetrieveSymbolRequest) (res SchematicRetrieveSymbolResponse, err error) {
	q := s.internal.Symbol.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	err = q.Entries(&res.Symbols).Exec(ctx, nil)
	if err != nil {
		return SchematicRetrieveSymbolResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: symbol.OntologyIDsFromSymbols(res.Symbols),
	}); err != nil {
		return SchematicRetrieveSymbolResponse{}, err
	}
	return res, err
}

type SchematicRenameSymbolRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  uuid.UUID `json:"key" msgpack:"key"`
}

func (s *SchematicService) RenameSymbol(ctx context.Context, req SchematicRenameSymbolRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{symbol.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.Symbol.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type SchematicDeleteSymbolRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *SchematicService) DeleteSymbol(ctx context.Context, req SchematicDeleteSymbolRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: symbol.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.Symbol.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type SchematicRetrieveSymbolGroupRequest struct{}

type SchematicRetrieveSymbolGroupResponse struct {
	Group group.Group `json:"group" msgpack:"group"`
}

func (s *SchematicService) RetrieveSymbolGroup(
	ctx context.Context,
	_ SchematicRetrieveSymbolGroupRequest,
) (SchematicRetrieveSymbolGroupResponse, error) {
	g := s.internal.Symbol.Group()
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{g.OntologyID()},
	}); err != nil {
		return SchematicRetrieveSymbolGroupResponse{}, err
	}
	return SchematicRetrieveSymbolGroupResponse{Group: g}, nil
}
