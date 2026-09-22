// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// FactoryConfig is the configuration for the MQTT task factory.
type FactoryConfig struct {
	// Device retrieves broker devices.
	//
	// [REQUIRED]
	Device *device.Service
	// Channel retrieves the channels that tasks read and write.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer opens the writers and streamers of tasks.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status writes task and device statuses.
	//
	// [REQUIRED]
	Status *status.Service
	alamos.Instrumentation
	// QueueSize is the count of messages that the queue of one read task holds
	// before it drops the oldest.
	//
	// [OPTIONAL] - Defaults to 4096.
	QueueSize int
	// RebirthInterval is the shortest time between two Sparkplug B rebirth requests to
	// one edge node.
	//
	// [OPTIONAL] - Defaults to 5s.
	RebirthInterval time.Duration
	// BirthGrace is how long an edge node has to answer a rebirth request before a
	// read task reports it as offline. It must be longer than RebirthInterval, or a
	// request that waits out the interval gives a false warning.
	//
	// [OPTIONAL] - Defaults to 8s.
	BirthGrace time.Duration
}

var (
	_ config.Config[FactoryConfig] = FactoryConfig{}
	// DefaultFactoryConfig is the default configuration for the MQTT task factory.
	DefaultFactoryConfig = FactoryConfig{
		QueueSize:       4096,
		RebirthInterval: sparkplug.DefaultRebirthInterval,
		BirthGrace:      sparkplug.DefaultRebirthInterval + 3*time.Second,
	}
)

// Override implements config.Config.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Device = override.Nil(c.Device, other.Device)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Status = override.Nil(c.Status, other.Status)
	c.QueueSize = override.Numeric(c.QueueSize, other.QueueSize)
	c.RebirthInterval = override.Numeric(c.RebirthInterval, other.RebirthInterval)
	c.BirthGrace = override.Numeric(c.BirthGrace, other.BirthGrace)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("mqtt.factory")
	v.NotNil("device", c.Device)
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NotNil("status", c.Status)
	v.Positive("queue_size", c.QueueSize)
	v.Positive("rebirth_interval", c.RebirthInterval)
	v.Ternary(
		"birth_grace",
		c.BirthGrace <= c.RebirthInterval,
		"must be longer than rebirth_interval",
	)
	return v.Error()
}

type factory struct {
	pool *pool
	cfg  FactoryConfig
}

var _ driver.Factory = (*factory)(nil)

// NewFactory creates the factory for the MQTT task types. Its tasks share one
// connection for each broker device.
func NewFactory(cfgs ...FactoryConfig) (driver.Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &factory{
		cfg:  cfg,
		pool: newPool(cfg.Instrumentation, cfg.RebirthInterval),
	}, nil
}

// Name implements driver.Factory.
func (f *factory) Name() string { return Make }

// InitialTasks implements driver.Factory.
func (f *factory) InitialTasks() []task.Task {
	return []task.Task{{Name: "MQTT Scanner", Type: ScanTaskType}}
}

// autoStarter is a task that a factory starts when its config asks for it.
type autoStarter interface {
	driver.Task
	Start(ctx context.Context, cmdKey string) error
}

// ConfigureTask implements driver.Factory.
func (f *factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	var (
		configured autoStarter
		autoStart  bool
		err        error
	)
	switch t.Type {
	case ReadTaskType:
		configured, autoStart, err = f.configureRead(ctx, t)
	case WriteTaskType:
		configured, autoStart, err = f.configureWrite(ctx, t)
	case ScanTaskType:
		configured, autoStart, err = f.configureScan(t)
	default:
		return nil, driver.ErrTaskNotHandled
	}
	if err != nil {
		driver.ReportConfigError(
			ctx, f.cfg.Instrumentation, f.cfg.Status, t, cmdKey, autoStart, err,
		)
		return nil, err
	}
	// A successful configure writes no status: the start that follows it answers the
	// command, and a "configured" status would answer it first with running false.
	if autoStart {
		if err = configured.Start(ctx, driver.NoCommand); err != nil {
			return nil, err
		}
	}
	return configured, nil
}

func (f *factory) retrieveDevice(
	ctx context.Context,
	key device.Key,
) (device.Device, error) {
	var dev device.Device
	if key == "" {
		return dev, errors.Wrap(validate.ErrValidation, "device: a broker is required")
	}
	err := f.cfg.Device.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&dev).
		Exec(ctx, nil)
	if errors.Is(err, query.ErrNotFound) {
		return dev, errors.Wrapf(
			validate.ErrValidation, "device: broker %s does not exist", key,
		)
	}
	if err != nil {
		return dev, err
	}
	if dev.Make != Make {
		return dev, errors.Wrapf(
			validate.ErrValidation, "device: %s is not an MQTT broker", dev.Name,
		)
	}
	// Bad properties fail the configure, not the first start.
	_, err = newClientConfig(dev)
	return dev, err
}

func (f *factory) retrieveChannels(
	ctx context.Context,
	keys channel.Keys,
) (map[channel.Key]channel.Channel, error) {
	var channels []channel.Channel
	if len(keys) > 0 {
		if err := f.cfg.Channel.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Entries(&channels).
			Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
			return nil, err
		}
	}
	byKey := make(map[channel.Key]channel.Channel, len(channels))
	for _, ch := range channels {
		byKey[ch.Key()] = ch
	}
	return byKey, nil
}

func (f *factory) configureRead(
	ctx context.Context,
	t task.Task,
) (autoStarter, bool, error) {
	var cfg ReadConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	dev, err := f.retrieveDevice(ctx, cfg.Device)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	var (
		enabled []ReadEntryVariant
		keys    channel.Keys
	)
	for _, e := range cfg.Entries {
		switch entry := e.Variant.(type) {
		case PlainReadEntry:
			if entry.Disabled {
				continue
			}
			for _, field := range entry.Fields {
				if !field.Disabled {
					keys = append(keys, field.Channel)
				}
			}
		case SparkplugReadEntry:
			if entry.Disabled {
				continue
			}
			keys = append(keys, entry.Channel)
		}
		enabled = append(enabled, e.Variant)
	}
	if len(enabled) == 0 {
		return nil, cfg.AutoStart, errors.Wrap(
			validate.ErrValidation, "entries: the task has no enabled entries",
		)
	}
	channels, err := f.retrieveChannels(ctx, keys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	src := &readSource{
		pool:       f.pool,
		dev:        dev,
		queueSize:  f.cfg.QueueSize,
		birthGrace: f.cfg.BirthGrace,
		topics:     make(map[string]readTopic),
		tags:       make(map[tagID]readTag),
		lastStamp:  make(map[channel.Key]telem.TimeStamp),
	}
	var (
		// written names the entry that writes to each channel.
		written    = make(map[channel.Key]string)
		writerKeys channel.Keys
	)
	for _, variant := range enabled {
		var (
			name      string
			entryKeys channel.Keys
		)
		switch entry := variant.(type) {
		case PlainReadEntry:
			topic, err := newReadTopic(entry, channels)
			if err != nil {
				return nil, cfg.AutoStart, err
			}
			if _, ok := src.topics[topic.topic]; ok {
				return nil, cfg.AutoStart, errors.Wrapf(
					validate.ErrValidation,
					"entries: topic %s appears more than once", topic.topic,
				)
			}
			src.topics[topic.topic] = topic
			name, entryKeys = "topic "+topic.topic, topic.keys
		case SparkplugReadEntry:
			tag, err := newReadTag(entry, channels)
			if err != nil {
				return nil, cfg.AutoStart, err
			}
			if _, ok := src.tags[tag.tagID]; ok {
				return nil, cfg.AutoStart, errors.Wrapf(
					validate.ErrValidation, "entries: %s appears more than once", tag,
				)
			}
			src.tags[tag.tagID] = tag
			name, entryKeys = tag.String(), tag.keys()
		}
		// Two entries on one index would race for the rising order of its timestamps.
		for _, key := range entryKeys {
			if other, ok := written[key]; ok {
				return nil, cfg.AutoStart, errors.Wrapf(
					validate.ErrValidation,
					"entries: %s and %s write to the same channel %d",
					other, name, key,
				)
			}
			written[key] = name
			writerKeys = append(writerKeys, key)
		}
	}
	mode := framer.WriterModePersistStream
	if cfg.DataSavingDisabled {
		mode = framer.WriterModeStreamOnly
	}
	rt, err := driver.NewReadTask(driver.ReadTaskConfig{
		Instrumentation: f.cfg.Child(t.Key.String()),
		Source:          src,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        writerKeys,
		Task:            t,
		Mode:            mode,
	})
	return rt, cfg.AutoStart, err
}

func (f *factory) configureWrite(
	ctx context.Context,
	t task.Task,
) (autoStarter, bool, error) {
	var cfg WriteConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	dev, err := f.retrieveDevice(ctx, cfg.Device)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	var keys channel.Keys
	for _, tg := range cfg.Targets {
		switch variant := tg.Variant.(type) {
		case PlainWriteTarget:
			if !variant.Disabled {
				keys = append(keys, variant.Channel.Channel)
			}
		case SparkplugWriteTarget:
			if !variant.Disabled {
				keys = append(keys, variant.Channel)
			}
		}
	}
	if len(keys) == 0 {
		return nil, cfg.AutoStart, errors.Wrap(
			validate.ErrValidation, "targets: the task has no enabled targets",
		)
	}
	channels, err := f.retrieveChannels(ctx, keys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	sink := &writeSink{
		pool:    f.pool,
		dev:     dev,
		targets: make(map[channel.Key][]target, len(keys)),
	}
	for _, tg := range cfg.Targets {
		var (
			key  channel.Key
			name string
		)
		switch variant := tg.Variant.(type) {
		case PlainWriteTarget:
			if variant.Disabled {
				continue
			}
			key, name = variant.Channel.Channel, variant.Topic
		case SparkplugWriteTarget:
			if variant.Disabled {
				continue
			}
			key, name = variant.Channel, variant.Tag
		}
		ch, ok := channels[key]
		if !ok {
			return nil, cfg.AutoStart, errors.Wrapf(
				validate.ErrValidation,
				"target %s: channel %d does not exist", name, key,
			)
		}
		var built target
		switch variant := tg.Variant.(type) {
		case PlainWriteTarget:
			built, err = newWriteTarget(variant, ch)
		case SparkplugWriteTarget:
			built, err = newCommandTarget(variant, ch)
		}
		if err != nil {
			return nil, cfg.AutoStart, err
		}
		sink.targets[key] = append(sink.targets[key], built)
	}
	wt, err := driver.NewWriteTask(driver.WriteTaskConfig{
		Instrumentation: f.cfg.Child(t.Key.String()),
		Sink:            sink,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        keys.Unique(),
		Task:            t,
	})
	return wt, cfg.AutoStart, err
}

func (f *factory) configureScan(t task.Task) (autoStarter, bool, error) {
	var cfg ScanConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, true, err
	}
	scanCfg := driver.ScanTaskConfig{
		Instrumentation:    f.cfg.Child("scan"),
		Scanner:            &scanner{pool: f.pool, device: f.cfg.Device},
		Status:             f.cfg.Status,
		Device:             f.cfg.Device,
		Make:               Make,
		ReachableMessage:   "Broker connected",
		UnreachableMessage: "Failed to reach broker",
		Task:               t,
	}
	if cfg.Rate > 0 {
		scanCfg.Interval = cfg.Rate.Period().Duration()
	}
	st, err := driver.NewScanTask(scanCfg)
	return st, !cfg.Disabled, err
}
