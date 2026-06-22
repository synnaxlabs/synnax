// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package task implements the driver task and factory infrastructure for
// executing compiled Arc programs. It is split out from arc/runtime so that
// arc/runtime (the core the calculation compiler depends on) does not import
// the arc service package, which would otherwise form an import cycle once the
// arc service depends on the action-dispatch stack.
package task

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Type is the type identifier for Arc tasks.
const Type = "arc"

// Config is the configuration for an Arc task.
type Config struct {
	alamos.Instrumentation
	// ArcKey is the UUID of the Arc program to execute.
	ArcKey arc.Key `json:"arc_key"`
	// AutoStart sets whether the task should start automatically when configured.
	AutoStart bool `json:"auto_start"`
}

// GetProgramFunc retrieves an Arc with its compiled Program by key.
type GetProgramFunc func(context.Context, arc.Key) (arc.Arc, error)

// FactoryConfig is the configuration for creating an Arc factory.
type FactoryConfig struct {
	// Channel is used for retrieving channel information.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer is used for reading/writing telemetry.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status is used by the Arc status module.
	//
	// [REQUIRED]
	Status *status.Service
	// GetProgram retrieves an Arc with its compiled Program by key.
	//
	// [REQUIRED]
	GetProgram GetProgramFunc
	// Ranger is used by the Arc ranges module to create and update ranges.
	//
	// [REQUIRED]
	Ranger *ranger.Service
	alamos.Instrumentation
}

var (
	_                    config.Config[FactoryConfig] = FactoryConfig{}
	DefaultFactoryConfig                              = FactoryConfig{}
)

func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Status = override.Nil(c.Status, other.Status)
	c.GetProgram = override.Nil(c.GetProgram, other.GetProgram)
	c.Ranger = override.Nil(c.Ranger, other.Ranger)
	return c
}

func (c FactoryConfig) Validate() error {
	v := validate.New("arc.task.factory")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "get_program", c.GetProgram)
	validate.NotNil(v, "ranger", c.Ranger)
	return v.Error()
}

type factory struct{ cfg FactoryConfig }

var _ driver.Factory = (*factory)(nil)

// NewFactory creates a new Arc factory.
func NewFactory(cfgs ...FactoryConfig) (driver.Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &factory{cfg: cfg}, nil
}

func (f *factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
) (driver.Task, error) {
	if t.Type != Type {
		return nil, driver.ErrTaskNotHandled
	}
	var cfg Config
	if err := t.Config.Unmarshal(&cfg); err != nil {
		f.setConfigStatus(ctx, t, status.VariantError, err.Error())
		return nil, err
	}
	prog, err := f.cfg.GetProgram(ctx, cfg.ArcKey)
	if err != nil {
		f.setConfigStatus(ctx, t, status.VariantError, err.Error())
		return nil, err
	}
	arcTask := &impl{
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		prog:       prog,
	}
	if cfg.AutoStart {
		if err := arcTask.Exec(ctx, task.Command{Type: "start"}); err != nil {
			return nil, err
		}
	} else {
		f.setConfigStatus(
			ctx, t, status.VariantSuccess, "Task configured successfully",
		)
	}
	return arcTask, nil
}

func (f *factory) setConfigStatus(
	ctx context.Context,
	t task.Task,
	variant status.Variant,
	message string,
) {
	stat := task.Status{
		Key:     task.OntologyID(t.Key).String(),
		Name:    t.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{
			Task:    t.Key,
			Running: false,
		},
	}
	if err := status.
		NewWriter[task.StatusDetails](f.cfg.Status, nil).
		Set(ctx, &stat); err != nil {
		f.cfg.L.Error(
			"failed to set configuration status for task",
			zap.Uint64("key", uint64(t.Key)),
			zap.String("name", t.Name),
			zap.Error(err),
		)
	}
}

func (f *factory) Name() string { return "arc" }
