// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package godriver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// Driver is the Go task executor that handles task lifecycle and command processing.
type Driver struct {
	Config
	rack rack.Rack
	ctx  Context
	mu   struct {
		sync.RWMutex
		tasks map[task.Key]Task
	}
	shutdown              io.Closer
	disconnectObserver    observe.Disconnect
	closeStreamerRequests confluence.Inlet[framer.StreamerRequest]
}

// commandSink is a confluence sink that processes incoming command frames.
type commandSink struct {
	confluence.UnarySink[framer.StreamerResponse]
	driver *Driver
}

// Open creates and starts a new Go driver. The driver is fully initialized and ready
// to receive task changes when this function returns. Background goroutines for command
// streaming are started automatically.
func Open(ctx context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	d := &Driver{Config: cfg}
	d.ctx = NewContext(ctx, cfg.Status)
	d.mu.tasks = make(map[task.Key]Task)

	d.rack = rack.Rack{
		Name:     fmt.Sprintf("Node %d Core", cfg.Host.HostKey()),
		Embedded: true,
	}
	if err = cfg.Rack.NewWriter(nil).Create(ctx, &d.rack); err != nil {
		return nil, err
	}
	cfg.L.Info("created go driver rack", zap.Stringer("key", d.rack.Key))

	// Set up background context for goroutines
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	d.shutdown = signal.NewGracefulShutdown(sCtx, cancel)

	// Set up gorp observer for task changes - this is registered synchronously
	// so the driver is ready to receive task changes when Open returns
	taskObs := gorp.Observe[task.Key, task.Task](cfg.DB)
	d.disconnectObserver = taskObs.OnChange(d.handleTaskChange)

	// Set up command streaming (optional - gracefully degrades if channel doesn't exist)
	d.setupCommandStreaming(ctx, sCtx)

	return d, nil
}

// setupCommandStreaming initializes the command channel streamer. This is optional
// and will log warnings if the command channel doesn't exist or streaming fails.
func (d *Driver) setupCommandStreaming(ctx context.Context, sCtx signal.Context) {
	var cmdCh channel.Channel
	err := d.Channel.NewRetrieve().WhereNames("sy_task_cmd").Entry(&cmdCh).Exec(ctx, nil)
	if err != nil {
		d.L.Warn("failed to retrieve sy_task_cmd channel, command streaming disabled", zap.Error(err))
		return
	}

	streamer, err := d.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys: channel.Keys{cmdCh.Key()},
	})
	if err != nil {
		d.L.Warn("failed to create command streamer", zap.Error(err))
		return
	}
	p := plumber.New()
	plumber.SetSegment[framer.StreamerRequest, framer.StreamerResponse](p, "streamer", streamer)
	sink := &commandSink{driver: d}
	sink.Sink = sink.process
	plumber.SetSink[framer.StreamerResponse](p, "driver", sink)

	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "driver", 10)

	streamerRequests := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(streamerRequests)
	d.closeStreamerRequests = streamerRequests

	sink.Flow(sCtx, confluence.CloseOutputInletsOnExit())
}

func (s *commandSink) process(ctx context.Context, res framer.StreamerResponse) error {
	s.driver.processCommand(ctx, res.Frame)
	return nil
}

func (d *Driver) processCommand(ctx context.Context, frame framer.Frame) {
	for series := range frame.Series() {
		for i := 0; i < int(series.Len()); i++ {
			data := series.At(i)
			var cmd task.Command
			if err := json.Unmarshal(data, &cmd); err != nil {
				d.L.Error("failed to unmarshal command", zap.Error(err))
				continue
			}
			if cmd.Task.Rack() != d.rack.Key {
				continue
			}
			d.mu.RLock()
			t, ok := d.mu.tasks[cmd.Task]
			d.mu.RUnlock()
			if !ok {
				d.L.Warn("received command for unknown task", zap.Stringer("task", cmd.Task))
				continue
			}
			if err := t.Exec(ctx, cmd); err != nil {
				d.L.Error("failed to execute command",
					zap.Stringer("task", cmd.Task),
					zap.String("type", cmd.Type),
					zap.Error(err),
				)
			}
		}
	}
}

func (d *Driver) handleTaskChange(ctx context.Context, reader gorp.TxReader[task.Key, task.Task]) {
	for ch := range reader {
		if ch.Key.Rack() == d.rack.Key {
			if ch.Variant == change.Set {
				d.configure(ctx, ch.Value)
			} else {
				d.delete(ctx, ch.Key)
			}
		}
	}
}

func (d *Driver) configure(ctx context.Context, t task.Task) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if existing, ok := d.mu.tasks[t.Key]; ok {
		if err := existing.Stop(ctx, true); err != nil {
			d.L.Error("failed to stop existing task for reconfiguration",
				zap.Stringer("task", t.Key),
				zap.Error(err),
			)
		}
		delete(d.mu.tasks, t.Key)
	}
	// Create a new context with the current context for this configuration
	taskCtx := NewContext(ctx, d.Status)
	newTask, ok, err := d.Factory.ConfigureTask(taskCtx, t)
	if err != nil {
		d.L.Error("factory failed to configure task",
			zap.Stringer("task", t.Key),
			zap.String("type", t.Type),
			zap.Error(err),
		)
		return
	}
	if !ok {
		d.L.Warn("no factory handled task type",
			zap.Stringer("task", t.Key),
			zap.String("type", t.Type),
		)
		return
	}
	d.mu.tasks[t.Key] = newTask
	d.L.Info("configured task",
		zap.Stringer("task", t.Key),
		zap.String("type", t.Type),
		zap.String("name", t.Name),
	)
}

func (d *Driver) delete(ctx context.Context, key task.Key) {
	d.mu.Lock()
	defer d.mu.Unlock()

	t, ok := d.mu.tasks[key]
	if !ok {
		return
	}
	if err := t.Stop(ctx, false); err != nil {
		d.L.Error("failed to stop task during deletion",
			zap.Stringer("task", key),
			zap.Error(err),
		)
	}
	delete(d.mu.tasks, key)
	d.L.Info("deleted task", zap.Stringer("task", key))
}

func (d *Driver) RackKey() rack.Key {
	return d.rack.Key
}

func (d *Driver) Close() error {
	d.mu.Lock()
	ctx := context.TODO()
	for key, t := range d.mu.tasks {
		if err := t.Stop(ctx, false); err != nil {
			d.L.Error("failed to stop task during shutdown",
				zap.Stringer("task", key),
				zap.Error(err),
			)
		}
	}
	d.mu.tasks = make(map[task.Key]Task)
	d.mu.Unlock()
	d.disconnectObserver()
	if d.closeStreamerRequests != nil {
		d.closeStreamerRequests.Close()
	}
	return d.shutdown.Close()
}
