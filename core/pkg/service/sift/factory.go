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
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
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
	v.Positive("timeout", c.Timeout)
	v.Positive("checkpoint_interval", c.CheckpointInterval)
	return v.Error()
}

type factory struct {
	cfg FactoryConfig
	mu  sync.Mutex
	// uploaders holds the scan task that runs the uploads of each rack.
	uploaders map[rack.Key]task.Key
}

var _ driver.Factory = (*factory)(nil)

// NewFactory returns a driver factory for Sift tasks.
func NewFactory(cfgs ...FactoryConfig) (driver.Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &factory{cfg: cfg, uploaders: make(map[rack.Key]task.Key)}, nil
}

// Name implements driver.Factory.
func (f *factory) Name() string { return Make }

// InitialTasks implements driver.Factory.
func (f *factory) InitialTasks() []task.Task {
	return []task.Task{{
		Name:   scanTaskName,
		Type:   ScanTaskType,
		Config: msgpack.EncodedJSON{},
	}}
}

// ConfigureTask implements driver.Factory.
func (f *factory) ConfigureTask(
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

func (f *factory) reporter() driver.ConfigReporter {
	return driver.ConfigReporter{
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Instrumentation: f.cfg.Instrumentation,
	}
}

func (f *factory) configureStreamTask(
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

func (f *factory) configureScanTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	st, autoStart, err := f.configureScan(t)
	if err == nil {
		err = f.claim(t)
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
		ScanTask: st,
		uploads:  openUploader(f.cfg, t.Rack),
		release:  func() { f.release(t) },
	}, nil
}

// claim makes t the scan task that runs the uploads of its rack. It fails when another
// scan task runs them, since two would send each upload twice.
func (f *factory) claim(t task.Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if key, ok := f.uploaders[t.Rack]; ok && key != t.Key {
		return validate.Errorf("the rack already has a Sift scan task")
	}
	f.uploaders[t.Rack] = t.Key
	return nil
}

func (f *factory) release(t task.Task) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.uploaders[t.Rack] == t.Key {
		delete(f.uploaders, t.Rack)
	}
}

// scanTask is the sift_scan task. Besides checking devices, it runs the uploads of
// its rack whether or not scanning is on.
type scanTask struct {
	*driver.ScanTask
	uploads *uploader
	release func()
}

// Stop implements driver.Task.
func (t scanTask) Stop(sendStatus bool) error {
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
func (f *factory) configureScan(t task.Task) (*driver.ScanTask, bool, error) {
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
