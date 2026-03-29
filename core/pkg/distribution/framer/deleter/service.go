// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package deleter provides a service for deleting data from a Synnax cluster through
// deleting certain time ranges from channels.
package deleter

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Service is the distribution layer interface for deleting data from a Synnax cluster
// through deleting certain time ranges from channels.
type Service struct{ proxy *leaseProxy }

// ServiceConfig is the configuration for the Service.
type ServiceConfig struct {
	HostResolver cluster.HostResolver
	TSChannel    *ts.DB
	Transport    Transport
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate validates the ServiceConfig.
func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.deleter")
	validate.NotNil(v, "host_resolver", c.HostResolver)
	validate.NotNil(v, "ts_channel", c.TSChannel)
	validate.NotNil(v, "transport", c.Transport)
	return v.Error()
}

// Override overrides the ServiceConfig with the other ServiceConfig.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	return c
}

// NewService creates a new Service from the given ServiceConfig(s).
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	proxy, err := newLeaseProxy(cfg)
	if err != nil {
		return nil, err
	}
	cfg.Transport.Server().BindHandler(func(ctx context.Context, req Request) (types.Nil, error) {
		return types.Nil{}, cfg.TSChannel.DeleteTimeRange(ctx, req.Keys.Storage(), req.Bounds)
	})
	return &Service{proxy: proxy}, nil
}

// DeleteTimeRange deletes a time range in the specified channels. It is idempotent: if
// no data is found in the range, that channel is skipped.
//
// It is NOT atomic: if any deletion fails after others have succeeded, the operation is
// abandoned midway.
func (s *Service) DeleteTimeRange(
	ctx context.Context,
	keys channel.Keys,
	tr telem.TimeRange,
) error {
	return s.proxy.deleteTimeRange(ctx, keys, tr)
}
