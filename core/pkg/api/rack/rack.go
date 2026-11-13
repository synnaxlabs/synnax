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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
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
	// Task stores and retrieves tasks.
	//
	// [REQUIRED]
	Task *task.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC checks the read access that a create response needs.
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
	c.Task = override.Nil(c.Task, other.Task)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.rack")
	v.NotNil("rack", c.Rack)
	v.NotNil("task", c.Task)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access *rbac.Service
	rack   *rack.Service
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

// Create creates the racks. It returns an error wrapping validate.ErrValidation when a
// rack with the key of one exists. The response holds the status of each rack that the
// subject can retrieve.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.rack.NewWriter(tx).CreateMany(ctx, &req.Racks); err != nil {
		return CreateResponse{}, err
	}
	if err := s.attachReadableStatuses(ctx, tx, req.Racks); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

func (s *Service) attachReadableStatuses(
	ctx context.Context,
	tx gorp.Tx,
	racks []rack.Rack,
) error {
	enforcer := s.access.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	for i := range racks {
		err := enforcer.Enforce(ctx, access.Request{
			Subject: subject,
			Action:  access.ActionRetrieve,
			Objects: []resource.ID{racks[i].ResourceID()},
		})
		// Create access does not grant read access.
		if errors.Is(err, access.ErrDenied) {
			racks[i].Status = nil
			continue
		}
		if err != nil {
			return err
		}
		if err := s.attachStatuses(ctx, tx, racks[i:i+1]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) attachStatuses(
	ctx context.Context,
	snap gorp.Snapshot,
	racks []rack.Rack,
) error {
	statuses, err := s.status.Join[rack.StatusDetails](
		ctx,
		snap,
		rack.ResourceIDsFromRacks(racks),
	)
	if err != nil {
		return err
	}
	for i, st := range statuses {
		racks[i].Status = st
	}
	return nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the racks that the top level of Where selects
// by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, rack.Key.ResourceID)
}

// Retrieve returns the racks that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.rack.NewRetrieve().Entries(&res.Racks))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, rack.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeStatus {
		if err := s.attachStatuses(ctx, snap, res.Racks); err != nil {
			return RetrieveResponse{}, err
		}
	}
	return res, nil
}

// DispatchRequest carries an action sequence to apply to a single rack. DispatchKey is
// a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[rack.Key, rack.Action]

// Dispatch applies the action sequence to the target rack atomically. Subscribers to
// the rack action channel receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.rack.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

type DeleteRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the racks with the internal tasks, API keys, and sessions they hold.
// It refuses a rack that is embedded or holds a device or a task that is not internal.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.task.NewWriter(tx).DeleteRacks(ctx, req.Keys...)
}
