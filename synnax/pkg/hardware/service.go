/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package hardware

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/hardware/state"
	"github.com/synnaxlabs/synnax/pkg/hardware/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
)

type Config = rack.Config

var DefaultConfig = rack.DefaultConfig

type Service struct {
	Rack   *rack.Service
	Task   *task.Service
	Device *device.Service
	CDC    *signals.Provider
	State  *state.Tracker
}

func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New[Config](DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	rackSvc, err := rack.OpenService(ctx, cfg)
	if err != nil {
		return nil, err
	}

	deviceSvc, err := device.OpenService(ctx, device.Config{
		DB:       cfg.DB,
		Ontology: cfg.Ontology,
		Group:    cfg.Group,
		Signals:  cfg.Signals,
	})
	if err != nil {
		return nil, err
	}

	taskSvc, err := task.OpenService(ctx, task.Config{
		DB:           cfg.DB,
		Ontology:     cfg.Ontology,
		Group:        cfg.Group,
		Rack:         rackSvc,
		Signals:      cfg.Signals,
		Channel:      cfg.Channel,
		HostProvider: cfg.HostProvider,
	})
	if err != nil {
		return nil, err
	}

	stateSvc, err := state.OpenTracker(ctx, state.TrackerConfig{
		Instrumentation: cfg.Instrumentation,
		DB:              cfg.DB,
		Rack:            rackSvc,
		Task:            taskSvc,
		Signals:         cfg.Signals,
		HostProvider:    cfg.HostProvider,
		Channels:        cfg.Channel,
	})
	if err != nil {
		return nil, err
	}

	svc := &Service{
		Rack:   rackSvc,
		Task:   taskSvc,
		Device: deviceSvc,
		State:  stateSvc,
	}
	return svc, nil
}

func (s *Service) Close() error {
	e := errors.NewCatcher(errors.WithAggregation())
	e.Exec(s.Device.Close)
	e.Exec(s.Task.Close)
	e.Exec(s.State.Close)
	return e.Error()
}
