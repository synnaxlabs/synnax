// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
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

// TaskType is the type identifier for Arc tasks.
const TaskType = "arc"

// TaskConfig is the configuration for an Arc taskImpl.
type TaskConfig struct {
	alamos.Instrumentation
	// ArcKey is the UUID of the Arc program to execute.
	ArcKey uuid.UUID `json:"arc_key"`
	// AutoStart sets whether the taskImpl should start automatically when configured.
	AutoStart bool `json:"auto_start"`
}

// GetModuleFunc retrieves an Arc with its compiled Module by key.
type GetModuleFunc func(ctx context.Context, key uuid.UUID) (arc.Arc, error)

// FactoryConfig is the configuration for creating an Arc factory.
type FactoryConfig struct {
	// Channel is used for retrieving channel information.
	// [REQUIRED]
	Channel *channel.Service
	// Framer is used for reading/writing telemetry.
	// [REQUIRED]
	Framer *framer.Service
	// Status is used for Arc graph nodes (set_status) to update statuses.
	// [REQUIRED]
	Status *status.Service
	// GetModule retrieves an Arc with its compiled Module by key.
	// [REQUIRED]
	GetModule GetModuleFunc
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
	c.GetModule = override.Nil(c.GetModule, other.GetModule)
	return c
}

func (c FactoryConfig) Validate() error {
	v := validate.New("arc.runtime.factory")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "get_module", c.GetModule)
	return v.Error()
}

// Factory creates Arc tasks from taskImpl definitions.
type Factory struct {
	cfg FactoryConfig
}

var _ driver.Factory = (*Factory)(nil)

// NewFactory creates a new Arc factory.
func NewFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Factory{cfg: cfg}, nil
}

// ConfigureTask creates an Arc taskImpl if this factory handles the taskImpl type.
func (f *Factory) ConfigureTask(
	ctx driver.Context,
	t task.Task,
) (driver.Task, bool, error) {
	if t.Type != TaskType {
		return nil, false, nil
	}
	var cfg TaskConfig
	if err := json.Unmarshal([]byte(t.Config), &cfg); err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, true, err
	}
	prog, err := f.cfg.GetModule(ctx, cfg.ArcKey)
	if err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, true, err
	}
	arcTask := &taskImpl{
		ctx:        ctx,
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		prog:       prog,
	}
	if cfg.AutoStart {
		if err := arcTask.Exec(ctx, task.Command{Type: "start"}); err != nil {
			return arcTask, true, err
		}
	} else {
		f.setConfigStatus(ctx, t, xstatus.VariantSuccess, "Task configured successfully")
	}
	return arcTask, true, nil
}

func (f *Factory) setConfigStatus(ctx driver.Context, t task.Task, variant xstatus.Variant, message string) {
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
	if err := ctx.SetStatus(stat); err != nil {
		f.cfg.L.Error(
			"failed to set configuration status for task",
			zap.Uint64("key", uint64(t.Key)),
			zap.String("name", t.Name),
			zap.Error(err),
		)
	}
}

// Name returns the factory name.
func (f *Factory) Name() string { return "Arc" }
