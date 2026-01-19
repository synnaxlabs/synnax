// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// Driver is the Go task executor that handles task lifecycle and command processing.
type Driver struct {
	cfg                Config
	shutdownCommands   io.Closer
	shutdownHeartbeat  io.Closer
	disconnectObserver observe.Disconnect
	streamerRequests   confluence.Inlet[framer.StreamerRequest]
	rack               rack.Rack
	mu                 struct {
		tasks map[task.Key]Task
		sync.RWMutex
	}
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
	d := &Driver{cfg: cfg}
	d.mu.tasks = make(map[task.Key]Task)

	d.rack = rack.Rack{
		Name:     fmt.Sprintf("Node %d", cfg.Host.HostKey()),
		Embedded: true,
	}
	if err = cfg.Rack.NewWriter(nil).Create(ctx, &d.rack); err != nil {
		return nil, err
	}
	cfg.L.Info("created go driver rack", zap.Stringer("key", d.rack.Key))

	d.startHeartbeat(ctx)
	taskObs := gorp.Observe[task.Key, task.Task](cfg.DB)
	d.disconnectObserver = taskObs.OnChange(d.handleTaskChange)
	if err = d.startCommandStreaming(ctx); err != nil {
		return d, nil
	}
	return d, nil
}

func (d *Driver) startHeartbeat(ctx context.Context) {
	statusWriter := status.NewWriter[rack.StatusDetails](d.cfg.Status, nil)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(d.cfg.Instrumentation))
	d.shutdownHeartbeat = signal.NewHardShutdown(sCtx, cancel)
	signal.GoTick(
		sCtx,
		d.cfg.HeartbeatInterval,
		func(ctx context.Context, _ time.Time) error {
			if err := statusWriter.Set(ctx, &rack.Status{
				Key:     rack.StatusKey(d.rack.Key),
				Name:    d.rack.Name,
				Time:    telem.Now(),
				Variant: xstatus.VariantSuccess,
				Message: "Driver is running",
				Details: rack.StatusDetails{Rack: d.rack.Key},
			}); err != nil {
				d.cfg.L.Error("failed to update rack status", zap.Error(err))
			}
			return nil
		})
}

// startCommandStreaming initializes the command channel streamer. This is optional
// and will log warnings if the command channel doesn't exist or streaming fails.
func (d *Driver) startCommandStreaming(ctx context.Context) error {
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(d.cfg.Instrumentation))
	d.shutdownCommands = signal.NewGracefulShutdown(sCtx, cancel)
	streamer, err := d.cfg.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys: channel.Keys{d.cfg.Task.CommandChannelKey()},
	})
	if err != nil {
		return err
	}
	p := plumber.New()
	plumber.SetSegment[framer.StreamerRequest, framer.StreamerResponse](p, "streamer", streamer)
	sink := &commandSink{driver: d}
	sink.Sink = sink.process
	plumber.SetSink[framer.StreamerResponse](p, "driver", sink)
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "driver", 10)
	streamerRequests := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(streamerRequests)
	d.streamerRequests = streamerRequests
	sink.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return nil
}

func (s *commandSink) process(ctx context.Context, res framer.StreamerResponse) error {
	s.driver.processCommand(ctx, res.Frame)
	return nil
}

func (d *Driver) processCommand(ctx context.Context, frame framer.Frame) {
	var cmd task.Command
	for series := range frame.Series() {
		for s := range series.Samples() {
			if err := json.Unmarshal(s, &cmd); err != nil {
				d.cfg.L.Error("failed to unmarshal command", zap.Error(err))
				continue
			}
			if cmd.Task.Rack() != d.rack.Key {
				continue
			}
			d.mu.RLock()
			t, ok := d.mu.tasks[cmd.Task]
			d.mu.RUnlock()
			if ok {
				d.cfg.L.Warn("received command for unknown task", zap.Stringer("task", cmd.Task))
				continue
			}
			if err := t.Exec(ctx, cmd); err != nil {
				d.cfg.L.Error("failed to execute command",
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
			if ch.Variant == change.VariantSet {
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
			d.cfg.L.Error("failed to stop existing task for reconfiguration",
				zap.Stringer("task", t.Key),
				zap.Error(err),
			)
		}
		delete(d.mu.tasks, t.Key)
	}
	taskCtx := NewContext(ctx, d.cfg.Status)
	newTask, ok, err := d.cfg.Factory.ConfigureTask(taskCtx, t)
	if err != nil {
		d.cfg.L.Error("factory failed to configure task",
			zap.Stringer("task", t.Key),
			zap.String("type", t.Type),
			zap.Error(err),
		)
		return
	}
	if !ok {
		d.cfg.L.Warn("no factory handled task type",
			zap.Stringer("task", t.Key),
			zap.String("type", t.Type),
		)
		return
	}
	d.mu.tasks[t.Key] = newTask
	d.cfg.L.Info("configured task",
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
		d.cfg.L.Error("failed to stop task during deletion",
			zap.Stringer("task", key),
			zap.Error(err),
		)
	}
	delete(d.mu.tasks, key)
	d.cfg.L.Info("deleted task", zap.Stringer("task", key))
}

func (d *Driver) RackKey() rack.Key {
	return d.rack.Key
}

func (d *Driver) Close() error {
	d.mu.Lock()
	for key, t := range d.mu.tasks {
		if err := t.Stop(context.TODO(), false); err != nil {
			d.cfg.L.Error("failed to stop task during shutdown",
				zap.Stringer("task", key),
				zap.Error(err),
			)
		}
	}
	d.mu.tasks = nil
	d.mu.Unlock()
	d.disconnectObserver()
	if d.streamerRequests != nil {
		d.streamerRequests.Close()
	}
	heartbeatErr := d.shutdownHeartbeat.Close()
	return errors.Combine(d.shutdownCommands.Close(), heartbeatErr)
}
