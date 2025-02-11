// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

import (
	"context"
	"encoding/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	binaryx "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type RackState struct {
	Key       rack.Key                    `json:"key" msgpack:"key"`
	Heartbeat uint64                      `json:"heartbeat" msgpack:"heartbeat"`
	Tasks     map[task.Key]task.TaskState `json:"tasks" msgpack:"tasks"`
}

type Tracker struct {
	mu struct {
		sync.RWMutex
		Racks map[rack.Key]*RackState
	}
	stopListeners     io.Closer
	saveNotifications chan task.Key
}

type TrackerConfig struct {
	alamos.Instrumentation
	Rack         *rack.Service
	Task         *task.Service
	Signals      *signals.Provider
	Channels     channel.Writeable
	HostProvider core.HostProvider
	DB           *gorp.DB
}

var (
	_             config.Config[TrackerConfig] = TrackerConfig{}
	DefaultConfig                              = TrackerConfig{}
)

func (c TrackerConfig) Override(other TrackerConfig) TrackerConfig {
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Task = override.Nil(c.Task, other.Task)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.DB = override.Nil(c.DB, other.DB)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Channels = override.Nil(c.Channels, other.Channels)
	return c
}

func (c TrackerConfig) Validate() error {
	v := validate.New("hardware.state")
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "task", c.Task)
	validate.NotNil(v, "signals", c.Signals)
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "host", c.HostProvider)
	validate.NotNil(v, "channels", c.Channels)
	return v.Error()
}

func OpenTracker(ctx context.Context, configs ...TrackerConfig) (t *Tracker, err error) {
	cfg, err := config.New[TrackerConfig](DefaultConfig, configs...)
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
	t = &Tracker{}
	t.mu.Racks = make(map[rack.Key]*RackState, len(racks))
	for _, r := range racks {
		var tasks []task.Task
		if err = cfg.Task.NewRetrieve().
			WhereRacks(r.Key).
			Entries(&tasks).
			Exec(ctx, nil); err != nil {
			return
		}
		rck := &RackState{Key: r.Key, Tasks: make(map[task.Key]task.TaskState, len(tasks))}
		for _, tsk := range tasks {
			// try to fetch the task state
			taskState := task.TaskState{Task: tsk.Key, Variant: task.StatusInfo}
			if err = gorp.NewRetrieve[task.Key, task.TaskState]().
				WhereKeys(tsk.Key).
				Entry(&taskState).
				Exec(ctx, cfg.DB); err != nil && !errors.Is(err, query.NotFound) {
				return
			}
			rck.Tasks[tsk.Key] = taskState
		}
		t.mu.Racks[rck.Key] = rck
	}
	if err = cfg.Channels.CreateMany(
		ctx,
		&[]channel.Channel{
			{
				Name:        "sy_rack_heartbeat",
				DataType:    telem.Uint64T,
				Leaseholder: cfg.HostProvider.HostKey(),
				Virtual:     true,
				Internal:    true,
			},
			{
				Name:        "sy_task_state",
				DataType:    telem.JSONT,
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
			},
		},
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}

	heartBeatObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: "sy_rack_heartbeat",
	})
	if err != nil {
		return nil, err
	}
	taskStateObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: "sy_task_state",
	})
	taskObs := gorp.Observe[task.Key, task.Task](cfg.DB)
	rackObs := gorp.Observe[rack.Key, rack.Rack](cfg.DB)
	dcTaskObs := taskObs.OnChange(func(ctx context.Context, r gorp.TxReader[task.Key, task.Task]) {
		t.mu.Lock()
		defer t.mu.Unlock()
		for c, ok := r.Next(ctx); ok; c, ok = r.Next(ctx) {
			if c.Variant == change.Delete {
				if _, rackOk := t.mu.Racks[c.Key.Rack()]; rackOk {
					delete(t.mu.Racks[c.Key.Rack()].Tasks, c.Key)
				}
			} else {
				rackKey := c.Key.Rack()
				rck, rckOk := t.mu.Racks[rackKey]
				if !rckOk {
					rck = &RackState{Key: rackKey, Tasks: make(map[task.Key]task.TaskState)}
					t.mu.Racks[rackKey] = rck
				}
				if _, tskOk := rck.Tasks[c.Key]; !tskOk {
					rck.Tasks[c.Key] = task.TaskState{Task: c.Key, Variant: task.StatusInfo}
				}
			}
		}
	})
	dcRackObs := rackObs.OnChange(func(ctx context.Context, r gorp.TxReader[rack.Key, rack.Rack]) {
		t.mu.Lock()
		defer t.mu.Unlock()
		for c, ok := r.Next(ctx); ok; c, ok = r.Next(ctx) {
			if c.Variant == change.Delete {
				delete(t.mu.Racks, c.Key)
			} else {
				if _, rackOk := t.mu.Racks[c.Key]; !rackOk {
					t.mu.Racks[c.Key] = &RackState{Key: c.Key, Tasks: make(map[task.Key]task.TaskState)}
				}
			}
		}
	})
	if err != nil {
		return nil, err
	}
	heartBeatObs.OnChange(func(ctx context.Context, changes []change.Change[[]byte, struct{}]) {
		t.mu.Lock()
		defer t.mu.Unlock()
		for _, ch := range changes {
			b := binary.LittleEndian.Uint64(ch.Key)
			// first 32 bits is the rack key, second
			// 32 bits is the heartbeat
			rackKey := rack.Key(b >> 32)
			heartbeat := b
			r, ok := t.mu.Racks[rackKey]
			if !ok {
				cfg.L.Warn("rack not found for heartbeat update", zap.Uint64("heartbeat", heartbeat), zap.Uint32("rack", uint32(rackKey)))
				continue
			}
			r.Heartbeat = heartbeat
		}
	})
	taskStateObs.OnChange(func(ctx context.Context, changes []change.Change[[]byte, struct{}]) {
		t.mu.Lock()
		defer t.mu.Unlock()
		// Assume the tasks state is encoded as JSON
		decoder := &binaryx.JSONCodec{}
		for _, ch := range changes {
			var taskState task.TaskState
			if err = decoder.Decode(ctx, ch.Key, &taskState); err != nil {
				cfg.L.Warn("failed to decode task state", zap.Error(err))
			}
			rackKey := taskState.Task.Rack()
			r, ok := t.mu.Racks[rackKey]
			if !ok {
				cfg.L.Warn("rack not found for task state update", zap.Uint64("task", uint64(taskState.Task)))
			} else {
				r.Tasks[taskState.Task] = taskState
			}
			t.sendSaveSignal(taskState.Task)
		}
	})
	t.saveNotifications = make(chan task.Key, 10)
	signal.GoRange[task.Key](sCtx, t.saveNotifications, func(ctx context.Context, taskKey task.Key) error {
		tsk, ok := t.GetTask(ctx, taskKey)
		if !ok {
			return nil
		}
		if err := gorp.NewCreate[task.Key, task.TaskState]().Entry(&tsk).Exec(ctx, cfg.DB); err != nil {
			cfg.L.Warn("failed to save task state", zap.Error(err))
		}
		return nil
	})
	t.stopListeners = xio.MultiCloser{
		signal.NewShutdown(sCtx, cancel),
		xio.NopCloserFunc(dcRackObs),
		xio.NopCloserFunc(dcTaskObs),
	}
	return
}

func (t *Tracker) sendSaveSignal(taskKey task.Key) {
	select {
	case t.saveNotifications <- taskKey:
	default:
	}
}

func (t *Tracker) GetTask(ctx context.Context, key task.Key) (task.TaskState, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.mu.Racks[key.Rack()]
	if !ok {
		return task.TaskState{}, false
	}
	tsk, ok := r.Tasks[key]
	return tsk, ok
}

func (t *Tracker) GetRack(ctx context.Context, key rack.Key) (RackState, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.mu.Racks[key]
	if !ok {
		return RackState{}, false
	}
	return *r, true
}

func (t *Tracker) Close() error {
	return t.stopListeners.Close()
}
