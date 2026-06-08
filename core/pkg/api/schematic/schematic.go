// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *schematic.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		internal: cfg.Service.Schematic,
		access:   cfg.Service.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
		Workspace  workspace.Key         `json:"workspace" msgpack:"workspace"`
	}
	CreateResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	var res CreateResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionCreate,
			Objects: schematic.OntologyIDsFromSchematics(req.Schematics),
		}); err != nil {
			return err
		}
		for i, sch := range req.Schematics {
			if err := s.internal.NewWriter(tx).Create(ctx, req.Workspace, &sch); err != nil {
				return err
			}
			req.Schematics[i] = sch
		}
		res.Schematics = req.Schematics
		return nil
	}); err != nil {
		return CreateResponse{}, err
	}
	return res, nil
}

// DispatchRequest carries an action sequence to apply to a single schematic.
// DispatchKey is a client-generated identifier for the batch, registered as outstanding
// on the originator before the request is sent. The server echoes it verbatim on the
// broadcast frame so the originator can recognize its own echo race-safely.
type DispatchRequest = actions.DispatchRequest[schematic.Key, schematic.Action]

// Dispatch applies the action sequence to the target schematic atomically. Subscribers
// to the schematic action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	req DispatchRequest,
) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []ontology.ID{schematic.OntologyID(req.Key)},
		}); err != nil {
			return err
		}
		return s.internal.NewWriter(tx).
			Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
	})
}

type (
	RetrieveRequest struct {
		Keys []schematic.Key `json:"keys" msgpack:"keys"`
	}
	RetrieveResponse struct {
		Schematics []schematic.Schematic `json:"schematics" msgpack:"schematics"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context, req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewRetrieve().
			Where(schematic.MatchKeys(req.Keys...)).Entries(&res.Schematics).Exec(ctx, tx); err != nil {
			return err
		}
		return s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionRetrieve,
			Objects: schematic.OntologyIDs(req.Keys),
		})
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []schematic.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionDelete,
			Objects: schematic.OntologyIDs(req.Keys),
		}); err != nil {
			return err
		}
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type (
	CopyRequest struct {
		Name     string        `json:"name" msgpack:"name"`
		Key      schematic.Key `json:"key" msgpack:"key"`
		Snapshot bool          `json:"snapshot" msgpack:"snapshot"`
	}
	CopyResponse struct {
		Schematic schematic.Schematic `json:"schematic" msgpack:"schematic"`
	}
)

func (s *Service) Copy(ctx context.Context, req CopyRequest) (CopyResponse, error) {
	var res CopyResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionRetrieve,
			Objects: []ontology.ID{schematic.OntologyID(req.Key)},
		}); err != nil {
			return err
		}
		if err := s.internal.NewWriter(tx).Copy(ctx, req.Key, req.Name, req.Snapshot, &res.Schematic); err != nil {
			return err
		}
		return s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionCreate,
			Objects: []ontology.ID{schematic.OntologyID(res.Schematic.Key)},
		})
	}); err != nil {
		return CopyResponse{}, err
	}
	return res, nil
}

type (
	CreateSymbolRequest struct {
		Parent  ontology.ID     `json:"parent" msgpack:"parent"`
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
	CreateSymbolResponse struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

func (s *Service) CreateSymbol(ctx context.Context, req CreateSymbolRequest) (CreateSymbolResponse, error) {
	var res CreateSymbolResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		enforcer := s.access.NewEnforcer(tx)
		if err := enforcer.Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionCreate,
			Objects: symbol.OntologyIDsFromSymbols(req.Symbols),
		}); err != nil {
			return err
		}
		if !req.Parent.IsZero() {
			if err := enforcer.Enforce(ctx, access.Request{
				Subject: auth.GetSubject(ctx),
				Action:  access.ActionUpdate,
				Objects: []ontology.ID{req.Parent},
			}); err != nil {
				return err
			}
		}
		writer := s.internal.Symbol.NewWriter(tx)
		for i, sym := range req.Symbols {
			if err := writer.Create(ctx, &sym, req.Parent); err != nil {
				return err
			}
			req.Symbols[i] = sym
		}
		res.Symbols = req.Symbols
		return nil
	}); err != nil {
		return CreateSymbolResponse{}, err
	}
	return res, nil
}

type (
	RetrieveSymbolRequest struct {
		SearchTerm string       `json:"search_term" msgpack:"search_term"`
		Keys       []symbol.Key `json:"keys" msgpack:"keys"`
	}
	RetrieveSymbolResponse struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

func (s *Service) RetrieveSymbol(
	ctx context.Context, req RetrieveSymbolRequest,
) (RetrieveSymbolResponse, error) {
	var res RetrieveSymbolResponse
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		q := s.internal.Symbol.NewRetrieve()
		if len(req.Keys) > 0 {
			q = q.Where(symbol.MatchKeys(req.Keys...))
		}
		if req.SearchTerm != "" {
			q = q.Search(req.SearchTerm)
		}
		if err := q.Entries(&res.Symbols).Exec(ctx, tx); err != nil {
			return err
		}
		return s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionRetrieve,
			Objects: symbol.OntologyIDsFromSymbols(res.Symbols),
		})
	}); err != nil {
		return RetrieveSymbolResponse{}, err
	}
	return res, nil
}

type RenameSymbolRequest struct {
	Name string     `json:"name" msgpack:"name"`
	Key  symbol.Key `json:"key" msgpack:"key"`
}

func (s *Service) RenameSymbol(
	ctx context.Context,
	req RenameSymbolRequest,
) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []ontology.ID{symbol.OntologyID(req.Key)},
		}); err != nil {
			return err
		}
		return s.internal.Symbol.NewWriter(tx).Rename(ctx, req.Key, req.Name)
	})
}

type DeleteSymbolRequest struct {
	Keys []symbol.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) DeleteSymbol(ctx context.Context, req DeleteSymbolRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionDelete,
			Objects: symbol.OntologyIDs(req.Keys),
		}); err != nil {
			return err
		}
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
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionRetrieve,
			Objects: []ontology.ID{g.OntologyID()},
		})
	}); err != nil {
		return RetrieveSymbolGroupResponse{}, err
	}
	return RetrieveSymbolGroupResponse{Group: g}, nil
}
