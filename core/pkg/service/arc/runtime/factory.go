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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// TaskType is the type identifier for Arc tasks.
const TaskType = "arc"

// TaskConfig is the configuration for an Arc task.
type TaskConfig struct {
	// ArcKey is the UUID of the Arc program to execute.
	ArcKey uuid.UUID `json:"arc_key"`
	// AutoStart sets whether the task should start automatically when configured.
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
}

var (
	_             config.Config[FactoryConfig] = FactoryConfig{}
	DefaultConfig                              = FactoryConfig{}
)

func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
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

// Factory creates Arc tasks from task definitions.
type Factory struct {
	cfg FactoryConfig
}

// NewFactory creates a new Arc factory.
func NewFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Factory{cfg: cfg}, nil
}

// ConfigureTask creates an Arc task if this factory handles the task type.
func (f *Factory) ConfigureTask(
	ctx driver.Context,
	t task.Task,
) (driver.Task, bool, error) {
	if t.Type != TaskType {
		return nil, false, nil
	}
	var cfg TaskConfig
	if err := json.Unmarshal([]byte(t.Config), &cfg); err != nil {
		return nil, true, err
	}
	prog, err := f.cfg.GetModule(ctx, cfg.ArcKey)
	if err != nil {
		return nil, true, err
	}
	arcTask := newTask(t.Key, prog, cfg, ctx, f.cfg)
	return arcTask, true, nil
}

// Name returns the factory name.
func (f *Factory) Name() string { return "Arc" }
