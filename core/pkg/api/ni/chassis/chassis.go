// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package chassis serves the NI chassis routes.
package chassis

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the NI chassis API service.
type ServiceConfig struct {
	// Chassis stores and retrieves chassis.
	// [REQUIRED]
	Chassis *chassis.Service
	// Status retrieves chassis statuses.
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Chassis = override.Nil(c.Chassis, other.Chassis)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ni.chassis")
	v.NotNil("chassis", c.Chassis)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service serves the NI chassis routes.
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
	// CreateRequest carries the chassis to create or replace.
	CreateRequest struct {
		// Chassis are the chassis to store. A nil key mints a new one.
		Chassis []chassis.Chassis `json:"chassis" msgpack:"chassis"`
	}
	// CreateResponse carries the created chassis with their keys.
	CreateResponse struct {
		// Chassis are the stored chassis, without their statuses.
		Chassis []chassis.Chassis `json:"chassis" msgpack:"chassis"`
	}
)

// Create stores the given chassis.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.cfg.Chassis.NewWriter(tx)
	for i := range req.Chassis {
		req.Chassis[i].Status = nil
		if err := w.Create(ctx, &req.Chassis[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the chassis that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, chassis.ResourceID)
}

// Retrieve returns the chassis that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.cfg.Chassis.NewRetrieve().Entries(&res.Chassis))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, chassis.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	ids := lo.Map(res.Chassis, func(c chassis.Chassis, _ int) resource.ID {
		return c.ResourceID()
	})
	statuses, err := s.cfg.Status.Join[chassis.StatusDetails](ctx, snap, ids)
	if err != nil {
		return RetrieveResponse{}, err
	}
	for i, st := range statuses {
		res.Chassis[i].Status = st
	}
	return res, nil
}

// UpdateRequest names a chassis and the one field group to write to it.
type UpdateRequest struct {
	// Update is the field group to write.
	Update chassis.Update `json:"update" msgpack:"update"`
	// Key selects the chassis to update.
	Key chassis.Key `json:"key" msgpack:"key"`
}

// Update writes one field group to a chassis and keeps the other as stored. It returns
// a validation error when the request carries no group.
func (s *Service) Update(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Chassis.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the chassis to delete.
type DeleteRequest struct {
	// Keys selects the chassis to delete.
	Keys []chassis.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given chassis with their modules and statuses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Chassis.NewWriter(tx).Delete(ctx, req.Keys...)
}
