// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the device API service.
type ServiceConfig struct {
	// Device stores and retrieves devices.
	//
	// [REQUIRED]
	Device *device.Service
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
	// DefaultServiceConfig is the default configuration for the device API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Device = override.Nil(c.Device, other.Device)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.device")
	v.NotNil("device", c.Device)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access *rbac.Service
	device *device.Service
	status *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		device: cfg.Device,
		status: cfg.Status,
		access: cfg.RBAC,
	}, nil
}

type (
	CreateRequest struct {
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
	CreateResponse struct {
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
)

// Create creates the devices. It returns an error wrapping validate.ErrValidation when
// a device with the key of one exists. The response holds the status of each device
// that the subject can retrieve.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.device.NewWriter(tx).CreateMany(ctx, &req.Devices); err != nil {
		return CreateResponse{}, err
	}
	if err := s.attachReadableStatuses(ctx, tx, req.Devices); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

func (s *Service) attachReadableStatuses(
	ctx context.Context,
	tx gorp.Tx,
	devices []device.Device,
) error {
	enforcer := s.access.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	for i := range devices {
		err := enforcer.Enforce(ctx, access.Request{
			Subject: subject,
			Action:  access.ActionRetrieve,
			Objects: []resource.ID{devices[i].ResourceID()},
		})
		// Create access does not grant read access.
		if errors.Is(err, access.ErrDenied) {
			devices[i].Status = nil
			continue
		}
		if err != nil {
			return err
		}
		if err := s.attachStatuses(ctx, tx, devices[i:i+1]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) attachStatuses(
	ctx context.Context,
	snap gorp.Snapshot,
	devices []device.Device,
) error {
	statuses, err := s.status.Join[device.StatusDetails](
		ctx,
		snap,
		device.ResourceIDsFromDevices(devices),
	)
	if err != nil {
		return err
	}
	for i, st := range statuses {
		devices[i].Status = st
	}
	return nil
}

// DispatchRequest carries an action sequence to apply to a single device. DispatchKey
// is a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[device.Key, device.Action]

// Dispatch applies the action sequence to the target device atomically. Subscribers to
// the device action channel receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.device.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
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
	q, err := ApplyRetrieve(req, s.device.NewRetrieve().Entries(&res.Devices))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, device.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeStatus {
		if err := s.attachStatuses(ctx, snap, res.Devices); err != nil {
			return RetrieveResponse{}, err
		}
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []device.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the devices, their statuses, and the devices they hold.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	w := s.device.NewWriter(tx)
	for _, k := range req.Keys {
		if err := w.Delete(ctx, k); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}
