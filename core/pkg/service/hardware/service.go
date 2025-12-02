// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package hardware

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/tracker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening the hardware service.
type Config struct {
	alamos.Instrumentation
	// DB is the gorp database that all meta-data structures will be stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between hardware resources in the cluster.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create hardware related groups of ontology resources.
	// [REQUIRED]
	Group *group.Service
	// HostProvider is used to add cluster topology information to hardware resources.
	HostProvider cluster.HostProvider
	// Signals is used to propagate changes to meta-data throughout the cluster.
	Signals *signals.Provider
	// Channel is used to create channels necessary for hardware communication.
	Channel *channel.Service
	// Framer is used for writing hardware telemetry data.
	Framer *framer.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	return c
}

func (c Config) Validate() error {
	v := validate.New("hardware")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "host_provider", c.HostProvider)
	validate.NotNil(v, "signals", c.Signals)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "framer", c.Framer)
	return v.Error()
}

type Service struct {
	Rack   *rack.Service
	Task   *task.Service
	Device *device.Service
	CDC    *signals.Provider
	State  *tracker.Tracker
}

func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	rackSvc, err := rack.OpenService(ctx, rack.Config{
		DB:           cfg.DB,
		Ontology:     cfg.Ontology,
		Group:        cfg.Group,
		HostProvider: cfg.HostProvider,
		Signals:      cfg.Signals,
	})
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

	stateSvc, err := tracker.Open(ctx, tracker.Config{
		Instrumentation: cfg.Child("tracker"),
		DB:              cfg.DB,
		Rack:            rackSvc,
		Task:            taskSvc,
		Device:          deviceSvc,
		Signals:         cfg.Signals,
		HostProvider:    cfg.HostProvider,
		Channel:         cfg.Channel,
		Framer:          cfg.Framer,
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
	e.Exec(s.Rack.Close)
	e.Exec(s.Device.Close)
	e.Exec(s.Task.Close)
	e.Exec(s.State.Close)
	return e.Error()
}
