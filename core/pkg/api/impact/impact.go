// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package impact serves the plan and apply routes of changes to channels.
package impact

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the impact API service.
type ServiceConfig struct {
	// DB reads the policies that decide which records a subject may see.
	//
	// [REQUIRED]
	DB *gorp.DB
	// RBAC checks which records a subject may see.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Impact plans and applies changes to channels.
	//
	// [REQUIRED]
	Impact *impact.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the impact API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Impact = override.Nil(c.Impact, other.Impact)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.impact")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("impact", c.Impact)
	return v.Error()
}

// Service serves the plan and apply routes.
type Service struct{ cfg ServiceConfig }

// NewService opens a Service from cfgs.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// PlanRequest asks what a change would do.
type PlanRequest struct {
	// Change is the change to plan.
	Change impact.Change `json:"change" msgpack:"change"`
}

// Named implements bind.Named: the subject needs retrieve on each channel it names.
func (r PlanRequest) Named() []resource.ID { return named(r.Change) }

// ApplyRequest carries out a change with the caller's picks.
type ApplyRequest struct {
	// Change is the change to apply.
	Change impact.Change `json:"change" msgpack:"change"`
	// Token is the token of the plan that the caller saw. Empty skips the check.
	Token string `json:"token" msgpack:"token"`
	// Picks maps the text form of the record of a visible entry to the pick for it.
	Picks map[string]impact.Choice `json:"picks" msgpack:"picks"`
	// Hidden is the pick for every entry that the caller cannot see.
	Hidden *impact.Choice `json:"hidden,omitzero" msgpack:"hidden,omitempty"`
}

func named(c impact.Change) []resource.ID {
	if d, ok := c.Variant.(impact.DeleteChannelsChange); ok {
		return channel.ResourceIDsFromKeys(d.Keys)
	}
	return nil
}

// Plan returns every record that the change touches, with an entry for each that the
// subject can retrieve and a count of the others.
func (s *Service) Plan(
	ctx context.Context,
	snap bind.Snapshot,
	req PlanRequest,
) (impact.Plan, error) {
	return s.cfg.Impact.Plan(ctx, snap, req.Change, impact.PlanOptions{
		Visible: snap.Allowed,
	})
}

// ApplyHandler returns the apply route on binder: it plans and applies the change and
// checks access on what it wrote, in one transaction. A denial on a record that the
// subject cannot retrieve names no record.
func (s *Service) ApplyHandler(
	binder *bind.Service,
) freighter.UnaryHandler[ApplyRequest, impact.Plan] {
	apply := bind.Write(binder, s.apply)
	return func(ctx context.Context, req ApplyRequest) (impact.Plan, error) {
		plan, err := apply(ctx, req)
		denied, ok := errors.AsType[access.DeniedError](err)
		if !ok {
			return plan, err
		}
		visible, vErr := s.cfg.RBAC.NewEnforcer(s.cfg.DB).
			Allowed(ctx, denied.Subject, access.ActionRetrieve)
		if vErr != nil {
			return impact.Plan{}, vErr
		}
		if visible(denied.Object) {
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

func (s *Service) apply(
	ctx context.Context,
	tx gorp.Tx,
	req ApplyRequest,
) (impact.Plan, error) {
	visible, err := s.cfg.RBAC.NewEnforcer(tx).
		Allowed(ctx, auth.GetSubject(ctx), access.ActionRetrieve)
	if err != nil {
		return impact.Plan{}, err
	}
	return s.cfg.Impact.Apply(ctx, tx, req.Change, impact.ApplyOptions{
		Token:   req.Token,
		Picks:   req.Picks,
		Hidden:  req.Hidden,
		Visible: visible,
	})
}
