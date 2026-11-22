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
	"slices"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
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
	// RBAC checks that the subject of a save can read the channels it reads.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Calculation = override.Nil(c.Calculation, other.Calculation)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.calculation")
	v.NotNil("calculation", c.Calculation)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service is the API for calculations.
type Service struct {
	internal *calculation.Service
	status   *status.Service
	access   *rbac.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Calculation,
		status:   cfg.Status,
		access:   cfg.RBAC,
	}, nil
}

// Input is a calculation to create, with the output to make for it.
type Input struct {
	// Calculation is the calculation to create. When its output is 0, the create makes
	// a virtual output for it, typed by its expression.
	Calculation calculation.Calculation `json:"calculation" msgpack:"calculation"`
	// Name is the name of the output to make. Set it when, and only when, the output of
	// Calculation is 0.
	Name string `json:"name" msgpack:"name"`
	// Group is the group to place the made output in, or nil for the root. Set it only
	// when the output of Calculation is 0.
	Group *group.Key `json:"group" msgpack:"group"`
}

// CreateRequest holds the calculations to create.
type CreateRequest struct {
	// Calculations are the calculations to create.
	Calculations []Input `json:"calculations" msgpack:"calculations"`
	// DispatchKey is stamped onto the create broadcasts so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

// CreateResponse holds the created calculations.
type CreateResponse struct {
	// Calculations are the created calculations, each with its key.
	Calculations []calculation.Calculation `json:"calculations" msgpack:"calculations"`
}

// Create creates the calculations, stopping at the first error. It returns a
// validation error when a calculation is not valid, when its output is not a virtual
// channel that no other record writes, or when an input both names an output and sets
// the fields of one to make.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.internal.NewWriter(tx).WithDispatchKey(req.DispatchKey)
	created := make([]calculation.Calculation, len(req.Calculations))
	outputs := make(channel.Keys, len(req.Calculations))
	for i, in := range req.Calculations {
		c := in.Calculation
		var err error
		switch {
		case c.Output == 0:
			_, err = w.CreateOutput(ctx, &c, in.Name, in.Group)
		case in.Name != "":
			err = madeOutputError("name")
		case in.Group != nil:
			err = madeOutputError("group")
		default:
			err = w.Create(ctx, &c)
		}
		if err != nil {
			return CreateResponse{}, validate.PathedError(
				err,
				"calculations",
				strconv.Itoa(i),
			)
		}
		created[i], outputs[i] = c, c.Output
	}
	if err := s.enforceInputs(ctx, tx, outputs...); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Calculations: created}, nil
}

// madeOutputError returns the error for an input that names its output and also sets
// field, a field of an output to make.
func madeOutputError(field string) error {
	return validate.PathedError(errors.Wrap(
		validate.ErrValidation,
		"only a calculation that names no output makes one",
	), field)
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
	named := req.Named()
	allowed := snap.Allowed
	if err = q.Where(calculation.Match(func(
		_ gorp.Context,
		_ calculation.Retrieve,
		c *calculation.Calculation,
	) (bool, error) {
		id, denied := s.internal.Denied(*c, allowed)
		if !denied {
			return true, nil
		}
		if len(named) > 0 && slices.Contains(named, calculation.ResourceID(c.Key)) {
			return false, snap.Enforce(id)
		}
		return false, nil
	})).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeStatus {
		if err = s.attachStatuses(ctx, snap, res.Calculations); err != nil {
			return RetrieveResponse{}, err
		}
	}
	return res, nil
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

// DispatchResponse holds the actions that the Core adds to a dispatch, such as a set of
// the reads that a new expression changed.
type DispatchResponse = actions.DispatchResponse[calculation.Action]

// Dispatch applies the actions to the calculation atomically. Subscribers to the
// calculation action signals receive them after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (DispatchResponse, error) {
	computed, err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
	if err != nil {
		return DispatchResponse{}, err
	}
	var c calculation.Calculation
	if err = s.internal.NewRetrieve().
		Where(calculation.MatchKeys(req.Key)).
		Entry(&c).
		Exec(ctx, tx); err != nil {
		return DispatchResponse{}, err
	}
	if err = s.enforceInputs(ctx, tx, c.Output); err != nil {
		return DispatchResponse{}, err
	}
	return DispatchResponse{Actions: computed}, nil
}

// enforceInputs fails when the subject of ctx cannot read a channel that the
// calculations of outputs read, as tx sees them.
func (s *Service) enforceInputs(
	ctx context.Context,
	tx gorp.Tx,
	outputs ...channel.Key,
) error {
	inputs, err := s.internal.Inputs(ctx, tx, outputs, nil)
	if err != nil {
		return err
	}
	return s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: framer.ResourceIDs(inputs),
	})
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
