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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// TS is the storage-layer time-series database that arbitrates control over
	// channels.
	//
	// [REQUIRED]
	TS *ts.DB
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.control")
	v.NotNil("ts", c.TS)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.TS = override.Nil(c.TS, other.TS)
	return c
}

// Service reads and observes the control state of channels, which the storage engine
// arbitrates.
//
// The Service must be closed after use.
type Service struct {
	cfg     ServiceConfig
	updates observe.Observer[Update]
	// disconnectStorage stops the storage subscription.
	disconnectStorage observe.Disconnect
}

// OpenService opens a new Service using the provided configuration(s). Fields defined
// in each subsequent configuration override those in previous configurations.
//
// If the returned error is nil, the Service must be closed after use.
func OpenService(_ context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, updates: observe.New[Update]()}
	s.disconnectStorage = cfg.TS.OnControlUpdate(
		func(ctx context.Context, u ts.ControlUpdate) {
			s.updates.Notify(ctx, updateFromStorage(u))
		},
	)
	return s, nil
}

// Retrieve returns the control state of the given channels. Passing no keys returns the
// state of every controlled channel. Channels that no subject controls are absent from
// the result.
func (s *Service) Retrieve(_ context.Context, keys ...channel.Key) ([]State, error) {
	states := statesFromStorage(s.cfg.TS.ControlStates())
	if len(keys) == 0 {
		return states, nil
	}
	requested := set.New(keys...)
	out := make([]State, 0, len(keys))
	for _, state := range states {
		if requested.Contains(state.Resource) {
			out = append(out, state)
		}
	}
	return out, nil
}

// OnChange registers handler to receive every control transfer that storage arbitrates.
// The returned function unregisters the handler.
func (s *Service) OnChange(
	handler func(context.Context, Update),
) observe.Disconnect {
	return s.updates.OnChange(handler)
}

// Close stops the Service.
func (s *Service) Close() error {
	s.disconnectStorage()
	return nil
}
