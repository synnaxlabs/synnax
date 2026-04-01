// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// FactoryConfig is the configuration for the PagerDuty factory.
type FactoryConfig struct {
	// Status is the status service for observing status changes.
	//
	// [REQUIRED]
	Status *status.Service
	// Sender is an optional EventSender for testability.
	//
	// [OPTIONAL] - Defaults to the real PagerDuty API.
	Sender EventSender
	alamos.Instrumentation
}

var _ config.Config[FactoryConfig] = FactoryConfig{}

// Override overrides the factory configuration with the given other configuration.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Status = override.Nil(c.Status, other.Status)
	c.Sender = override.Nil(c.Sender, other.Sender)
	return c
}

// Validate validates the factory configuration.
func (c FactoryConfig) Validate() error {
	v := validate.New("pagerduty.factory")
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "sender", c.Sender)
	return v.Error()
}

type factory struct{ cfg FactoryConfig }

var _ driver.Factory = (*factory)(nil)

// NewFactory creates a new PagerDuty factory.
func NewFactory(cfgs ...FactoryConfig) (driver.Factory, error) {
	cfg, err := config.New(FactoryConfig{Sender: defaultEventSender}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &factory{cfg: cfg}, nil
}

func (f *factory) ConfigureTask(
	ctx driver.Context,
	t task.Task,
) (driver.Task, error) {
	if t.Type != AlertTaskType {
		return nil, driver.ErrTaskNotHandled
	}
	var cfg AlertTaskConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}
	v := validate.New("pagerduty.alert_task_config")
	v.Ternary("routing_key", len(cfg.RoutingKey) != 32, "must be exactly 32 characters")
	var hasEnabled bool
	for _, a := range cfg.Alerts {
		if a.Enabled {
			hasEnabled = true
			break
		}
	}
	v.Ternary("alerts", !hasEnabled, "at least one alert must be enabled")
	if err := v.Error(); err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}
	pdTask := &alertTaskImpl{factoryCfg: f.cfg, task: t, cfg: cfg}
	if cfg.AutoStart {
		if err := pdTask.start(ctx); err != nil {
			return nil, err
		}
	} else {
		f.setConfigStatus(
			ctx, t, xstatus.VariantSuccess, "Task configured successfully",
		)
	}
	return pdTask, nil
}

func (f *factory) setConfigStatus(
	ctx context.Context,
	t task.Task,
	variant xstatus.Variant,
	message string,
) {
	stat := task.Status{
		Key:     task.OntologyID(t.Key).String(),
		Name:    t.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.Key, Running: false},
	}
	err := status.NewWriter[task.StatusDetails](f.cfg.Status, nil).
		Set(ctx, &stat)
	if err != nil {
		f.cfg.L.Error(
			"failed to set configuration status",
			zap.Stringer("task", t),
			zap.Stringer("status", stat),
			zap.Error(err),
		)
	}
}

func (f *factory) Name() string { return "pagerduty" }
