// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package impact plans each change to channels: every record that the change touches,
// the picks for each, and the default. It applies a change with the caller's picks.
package impact

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening an impact service.
type ServiceConfig struct {
	// Channel is the service of the channels that a change touches.
	//
	// [REQUIRED]
	Channel *channel.Service
	// RecordOf returns the record that the holder of a reference stands for.
	//
	// [REQUIRED]
	RecordOf func(holder gorp.Target) resource.ID
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.RecordOf = override.Nil(c.RecordOf, other.RecordOf)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("impact")
	v.NotNil("channel", c.Channel)
	v.NotNil("record_of", c.RecordOf)
	return v.Error()
}

// Service plans and applies changes to channels.
type Service struct{ cfg ServiceConfig }

// OpenService opens a Service from cfgs.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// Plan returns every record that c touches, as seen by snap, with the picks for each.
// It writes nothing.
func (s *Service) Plan(
	ctx context.Context,
	snap gorp.Snapshot,
	c Change,
) (Plan, error) {
	keys, err := deletedKeys(c)
	if err != nil {
		return Plan{}, err
	}
	p, err := s.planDelete(ctx, snap, keys)
	return p.Plan, err
}

// deletedKeys returns the keys of the channels that c deletes.
func deletedKeys(c Change) ([]channel.Key, error) {
	d, ok := c.Variant.(DeleteChannelsChange)
	if !ok {
		return nil, errors.Wrapf(validate.ErrValidation, "unknown change %T", c.Variant)
	}
	return d.Keys, nil
}
