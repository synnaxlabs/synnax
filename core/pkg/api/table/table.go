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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the table API service.
type ServiceConfig struct {
	// Table stores and retrieves tables.
	//
	// [REQUIRED]
	Table *table.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the table API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Table = override.Nil(c.Table, other.Table)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.table")
	v.NotNil("table", c.Table)
	return v.Error()
}

type Service struct {
	internal *table.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Table,
	}, nil
}

type (
	CreateRequest struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Tables []table.Table `json:"tables" msgpack:"tables"`
	}
)

// Create creates the tables. It returns an error wrapping validate.ErrValidation when a
// table with one of their keys exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Tables); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Tables: req.Tables}, nil
}

// DispatchRequest carries an action sequence to apply to a single table. DispatchKey is
// a client-generated identifier for the batch, registered as outstanding on the
// originator before the request is sent. The server echoes it verbatim on the broadcast
// frame so the originator can recognize its own echo race-safely.
type DispatchRequest = actions.DispatchRequest[table.Key, table.Action]

// DispatchResponse carries the actions that the Core added to a dispatch, such as a set
// of the project that a move into a group of another project needs.
type DispatchResponse = actions.DispatchResponse[table.Action]

// Dispatch applies the action sequence to the target table in tx. Subscribers to the
// table action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (DispatchResponse, error) {
	added, err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
	if err != nil {
		return DispatchResponse{}, err
	}
	return DispatchResponse{Actions: added}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the tables that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, table.ResourceID)
}

// Retrieve returns the tables that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Tables))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, table.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []table.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the tables with the given keys. A key that matches no table is
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
