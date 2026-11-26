// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package server serves the OPC UA server routes.
package server

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the OPC UA server API service.
type ServiceConfig struct {
	// Server stores and retrieves servers.
	//
	// [REQUIRED]
	Server *server.Service
	// Status retrieves server statuses.
	//
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Server = override.Nil(c.Server, other.Server)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.opcua.server")
	v.NotNil("server", c.Server)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service serves the OPC UA server routes.
type Service struct {
	cfg ServiceConfig
}

// NewService returns a Service. It returns a validation error when the configuration
// is invalid.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

type (
	// CreateRequest carries the servers to create.
	CreateRequest struct {
		// Servers are the servers to store. A nil key mints a new one.
		Servers []server.Server `json:"servers" msgpack:"servers"`
	}
	// CreateResponse carries the created servers with their keys.
	CreateResponse struct {
		// Servers are the stored servers, without their statuses.
		Servers []server.Server `json:"servers" msgpack:"servers"`
	}
)

// Create stores the given servers.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.cfg.Server.NewWriter(tx)
	for i := range req.Servers {
		req.Servers[i].Status = nil
		if err := w.Create(ctx, &req.Servers[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

// UpdateRequest carries one field group to write to a stored server.
type UpdateRequest struct {
	// Update is the field group to write.
	Update server.Update `json:"update" msgpack:"update"`
	// Key selects the server to update.
	Key server.Key `json:"key" msgpack:"key"`
}

// Update writes the field group of req to the server stored under req.Key, as
// server.Writer.Apply does. It returns a validation error when the request carries no
// group.
func (s *Service) Update(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Server.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the servers to delete.
type DeleteRequest struct {
	// Keys selects the servers to delete.
	Keys []server.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given servers with their statuses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Server.NewWriter(tx).Delete(ctx, req.Keys...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the servers that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, server.ResourceID)
}

// Retrieve returns the servers that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.cfg.Server.NewRetrieve().Entries(&res.Servers))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, server.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	ids := lo.Map(res.Servers, func(srv server.Server, _ int) resource.ID {
		return srv.ResourceID()
	})
	statuses, err := s.cfg.Status.Join[server.StatusDetails](ctx, snap, ids)
	if err != nil {
		return RetrieveResponse{}, err
	}
	for i, st := range statuses {
		res.Servers[i].Status = st
	}
	return res, nil
}
