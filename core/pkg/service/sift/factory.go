// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"context"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ScanTaskType is the type of the internal task that checks Sift devices and answers
// their commands.
const ScanTaskType = "sift_scan"

const scanTaskName = "Sift Scanner"

// FactoryConfig is the configuration for the Sift driver factory.
type FactoryConfig struct {
	alamos.Instrumentation
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Device retrieves Sift devices.
	//
	// [REQUIRED]
	Device *device.Service
	// Status writes the statuses of tasks and devices.
	//
	// [REQUIRED]
	Status *status.Service
	// Secret reads the API keys of Sift devices.
	//
	// [REQUIRED]
	Secret *secret.Service
	// Ranger retrieves the ranges that uploads send and that streams make runs of.
	//
	// [REQUIRED]
	Ranger *ranger.Service
	// Channel retrieves the channels that uploads and streams send.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer reads the samples that uploads and streams send.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Label reads the labels of ranges, which become the tags of Sift runs.
	//
	// [REQUIRED]
	Label *label.Service
	// Task follows the deletes and renames of stream tasks, which their progress
	// statuses follow.
	//
	// [REQUIRED]
	Task *task.Service
	// Timeout bounds one call to Sift from a check or a command.
	//
	// [OPTIONAL] - Defaults to 10 s.
	Timeout telem.TimeSpan
	// CheckpointInterval is how long uploads and streams send before they wait for
	// Sift to confirm what they sent and save their progress.
	//
	// [OPTIONAL] - Defaults to 10 s.
	CheckpointInterval telem.TimeSpan
}

var _ config.Config[FactoryConfig] = FactoryConfig{}

// DefaultFactoryConfig is the default configuration for the Sift driver factory.
var DefaultFactoryConfig = FactoryConfig{
	Timeout:            10 * telem.Second,
	CheckpointInterval: 10 * telem.Second,
}

// Override implements config.Config.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Device = override.Nil(c.Device, other.Device)
	c.Status = override.Nil(c.Status, other.Status)
	c.Secret = override.Nil(c.Secret, other.Secret)
	c.Ranger = override.Nil(c.Ranger, other.Ranger)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Label = override.Nil(c.Label, other.Label)
	c.Task = override.Nil(c.Task, other.Task)
	c.Timeout = override.Numeric(c.Timeout, other.Timeout)
	c.CheckpointInterval = override.Numeric(
		c.CheckpointInterval,
		other.CheckpointInterval,
	)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("sift.factory")
	v.NotNil("db", c.DB)
	v.NotNil("device", c.Device)
	v.NotNil("status", c.Status)
	v.NotNil("secret", c.Secret)
	v.NotNil("ranger", c.Ranger)
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NotNil("label", c.Label)
	v.NotNil("task", c.Task)
	v.Positive("timeout", c.Timeout)
	v.Positive("checkpoint_interval", c.CheckpointInterval)
	return v.Error()
}

// Factory configures the Sift task types. It also deletes and renames the progress
// status of each stream task as the task is deleted or renamed, whether or not the task
// runs, and repairs the changes it missed when it opens. Safe for concurrent use.
type Factory struct {
	cfg FactoryConfig
	mu  sync.Mutex
	// uploads holds the uploads of each rack.
	uploads map[rack.Key]*rackUploads
	// stopping holds, for each rack whose scan task stops, a channel that closes when
	// the task releases the rack.
	stopping map[rack.Key]chan struct{}
	// wake signals that a task was deleted or a stream task was set.
	wake       chan struct{}
	disconnect observe.Disconnect
	shutdown   io.Closer
}

var _ driver.Factory = (*Factory)(nil)

// OpenFactory opens the factory for the Sift task types. The caller must Close it after
// the last of its tasks stops.
func OpenFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	f := &Factory{
		cfg:      cfg,
		uploads:  make(map[rack.Key]*rackUploads),
		stopping: make(map[rack.Key]chan struct{}),
		wake:     make(chan struct{}, 1),
	}
	f.disconnect = cfg.Task.Observe().OnChange(f.onTaskChange)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(f.followTasks, signal.RecoverWithErrOnPanic())
	f.shutdown = signal.NewHardShutdown(sCtx, cancel)
	return f, nil
}

// Close stops following the deletes and renames of stream tasks.
func (f *Factory) Close() error {
	f.disconnect()
	return f.shutdown.Close()
}

// onTaskChange runs in the commit of a task change, so it wakes followTasks and never
// waits.
func (f *Factory) onTaskChange(
	_ context.Context,
	r gorp.TxReader[task.Key, task.Task],
) {
	for ch := range r {
		if ch.Kind == gorp.ChangeDelete || ch.Value.Type == StreamTaskType {
			select {
			case f.wake <- struct{}{}:
			default:
			}
			return
		}
	}
}

// followTasks repairs the progress statuses at once and after each wake until ctx is
// cancelled.
func (f *Factory) followTasks(ctx context.Context) error {
	for {
		f.repairProgress(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-f.wake:
		}
	}
}

// repairProgress deletes each progress status that no stream task has, and gives each
// other one the name of its task. It logs each status it fails to repair, which the
// next repair tries again.
func (f *Factory) repairProgress(ctx context.Context) {
	var (
		statuses []status.Status[any]
		tasks    []task.Task
	)
	if err := f.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		if err := f.cfg.Status.NewRetrieve[any]().
			WherePrefix(streamKeyPrefix).
			Entries(&statuses).
			Exec(ctx, snap); err != nil {
			return err
		}
		return f.cfg.Task.NewRetrieve().
			Where(task.MatchTypes(StreamTaskType)).
			Entries(&tasks).
			Exec(ctx, snap)
	}); err != nil {
		f.cfg.L.Error("failed to read stream progress", zap.Error(err))
		return
	}
	names := make(map[status.Key]string, len(tasks))
	for _, t := range tasks {
		names[ProgressKey(t.Key)] = t.Name
	}
	for _, s := range statuses {
		name, ok := names[s.Key]
		if ok && name == s.Name {
			continue
		}
		err := f.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			w := f.cfg.Status.NewWriter(tx)
			if !ok {
				return w.Delete(ctx, s.Key)
			}
			return w.Rename(ctx, s.Key, name)
		})
		// A user can delete the progress status of a task that still exists.
		if errors.Skip(err, query.ErrNotFound) != nil {
			f.cfg.L.Error(
				"failed to repair stream progress",
				zap.String("key", s.Key),
				zap.Error(err),
			)
		}
	}
}

// Name implements driver.Factory.
func (f *Factory) Name() string { return Make }

// InitialTasks implements driver.Factory.
func (f *Factory) InitialTasks() []task.Task {
	return []task.Task{{
		Name:   scanTaskName,
		Type:   ScanTaskType,
		Config: msgpack.EncodedJSON{},
	}}
}

// ConfigureTask implements driver.Factory.
func (f *Factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	switch t.Type {
	case ScanTaskType:
		return f.configureScanTask(ctx, t, cmdKey)
	case StreamTaskType:
		return f.configureStreamTask(ctx, t, cmdKey)
	default:
		return nil, driver.ErrTaskNotHandled
	}
}

func (f *Factory) reporter() driver.ConfigReporter {
	return driver.ConfigReporter{
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Instrumentation: f.cfg.Instrumentation,
	}
}

func (f *Factory) configureStreamTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	var cfg StreamConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		f.reporter().Report(ctx, t, cmdKey, err)
		return nil, err
	}
	if err := validateStreamConfig(cfg); err != nil {
		f.reporter().Report(ctx, t, cmdKey, err)
		return nil, err
	}
	st, err := newStreamTask(f.cfg, t, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.AutoStart && cmdKey == driver.NoCommand {
		if err = st.Start(ctx, cmdKey); err != nil {
			return nil, err
		}
	}
	return st, nil
}

func (f *Factory) configureScanTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	st, autoStart, err := f.configureScan(t)
	if err == nil {
		err = f.claim(ctx, t)
	}
	if err != nil {
		f.reporter().Report(ctx, t, cmdKey, err)
		return nil, err
	}
	if autoStart && cmdKey == driver.NoCommand {
		if err = st.Start(ctx, cmdKey); err != nil {
			return nil, errors.Combine(err, f.release(t))
		}
	}
	return scanTask{
		ScanTask:  st,
		releasing: func() { f.releasing(t) },
		release:   func() error { return f.release(t) },
	}, nil
}

// rackUploads is the uploader of a rack and the scan task that runs it. Each instance
// of the task holds a reference, so an instance built while another still runs shares
// its uploader.
type rackUploads struct {
	task     task.Key
	uploader *uploader
	refs     int
}

// claim makes t the scan task that runs the uploads of its rack, and opens the uploader
// of the rack when no instance of t holds it. It waits for a scan task that stops to
// release the rack, and fails when another scan task runs the uploads, since two would
// send each upload twice.
func (f *Factory) claim(ctx context.Context, t task.Task) error {
	for {
		f.mu.Lock()
		u, held := f.uploads[t.Rack]
		released := f.stopping[t.Rack]
		switch {
		case !held:
			f.uploads[t.Rack] = &rackUploads{
				task:     t.Key,
				uploader: openUploader(f.cfg, t.Rack),
				refs:     1,
			}
			f.mu.Unlock()
			return nil
		case u.task == t.Key && u.refs > 0:
			u.refs++
			f.mu.Unlock()
			return nil
		}
		f.mu.Unlock()
		if released == nil {
			return validate.Errorf("the rack already has a Sift scan task")
		}
		select {
		case <-released:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// releasing makes claim wait for the last instance of t to release its rack instead of
// failing.
func (f *Factory) releasing(t task.Task) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.uploads[t.Rack]
	if ok && u.task == t.Key && u.refs == 1 && f.stopping[t.Rack] == nil {
		f.stopping[t.Rack] = make(chan struct{})
	}
}

// release drops the reference of an instance of t to the uploads of its rack. The last
// one closes the uploader before it releases the rack, so a waiting claim opens its
// uploader only after the old one saves its progress.
func (f *Factory) release(t task.Task) error {
	f.mu.Lock()
	u, ok := f.uploads[t.Rack]
	if !ok || u.task != t.Key {
		f.mu.Unlock()
		return nil
	}
	u.refs--
	if u.refs > 0 {
		f.unblock(t.Rack)
		f.mu.Unlock()
		return nil
	}
	if f.stopping[t.Rack] == nil {
		f.stopping[t.Rack] = make(chan struct{})
	}
	f.mu.Unlock()
	// Close waits for Sift to confirm each upload in flight.
	err := u.uploader.Close()
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.uploads, t.Rack)
	f.unblock(t.Rack)
	return err
}

// unblock wakes each claim that waits for the rack rk. f.mu must be held.
func (f *Factory) unblock(rk rack.Key) {
	if released, ok := f.stopping[rk]; ok {
		close(released)
		delete(f.stopping, rk)
	}
}

// scanTask is the sift_scan task. Besides checking devices, it runs the uploads of
// its rack whether or not scanning is on.
type scanTask struct {
	*driver.ScanTask
	releasing func()
	release   func() error
}

// Stop implements driver.Task.
func (t scanTask) Stop(sendStatus bool) error {
	t.releasing()
	return errors.Combine(t.ScanTask.Stop(sendStatus), t.release())
}

func validateScanConfig(cfg *ScanConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	v := validate.New("sift.scan")
	v.Ternaryf("rate", cfg.Rate.Period() <= 0, "%s is too high", cfg.Rate)
	return v.Error()
}

// configureScan returns the scan task of t, and whether the factory starts it.
func (f *Factory) configureScan(t task.Task) (*driver.ScanTask, bool, error) {
	var cfg ScanConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	if err := validateScanConfig(&cfg); err != nil {
		return nil, false, err
	}
	st, err := driver.NewScanTask(driver.ScanTaskConfig{
		Instrumentation:    f.cfg.Child("scan"),
		Scanner:            scanner{cfg: f.cfg, rack: t.Rack},
		DB:                 f.cfg.DB,
		Status:             f.cfg.Status,
		Device:             f.cfg.Device,
		Make:               Make,
		ReachableMessage:   "Connected to Sift",
		UnreachableMessage: "Failed to connect to Sift",
		Task:               t,
		Interval:           cfg.Rate.Period().Duration(),
		CheckTimeout:       f.cfg.Timeout.Duration(),
	})
	if err != nil {
		return nil, false, err
	}
	return st, !cfg.Disabled, nil
}
