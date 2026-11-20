// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the calculation API service.
type ServiceConfig struct {
	// Calculation stores and retrieves calculations.
	//
	// [REQUIRED]
	Calculation *calculation.Service
	// Status joins the status of each calculation on retrieve.
	//
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Calculation = override.Nil(c.Calculation, other.Calculation)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.calculation")
	v.NotNil("calculation", c.Calculation)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service is the API for calculations.
type Service struct {
	internal *calculation.Service
	status   *status.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Calculation, status: cfg.Status}, nil
}

// CreateRequest holds the calculations to create.
type CreateRequest struct {
	// Calculations are the calculations to create.
	Calculations []calculation.Calculation `json:"calculations" msgpack:"calculations"`
	// DispatchKey is stamped onto the create broadcasts so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

// CreateResponse holds the created calculations.
type CreateResponse struct {
	// Calculations are the created calculations, each with its key.
	Calculations []calculation.Calculation `json:"calculations" msgpack:"calculations"`
}

// Create creates the calculations, stopping at the first error. It returns a
// validation error when a calculation is not valid or its output is not a virtual
// channel that no other record writes.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.internal.NewWriter(tx).WithDispatchKey(req.DispatchKey)
	for i := range req.Calculations {
		if err := w.Create(ctx, &req.Calculations[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse{Calculations: req.Calculations}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the calculations that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, calculation.ResourceID)
}

// Retrieve returns the calculations that req selects and that the subject can
// retrieve, with the output and each primary and reset channel they read.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Calculations))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(calculation.And(
		bind.Filter(snap, calculation.Match),
		calculation.Match(func(
			_ gorp.Context,
			_ calculation.Retrieve,
			c *calculation.Calculation,
		) (bool, error) {
			return channelsAllowed(snap, *c), nil
		}),
	)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeStatus {
		if err = s.attachStatuses(ctx, snap, res.Calculations); err != nil {
			return RetrieveResponse{}, err
		}
	}
	return res, nil
}

// channelsAllowed returns true when the subject can retrieve the output of c and each
// primary and reset channel of c.
func channelsAllowed(snap bind.Snapshot, c calculation.Calculation) bool {
	if !snap.Allowed(channel.ResourceID(c.Output)) {
		return false
	}
	if c.Primary != nil && !snap.Allowed(channel.ResourceID(*c.Primary)) {
		return false
	}
	for _, reset := range calculation.Resets(c) {
		if !snap.Allowed(channel.ResourceID(reset)) {
			return false
		}
	}
	return true
}

func (s *Service) attachStatuses(
	ctx context.Context,
	snap gorp.Snapshot,
	calculations []calculation.Calculation,
) error {
	ids := make([]resource.ID, len(calculations))
	for i, c := range calculations {
		ids[i] = calculation.ResourceID(c.Key)
	}
	statuses, err := s.status.Join[struct{}](ctx, snap, ids)
	if err != nil {
		return err
	}
	for i, st := range statuses {
		calculations[i].Status = st
	}
	return nil
}

// DispatchRequest carries an action sequence to apply to one calculation. DispatchKey
// is echoed on the broadcast so the sender recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[calculation.Key, calculation.Action]

// Dispatch applies the actions to the calculation atomically. Subscribers to the
// calculation action signals receive them after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

// DeleteRequest selects the calculations to delete.
type DeleteRequest struct {
	// Keys are the keys of the calculations to delete.
	Keys []calculation.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the calculations. A key that matches no calculation is ignored. The
// output of each one stays as a channel with no producer.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
