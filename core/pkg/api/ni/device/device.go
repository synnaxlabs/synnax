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
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
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
	// RBAC enforces access control on every request.
	// [REQUIRED]
	RBAC *rbac.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Device = override.Nil(c.Device, other.Device)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ni.device")
	v.NotNil("device", c.Device)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
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

func (s *Service) enforce(
	ctx context.Context,
	tx gorp.Tx,
	action access.Action,
	objects ...ontology.ID,
) error {
	return s.cfg.RBAC.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  action,
		Objects: objects,
	})
}

type (
	// CreateRequest carries the devices to create or replace.
	CreateRequest struct {
		// Devices are the devices to store. A nil key mints a new one.
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
	// CreateResponse carries the created devices with their keys.
	CreateResponse struct {
		// Devices are the stored devices.
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
)

// enforceCreate requires update access on each device that already exists, because
// Create replaces it, and create access on the type when any device is new.
func (s *Service) enforceCreate(
	ctx context.Context,
	tx gorp.Tx,
	devices []device.Device,
) error {
	keys := make([]device.Key, 0, len(devices))
	for _, d := range devices {
		if d.Key != uuid.Nil() {
			keys = append(keys, d.Key)
		}
	}
	var existing []device.Device
	if len(keys) > 0 {
		if err := s.cfg.Device.NewRetrieve().
			Where(device.MatchKeys(keys...)).
			Entries(&existing).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
	}
	found := make(set.Set[device.Key], len(existing))
	ids := make([]ontology.ID, len(existing))
	for i, d := range existing {
		found.Add(d.Key)
		ids[i] = d.OntologyID()
	}
	if len(ids) > 0 {
		if err := s.enforce(ctx, tx, access.ActionUpdate, ids...); err != nil {
			return err
		}
	}
	if !slices.ContainsFunc(
		devices,
		func(d device.Device) bool { return !found.Contains(d.Key) },
	) {
		return nil
	}
	return s.enforce(
		ctx,
		tx,
		access.ActionCreate,
		ontology.ID{Type: ontology.ResourceTypeNiDevice},
	)
}

// Create stores the given devices. It requires create access on the device type for a
// new device and update access on a device it replaces.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.enforceCreate(ctx, tx, req.Devices); err != nil {
		return CreateResponse{}, err
	}
	w := s.cfg.Device.NewWriter(tx)
	for i := range req.Devices {
		if err := w.Create(ctx, &req.Devices[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

// RetrieveRequest selects the devices to retrieve. Empty fields do not filter.
type RetrieveRequest struct {
	// SearchTerm matches devices by name.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Keys selects devices by key.
	Keys []device.Key `json:"keys" msgpack:"keys"`
	// Names selects devices by name.
	Names []string `json:"names" msgpack:"names"`
	// Serials selects devices by serial number.
	Serials []string `json:"serials" msgpack:"serials"`
	// Models selects devices by product name.
	Models []string `json:"models" msgpack:"models"`
	// Racks selects devices by rack.
	Racks []rack.Key `json:"racks" msgpack:"racks"`
	// Limit caps the number of devices returned. Zero means no cap.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset skips this many matching devices.
	Offset int `json:"offset" msgpack:"offset"`
	// IgnoreNotFound returns no devices instead of a not found error.
	IgnoreNotFound bool `json:"ignore_not_found" msgpack:"ignore_not_found"`
	// IncludeStatus attaches each device's status.
	IncludeStatus bool `json:"include_status" msgpack:"include_status"`
}

// RetrieveResponse carries the retrieved devices.
type RetrieveResponse struct {
	// Devices are the retrieved devices.
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

// Retrieve returns the devices that req selects. It requires retrieve access on them.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.cfg.Device.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.Where(device.MatchKeys(req.Keys...))
	}
	if len(req.SearchTerm) > 0 {
		q = q.Search(req.SearchTerm)
	}
	if len(req.Names) > 0 {
		q = q.Where(device.MatchNames(req.Names...))
	}
	if len(req.Serials) > 0 {
		q = q.Where(device.MatchSerials(req.Serials...))
	}
	if len(req.Models) > 0 {
		q = q.Where(device.MatchModels(req.Models...))
	}
	if len(req.Racks) > 0 {
		q = q.Where(device.MatchRacks(req.Racks...))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	var res RetrieveResponse
	if err := q.Entries(&res.Devices).Exec(ctx, nil); err != nil &&
		(!req.IgnoreNotFound || !errors.Is(err, query.ErrNotFound)) {
		return RetrieveResponse{}, err
	}
	keys := lo.Map(res.Devices, func(d device.Device, _ int) device.Key {
		return d.Key
	})
	ids := device.OntologyIDs(keys)
	if err := s.enforce(ctx, nil, access.ActionRetrieve, ids...); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	statuses := make([]device.Status, 0, len(res.Devices))
	if err := errors.Skip(s.cfg.Status.NewRetrieve[device.StatusDetails]().
		Where(status.MatchKeys[device.StatusDetails](ontology.IDsToKeys(ids)...)).
		Entries(&statuses).
		Exec(ctx, nil), query.ErrNotFound); err != nil {
		return RetrieveResponse{}, err
	}
	byKey := lo.SliceToMap(statuses, func(st device.Status) (status.Key, device.Status) {
		return st.Key, st
	})
	for i, id := range ids {
		if st, ok := byKey[id.String()]; ok {
			res.Devices[i].Status = &st
		}
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

// Update writes one field group to a device and keeps the other as stored. It requires
// update access on the device and returns a validation error when the request carries
// no group.
func (s *Service) Update(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateRequest,
) (struct{}, error) {
	if err := s.enforce(
		ctx,
		tx,
		access.ActionUpdate,
		device.OntologyID(req.Key),
	); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.cfg.Device.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the devices to delete.
type DeleteRequest struct {
	// Keys selects the devices to delete.
	Keys []device.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given devices. It requires delete access on them and returns a
// validation error when a task references one.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.enforce(
		ctx,
		tx,
		access.ActionDelete,
		device.OntologyIDs(req.Keys)...,
	); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.cfg.Device.NewWriter(tx).Delete(ctx, req.Keys...)
}
