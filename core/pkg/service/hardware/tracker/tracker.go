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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// RackStatus is the state of a hardware rack. Unfortunately, we can't put this into the
// rack package because it would create a circular dependency.
type RackStatus struct {
	rack.Status
	// TaskStatuses is the state of the tasks associated with the rack.
	TaskStatuses map[task.Key]task.Status `json:"taskstatuses" msgpack:"taskstatuses"`
}

// Alive returns true if the rack is alive.
func (r RackStatus) Alive(threshold telem.TimeSpan) bool {
	return telem.Since(r.Time) < threshold
}

// Tracker is used to track the state of hardware racks and tasks.
type Tracker struct {
	cfg Config
	// mu is a read-write lock used to protect the state of the tracker. Any fields
	// inside of this struct should be accessed while holding the lock.
	mu struct {
		sync.RWMutex
		// Racks is the map of racks to their corresponding state.
		Racks map[rack.Key]*RackStatus
		// Devices is the map of devices to their corresponding state.
		Devices map[string]device.Status
	}
	// saveNotifications is used to signal an observing go-routine to save the state of
	// a task to gorp. This ensures that the most recent task state is persisted across
	// reloads.
	saveNotifications chan task.Key
	// deviceSaveNotifications is used to signal an observing go-routine to save the
	// state of a device to gorp. This ensures that the most recent device state is
	// persisted across reloads.
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
	//
	// [OPTIONAL]
	alamos.Instrumentation
	// Rack is the service used to retrieve rack information.
	//
	// [REQUIRED]
	Rack *rack.Service
	// Task is the service used to retrieve task information.
	//
	// [REQUIRED]
	Task *task.Service
	// Device is the service used to retrieve device information.
	//
	// [REQUIRED]
	Device *device.Service
	// Signals is used to subscribe to changes in rack and task state.
	//
	// [REQUIRED]
	Signals *signals.Provider
	// Channels is used to create channels for the tracker service.
	//
	// [REQUIRED]
	Channels channel.ReadWriteable
	// HostProvider returns information about the cluster host.
	//
	// [REQUIRED]
	HostProvider cluster.HostProvider
	// DB is used to persist and retrieve information about rack and task state.
	//
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
	DefaultConfig = Config{RackStateAliveThreshold: telem.Second * 3}
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
	c.RackStateAliveThreshold =
		override.Numeric(c.RackStateAliveThreshold, other.RackStateAliveThreshold)
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

const (
	DeviceStatusChannelName = "sy_device_status"
	RackStatusChannelName   = "sy_rack_status"
	TaskStatusChannelName   = "sy_task_status"
)

var legacyChannelNames = map[string]string{
	"sy_device_state": DeviceStatusChannelName,
	"sy_rack_state":   RackStatusChannelName,
	"sy_task_state":   TaskStatusChannelName,
}

// Open opens a new hardware state tracker with the provided configuration. The Tracker
// must be closed after use.
func Open(ctx context.Context, configs ...Config) (*Tracker, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	var racks []rack.Rack
	if err = cfg.Rack.NewRetrieve().
		WhereNode(cfg.HostProvider.HostKey()).
		Entries(&racks).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	t := &Tracker{cfg: cfg}
	t.mu.Racks = make(map[rack.Key]*RackStatus, len(racks))
	t.mu.Devices = make(map[string]device.Status)

	for _, r := range racks {
		rck := &RackStatus{
			Status:       newUnknownRackStatus(r.Key),
			TaskStatuses: make(map[task.Key]task.Status),
		}

		// Fetch and initialize tasks for this rack
		var tasks []task.Task
		if err = cfg.Task.NewRetrieve().
			WhereRacks(r.Key).
			Entries(&tasks).
			Exec(ctx, nil); err != nil {
			return nil, err
		}

		for _, tsk := range tasks {
			if tsk.Snapshot {
				continue
			}
			taskState := newUnknownTaskStatus(tsk.Key)
			if err = gorp.NewRetrieve[task.Key, task.Status]().
				WhereKeys(tsk.Key).
				Entry(&taskState).
				Exec(ctx, cfg.DB); err != nil && !errors.IsAny(err, query.NotFound, binary.DecodeError) {
				return nil, err
			}
			rck.TaskStatuses[tsk.Key] = taskState
		}

		t.mu.Racks[r.Key] = rck
	}

	// Fetch and initialize all devices
	var allDevices []device.Device
	if err = cfg.Device.NewRetrieve().
		Entries(&allDevices).
		Exec(ctx, cfg.DB); err != nil {
		return nil, err
	}

	for _, dev := range allDevices {
		deviceState := newUnknownDeviceStatus(dev.Key, dev.Rack)
		if err = gorp.NewRetrieve[string, device.Status]().
			WhereKeys(dev.Key).
			Entry(&deviceState).
			Exec(ctx, cfg.DB); err != nil && !errors.IsAny(err, query.NotFound, binary.DecodeError) {
			return nil, err
		}
		t.mu.Devices[dev.Key] = deviceState
	}

	if err =
		cfg.Channels.DeleteByName(ctx, "sy_rack_heartbeat", true); err != nil {
		return nil, err
	}
	channels := []channel.Channel{
		{
			Name:        TaskStatusChannelName,
			DataType:    telem.JSONT,
			Leaseholder: cfg.HostProvider.HostKey(),
			Virtual:     true,
			Internal:    true,
		},
		{
			Name:        RackStatusChannelName,
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
			Name:        DeviceStatusChannelName,
			DataType:    telem.JSONT,
			Leaseholder: cfg.HostProvider.HostKey(),
			Virtual:     true,
			Internal:    true,
		},
	}
	tx := cfg.DB.OpenTx()
	defer func() {
		err = errors.Combine(err, tx.Close())
	}()
	w := cfg.Channels.NewWriter(tx)
	if err = w.MapRename(ctx, legacyChannelNames, true); err != nil {
		return nil, err
	}
	if err = w.CreateMany(
		ctx,
		&channels,
		channel.OverwriteIfNameExistsAndDifferentProperties(),
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
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
		SetChannelName: RackStatusChannelName,
	})
	if err != nil {
		return nil, err
	}
	dcRackStateObs := rackStateObs.OnChange(t.handleRackState)
	taskStateObs, closeTaskStateObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: TaskStatusChannelName,
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
	t.stateWriter = taskStateWriterStream
	obs := confluence.NewObservableSubscriber[framer.WriterResponse]()
	obs.OnChange(func(ctx context.Context, r framer.WriterResponse) {
		cfg.L.Error("unexpected writer error", zap.Int("seq_num", r.SeqNum), zap.Error(r.Err))
	})
	outlets := confluence.NewStream[framer.WriterResponse](1)
	obs.InFrom(outlets)
	stateWriter.OutTo(outlets)
	stateWriter.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	dcTaskStateObs := taskStateObs.OnChange(t.handleTaskState)
	t.saveNotifications = make(chan task.Key, 10)
	signal.GoRange(sCtx, t.saveNotifications, t.saveTaskState, signal.WithKey("save_task_state"))
	t.deviceSaveNotifications = make(chan string, 10)
	signal.GoRange(sCtx, t.deviceSaveNotifications, t.saveDeviceState, signal.WithKey("save_device_state"))
	deviceStateObs, closeDeviceStateObs, err := cfg.Signals.Subscribe(sCtx, signals.ObservableSubscriberConfig{
		SetChannelName: DeviceStatusChannelName,
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
	}, signal.WithKey("check_rack_state"))
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
	return t, nil
}

// GetTask returns the state of a task by its key. If the task is not found, the second
// return value will be false.
func (t *Tracker) GetTask(_ context.Context, key task.Key) (task.Status, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, exists := t.mu.Racks[key.Rack()]
	if !exists {
		return task.Status{}, false
	}
	tsk, exists := r.TaskStatuses[key]
	return tsk, exists
}

// GetRack returns the state of a rack by its key. If the rack is not found, the second
// return value will be false.
func (t *Tracker) GetRack(_ context.Context, key rack.Key) (RackStatus, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, exists := t.mu.Racks[key]
	if !exists {
		return RackStatus{}, false
	}
	return *r, true
}

// GetDevice returns the state of a device by its key. If the device is not found, the second
// return value will be false.
func (t *Tracker) GetDevice(_ context.Context, deviceKey string) (device.Status, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	d, exists := t.mu.Devices[deviceKey]
	return d, exists
}

// Close closes the tracker, freeing all associated go-routines and resources.
// The tracker must not be used after it is closed.
func (t *Tracker) Close() error {
	return t.closer.Close()
}

// handleTaskChanges handles changes to tasks in the DB.
func (t *Tracker) handleTaskChanges(ctx context.Context, r gorp.TxReader[task.Key, task.Task]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for c := range r {
		rackKey := c.Key.Rack()
		if c.Variant == change.Delete {
			delete(t.mu.Racks[rackKey].TaskStatuses, c.Key)
			continue
		}
		rackState, rackExists := t.mu.Racks[rackKey]
		if !rackExists {
			rackState = &RackStatus{
				Status:       newUnknownRackStatus(rackKey),
				TaskStatuses: make(map[task.Key]task.Status),
			}
			t.mu.Racks[rackKey] = rackState
		}
		if _, taskExists := rackState.TaskStatuses[c.Key]; !taskExists {
			rackState.TaskStatuses[c.Key] = newUnknownTaskStatus(c.Key)
		}
		alive := rackState.Alive(t.cfg.RackStateAliveThreshold)
		if !rackExists || !alive {
			s := task.Status{
				Time:    telem.Now(),
				Message: "rack is not alive",
				Variant: status.WarningVariant,
				Details: task.StatusDetails{Task: c.Key, Running: true},
			}
			if rackExists {
				var rck rack.Rack
				if err := gorp.NewRetrieve[rack.Key, rack.Rack]().
					WhereKeys(rackKey).
					Entry(&rck).
					Exec(ctx, t.cfg.DB); err != nil {
					t.cfg.L.Warn("failed to retrieve rack", zap.Error(err))
				}
				s.Details.Running = false
				s.Message = fmt.Sprintf(
					"Synnax Driver on %s is not running, so the task may fail to configure. Driver was last alive %s ago.",
					rck.Name,
					telem.Since(rackState.Time).Truncate(telem.Second))
			}
			t.stateWriter.Inlet() <- framer.WriterRequest{
				Command: writer.Write,
				Frame:   core.UnaryFrame(t.taskStateChannelKey, telem.NewSeriesStaticJSONV(s)),
			}
		}
	}
}

// handleRackChanges handles changes to racks in the DB.
func (t *Tracker) handleRackChanges(ctx context.Context, r gorp.TxReader[rack.Key, rack.Rack]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for c := range r {
		if c.Variant == change.Delete {
			delete(t.mu.Racks, c.Key)
			continue
		}
		if _, rackExists := t.mu.Racks[c.Key]; !rackExists {
			t.mu.Racks[c.Key] = &RackStatus{
				TaskStatuses: make(map[task.Key]task.Status),
				Status:       newUnknownRackStatus(c.Key),
			}

		}
	}
}

func (t *Tracker) checkRackState(ctx context.Context) {
	rackStatuses := make([]rack.Status, 0, len(t.mu.Racks))
	taskStatuses := make([]task.Status, 0, len(t.mu.Racks))
	deviceStatuses := make([]device.Status, 0)

	for _, rackStatus := range t.mu.Racks {
		if rackStatus.Alive(t.cfg.RackStateAliveThreshold) {
			continue
		}
		rackStatus.Time = telem.Now()
		rackStatus.Variant = status.WarningVariant
		rackStatus.Message = fmt.Sprintf("Driver %s is not alive", rackStatus.Key)
		rackStatuses = append(rackStatuses, rackStatus.Status)

		var rck rack.Rack
		if err := gorp.NewRetrieve[rack.Key, rack.Rack]().
			WhereKeys(rackStatus.Details.Rack).
			Entry(&rck).
			Exec(ctx, t.cfg.DB); err != nil {
			t.cfg.L.Warn("failed to retrieve rack", zap.Error(err))
			continue
		}

		msg := fmt.Sprintf(
			"Synnax Driver on %s is not running. Driver was last alive %s ago.",
			rck.Name,
			telem.Since(rackStatus.Time).Truncate(telem.Second),
		)
		for _, taskState := range rackStatus.TaskStatuses {
			taskState.Time = telem.Now()
			taskState.Variant = status.WarningVariant
			taskState.Message = msg
			taskState.Details.Running = false
			taskStatuses = append(taskStatuses, taskState)
		}

		for _, dev := range t.mu.Devices {
			if dev.Details.Rack == rackStatus.Details.Rack {
				dev.Time = telem.Now()
				dev.Variant = status.WarningVariant
				dev.Message = msg
				deviceStatuses = append(deviceStatuses, dev)
			}
		}

	}

	fr := core.Frame{}
	if len(rackStatuses) > 0 {
		fr = fr.Append(t.rackStateChannelKey, telem.NewSeriesStaticJSONV(rackStatuses...))
	}
	if len(taskStatuses) > 0 {
		fr = fr.Append(t.taskStateChannelKey, telem.NewSeriesStaticJSONV(taskStatuses...))
	}
	if len(deviceStatuses) > 0 {
		fr = fr.Append(t.deviceStateChannelKey, telem.NewSeriesStaticJSONV(deviceStatuses...))
	}
	if fr.Empty() {
		return
	}

	t.stateWriter.Inlet() <- framer.WriterRequest{Command: writer.Write, Frame: fr}
}

// handleRackState handles heartbeat changes.
func (t *Tracker) handleRackState(_ context.Context, changes []change.Change[[]byte, struct{}]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	decoder := &binary.JSONCodec{}
	for _, ch := range changes {
		var rackState rack.Status
		if err := decoder.Decode(context.Background(), ch.Key, &rackState); err != nil {
			t.cfg.L.Warn("failed to decode rack state", zap.Error(err))
			continue
		}
		r, ok := t.mu.Racks[rackState.Details.Rack]
		if !ok {
			t.cfg.L.Warn(
				"rack not found for state update",
				zap.Uint32("rack", uint32(rackState.Details.Rack)),
			)
			continue
		}
		r.Status = rackState
		r.Time = telem.Now()
	}
}

// handleTaskState handles task state changes.
func (t *Tracker) handleTaskState(ctx context.Context, changes []change.Change[[]byte, struct{}]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	decoder := &binary.JSONCodec{}
	for _, ch := range changes {
		var taskState task.Status
		if err := decoder.Decode(ctx, ch.Key, &taskState); err != nil {
			t.cfg.L.Warn("failed to decode task state", zap.Error(err))
			continue
		}
		rackKey := taskState.Details.Task.Rack()
		r, ok := t.mu.Racks[rackKey]
		if !ok {
			t.cfg.L.Warn("rack not found for task state update", zap.Uint64("task", uint64(taskState.Details.Task)))
		} else {
			r.TaskStatuses[taskState.Details.Task] = taskState
		}
		select {
		case t.saveNotifications <- taskState.Details.Task:
		default:
		}
	}
}

func (t *Tracker) saveTaskState(ctx context.Context, taskKey task.Key) error {
	state, ok := t.GetTask(ctx, taskKey)
	if !ok {
		return nil
	}
	if err := gorp.NewCreate[task.Key, task.Status]().Entry(&state).Exec(ctx, t.cfg.DB); err != nil {
		t.cfg.L.Warn("failed to save task state", zap.Error(err))
	}
	return nil
}

// handleDeviceState handles device state changes.
func (t *Tracker) handleDeviceState(ctx context.Context, changes []change.Change[[]byte, struct{}]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	decoder := &binary.JSONCodec{}
	for _, ch := range changes {
		var incomingState device.Status
		if err := decoder.Decode(ctx, ch.Key, &incomingState); err != nil {
			t.cfg.L.Warn("failed to decode device state", zap.Error(err))
			continue
		}

		devKey := incomingState.Details.Device
		existingState, exists := t.mu.Devices[devKey]
		if exists && existingState.Details.Rack != incomingState.Details.Rack {
			var racks []rack.Rack
			if err := gorp.NewRetrieve[rack.Key, rack.Rack]().
				WhereKeys(incomingState.Details.Rack, existingState.Details.Rack).
				Entries(&racks).
				Exec(ctx, t.cfg.DB); err != nil {
				t.cfg.L.Warn("failed to retrieve rack", zap.Error(err))
				return
			}
			t.cfg.L.Warn(
				"device state update with different rack key",
				zap.String("device", incomingState.Key),
				zap.Uint32("incoming_rack", uint32(existingState.Details.Rack)),
				zap.String("incoming_rack_name", racks[0].Name),
				zap.Uint32("valid_rack", uint32(incomingState.Details.Rack)),
				zap.String("valid_rack_name", racks[1].Name),
			)
			return
		}

		t.mu.Devices[devKey] = incomingState

		select {
		case t.deviceSaveNotifications <- devKey:
		default:
		}
	}
}

// handleDeviceChanges handles changes to devices in the DB.
func (t *Tracker) handleDeviceChanges(ctx context.Context, r gorp.TxReader[string, device.Device]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for c := range r {
		if c.Variant == change.Delete {
			delete(t.mu.Devices, c.Key)
			continue
		}
		existing, hasState := t.mu.Devices[c.Key]
		existing.Time = telem.Now()
		existing.Key = c.Value.Key
		existing.Details.Rack = c.Value.Rack
		existing.Details.Device = c.Key
		if !hasState {
			existing.Variant = status.InfoVariant
		}
		t.mu.Devices[c.Key] = existing

	}
}

func (t *Tracker) saveDeviceState(ctx context.Context, deviceKey string) error {
	state, exists := t.GetDevice(ctx, deviceKey)
	if !exists {
		return nil
	}
	if err := gorp.NewCreate[string, device.Status]().Entry(&state).Exec(ctx, t.cfg.DB); err != nil {
		t.cfg.L.Warn("failed to save device state", zap.Error(err))
	}
	return nil
}

func newUnknownTaskStatus(key task.Key) task.Status {
	s := task.Status{}
	s.Key = key.String()
	s.Time = telem.Now()
	s.Details.Task = key
	s.Variant = status.WarningVariant
	s.Message = "Task state unknown"
	return s
}

func newUnknownRackStatus(key rack.Key) rack.Status {
	s := rack.Status{}
	s.Key = key.String()
	s.Details.Rack = key
	s.Time = telem.Now()
	s.Variant = status.WarningVariant
	s.Message = "Rack state unknown"
	return s
}

func newUnknownDeviceStatus(devKey string, rackKey rack.Key) device.Status {
	s := device.Status{}
	s.Key = devKey
	s.Time = telem.Now()
	s.Variant = status.WarningVariant
	s.Message = "Device state unknown"
	s.Details.Rack = rackKey
	s.Details.Device = devKey
	return s
}
