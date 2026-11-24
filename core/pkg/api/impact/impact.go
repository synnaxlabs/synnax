// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package impact serves the plan and apply routes of changes to channels and groups.
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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the impact API service.
type ServiceConfig struct {
	// RBAC checks which records a subject may see.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Impact plans and applies changes to channels and groups.
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

// Named implements bind.Named: the subject needs retrieve on each channel or group it
// names. A request with no change names nothing, and the plan refuses it.
func (r PlanRequest) Named() []resource.ID { return named(r.Change) }

func named(c impact.Change) []resource.ID {
	switch v := c.Variant.(type) {
	case nil:
		return nil
	case impact.DeleteChannelsChange:
		return channel.ResourceIDsFromKeys(v.Keys)
	case impact.DeleteGroupsChange:
		return group.ResourceIDs(v.Keys)
	}
	// A variant that this switch misses would plan with no access check.
	panic("[api.impact] - named covers no such change")
}

// Plan returns every record that the change touches with the subject's picks, with an
// entry for each that the subject can retrieve and a count of the others.
func (s *Service) Plan(
	ctx context.Context,
	snap bind.Snapshot,
	req PlanRequest,
) (impact.Plan, error) {
	return s.cfg.Impact.Plan(ctx, snap, req.Change, impact.PlanOptions{
		Picks:   req.Picks,
		Hidden:  req.Hidden,
		Visible: snap.Allowed,
	})
}

// ApplyHandler returns the apply route on binder: it plans and applies the change and
// checks access on what it wrote, in one transaction. A denial counts the records that
// the subject cannot retrieve and names none of them.
func (s *Service) ApplyHandler(
	binder *bind.Service,
) freighter.UnaryHandler[ApplyRequest, impact.Plan] {
	apply := func(
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
	return bind.Write(binder, apply, bind.HideDenied("delete"))
}
