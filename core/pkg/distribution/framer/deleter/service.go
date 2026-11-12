// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package deleter deletes time ranges of data from channels.
package deleter

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Service is the distribution layer interface for deleting time ranges of data from
// channels.
type Service struct{ cfg ServiceConfig }

// ServiceConfig is the configuration for the Service.
type ServiceConfig struct {
	// TS is the storage-layer time-series database to delete data from.
	//
	// [REQUIRED]
	TS *ts.DB
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate validates the ServiceConfig.
func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.deleter")
	v.NotNil("ts", c.TS)
	return v.Error()
}

// Override overrides the ServiceConfig with the other ServiceConfig.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.TS = override.Nil(c.TS, other.TS)
	return c
}

// NewService creates a new Service from the given ServiceConfig(s).
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// DeleteTimeRange deletes a time range in the specified channels. It is idempotent: if
// no data is found in the range, that channel is skipped. Free channels have no stored
// data and are skipped.
//
// It is NOT atomic: if any deletion fails after others have succeeded, the operation is
// abandoned midway.
func (s *Service) DeleteTimeRange(
	ctx context.Context,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	stored := channel.Keys(lo.Reject(keys, func(k channel.Key, _ int) bool {
		return k.Free()
	}))
	return s.cfg.TS.DeleteTimeRange(ctx, stored.Storage(), tr)
}
