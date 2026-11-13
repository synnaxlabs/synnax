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
	"github.com/synnaxlabs/x/set"
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
// runs. Safe for concurrent use.
type Factory struct {
	cfg FactoryConfig
	mu  sync.Mutex
	// uploaders holds the scan task that runs the uploads of each rack.
	uploaders map[rack.Key]task.Key
	// stopping holds, for each rack whose scan task stops, a channel that closes when
	// the task releases the rack.
	stopping map[rack.Key]chan struct{}
	// changed holds the tasks that were deleted, and the latest name of each stream
	// task that was set, since followTasks last ran.
	changed struct {
		sync.Mutex
		deleted set.Set[task.Key]
		renamed map[task.Key]string
	}
	// wake signals that changed holds a task.
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
		cfg:       cfg,
		uploaders: make(map[rack.Key]task.Key),
		stopping:  make(map[rack.Key]chan struct{}),
		wake:      make(chan struct{}, 1),
	}
	f.changed.deleted = make(set.Set[task.Key])
	f.changed.renamed = make(map[task.Key]string)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(f.followTasks, signal.RecoverWithErrOnPanic())
	f.shutdown = signal.NewHardShutdown(sCtx, cancel)
	f.disconnect = cfg.Task.Observe().OnChange(f.onTaskChange)
	return f, nil
}

// Close stops following the deletes and renames of stream tasks.
func (f *Factory) Close() error {
	f.disconnect()
	return f.shutdown.Close()
}

// onTaskChange runs in the commit of a task change, so it hands the changed stream
// tasks to followTasks and never waits.
func (f *Factory) onTaskChange(
	_ context.Context,
	r gorp.TxReader[task.Key, task.Task],
) {
	f.changed.Lock()
	defer f.changed.Unlock()
	for ch := range r {
		switch {
		case ch.Kind == gorp.ChangeDelete:
			f.changed.deleted.Add(ch.Key)
			delete(f.changed.renamed, ch.Key)
		case ch.Value.Type == StreamTaskType:
			f.changed.deleted.Remove(ch.Key)
			f.changed.renamed[ch.Key] = ch.Value.Name
		default:
			continue
		}
		select {
		case f.wake <- struct{}{}:
		default:
		}
	}
}

// followTasks deletes the progress status of each deleted task and renames the progress
// status of each renamed stream task until ctx is cancelled.
func (f *Factory) followTasks(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.wake:
		}
		f.changed.Lock()
		deleted, renamed := f.changed.deleted, f.changed.renamed
		f.changed.deleted = make(set.Set[task.Key])
		f.changed.renamed = make(map[task.Key]string)
		f.changed.Unlock()
		if err := f.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return f.updateProgress(ctx, tx, deleted, renamed)
		}); err != nil {
			f.cfg.L.Error("failed to update stream progress", zap.Error(err))
		}
	}
}

func (f *Factory) updateProgress(
	ctx context.Context,
	tx gorp.Tx,
	deleted set.Set[task.Key],
	renamed map[task.Key]string,
) error {
	w := f.cfg.Status.NewWriter(tx)
	keys := make([]status.Key, 0, len(deleted))
	for key := range deleted {
		keys = append(keys, ProgressKey(key))
	}
	if err := w.Delete(ctx, keys...); err != nil {
		return err
	}
	for key, name := range renamed {
		var s ProgressStatus
		err := f.cfg.Status.NewRetrieve[StreamDetails]().
			Where(status.MatchKeys[StreamDetails](ProgressKey(key))).
			Entry(&s).
			Exec(ctx, tx)
		// A stream task that never started has no progress.
		if errors.Is(err, query.ErrNotFound) || (err == nil && s.Name == name) {
			continue
		}
		if err != nil {
			return err
		}
		if err = w.Rename(ctx, s.Key, name); err != nil {
			return err
		}
	}
	return nil
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
		f.reporter().Report(ctx, t, cmdKey, false, err)
		return nil, err
	}
	if err := validateStreamConfig(cfg); err != nil {
		f.reporter().Report(ctx, t, cmdKey, cfg.AutoStart, err)
		return nil, err
	}
	st, err := newStreamTask(f.cfg, t, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.AutoStart {
		if err = st.Start(ctx, driver.NoCommand); err != nil {
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
		f.reporter().Report(ctx, t, cmdKey, false, err)
		return nil, err
	}
	if autoStart {
		if err = st.Start(ctx, driver.NoCommand); err != nil {
			f.release(t)
			return nil, err
		}
	}
	return scanTask{
		ScanTask:  st,
		uploads:   openUploader(f.cfg, t.Rack),
		releasing: func() { f.releasing(t) },
		release:   func() { f.release(t) },
	}, nil
}

// claim makes t the scan task that runs the uploads of its rack. It waits for a scan
// task that stops to release the rack, and fails when another scan task runs the
// uploads, since two would send each upload twice.
func (f *Factory) claim(ctx context.Context, t task.Task) error {
	for {
		f.mu.Lock()
		key, held := f.uploaders[t.Rack]
		released := f.stopping[t.Rack]
		if !held || key == t.Key {
			f.uploaders[t.Rack] = t.Key
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

// releasing makes claim wait for t to release its rack instead of failing.
func (f *Factory) releasing(t task.Task) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.uploaders[t.Rack] == t.Key && f.stopping[t.Rack] == nil {
		f.stopping[t.Rack] = make(chan struct{})
	}
}

func (f *Factory) release(t task.Task) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.uploaders[t.Rack] != t.Key {
		return
	}
	delete(f.uploaders, t.Rack)
	if released, ok := f.stopping[t.Rack]; ok {
		close(released)
		delete(f.stopping, t.Rack)
	}
}

// scanTask is the sift_scan task. Besides checking devices, it runs the uploads of
// its rack whether or not scanning is on.
type scanTask struct {
	*driver.ScanTask
	uploads   *uploader
	releasing func()
	release   func()
}

// Stop implements driver.Task.
func (t scanTask) Stop(sendStatus bool) error {
	t.releasing()
	defer t.release()
	return errors.Combine(t.uploads.Close(), t.ScanTask.Stop(sendStatus))
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
