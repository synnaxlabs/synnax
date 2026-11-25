// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package impact serves the plan and apply routes of the delete of records.
package impact

import (
	"context"
	"slices"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the impact API service.
type ServiceConfig struct {
	// RBAC checks which records a subject may see.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Impact plans and applies the delete of records.
	//
	// [REQUIRED]
	Impact *impact.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Impact = override.Nil(c.Impact, other.Impact)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.impact")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("impact", c.Impact)
	return v.Error()
}

// Service serves the plan and apply routes.
type Service struct{ cfg ServiceConfig }

// NewService opens a Service from cfgs.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// Named implements bind.Named: the subject needs retrieve on each record it names.
func (r PlanRequest) Named() []resource.ID { return named(r.Change) }

// named returns the records that c names.
func named(c Change) []resource.ID {
	if d, ok := c.Variant.(DeleteChange); ok {
		return d.Records
	}
	return nil
}

// deleted returns the records that c deletes. Any other change fails with a
// validation error.
func deleted(c Change) ([]resource.ID, error) {
	d, ok := c.Variant.(DeleteChange)
	if !ok {
		return nil, errors.Wrap(validate.ErrValidation, "unknown change")
	}
	return d.Records, nil
}

// Plan returns every record that the change touches with the subject's picks, with an
// entry for each that the subject can retrieve and a count of the others.
func (s *Service) Plan(
	ctx context.Context,
	snap bind.Snapshot,
	req PlanRequest,
) (impact.Plan, error) {
	records, err := deleted(req.Change)
	if err != nil {
		return impact.Plan{}, err
	}
	return s.cfg.Impact.Plan(ctx, snap, records, impact.PlanOptions{
		Picks:   req.Picks,
		Hidden:  req.Hidden,
		Visible: snap.Allowed,
	})
}

// ApplyHandler returns the apply route on binder: it plans and applies the change and
// checks access on what it wrote, in one transaction. As the plan route does, it
// refuses a change that names a record the subject cannot retrieve, before it plans. A
// denial on a record that the subject cannot retrieve and did not name names no record.
func (s *Service) ApplyHandler(
	binder *bind.Service,
) freighter.UnaryHandler[ApplyRequest, impact.Plan] {
	return func(ctx context.Context, req ApplyRequest) (impact.Plan, error) {
		var visible func(resource.ID) bool
		apply := func(
			ctx context.Context,
			tx gorp.Tx,
			req ApplyRequest,
		) (impact.Plan, error) {
			records, err := deleted(req.Change)
			if err != nil {
				return impact.Plan{}, err
			}
			subject := auth.GetSubject(ctx)
			visible, err = s.cfg.RBAC.NewEnforcer(tx).
				Allowed(ctx, subject, access.ActionRetrieve)
			if err != nil {
				return impact.Plan{}, err
			}
			for _, r := range records {
				if !visible(r) {
					return impact.Plan{}, access.Denied(
						subject,
						access.ActionRetrieve,
						r,
					)
				}
			}
			return s.cfg.Impact.Apply(ctx, tx, records, impact.ApplyOptions{
				Token:   req.Token,
				Picks:   req.Picks,
				Hidden:  req.Hidden,
				Visible: visible,
			})
		}
		plan, err := bind.Write(binder, apply)(ctx, req)
		denied, ok := errors.AsType[access.DeniedError](err)
		if !ok || visible == nil || visible(denied.Object) ||
			slices.Contains(named(req.Change), denied.Object) {
			return plan, err
		}
		return impact.Plan{}, errors.Wrapf(
			access.ErrDenied,
			"%s cannot %s a record that it cannot see",
			denied.Subject,
			denied.Action,
		)
	}
}
