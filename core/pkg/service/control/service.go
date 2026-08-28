// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig configures the service-layer control service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Control is the distribution-layer control service that reads and observes control
	// state across the cluster.
	//
	// [REQUIRED]
	Control *control.Service
	// Signals opens the publisher that writes updates to the control channel.
	//
	// [REQUIRED]
	Signals *signals.Provider
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("control")
	v.NotNil("control", c.Control)
	v.NotNil("signals", c.Signals)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Control = override.Nil(c.Control, other.Control)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Service answers control state reads and publishes every control transfer in the
// cluster on the sy_control channel.
//
// The Service must be closed after use.
type Service struct {
	cfg      ServiceConfig
	shutdown io.Closer
}

// OpenService opens the control service using the provided configuration(s). Fields
// defined in each subsequent configuration override those in previous configurations.
//
// If the returned error is nil, the Service must be closed after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.shutdown, err = cfg.Signals.PublishJSON(
		ctx,
		signals.JSONPublisherConfig[Update]{
			Observable: cfg.Control,
			SetName:    channelName,
		},
	); err != nil {
		return nil, err
	}
	return s, nil
}

// Retrieve returns the control state of the given channels from across the cluster.
// Passing no keys returns the state of every controlled channel. Channels that no
// subject controls are absent from the result.
func (s *Service) Retrieve(
	ctx context.Context,
	keys ...channel.Key,
) ([]State, error) {
	return s.cfg.Control.Retrieve(ctx, keys...)
}

// Close stops publishing updates and releases the service's resources.
func (s *Service) Close() error { return s.shutdown.Close() }
