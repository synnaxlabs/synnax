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
	v.NotNil("status", c.Status)
	v.NotNil("sender", c.Sender)
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
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	if t.Type != AlertTaskType {
		return nil, driver.ErrTaskNotHandled
	}
	var cfg TaskConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		if cmdKey == driver.NoCommand {
			f.cfg.L.Warn("failed to configure task",
				zap.Stringer("task", t),
				zap.Error(err),
			)
		} else {
			f.setConfigStatus(ctx, t, cmdKey, status.VariantError, err.Error())
		}
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		if cmdKey == driver.NoCommand && !cfg.AutoStart {
			f.cfg.L.Warn("failed to configure task",
				zap.Stringer("task", t),
				zap.Error(err),
			)
		} else {
			f.setConfigStatus(ctx, t, cmdKey, status.VariantError, err.Error())
		}
		return nil, err
	}
	pdTask := &alertTask{
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		status:     driver.NewStatusHandler(f.cfg.Status, t),
	}
	// A successful configure writes no status: the start that follows it answers the
	// command, and a "configured" status would answer it first with running false.
	if cfg.AutoStart {
		if err := pdTask.start(ctx, driver.NoCommand); err != nil {
			return nil, err
		}
	}
	return pdTask, nil
}

func (f *factory) setConfigStatus(
	ctx context.Context,
	t task.Task,
	cmdKey string,
	variant status.Variant,
	message string,
) {
	details := task.NewStatusDetails(t, false)
	details.Cmd = cmdKey
	stat := task.Status{
		Key:     t.OntologyID().String(),
		Name:    t.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: details,
	}
	if err := f.cfg.Status.NewWriter(nil).
		Set(ctx, &stat); err != nil {
		f.cfg.L.Error(
			"failed to set configuration status",
			zap.Stringer("task", t),
			zap.Stringer("status", stat),
			zap.Error(err),
		)
	}
}

func (f *factory) Name() string { return "pagerduty" }
