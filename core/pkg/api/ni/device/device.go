// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package device serves the NI device routes.
package device

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the NI device API service.
type ServiceConfig struct {
	// Device stores and retrieves devices.
	// [REQUIRED]
	Device *device.Service
	// Status retrieves device statuses.
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Device = override.Nil(c.Device, other.Device)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ni.device")
	v.NotNil("device", c.Device)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service serves the NI device routes.
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
	// CreateRequest carries the devices to create or replace.
	CreateRequest struct {
		// Devices are the devices to store. A nil key mints a new one.
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
	// CreateResponse carries the created devices with their keys.
	CreateResponse struct {
		// Devices are the stored devices, without their statuses.
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
)

// Create stores the given devices.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.cfg.Device.NewWriter(tx)
	for i := range req.Devices {
		req.Devices[i].Status = nil
		if err := w.Create(ctx, &req.Devices[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the devices that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, device.ResourceID)
}

// Retrieve returns the devices that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.cfg.Device.NewRetrieve().Entries(&res.Devices))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, device.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	ids := lo.Map(res.Devices, func(d device.Device, _ int) resource.ID {
		return d.ResourceID()
	})
	statuses, err := s.cfg.Status.Join[device.StatusDetails](ctx, snap, ids)
	if err != nil {
		return RetrieveResponse{}, err
	}
	for i, st := range statuses {
		res.Devices[i].Status = st
	}
	return res, nil
}

// UpdateRequest names a device and the one field group to write to it.
type UpdateRequest struct {
	// Update is the field group to write.
	Update device.Update `json:"update" msgpack:"update"`
	// Key selects the device to update.
	Key device.Key `json:"key" msgpack:"key"`
}

// Update writes one field group to a device and keeps the other as stored. It returns
// a validation error when the request carries no group.
func (s *Service) Update(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Device.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the devices to delete.
type DeleteRequest struct {
	// Keys selects the devices to delete.
	Keys []device.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given devices with their statuses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Device.NewWriter(tx).Delete(ctx, req.Keys...)
}
