// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	arclib "github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	godriver "github.com/synnaxlabs/synnax/pkg/service/driver/go"
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

// Factory creates Arc tasks from task definitions.
type Factory struct {
	svc *Service
}

// NewFactory creates a new Arc factory.
func NewFactory(svc *Service) *Factory {
	return &Factory{svc: svc}
}

// ConfigureTask creates an Arc task if this factory handles the task type.
func (f *Factory) ConfigureTask(ctx context.Context, t task.Task) (godriver.Task, bool, error) {
	if t.Type != TaskType {
		return nil, false, nil
	}

	var cfg TaskConfig
	if err := json.Unmarshal([]byte(t.Config), &cfg); err != nil {
		return nil, true, err
	}

	// Retrieve the Arc program
	var prog Arc
	if err := f.svc.NewRetrieve().WhereKeys(cfg.ArcKey).Entry(&prog).Exec(ctx, nil); err != nil {
		return nil, true, err
	}

	// Compile the Arc program
	mod, err := arclib.CompileGraph(ctx, prog.Graph, arclib.WithResolver(f.svc.symbolResolver))
	if err != nil {
		return nil, true, err
	}

	// Create the task
	arcTask := newTask(t.Key, prog, mod, cfg, f.svc.cfg)
	return arcTask, true, nil
}

// Name returns the factory name.
func (f *Factory) Name() string { return "arc" }

// newTask creates a new Arc task with the given parameters.
func newTask(key task.Key, prog Arc, mod arclib.Module, cfg TaskConfig, svcCfg ServiceConfig) *Task {
	return &Task{
		key:    key,
		prog:   prog,
		mod:    mod,
		cfg:    cfg,
		svcCfg: svcCfg,
	}
}

// Task wraps an Arc runtime and implements the godriver.Task interface.
type Task struct {
	key     task.Key
	prog    Arc
	mod     arclib.Module
	cfg     TaskConfig
	svcCfg  ServiceConfig
	runtime *runtime.Runtime
	running bool
}

// Exec handles commands (start, stop, etc.)
func (t *Task) Exec(ctx context.Context, cmd godriver.Command) error {
	switch cmd.Type {
	case "start":
		return t.start(ctx)
	case "stop":
		return t.stop(ctx, false)
	default:
		return nil
	}
}

func (t *Task) start(ctx context.Context) error {
	if t.running {
		return nil
	}

	baseCfg := runtime.Config{
		Channel: t.svcCfg.Channel,
		Framer:  t.svcCfg.Framer,
		Status:  t.svcCfg.Status,
		Module:  t.mod,
		Name:    t.prog.Name,
	}

	r, err := runtime.Open(ctx, baseCfg)
	if err != nil {
		return err
	}

	t.runtime = r
	t.running = true
	return nil
}

func (t *Task) stop(ctx context.Context, willReconfigure bool) error {
	if !t.running || t.runtime == nil {
		return nil
	}

	err := t.runtime.Close()
	t.runtime = nil
	t.running = false
	return err
}

// Stop gracefully shuts down the task.
func (t *Task) Stop(ctx context.Context, willReconfigure bool) error {
	return t.stop(ctx, willReconfigure)
}

// Key returns the task key.
func (t *Task) Key() task.Key { return t.key }
