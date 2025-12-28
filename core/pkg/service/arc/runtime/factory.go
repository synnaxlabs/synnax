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
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/state"
	arctelem "github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	godriver "github.com/synnaxlabs/synnax/pkg/service/driver/go"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
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
	// Status is used for updating statuses.
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
func (f *Factory) ConfigureTask(ctx context.Context, t task.Task) (godriver.Task, bool, error) {
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
	arcTask := newTask(t.Key, prog, cfg, f.cfg)
	return arcTask, true, nil
}

// Name returns the factory name.
func (f *Factory) Name() string { return "arc" }

// newTask creates a new Arc task with the given parameters.
func newTask(key task.Key, prog arc.Arc, cfg TaskConfig, factoryCfg FactoryConfig) *Task {
	return &Task{
		key:        key,
		prog:       prog,
		cfg:        cfg,
		factoryCfg: factoryCfg,
	}
}

// Task implements the godriver.Task interface and manages Arc program execution.
// It combines the previous Task and Runtime structs into a single entity.
type Task struct {
	// Identity & compiled program
	key  task.Key
	prog arc.Arc // Contains Name, Graph, and compiled Module
	cfg  TaskConfig

	// Factory config (injected dependencies for runtime)
	factoryCfg FactoryConfig

	// Runtime state (nil/zero when stopped, initialized on start)
	running   bool
	scheduler *scheduler.Scheduler
	streamer  *streamerSeg
	writer    *writerSeg
	state     *state.State
	closer    io.Closer
	startTime telem.TimeStamp
}

// Exec handles commands (start, stop, etc.)
func (t *Task) Exec(ctx context.Context, cmd task.Command) error {
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

	mod := t.prog.Module

	// Build state config
	stateCfg, err := NewStateConfig(ctx, t.factoryCfg.Channel, mod)
	if err != nil {
		return err
	}
	t.state = state.New(stateCfg.State)

	// Create node factories
	telemFactory := arctelem.NewTelemFactory()
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
	opFactory := op.NewFactory()
	stableFactory := stable.NewFactory(stable.FactoryConfig{})
	statusFactory := arcstatus.NewFactory(t.factoryCfg.Status)

	f := node.MultiFactory{
		opFactory,
		telemFactory,
		selectFactory,
		constantFactory,
		stableFactory,
		statusFactory,
	}

	var closers xio.MultiCloser
	if len(mod.WASM) > 0 {
		wasmMod, err := wasm.OpenModule(ctx, wasm.ModuleConfig{
			Module: mod,
			State:  t.state,
		})
		if err != nil {
			return err
		}
		closers = append(closers, wasmMod)
		wasmFactory, err := wasm.NewFactory(wasmMod)
		if err != nil {
			return err
		}
		f = append(f, wasmFactory)
	}

	// Create nodes
	nodes := make(map[string]node.Node)
	for _, irNode := range mod.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:   irNode,
			Module: mod,
			State:  t.state.Node(irNode.Key),
		})
		if err != nil {
			return err
		}
		nodes[irNode.Key] = n
	}

	// Create scheduler
	t.scheduler = scheduler.New(mod.IR, nodes)

	// Initialize segments
	t.streamer = &streamerSeg{}
	t.writer = &writerSeg{}
	t.startTime = telem.Now()

	// Create stream pipeline
	streamPipeline, requests, err := createStreamPipeline(
		ctx, t, t.factoryCfg.Framer, stateCfg.Reads.Keys(),
	)
	if err != nil {
		return err
	}
	t.streamer.requests = requests

	// Create write pipeline if needed
	var writePipeline confluence.Flow
	if len(stateCfg.Writes) > 0 {
		writePipeline, err = createWritePipeline(
			ctx, t.prog.Name, t, t.factoryCfg.Framer, stateCfg.Writes.Keys(),
		)
		if err != nil {
			return err
		}
	}

	// Start pipelines
	sCtx, cancel := signal.Isolated()
	streamPipeline.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	if writePipeline != nil {
		writePipeline.Flow(
			sCtx,
			confluence.CloseOutputInletsOnExit(),
			confluence.RecoverWithErrOnPanic(),
		)
	}
	t.closer = append(closers, signal.NewGracefulShutdown(sCtx, cancel))

	t.running = true
	return nil
}

// processFrame is the core reactive loop: ingests telemetry, schedules nodes, flushes writes.
func (t *Task) processFrame(ctx context.Context, res framer.StreamerResponse) error {
	t.state.Ingest(res.Frame.ToStorage())
	t.scheduler.Next(ctx, telem.Since(t.startTime))
	fr, changed := t.state.FlushWrites(telem.Frame[uint32]{})
	if !changed {
		return nil
	}
	return t.writer.Write(ctx, core.NewFrameFromStorage(fr))
}

func (t *Task) stop(context.Context, bool) error {
	if !t.running {
		return nil
	}

	c := errors.NewCatcher(errors.WithAggregation())
	if t.streamer != nil {
		c.Exec(t.streamer.Close)
	}
	if t.writer != nil {
		c.Exec(t.writer.Close)
	}
	if t.closer != nil {
		c.Exec(t.closer.Close)
	}

	// Reset runtime state
	t.scheduler = nil
	t.streamer = nil
	t.writer = nil
	t.state = nil
	t.closer = nil
	t.running = false

	return c.Error()
}

// Stop gracefully shuts down the task.
func (t *Task) Stop(ctx context.Context, willReconfigure bool) error {
	return t.stop(ctx, willReconfigure)
}

// Key returns the task key.
func (t *Task) Key() task.Key { return t.key }
