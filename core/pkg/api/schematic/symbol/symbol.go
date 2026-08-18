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
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/gorp"
)

// Service is the API for schematic symbols and the groups that hold them. It enforces
// access control and delegates to the symbol service.
type Service struct {
	access   *rbac.Service
	internal *symbol.Service
}

// NewService opens a Service from the API layer's configuration.
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
	// CreateRequest carries the symbols to create and the resource to create them
	// under. A zero Parent leaves the symbols unattached.
	CreateRequest struct {
		// Parent is the resource to attach the symbols to. A zero value leaves them
		// unattached.
		Parent ontology.ID `json:"parent" msgpack:"parent"`
		// Symbols are the symbols to create. The Core mints a key for each.
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
	// CreateResponse carries the created symbols with their keys stamped.
	CreateResponse struct {
		// Symbols are the created symbols, carrying the keys the Core minted.
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

// Create persists the symbols in req. It requires create access on the symbol type,
// and update access on req.Parent when one is given.
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
	// RetrieveRequest filters symbols by key, by search term, or by both. An empty
	// request matches every symbol.
	RetrieveRequest struct {
		// SearchTerm fuzzy-matches symbol names. Empty applies no name filter.
		SearchTerm string `json:"search_term" msgpack:"search_term"`
		// Keys narrows the query to these symbols. Empty applies no key filter.
		Keys []symbol.Key `json:"keys" msgpack:"keys"`
	}
	// RetrieveResponse carries the matched symbols.
	RetrieveResponse struct {
		// Symbols are the matched symbols, left off the wire when there are none.
		Symbols []symbol.Symbol `json:"symbols,omitzero" msgpack:"symbols,omitzero"`
	}
)

// Retrieve returns the symbols matching req. It requires retrieve access on every
// match.
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

// RenameRequest names the symbol to rename and the name to give it.
type RenameRequest struct {
	// Name is the name to give the symbol.
	Name string `json:"name" msgpack:"name"`
	// Key identifies the symbol to rename.
	Key symbol.Key `json:"key" msgpack:"key"`
}

// Rename renames the symbol. It requires update access on it.
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

// DeleteRequest carries the keys of the symbols to delete.
type DeleteRequest struct {
	// Keys identify the symbols to delete.
	Keys []symbol.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the symbols. It requires delete access on every one.
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

// RetrieveGroupRequest is empty. The permanent symbol group is a singleton.
type RetrieveGroupRequest struct{}

// RetrieveGroupResponse carries the permanent group that holds every symbol group.
type RetrieveGroupResponse struct {
	// Group is the permanent group that holds every symbol group.
	Group group.Group `json:"group" msgpack:"group"`
}

// RetrieveGroup returns the permanent symbol group. It requires retrieve access on it.
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
	// ExportGroupRequest names the group to export.
	ExportGroupRequest struct {
		// Key identifies the group to export.
		Key group.Key `json:"key" msgpack:"key"`
		// Encoding names the serialization member files are written in. "JSON" is the
		// only supported value.
		Encoding string `json:"encoding" msgpack:"encoding"`
	}
	// ExportGroupResponse holds the bundle's contents keyed by file name. The HTTP
	// transport encodes it as a zip archive.
	ExportGroupResponse = zip.Files
)

// ExportGroup exports every symbol in the group as a bundle. It requires retrieve
// access on the group, which it enforces before it reads a symbol, and on every symbol
// the group holds.
func (s *Service) ExportGroup(
	ctx context.Context,
	req ExportGroupRequest,
) (ExportGroupResponse, error) {
	encoder, err := imex.ResolveEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}
	var (
		enforcer = s.access.NewEnforcer(nil)
		subject  = auth.GetSubject(ctx)
	)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return nil, err
	}
	files, members, err := s.internal.ExportGroup(ctx, req.Key, encoder)
	if err != nil {
		return nil, err
	}
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionRetrieve,
		Objects: members,
	}); err != nil {
		return nil, err
	}
	return files, nil
}

type (
	// ImportGroupRequest holds a bundle's contents keyed by file name. The HTTP
	// transport decodes it from a zip archive.
	ImportGroupRequest = zip.Files
	// ImportGroupResponse carries the group the import created.
	ImportGroupResponse struct {
		// Group is the created group holding the imported symbols.
		Group group.Group `json:"group" msgpack:"group"`
	}
)

// ImportGroup imports a symbol group bundle in a single transaction. It requires
// create access on the group and symbol types, and update access on the permanent
// symbol group, all enforced before any file decodes.
func (s *Service) ImportGroup(
	ctx context.Context,
	tx gorp.Tx,
	req ImportGroupRequest,
) (ImportGroupResponse, error) {
	var (
		enforcer = s.access.NewEnforcer(tx)
		subject  = auth.GetSubject(ctx)
	)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: []ontology.ID{
			{Type: ontology.ResourceTypeGroup},
			{Type: ontology.ResourceTypeSchematicSymbol},
		},
	}); err != nil {
		return ImportGroupResponse{}, err
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{s.internal.Group().OntologyID()},
	}); err != nil {
		return ImportGroupResponse{}, err
	}
	g, err := s.internal.ImportGroup(ctx, tx, req, imex.JSONCodec)
	if err != nil {
		return ImportGroupResponse{}, err
	}
	return ImportGroupResponse{Group: g}, nil
}

// DeleteGroupRequest names the group to delete.
type DeleteGroupRequest struct {
	// Key identifies the group to delete.
	Key group.Key `json:"key" msgpack:"key"`
}

// DeleteGroup deletes the group and every symbol in it in a single transaction. It
// requires delete access on the group, which it enforces before it reads a symbol, and
// on every symbol the group holds.
func (s *Service) DeleteGroup(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteGroupRequest,
) (types.Nil, error) {
	var (
		enforcer = s.access.NewEnforcer(tx)
		subject  = auth.GetSubject(ctx)
	)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionDelete,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return types.Nil{}, err
	}
	// Read through tx so the symbols enforced on and the symbols deleted come from one
	// snapshot.
	symbols, err := s.internal.RetrieveGroupSymbols(ctx, tx, req.Key)
	if err != nil {
		return types.Nil{}, err
	}
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionDelete,
		Objects: symbols,
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.internal.DeleteGroup(ctx, tx, req.Key, symbols)
}
