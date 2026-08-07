// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	access   *rbac.Service
	internal *symbol.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Service.Schematic.Symbol,
		access:   cfg.Service.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Parent  ontology.ID     `json:"parent"  msgpack:"parent"`
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
	CreateResponse struct {
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	enforcer := s.access.NewEnforcer(tx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeSchematicSymbol}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if !req.Parent.IsZero() {
		if err := enforcer.Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []ontology.ID{req.Parent},
		}); err != nil {
			return CreateResponse{}, err
		}
	}
	writer := s.internal.NewWriter(tx)
	if err := writer.CreateMany(ctx, &req.Symbols, req.Parent); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Symbols: req.Symbols}, nil
}

type (
	RetrieveRequest struct {
		SearchTerm string       `json:"search_term" msgpack:"search_term"`
		Keys       []symbol.Key `json:"keys"        msgpack:"keys"`
	}
	RetrieveResponse struct {
		Symbols []symbol.Symbol `json:"symbols,omitzero" msgpack:"symbols,omitzero"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.Where(symbol.MatchKeys(req.Keys...))
	}
	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	var res RetrieveResponse
	if err := q.Entries(&res.Symbols).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: symbol.OntologyIDsFromSymbols(res.Symbols),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type RenameRequest struct {
	Name string     `json:"name" msgpack:"name"`
	Key  symbol.Key `json:"key"  msgpack:"key"`
}

func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{symbol.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	Keys []symbol.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: symbol.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

type RetrieveGroupRequest struct{}

type RetrieveGroupResponse struct {
	Group group.Group `json:"group" msgpack:"group"`
}

func (s *Service) RetrieveGroup(
	ctx context.Context,
	_ RetrieveGroupRequest,
) (RetrieveGroupResponse, error) {
	g := s.internal.Group()
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{g.OntologyID()},
	}); err != nil {
		return RetrieveGroupResponse{}, err
	}
	return RetrieveGroupResponse{Group: g}, nil
}

type (
	ExportGroupRequest struct {
		Key group.Key `json:"key" msgpack:"key"`
	}
	// ExportGroupResponse is the exported bundle's contents keyed by file name. The
	// HTTP transport encodes it as a zip archive.
	ExportGroupResponse = zip.Files
)

// ExportGroup exports every symbol in the group as a bundle.
func (s *Service) ExportGroup(
	ctx context.Context,
	req ExportGroupRequest,
) (ExportGroupResponse, error) {
	bundle, err := s.internal.ExportGroup(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: append(bundle.Members, group.OntologyID(req.Key)),
	}); err != nil {
		return nil, err
	}
	return bundle.Files, nil
}
