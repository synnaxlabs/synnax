// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig configures the distribution-layer channel allocator.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// HostKey is the key of the host node. Channels leased to it get their keys from
	// its counter.
	//
	// [REQUIRED]
	HostKey node.Key
	// KV is the key-value store used to back the local-key counters.
	//
	// [REQUIRED]
	KV kv.ReadWriter
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	v.NonZero("host_key", c.HostKey)
	v.NotNil("kv", c.KV)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.HostKey = override.Numeric(c.HostKey, other.HostKey)
	c.KV = override.Nil(c.KV, other.KV)
	return c
}

// Service is the distribution-layer channel allocator. It assigns local channel keys.
type Service struct {
	cfg           ServiceConfig
	leasedCounter *counter
	freeCounter   *counter
}

// NewService opens the distribution-layer channel service.
func NewService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	leasedCounterKey := []byte(
		cfg.HostKey.String() + ".distribution.channel.leasedCounter",
	)
	if s.leasedCounter, err = newCounter(ctx, cfg.KV, leasedCounterKey); err != nil {
		return nil, err
	}
	freeCounterKey := []byte(
		cfg.HostKey.String() + ".distribution.channel.counter.free",
	)
	if s.freeCounter, err = newCounter(ctx, cfg.KV, freeCounterKey); err != nil {
		return nil, err
	}
	return s, nil
}
