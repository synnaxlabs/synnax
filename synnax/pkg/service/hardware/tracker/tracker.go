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
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	binaryx "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// RackState is the state of a hardware rack. Unfortunately, we can't put this into
// the rack package because it would create a circular dependency.
type RackState struct {
	rack.State
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
		// Devices is the map of devices to their corresponding state.
		Devices map[string]device.State
	}
	// saveNotifications is used to signal an observing go-routine to save the state of
	// a task to gorp. This ensures that the most recent task state is persisted
	// across reloads.
	saveNotifications chan task.Key
	// deviceSaveNotifications is used to signal an observing go-routine to save the state of
	// a device to gorp. This ensures that the most recent device state is persisted
	// across reloads.
	deviceSaveNotifications chan string
	// closer shuts down all go-routines needed to keep the tracker service running.
	closer io.Closer
	// stateWriter is used to write state changes to the database.
	stateWriter confluence.Inlet[framer.WriterRequest]
	// taskStateChannelKey is the key of the channel used to set task state.
	taskStateChannelKey channel.Key
	// rackStateChannelKey is the key of the channel used to set rack state.
	rackStateChannelKey channel.Key
	// deviceStateChannelKey is the key of the channel used to set device state.
	deviceStateChannelKey channel.Key
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
	// [REQUIRED]
	Task *task.Service
	// Device is the service used to retrieve device information.
	// [REQUIRED]
	Device *device.Service
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
		RackStateAliveThreshold: telem.Second * 3,
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
	c.Device = override.Nil(c.Device, other.Device)
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
	validate.NotNil(v, "device", c.Device)
	return v.Error()
}

// Open opens a new task/rack state tracker with the provided configuration. If error
// is nil, the Tracker must be closed after use.
func Open(ctx context.Context, configs ...Config) (t *Tracker, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
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
	t.mu.Devices = make(map[string]device.State)

	for _, r := range racks {
		// Initialize rack state with empty maps
		rck := &RackState{
			Tasks: make(map[task.Key]task.State),
		}
		rck.Key = r.Key

		// Fetch and initialize tasks for this rack
		var tasks []task.Task
		if err = cfg.Task.NewRetrieve().
			WhereRacks(r.Key).
			Entries(&tasks).
			Exec(ctx, nil); err != nil {
			return
		}

		for _, tsk := range tasks {
			if tsk.Snapshot {
				continue
			}
			taskState := task.State{Task: tsk.Key, Variant: status.InfoVariant}
			if err = gorp.NewRetrieve[task.Key, task.State]().
				WhereKeys(tsk.Key).
				Entry(&taskState).
				Exec(ctx, cfg.DB); err != nil && !errors.Is(err, query.NotFound) {
				return
			}
			rck.Tasks[tsk.Key] = taskState
		}

		t.mu.Racks[r.Key] = rck
	}

	// Fetch and initialize all devices
	var allDevices []device.Device
	if err = cfg.Device.NewRetrieve().
		Entries(&allDevices).
		Exec(ctx, cfg.DB); err != nil {
		return
	}

	for _, dev := range allDevices {
		deviceState := device.State{
			Key:     dev.Key,
			Variant: "info",
			Details: "",
			Rack:    dev.Rack,
		}
		if err = gorp.NewRetrieve[string, device.State]().
			WhereKeys(dev.Key).
			Entry(&deviceState).
			Exec(ctx, cfg.DB); err != nil && !errors.Is(err, query.NotFound) {
			return
		}
		t.mu.Devices[dev.Key] = deviceState
	}

	if err := cfg.Channels.DeleteByName(ctx, "sy_rack_heartbeat", true); err != nil {
		return nil, err
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
			Name:        "sy_rack_state",
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
		{
			Name:        "sy_device_state",
			DataType:    telem.JSONT,
			Leaseholder: cfg.HostProvider.HostKey(),
			Virtual:     true,
			Internal:    true,
		},
	}
	if err = cfg.Channels.CreateMany(
		ctx,
		&channels,
		channel.OverwriteIfNameExistsAndDifferentProperties(),
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}
	t.taskStateChannelKey = channels[0].Key()
	t.rackStateChannelKey = channels[1].Key()
	t.deviceStateChannelKey = channels[3].Key()
	taskObs := gorp.Observe[task.Key, task.Task](cfg.DB)
	rackObs := gorp.Observe[rack.Key, rack.Rack](cfg.DB)
	deviceObs := gorp.Observe[string, device.Device](cfg.DB)
	dcTaskObs := taskObs.OnChange(t.handleTaskChanges)
	dcRackObs := rackObs.OnChange(t.handleRackChanges)
	dcDeviceObs := deviceObs.OnChange(t.handleDeviceChanges)
	rackStateObs, closeRackStateObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: "sy_rack_state",
	})
	if err != nil {
		return nil, err
	}
	dcRackStateObs := rackStateObs.OnChange(t.handleRackState)
	taskStateObs, closeTaskStateObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: "sy_task_state",
	})
	if err != nil {
		return nil, err
	}
	stateWriter, err := cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Start: telem.Now(),
		Keys:  []channel.Key{t.taskStateChannelKey, t.rackStateChannelKey, t.deviceStateChannelKey},
	})
	if err != nil {
		return nil, err
	}
	taskStateWriterStream := confluence.NewStream[framer.WriterRequest](1)
	stateWriter.InFrom(taskStateWriterStream)
	// Try to delete a non-existent task
	t.stateWriter = taskStateWriterStream
	obs := confluence.NewObservableSubscriber[framer.WriterResponse]()
	obs.OnChange(func(ctx context.Context, r framer.WriterResponse) {
		cfg.L.Error("unexpected writer error", zap.Error(r.Error))
	})
	outlets := confluence.NewStream[framer.WriterResponse](1)
	obs.InFrom(outlets)
	stateWriter.OutTo(outlets)
	stateWriter.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	dcTaskStateObs := taskStateObs.OnChange(t.handleTaskState)
	t.saveNotifications = make(chan task.Key, 10)
	signal.GoRange(sCtx, t.saveNotifications, t.saveTaskState)
	t.deviceSaveNotifications = make(chan string, 10)
	signal.GoRange(sCtx, t.deviceSaveNotifications, func(ctx context.Context, notification string) error {
		return t.saveDeviceState(ctx, notification)
	})
	deviceStateObs, closeDeviceStateObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: "sy_device_state",
	})
	if err != nil {
		return nil, err
	}
	dcDeviceStateObs := deviceStateObs.OnChange(t.handleDeviceState)

	tickCtx, cancel := signal.WithCancel(sCtx)
	signal.GoTick(tickCtx, t.cfg.RackStateAliveThreshold.Duration(), func(ctx context.Context, _ time.Time) error {
		t.mu.RLock()
		defer t.mu.RUnlock()
		t.checkRackState(ctx)
		return nil
	})
	t.closer = xio.MultiCloser{
		xio.CloserFunc(func() error {
			defer cancel()
			t.stateWriter.Close()
			close(t.saveNotifications)
			close(t.deviceSaveNotifications)
			return sCtx.Wait()
		}),
		signal.NewHardShutdown(tickCtx, cancel),
		closeTaskStateObs,
		closeRackStateObs,
		closeDeviceStateObs,
		xio.NopCloserFunc(dcRackObs),
		xio.NopCloserFunc(dcTaskObs),
		xio.NopCloserFunc(dcDeviceObs),
		xio.NopCloserFunc(dcRackStateObs),
		xio.NopCloserFunc(dcTaskStateObs),
		xio.NopCloserFunc(dcDeviceStateObs),
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

// GetDevice returns the state of a device by its key. If the device is not found, the second
// return value will be false.
func (t *Tracker) GetDevice(_ context.Context, deviceKey string) (device.State, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	dev, ok := t.mu.Devices[deviceKey]
	return dev, ok
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
				rackState = &RackState{Tasks: make(map[task.Key]task.State)}
				rackState.Key = rackKey
				t.mu.Racks[rackKey] = rackState
			}
			if _, taskOk := rackState.Tasks[c.Key]; !taskOk {
				rackState.Tasks[c.Key] = task.State{Task: c.Key, Variant: status.InfoVariant}
			}
			alive := rackState.Alive(t.cfg.RackStateAliveThreshold)
			if !rckOk || !alive {
				state := task.State{
					Task:    c.Key,
					Variant: status.WarningVariant,
					Details: xjson.NewStaticString(ctx, map[string]interface{}{
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
					state.Details = xjson.NewStaticString(ctx, map[string]interface{}{
						"running": "false",
						"message": fmt.Sprintf("Synnax Driver on %s is not running, so the task may fail to configure. Driver was last alive %s ago.", rck.Name, telem.Since(rackState.LastReceived).Truncate(telem.Second)),
					})
				}
				t.stateWriter.Inlet() <- framer.WriterRequest{
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
				nState := &RackState{Tasks: make(map[task.Key]task.State)}
				nState.LastReceived = telem.Now()
				nState.Key = c.Key
				t.mu.Racks[c.Key] = nState
			}
		}
	}
}

func (t *Tracker) checkRackState(ctx context.Context) {
	rackStates := make([]rack.State, 0, len(t.mu.Racks))
	taskStates := make([]task.State, 0, len(t.mu.Racks))
	deviceStates := make([]device.State, 0)

	for _, r := range t.mu.Racks {
		if r.Alive(t.cfg.RackStateAliveThreshold) {
			continue
		}
		r.State.Variant = "warning"
		r.State.Message = fmt.Sprintf("Driver %s is not alive", r.Key)
		rackStates = append(rackStates, r.State)

		var rck rack.Rack
		if err := gorp.NewRetrieve[rack.Key, rack.Rack]().
			WhereKeys(r.Key).
			Entry(&rck).
			Exec(context.Background(), t.cfg.DB); err != nil {
			t.cfg.L.Warn("failed to retrieve rack", zap.Error(err))
			continue
		}

		msg := fmt.Sprintf("Synnax Driver on %s is not running. Driver was last alive %s ago.", rck.Name, telem.Since(r.LastReceived).Truncate(telem.Second))
		for _, taskState := range r.Tasks {
			taskState.Variant = status.WarningVariant
			taskState.Details = xjson.NewStaticString(ctx, map[string]interface{}{
				"message": msg,
				"running": false,
			})
			taskStates = append(taskStates, taskState)
		}

		for _, dev := range t.mu.Devices {
			if dev.Rack == r.Key {
				dev.Variant = status.WarningVariant
				dev.Details = xjson.NewStaticString(ctx, map[string]interface{}{
					"message": msg,
				})
				deviceStates = append(deviceStates, dev)
			}
		}

	}

	fr := core.Frame{}
	if len(rackStates) > 0 {
		fr.Keys = append(fr.Keys, t.rackStateChannelKey)
		fr.Series = append(fr.Series, telem.NewStaticJSONV(rackStates...))
	}
	if len(taskStates) > 0 {
		fr.Keys = append(fr.Keys, t.taskStateChannelKey)
		fr.Series = append(fr.Series, telem.NewStaticJSONV(taskStates...))
	}
	if len(deviceStates) > 0 {
		fr.Keys = append(fr.Keys, t.deviceStateChannelKey)
		fr.Series = append(fr.Series, telem.NewStaticJSONV(deviceStates...))
	}

	if len(fr.Keys) == 0 {
		return
	}

	t.stateWriter.Inlet() <- framer.WriterRequest{Command: writer.Data, Frame: fr}
}

// handleRackState handles heartbeat changes.
func (t *Tracker) handleRackState(_ context.Context, changes []change.Change[[]byte, struct{}]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	decoder := &binaryx.JSONCodec{}
	for _, ch := range changes {
		var rackState rack.State
		if err := decoder.Decode(context.Background(), ch.Key, &rackState); err != nil {
			t.cfg.L.Warn("failed to decode rack state", zap.Error(err))
			continue
		}
		r, ok := t.mu.Racks[rackState.Key]
		if !ok {
			t.cfg.L.Warn("rack not found for state update", zap.Uint32("rack", uint32(rackState.Key)))
			continue
		}
		r.State = rackState
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
			continue
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

// handleDeviceState handles device state changes.
func (t *Tracker) handleDeviceState(ctx context.Context, changes []change.Change[[]byte, struct{}]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	decoder := &binaryx.JSONCodec{}
	for _, ch := range changes {
		var incomingState device.State
		if err := decoder.Decode(ctx, ch.Key, &incomingState); err != nil {
			t.cfg.L.Warn("failed to decode device state", zap.Error(err))
			continue
		}

		existingState, exists := t.mu.Devices[incomingState.Key]
		if exists && existingState.Rack != incomingState.Rack {
			var racks []rack.Rack
			if err := gorp.NewRetrieve[rack.Key, rack.Rack]().
				WhereKeys(incomingState.Rack, existingState.Rack).
				Entries(&racks).
				Exec(ctx, t.cfg.DB); err != nil {
				t.cfg.L.Warn("failed to retrieve rack", zap.Error(err))
				return
			}
			t.cfg.L.Warn(
				"device state update with different rack key",
				zap.String("device", incomingState.Key),
				zap.Uint32("incoming_rack", uint32(existingState.Rack)),
				zap.String("incoming_rack_name", racks[0].Name),
				zap.Uint32("valid_rack", uint32(incomingState.Rack)),
				zap.String("valid_rack_name", racks[1].Name),
			)
			return
		}

		t.mu.Devices[incomingState.Key] = incomingState

		select {
		case t.deviceSaveNotifications <- incomingState.Key:
		default:
		}
	}
}

// handleDeviceChanges handles changes to devices in the DB.
func (t *Tracker) handleDeviceChanges(ctx context.Context, r gorp.TxReader[string, device.Device]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for c, ok := r.Next(ctx); ok; c, ok = r.Next(ctx) {
		if c.Variant == change.Delete {
			delete(t.mu.Devices, c.Key)
		} else {
			existing, hasState := t.mu.Devices[c.Key]
			existing.Key = c.Value.Key
			existing.Rack = c.Value.Rack
			if !hasState {
				existing.Variant = status.InfoVariant
			}
			t.mu.Devices[c.Key] = existing
		}
	}
}

func (t *Tracker) saveDeviceState(ctx context.Context, deviceKey string) error {
	state, ok := t.GetDevice(ctx, deviceKey)
	if !ok {
		return nil
	}
	if err := gorp.NewCreate[string, device.State]().Entry(&state).Exec(ctx, t.cfg.DB); err != nil {
		t.cfg.L.Warn("failed to save device state", zap.Error(err))
	}
	return nil
}
