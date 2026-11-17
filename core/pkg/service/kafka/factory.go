// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"context"
	"io"
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
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

// scanTaskName is the name of the internal scan task the factory creates on its rack.
const scanTaskName = "Kafka Scanner"

// FactoryConfig is the configuration for the Kafka task factory.
type FactoryConfig struct {
	// DB holds the records that the factory reads, and stores task statuses.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Device resolves the cluster devices tasks connect to.
	//
	// [REQUIRED]
	Device *device.Service
	// Channel resolves the channels tasks read and write.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer opens the streamers and writers tasks move samples through.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status receives task and device statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// ICD holds the ICDs whose messages read tasks decode.
	//
	// [REQUIRED]
	ICD *icd.Service
	// PingTimeout bounds a connection test or health check ping.
	//
	// [OPTIONAL] - Defaults to 10 seconds.
	PingTimeout telem.TimeSpan
	// FetchMaxWait is the longest a broker holds a read task's fetch when the topic
	// has no new records.
	//
	// [OPTIONAL] - Defaults to 5 seconds.
	FetchMaxWait telem.TimeSpan
	alamos.Instrumentation
}

var (
	_                    config.Config[FactoryConfig] = FactoryConfig{}
	DefaultFactoryConfig                              = FactoryConfig{
		PingTimeout:  10 * telem.Second,
		FetchMaxWait: 5 * telem.Second,
	}
)

// Override implements config.Config.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Device = override.Nil(c.Device, other.Device)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Status = override.Nil(c.Status, other.Status)
	c.ICD = override.Nil(c.ICD, other.ICD)
	c.PingTimeout = override.Numeric(c.PingTimeout, other.PingTimeout)
	c.FetchMaxWait = override.Numeric(c.FetchMaxWait, other.FetchMaxWait)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("kafka.factory")
	v.NotNil("db", c.DB)
	v.NotNil("device", c.Device)
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NotNil("status", c.Status)
	v.NotNil("icd", c.ICD)
	v.Positive("ping_timeout", c.PingTimeout)
	v.Positive("fetch_max_wait", c.FetchMaxWait)
	return v.Error()
}

// Factory configures the Kafka task types. A change of an ICD configures again each
// configured read task that reads it. Safe for concurrent use.
type Factory struct {
	cfg FactoryConfig
	// edited holds the ICDs that changed since the last reconfigure.
	edited struct {
		icds set.Set[icd.Key]
		sync.Mutex
	}
	// wake signals that edited holds a key.
	wake chan struct{}
	// reads holds the read tasks that the driver has not stopped.
	reads struct {
		tasks set.Set[*readTask]
		sync.Mutex
	}
	disconnect observe.Disconnect
	shutdown   io.Closer
}

var (
	_ driver.Factory = (*Factory)(nil)
	_ io.Closer      = (*Factory)(nil)
)

func (f *Factory) reporter() driver.ConfigReporter {
	return driver.ConfigReporter{
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Instrumentation: f.cfg.Instrumentation,
	}
}

// OpenFactory opens a Factory for the Kafka scan, read, and write task types. Close it
// to stop following the changes of ICDs.
func OpenFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	f := &Factory{cfg: cfg, wake: make(chan struct{}, 1)}
	f.edited.icds = make(set.Set[icd.Key])
	f.reads.tasks = make(set.Set[*readTask])
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(f.reconfigureEdited, signal.RecoverWithErrOnPanic())
	f.shutdown = signal.NewHardShutdown(sCtx, cancel)
	f.disconnect = cfg.ICD.Observe().OnChange(f.onICDChange)
	return f, nil
}

// Close stops following the changes of ICDs.
func (f *Factory) Close() error {
	f.disconnect()
	return f.shutdown.Close()
}

// onICDChange runs in the commit of an ICD change, so it hands the changed ICDs to
// reconfigureEdited and never waits.
func (f *Factory) onICDChange(_ context.Context, r gorp.TxReader[icd.Key, icd.ICD]) {
	f.edited.Lock()
	for ch := range r {
		f.edited.icds.Add(ch.Key)
	}
	f.edited.Unlock()
	select {
	case f.wake <- struct{}{}:
	default:
	}
}

// reconfigureEdited configures again each read task that reads a changed ICD, until ctx
// is cancelled. The changes that arrive while it works go into one pass, which reads
// the latest version of each ICD.
func (f *Factory) reconfigureEdited(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.wake:
		}
		f.edited.Lock()
		icds := f.edited.icds
		f.edited.icds = make(set.Set[icd.Key])
		f.edited.Unlock()
		for _, t := range f.readsOf(icds) {
			f.reconfigureRead(ctx, t)
		}
	}
}

// readsOf returns the configured read tasks that read any of icds.
func (f *Factory) readsOf(icds set.Set[icd.Key]) []*readTask {
	f.reads.Lock()
	defer f.reads.Unlock()
	var tasks []*readTask
	for t := range f.reads.tasks {
		if slices.ContainsFunc(icds.Slice(), t.icds.Contains) {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

// forget stops following t, which the driver stopped.
func (f *Factory) forget(t *readTask) {
	f.reads.Lock()
	defer f.reads.Unlock()
	f.reads.tasks.Remove(t)
}

// reconfigureRead configures t again with the ICDs that it reads now. When the config
// no longer resolves, it stops t and gives it the status of a failed configure.
func (f *Factory) reconfigureRead(ctx context.Context, t *readTask) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.mu.stopped {
		return
	}
	records, groups, err := f.buildReadRecords(ctx, t.cfg)
	if err != nil && ctx.Err() != nil {
		return
	}
	t.reconfigure(records, groups, err)
	if err == nil {
		return
	}
	if stopErr := errors.Combine(t.stop(ctx, driver.NoCommand, false)); stopErr != nil {
		f.cfg.L.Error(
			"failed to stop read task",
			zap.Stringer("task", t.task.Key),
			zap.Error(stopErr),
		)
	}
	f.reporter().Report(ctx, t.task, driver.NoCommand, err)
}

// Name implements driver.Factory.
func (f *Factory) Name() string { return Make }

// InitialTasks implements driver.Factory, returning the rack's scan task.
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
		st, autoStart, err := f.configureScan(t)
		if err != nil {
			f.reporter().Report(ctx, t, cmdKey, err)
			return nil, err
		}
		// Start writes its own status when it fails.
		if autoStart {
			if err := st.Start(ctx, driver.NoCommand); err != nil {
				return nil, err
			}
		}
		return st, nil
	case ReadTaskType:
		rt, err := f.configureRead(ctx, t)
		if err != nil {
			f.reporter().Report(ctx, t, cmdKey, err)
			return nil, err
		}
		f.reads.Lock()
		f.reads.tasks.Add(rt)
		f.reads.Unlock()
		// A successful configure writes no status, so a start answers the command. An
		// auto-start that fails must answer it too, since no start follows.
		if rt.cfg.AutoStart {
			rt.mu.Lock()
			err := errors.Combine(rt.start(ctx, cmdKey))
			rt.mu.Unlock()
			if err != nil {
				// The driver drops a task that fails to start, so this stops it.
				return nil, errors.Combine(err, rt.Stop(false))
			}
		}
		return rt, nil
	case WriteTaskType:
		wt, err := f.configureWrite(ctx, t)
		if err != nil {
			f.reporter().Report(ctx, t, cmdKey, err)
			return nil, err
		}
		if wt.cfg.AutoStart {
			if err := errors.Combine(wt.start(ctx, cmdKey)); err != nil {
				return nil, err
			}
		}
		return wt, nil
	default:
		return nil, driver.ErrTaskNotHandled
	}
}

// resolveProperties retrieves the Kafka cluster device key and decodes its connection
// properties.
func resolveProperties(
	ctx context.Context,
	db *gorp.DB,
	svc *device.Service,
	key device.Key,
) (Properties, error) {
	var dev device.Device
	if err := db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return svc.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&dev).
			Exec(ctx, snap)
	}); err != nil {
		return Properties{}, errors.Wrapf(err, "retrieving device %s", key)
	}
	return decodeProperties(dev)
}

// retrieveChannels resolves keys to their channel records. A key with no channel is
// absent from the result.
func retrieveChannels(
	ctx context.Context,
	db *gorp.DB,
	svc *channel.Service,
	keys channel.Keys,
) (map[channel.Key]channel.Channel, error) {
	var channels []channel.Channel
	if err := db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&channels).
			Exec(ctx, snap)
	}); err != nil {
		return nil, err
	}
	byKey := make(map[channel.Key]channel.Channel, len(channels))
	for _, ch := range channels {
		byKey[ch.Key()] = ch
	}
	return byKey, nil
}

// retrieveICDs resolves keys to their ICDs. It returns an error wrapping
// validate.ErrValidation when a key does not exist.
func retrieveICDs(
	ctx context.Context,
	db *gorp.DB,
	svc *icd.Service,
	keys []icd.Key,
) (map[icd.Key]icd.ICD, error) {
	var icds []icd.ICD
	if err := db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return svc.NewRetrieve().
			Where(icd.MatchKeys(keys...)).
			Entries(&icds).
			Exec(ctx, snap)
	}); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return nil, errors.Wrap(validate.ErrValidation, err.Error())
		}
		return nil, err
	}
	byKey := make(map[icd.Key]icd.ICD, len(icds))
	for _, ic := range icds {
		byKey[ic.Key] = ic
	}
	return byKey, nil
}
