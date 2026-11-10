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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the symbol API service.
type ServiceConfig struct {
	// Symbol stores and retrieves schematic symbols.
	//
	// [REQUIRED]
	Symbol *symbol.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the symbol API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Symbol = override.Nil(c.Symbol, other.Symbol)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.schematic.symbol")
	v.NotNil("symbol", c.Symbol)
	return v.Error()
}

// Service is the API for schematic symbols and the groups that hold them.
type Service struct {
	internal *symbol.Service
}

// NewService opens a Service from the API layer's configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Symbol}, nil
}

type (
	// CreateRequest carries the symbols to create.
	CreateRequest struct {
		// Symbols are the symbols to create. The Core mints a key for each.
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
	// CreateResponse carries the created symbols with their keys stamped.
	CreateResponse struct {
		// Symbols are the created symbols, carrying the keys the Core minted.
		Symbols []symbol.Symbol `json:"symbols" msgpack:"symbols"`
	}
)

// Create creates the symbols in req. It returns an error wrapping
// validate.ErrValidation when a symbol with one of their keys exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Symbols); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

// DispatchRequest carries an action sequence to apply to a single symbol. DispatchKey
// is a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[symbol.Key, symbol.Action]

// Dispatch applies the action sequence to the target symbol atomically. Subscribers to
// the symbol action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the symbols that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, symbol.ResourceID)
}

// Retrieve returns the symbols that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Symbols))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, symbol.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DeleteRequest carries the keys of the symbols to delete.
type DeleteRequest struct {
	// Keys identify the symbols to delete.
	Keys []symbol.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the symbols. A key that matches no symbol is ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
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

var _ bind.Named = ExportGroupRequest{}

// Named implements bind.Named.
func (r ExportGroupRequest) Named() []resource.ID {
	return []resource.ID{group.ResourceID(r.Key)}
}

// ExportGroup exports every symbol in the group as a bundle. It requires retrieve
// access on every symbol the group holds.
func (s *Service) ExportGroup(
	ctx context.Context,
	snap bind.Snapshot,
	req ExportGroupRequest,
) (ExportGroupResponse, error) {
	encoder, err := imex.ResolveEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}
	files, members, err := s.internal.ExportGroup(ctx, snap, req.Key, encoder)
	if err != nil {
		return nil, err
	}
	if err = snap.Enforce(members...); err != nil {
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

// ImportGroup imports a symbol group bundle into a new top-level group.
func (s *Service) ImportGroup(
	ctx context.Context,
	tx gorp.Tx,
	req ImportGroupRequest,
) (ImportGroupResponse, error) {
	g, err := s.internal.ImportGroup(ctx, tx, req)
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

// DeleteGroup deletes the group and every symbol in it. It returns a validation error
// when the group holds anything other than symbols.
func (s *Service) DeleteGroup(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteGroupRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.DeleteGroup(ctx, tx, req.Key)
}
