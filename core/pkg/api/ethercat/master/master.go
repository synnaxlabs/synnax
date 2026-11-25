// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package master serves the EtherCAT master routes.
package master

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/master"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the EtherCAT master API service.
type ServiceConfig struct {
	// Master stores and retrieves masters.
	// [REQUIRED]
	Master *master.Service
	// Status retrieves master statuses.
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Master = override.Nil(c.Master, other.Master)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ethercat.master")
	v.NotNil("master", c.Master)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service serves the EtherCAT master routes.
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
	// CreateRequest carries the masters to create or replace.
	CreateRequest struct {
		// Masters are the masters to store. A nil key mints a new one.
		Masters []master.Master `json:"masters" msgpack:"masters"`
	}
	// CreateResponse carries the created masters with their keys.
	CreateResponse struct {
		// Masters are the stored masters, without their statuses.
		Masters []master.Master `json:"masters" msgpack:"masters"`
	}
)

// Create stores the given masters.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.cfg.Master.NewWriter(tx)
	for i := range req.Masters {
		req.Masters[i].Status = nil
		if err := w.Create(ctx, &req.Masters[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the masters that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, master.ResourceID)
}

// Retrieve returns the masters that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.cfg.Master.NewRetrieve().Entries(&res.Masters))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, master.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	ids := lo.Map(res.Masters, func(m master.Master, _ int) resource.ID {
		return m.ResourceID()
	})
	statuses, err := s.cfg.Status.Join[master.StatusDetails](ctx, snap, ids)
	if err != nil {
		return RetrieveResponse{}, err
	}
	for i, st := range statuses {
		res.Masters[i].Status = st
	}
	return res, nil
}

// UpdateRequest names a master and the one field group to write to it.
type UpdateRequest struct {
	// Update is the field group to write.
	Update master.Update `json:"update" msgpack:"update"`
	// Key selects the master to update.
	Key master.Key `json:"key" msgpack:"key"`
}

// Update writes one field group to a master and keeps the other as stored. It returns
// a validation error when the request carries no group.
func (s *Service) Update(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Master.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the masters to delete.
type DeleteRequest struct {
	// Keys selects the masters to delete.
	Keys []master.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given masters with their slaves and statuses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Master.NewWriter(tx).Delete(ctx, req.Keys...)
}
