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
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
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

// ServiceConfig is the configuration for the NI chassis API service.
type ServiceConfig struct {
	// Chassis stores and retrieves chassis.
	// [REQUIRED]
	Chassis *chassis.Service
	// Status retrieves chassis statuses.
	// [REQUIRED]
	Status *status.Service
	// Device retrieves the modules that a chassis delete removes.
	// [REQUIRED]
	Device *device.Service
	// RBAC enforces access control on every request.
	// [REQUIRED]
	RBAC *rbac.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Chassis = override.Nil(c.Chassis, other.Chassis)
	c.Status = override.Nil(c.Status, other.Status)
	c.Device = override.Nil(c.Device, other.Device)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ni.chassis")
	v.NotNil("chassis", c.Chassis)
	v.NotNil("status", c.Status)
	v.NotNil("device", c.Device)
	v.NotNil("rbac", c.RBAC)
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
	// CreateRequest carries the chassis to create or replace.
	CreateRequest struct {
		// Chassis are the chassis to store. A nil key mints a new one.
		Chassis []chassis.Chassis `json:"chassis" msgpack:"chassis"`
	}
	// CreateResponse carries the created chassis with their keys.
	CreateResponse struct {
		// Chassis are the stored chassis.
		Chassis []chassis.Chassis `json:"chassis" msgpack:"chassis"`
	}
)

// enforceCreate requires update access on each chassis that already exists, because
// Create replaces it, and create access on the type when any chassis is new.
func (s *Service) enforceCreate(
	ctx context.Context,
	tx gorp.Tx,
	records []chassis.Chassis,
) error {
	keys := make([]chassis.Key, 0, len(records))
	for _, c := range records {
		if c.Key != uuid.Nil() {
			keys = append(keys, c.Key)
		}
	}
	var existing []chassis.Chassis
	if len(keys) > 0 {
		if err := s.cfg.Chassis.NewRetrieve().
			Where(chassis.MatchKeys(keys...)).
			Entries(&existing).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
	}
	found := make(set.Set[chassis.Key], len(existing))
	ids := make([]ontology.ID, len(existing))
	for i, c := range existing {
		found.Add(c.Key)
		ids[i] = c.OntologyID()
	}
	if len(ids) > 0 {
		if err := s.enforce(ctx, tx, access.ActionUpdate, ids...); err != nil {
			return err
		}
	}
	if !slices.ContainsFunc(
		records,
		func(c chassis.Chassis) bool { return !found.Contains(c.Key) },
	) {
		return nil
	}
	return s.enforce(
		ctx,
		tx,
		access.ActionCreate,
		ontology.ID{Type: ontology.ResourceTypeNiChassis},
	)
}

// Create stores the given chassis. It requires create access on the chassis type for a
// new chassis and update access on a chassis it replaces.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.enforceCreate(ctx, tx, req.Chassis); err != nil {
		return CreateResponse{}, err
	}
	w := s.cfg.Chassis.NewWriter(tx)
	for i := range req.Chassis {
		if err := w.Create(ctx, &req.Chassis[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

// RetrieveRequest selects the chassis to retrieve. Empty fields do not filter.
type RetrieveRequest struct {
	// SearchTerm matches chassis by name.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Keys selects chassis by key.
	Keys []chassis.Key `json:"keys" msgpack:"keys"`
	// Names selects chassis by name.
	Names []string `json:"names" msgpack:"names"`
	// Serials selects chassis by serial number.
	Serials []string `json:"serials" msgpack:"serials"`
	// Models selects chassis by product name.
	Models []string `json:"models" msgpack:"models"`
	// Racks selects chassis by rack.
	Racks []rack.Key `json:"racks" msgpack:"racks"`
	// Limit caps the number of chassis returned. Zero means no cap.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset skips this many matching chassis.
	Offset int `json:"offset" msgpack:"offset"`
	// IgnoreNotFound returns no chassis instead of a not found error.
	IgnoreNotFound bool `json:"ignore_not_found" msgpack:"ignore_not_found"`
	// IncludeStatus attaches each chassis's status.
	IncludeStatus bool `json:"include_status" msgpack:"include_status"`
}

// RetrieveResponse carries the retrieved chassis.
type RetrieveResponse struct {
	// Chassis are the retrieved chassis.
	Chassis []chassis.Chassis `json:"chassis" msgpack:"chassis"`
}

// Retrieve returns the chassis that req selects. It requires retrieve access on them.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.cfg.Chassis.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.Where(chassis.MatchKeys(req.Keys...))
	}
	if len(req.SearchTerm) > 0 {
		q = q.Search(req.SearchTerm)
	}
	if len(req.Names) > 0 {
		q = q.Where(chassis.MatchNames(req.Names...))
	}
	if len(req.Serials) > 0 {
		q = q.Where(chassis.MatchSerials(req.Serials...))
	}
	if len(req.Models) > 0 {
		q = q.Where(chassis.MatchModels(req.Models...))
	}
	if len(req.Racks) > 0 {
		q = q.Where(chassis.MatchRacks(req.Racks...))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	var res RetrieveResponse
	if err := q.Entries(&res.Chassis).Exec(ctx, nil); err != nil &&
		(!req.IgnoreNotFound || !errors.Is(err, query.ErrNotFound)) {
		return RetrieveResponse{}, err
	}
	keys := lo.Map(res.Chassis, func(c chassis.Chassis, _ int) chassis.Key {
		return c.Key
	})
	ids := chassis.OntologyIDs(keys)
	if err := s.enforce(ctx, nil, access.ActionRetrieve, ids...); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	statuses := make([]chassis.Status, 0, len(res.Chassis))
	if err := errors.Skip(s.cfg.Status.NewRetrieve[chassis.StatusDetails]().
		Where(status.MatchKeys[chassis.StatusDetails](ontology.IDsToKeys(ids)...)).
		Entries(&statuses).
		Exec(ctx, nil), query.ErrNotFound); err != nil {
		return RetrieveResponse{}, err
	}
	byKey := lo.SliceToMap(statuses, func(st chassis.Status) (status.Key, chassis.Status) {
		return st.Key, st
	})
	for i, id := range ids {
		if st, ok := byKey[id.String()]; ok {
			res.Chassis[i].Status = &st
		}
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

// Update writes one field group to a chassis and keeps the other as stored. It requires
// update access on the chassis and returns a validation error when the request carries
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
		chassis.OntologyID(req.Key),
	); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.cfg.Chassis.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the chassis to delete.
type DeleteRequest struct {
	// Keys selects the chassis to delete.
	Keys []chassis.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given chassis with their modules. It requires delete access on the
// chassis and on each module, and returns a validation error when a task references a
// module.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	var modules []device.Device
	if err := s.cfg.Device.NewRetrieve().
		Where(device.MatchChassis(req.Keys...)).
		Entries(&modules).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return struct{}{}, err
	}
	ids := chassis.OntologyIDs(req.Keys)
	for _, d := range modules {
		ids = append(ids, d.OntologyID())
	}
	if err := s.enforce(ctx, tx, access.ActionDelete, ids...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.cfg.Chassis.NewWriter(tx).Delete(ctx, req.Keys...)
}
