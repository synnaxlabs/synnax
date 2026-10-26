// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type (
	Key  = rack.Key
	Rack = rack.Rack
)

// ServiceConfig is the configuration for the rack API service.
type ServiceConfig struct {
	// Rack stores and retrieves racks.
	//
	// [REQUIRED]
	Rack *rack.Service
	// Device stores and retrieves devices.
	//
	// [REQUIRED]
	Device *device.Service
	// Task stores and retrieves tasks.
	//
	// [REQUIRED]
	Task *task.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the rack API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Device = override.Nil(c.Device, other.Device)
	c.Task = override.Nil(c.Task, other.Task)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.rack")
	v.NotNil("rack", c.Rack)
	v.NotNil("device", c.Device)
	v.NotNil("task", c.Task)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access *rbac.Service
	rack   *rack.Service
	device *device.Service
	task   *task.Service
	status *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		rack:   cfg.Rack,
		device: cfg.Device,
		task:   cfg.Task,
		status: cfg.Status,
		access: cfg.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
	CreateResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeRack}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.rack.NewWriter(tx).CreateMany(ctx, &req.Racks); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

type (
	RetrieveRequest struct {
		Embedded      *bool      `json:"embedded"       msgpack:"embedded"`
		HostIsNode    *bool      `json:"host_is_node"   msgpack:"host_is_node"`
		SearchTerm    string     `json:"search_term"    msgpack:"search_term"`
		Keys          []rack.Key `json:"keys"           msgpack:"keys"`
		Names         []string   `json:"names"          msgpack:"names"`
		Integration   string     `json:"integration"    msgpack:"integration"`
		Limit         int        `json:"limit"          msgpack:"limit"`
		Offset        int        `json:"offset"         msgpack:"offset"`
		IncludeStatus bool       `json:"include_status" msgpack:"include_status"`
		// IgnoreNotFoundError returns the racks that do exist instead of an error when
		// some keys are missing.
		IgnoreNotFoundError bool `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
	}
	RetrieveResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		hasSearch      = len(req.SearchTerm) > 0
		hasKeys        = len(req.Keys) > 0
		hasNames       = len(req.Names) > 0
		hasLimit       = req.Limit > 0
		hasOffset      = req.Offset > 0
		hasIntegration = req.Integration != ""
	)
	resRacks := make([]rack.Rack, 0, len(req.Keys)+len(req.Names))
	q := s.rack.NewRetrieve()
	if hasKeys {
		q = q.Where(rack.MatchKeys(req.Keys...))
	}
	if hasNames {
		q = q.Where(rack.MatchNames(req.Names...))
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if req.Embedded != nil {
		q = q.Where(rack.MatchEmbedded(*req.Embedded))
	}
	if req.HostIsNode != nil {
		q = q.Where(rack.MatchNodeIsHost(*req.HostIsNode))
	}
	if hasIntegration {
		q = q.Where(rack.MatchIntegration(req.Integration))
	}
	err := q.Entries(&resRacks).Exec(ctx, nil)
	if req.IgnoreNotFoundError {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if err != nil {
		return RetrieveResponse{}, err
	}

	if req.IncludeStatus {
		ids := rack.OntologyIDsFromRacks(resRacks)
		statuses := make([]rack.Status, 0, len(resRacks))
		// A status row can be missing, so the resource keeps a nil status.
		if err := errors.Skip(s.status.NewRetrieve[rack.StatusDetails]().
			Where(status.MatchKeys[rack.StatusDetails](ontology.IDsToKeys(ids)...)).
			Entries(&statuses).
			Exec(ctx, nil), query.ErrNotFound); err != nil {
			return RetrieveResponse{}, err
		}
		statusByKey := make(map[status.Key]*rack.Status, len(statuses))
		for i := range statuses {
			statusByKey[statuses[i].Key] = &statuses[i]
		}
		for i, id := range ids {
			resRacks[i].Status = statusByKey[id.String()]
		}
	}

	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rack.OntologyIDsFromRacks(resRacks),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Racks: resRacks}, nil
}

// RenameRequest names the rack to rename and the name to give it.
type RenameRequest struct {
	// Name is the new name. It must not be empty.
	Name string `json:"name" msgpack:"name"`
	// Key selects the rack to rename.
	Key rack.Key `json:"key" msgpack:"key"`
}

// Rename renames the rack. It requires update access on it.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{req.Key.OntologyID()},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.rack.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

func embeddedGuard(_ gorp.Context, r rack.Rack) error {
	if !r.Embedded {
		return nil
	}
	return errors.Wrapf(validate.ErrValidation, "cannot delete embedded rack")
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: rack.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	exists, err := s.device.NewRetrieve().
		Where(device.MatchRacks(req.Keys...)).
		Exists(ctx, tx)
	if err != nil {
		return struct{}{}, err
	}
	if exists {
		return struct{}{}, errors.Wrapf(
			validate.ErrValidation,
			"cannot delete rack when devices are still attached",
		)
	}
	exists, err = s.task.NewRetrieve().
		Where(task.And(task.MatchInternal(false), task.MatchRacks(req.Keys...))).
		Exists(ctx, tx)
	if err != nil {
		return struct{}{}, err
	}
	if exists {
		return struct{}{}, errors.Wrapf(
			validate.ErrValidation,
			"cannot delete rack when tasks are still attached",
		)
	}
	w := s.rack.NewWriter(tx)
	for _, k := range req.Keys {
		if err := w.DeleteGuard(ctx, k, embeddedGuard); err != nil {
			return struct{}{}, err
		}
	}
	// Only internal tasks are left: the check above rejects the delete while the user
	// still has tasks of their own on the rack.
	if err := s.task.NewWriter(tx).DeleteByRacks(ctx, req.Keys...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}
