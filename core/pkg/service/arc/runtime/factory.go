// Copyright 2025 Synnax Labs, Inc.
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
	godriver2 "github.com/synnaxlabs/synnax/pkg/driver/go"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
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

// Factory creates Arc tasks from task definitions.
type Factory struct {
	cfg FactoryConfig
}

// NewFactory creates a new Arc factory.
func NewFactory(cfg FactoryConfig) *Factory {
	return &Factory{cfg: cfg}
}

// ConfigureTask creates an Arc task if this factory handles the task type.
func (f *Factory) ConfigureTask(ctx godriver2.Context, t task.Task) (godriver2.Task, bool, error) {
	if t.Type != TaskType {
		return nil, false, nil
	}
	var cfg TaskConfig
	if err := json.Unmarshal([]byte(t.Config), &cfg); err != nil {
		return nil, true, err
	}
	// Use injected function to get Arc with compiled Module
	prog, err := f.cfg.GetModule(ctx, cfg.ArcKey)
	if err != nil {
		return nil, true, err
	}
	arcTask := newTask(t.Key, prog, cfg, ctx, f.cfg)
	return arcTask, true, nil
}

// Name returns the factory name.
func (f *Factory) Name() string { return "arc" }
