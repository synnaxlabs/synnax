// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tracker

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	binaryx "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"io"
	"sync"
)

// RackState is the state of a hardware rack. Unfortunately, we can't put this into
// the rack package because it would create a circular dependency.
type RackState struct {
	// Key is the key of the rack.
	Key rack.Key `json:"key" msgpack:"key"`
	// Heartbeat is a unit64 where the first 32 bits are the rack key and the second 32
	// bits are an incrementing heartbeat counter starting at 0 from when the rack
	// boots up. When the rack restarts, this counter will reset to 0.
	Heartbeat rack.Heartbeat `json:"heartbeat" msgpack:"heartbeat"`
	/// LastReceived is the last time the rack sent a heartbeat signal.
	LastReceived telem.TimeStamp `json:"last_received" msgpack:"last_received"`
	// Tasks is the state of the tasks associated with the rack.
	Tasks map[task.Key]task.State `json:"tasks" msgpack:"tasks"`
}

// Alive returns true if the rack is alive.
func (r RackState) Alive(threshold telem.TimeSpan) bool {
	return telem.Since(r.LastReceived) < threshold
}

// Tracker is used to track the state of hardware racks and tasks.
type Tracker struct {
	cfg Config
	// mu is a read-write lock used to protect the state of the tracker. Any fields
	// inside of this struct should be accessed while holding the lock.
	mu struct {
		sync.RWMutex
		// Racks is the map of racks to their corresponding state.
		Racks map[rack.Key]*RackState
	}
	// saveNotifications is used to signal an observing go-routine to save the state of
	// a task to gorp. This ensures that the most recent task state is persisted
	// across reloads.
	saveNotifications chan task.Key
	// closer shuts down all go-routines needed to keep the tracker service running.
	closer io.Closer
	// taskStateWriter is used to write task state changes to the database.
	taskStateWriter confluence.Inlet[framer.WriterRequest]
	// taskStateChannelKey is the key of the channel used to set task state.
	taskStateChannelKey channel.Key
}

// Config is the configuration for the Tracker service.
type Config struct {
	// Instrumentation used for logging, tracing, etc.
	// [OPTIONAL]
	alamos.Instrumentation
	// Rack is the service used to retrieve rack information.
	// [REQUIRED]
	Rack *rack.Service
	// Task is the service used to retrieve task information.
	// [TASK]
	Task *task.Service
	// Signals is used to subscribe to changes in rack and task state.
	// [REQUIRED]
	Signals *signals.Provider
	// Channels is used to create channels for the tracker service.
	// [REQUIRED]
	Channels channel.Writeable
	// HostProvider returns information about the cluster host.
	// [REQUIRED]
	HostProvider dcore.HostProvider
	// DB is used to persist and retrieve information about rack and task state.
	// [REQUIRED]
	DB     *gorp.DB
	Framer *framer.Service
	// rackStateAliveThreshold is the threshold for determining if a rack is alive.
	RackStateAliveThreshold telem.TimeSpan
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration or opening the tracker service. This
	// configuration is not valid on its own, and must be overridden with the required
	// fields detailed in the Config struct.
	DefaultConfig = Config{
		RackStateAliveThreshold: telem.Second * 15,
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Task = override.Nil(c.Task, other.Task)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channels = override.Nil(c.Channels, other.Channels)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.DB = override.Nil(c.DB, other.DB)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.RackStateAliveThreshold = override.Numeric(c.RackStateAliveThreshold, other.RackStateAliveThreshold)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("hardware.state")
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "task", c.Task)
	validate.NotNil(v, "signals", c.Signals)
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "host", c.HostProvider)
	validate.NotNil(v, "channels", c.Channels)
	validate.NotNil(v, "framer", c.Framer)
	return v.Error()
}

// Open opens a new task/rack state tracker with the provided configuration. If error
// is nil, the Tracker must be closed after use.
func Open(ctx context.Context, configs ...Config) (t *Tracker, err error) {
	cfg, err := config.New[Config](DefaultConfig, configs...)
	if err != nil {
		return
	}
	var racks []rack.Rack
	if err = cfg.Rack.NewRetrieve().
		WhereNode(cfg.HostProvider.HostKey()).
		Entries(&racks).
		Exec(ctx, nil); err != nil {
		return
	}
	sCtx, cancel := signal.Isolated()
	t = &Tracker{cfg: cfg}
	t.mu.Racks = make(map[rack.Key]*RackState, len(racks))
	for _, r := range racks {
		var tasks []task.Task
		if err = cfg.Task.NewRetrieve().
			WhereRacks(r.Key).
			Entries(&tasks).
			Exec(ctx, nil); err != nil {
			return
		}
		rck := &RackState{Key: r.Key, Tasks: make(map[task.Key]task.State, len(tasks))}
		for _, tsk := range tasks {
			// try to fetch the task state
			taskState := task.State{Task: tsk.Key, Variant: task.InfoStateVariant}
			if err = gorp.NewRetrieve[task.Key, task.State]().
				WhereKeys(tsk.Key).
				Entry(&taskState).
				Exec(ctx, cfg.DB); err != nil && !errors.Is(err, query.NotFound) {
				return
			}
			rck.Tasks[tsk.Key] = taskState
		}
		t.mu.Racks[rck.Key] = rck
	}
	channels := []channel.Channel{
		{
			Name:        "sy_task_state",
			DataType:    telem.JSONT,
			Leaseholder: cfg.HostProvider.HostKey(),
			Virtual:     true,
			Internal:    true,
		},
		{
			Name:        "sy_rack_heartbeat",
			DataType:    telem.Uint64T,
			Leaseholder: cfg.HostProvider.HostKey(),
			Virtual:     true,
			Internal:    true,
		},
		{
			Name:        "sy_task_cmd",
			DataType:    telem.JSONT,
			Leaseholder: cfg.HostProvider.HostKey(),
			Virtual:     true,
			Internal:    true,
		}}
	if err = cfg.Channels.CreateMany(
		ctx,
		&channels,
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}
	t.taskStateChannelKey = channels[0].Key()
	taskObs := gorp.Observe[task.Key, task.Task](cfg.DB)
	rackObs := gorp.Observe[rack.Key, rack.Rack](cfg.DB)
	dcTaskObs := taskObs.OnChange(t.handleTaskChanges)
	dcRackObs := rackObs.OnChange(t.handleRackChanges)
	heartBeatObs, closeHeartBeatObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: "sy_rack_heartbeat",
	})
	if err != nil {
		return nil, err
	}
	dcHeartbeatObs := heartBeatObs.OnChange(t.handleHeartbeat)
	taskStateObs, closeTaskStateObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: "sy_task_state",
	})
	if err != nil {
		return nil, err
	}
	taskStateWriter, err := cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Start: telem.Now(),
		Keys:  []channel.Key{t.taskStateChannelKey},
	})
	if err != nil {
		return nil, err
	}
	taskStateWriterStream := confluence.NewStream[framer.WriterRequest](1)
	taskStateWriter.InFrom(taskStateWriterStream)
	t.taskStateWriter = taskStateWriterStream
	obs := confluence.NewObservableSubscriber[framer.WriterResponse]()
	obs.OnChange(func(ctx context.Context, r framer.WriterResponse) {
		cfg.L.Error("unexpected writer error", zap.Error(r.Error))
	})
	outlets := confluence.NewStream[framer.WriterResponse](1)
	obs.InFrom(outlets)
	taskStateWriter.OutTo(outlets)
	taskStateWriter.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	dcTaskStateObs := taskStateObs.OnChange(t.handleTaskState)
	t.saveNotifications = make(chan task.Key, 10)
	signal.GoRange[task.Key](sCtx, t.saveNotifications, t.saveTaskState)
	// Closers get executed in reverse order, so we need to shut down all of our
	// observers before we shut down our signal context.
	t.closer = xio.MultiCloser{
		xio.CloserFunc(func() error {
			defer cancel()
			t.taskStateWriter.Close()
			close(t.saveNotifications)
			return sCtx.Wait()
		}),
		closeTaskStateObs,
		closeHeartBeatObs,
		xio.NopCloserFunc(dcRackObs),
		xio.NopCloserFunc(dcTaskObs),
		xio.NopCloserFunc(dcHeartbeatObs),
		xio.NopCloserFunc(dcTaskStateObs),
	}
	return
}

// GetTask returns the state of a task by its key. If the task is not found, the second
// return value will be false.
func (t *Tracker) GetTask(_ context.Context, key task.Key) (task.State, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.mu.Racks[key.Rack()]
	if !ok {
		return task.State{}, false
	}
	tsk, ok := r.Tasks[key]
	return tsk, ok
}

// GetRack returns the state of a rack by its key. If the rack is not found, the second
// return value will be false.
func (t *Tracker) GetRack(_ context.Context, key rack.Key) (RackState, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.mu.Racks[key]
	if !ok {
		return RackState{}, false
	}
	return *r, true
}

// Close closes the tracker, freeing all associated go-routines and resources.
// The tracker must not be used after it is closed.
func (t *Tracker) Close() error { return t.closer.Close() }

// handleTaskChanges handles changes to tasks in the DB.
func (t *Tracker) handleTaskChanges(ctx context.Context, r gorp.TxReader[task.Key, task.Task]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for c, ok := r.Next(ctx); ok; c, ok = r.Next(ctx) {
		if c.Variant == change.Delete {
			if _, rackOk := t.mu.Racks[c.Key.Rack()]; rackOk {
				delete(t.mu.Racks[c.Key.Rack()].Tasks, c.Key)
			}
		} else {
			rackKey := c.Key.Rack()
			rackState, rckOk := t.mu.Racks[rackKey]
			if !rckOk {
				rackState = &RackState{Key: rackKey, Tasks: make(map[task.Key]task.State)}
				t.mu.Racks[rackKey] = rackState
			}
			if _, taskOk := rackState.Tasks[c.Key]; !taskOk {
				rackState.Tasks[c.Key] = task.State{Task: c.Key, Variant: task.InfoStateVariant}
			}
			alive := rackState.Alive(t.cfg.RackStateAliveThreshold)
			if !rckOk || !alive {
				state := task.State{
					Task:    c.Key,
					Variant: task.WarningStateVariant,
					Details: task.NewStaticDetails(map[string]interface{}{
						"message": "rack is not alive",
						"running": false,
					}),
				}
				if rckOk {
					var rck rack.Rack
					if err := gorp.NewRetrieve[rack.Key, rack.Rack]().
						WhereKeys(rackKey).
						Entry(&rck).
						Exec(ctx, t.cfg.DB); err != nil {
						t.cfg.L.Warn("failed to retrieve rack", zap.Error(err))
					}
					state.Details = task.NewStaticDetails(map[string]interface{}{
						"running": "false",
						"message": fmt.Sprintf("Synnax Driver on %s is not running, so the task may fail to configure. Driver was last alive %s ago.", rck.Name, telem.Since(rackState.LastReceived)),
					})
				}
				t.taskStateWriter.Inlet() <- framer.WriterRequest{
					Command: writer.Data,
					Frame: core.Frame{
						Keys:   channel.Keys{t.taskStateChannelKey},
						Series: []telem.Series{telem.NewStaticJSONV(state)},
					},
				}
			}
		}
	}
}

// handleRackChanges handles changes to racks in the DB.
func (t *Tracker) handleRackChanges(ctx context.Context, r gorp.TxReader[rack.Key, rack.Rack]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for c, ok := r.Next(ctx); ok; c, ok = r.Next(ctx) {
		if c.Variant == change.Delete {
			delete(t.mu.Racks, c.Key)
		} else {
			if _, rackOk := t.mu.Racks[c.Key]; !rackOk {
				t.mu.Racks[c.Key] = &RackState{Key: c.Key, Tasks: make(map[task.Key]task.State), LastReceived: telem.Now()}
			}
		}
	}
}

// handleHeartbeat handles heartbeat changes.
func (t *Tracker) handleHeartbeat(_ context.Context, changes []change.Change[[]byte, struct{}]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, ch := range changes {
		b := binary.LittleEndian.Uint64(ch.Key)
		rackKey := rack.Key(b >> 32)
		r, ok := t.mu.Racks[rackKey]
		if !ok {
			t.cfg.L.Warn("rack not found for heartbeat update", zap.Uint64("heartbeat", b), zap.Uint32("rack", uint32(rackKey)))
			continue
		}
		r.Heartbeat = rack.Heartbeat(b)
		r.LastReceived = telem.Now()
	}
}

// handleTaskState handles task state changes.
func (t *Tracker) handleTaskState(ctx context.Context, changes []change.Change[[]byte, struct{}]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	decoder := &binaryx.JSONCodec{}
	for _, ch := range changes {
		var taskState task.State
		if err := decoder.Decode(ctx, ch.Key, &taskState); err != nil {
			t.cfg.L.Warn("failed to decode task state", zap.Error(err))
		}
		rackKey := taskState.Task.Rack()
		r, ok := t.mu.Racks[rackKey]
		if !ok {
			t.cfg.L.Warn("rack not found for task state update", zap.Uint64("task", uint64(taskState.Task)))
		} else {
			r.Tasks[taskState.Task] = taskState
		}
		select {
		case t.saveNotifications <- taskState.Task:
		default:
		}
	}
}

func (t *Tracker) saveTaskState(ctx context.Context, taskKey task.Key) error {
	state, ok := t.GetTask(ctx, taskKey)
	if !ok {
		return nil
	}
	if err := gorp.NewCreate[task.Key, task.State]().Entry(&state).Exec(ctx, t.cfg.DB); err != nil {
		t.cfg.L.Warn("failed to save task state", zap.Error(err))
	}
	return nil
}
