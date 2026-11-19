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
	"io"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// FactoryConfig is the configuration for the MQTT task factory.
type FactoryConfig struct {
	// DB holds the records that the factory reads, and stores task statuses.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Device retrieves and observes broker devices.
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
	// ICD retrieves the ICDs that ICD read tasks parse payloads with.
	//
	// [REQUIRED]
	ICD *icd.Service
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
	// NewClient builds the client of each connection of a read, write, or edge task.
	//
	// [OPTIONAL] - Defaults to paho.NewClient.
	NewClient func(*paho.ClientOptions) paho.Client
	// Now returns the wall clock that edge tasks stamp their commands from.
	//
	// [OPTIONAL] - Defaults to telem.Now.
	Now func() telem.TimeStamp
}

var (
	_ config.Config[FactoryConfig] = FactoryConfig{}
	// DefaultFactoryConfig is the default configuration for the MQTT task factory.
	DefaultFactoryConfig = FactoryConfig{
		QueueSize:       4096,
		RebirthInterval: sparkplug.DefaultRebirthInterval,
		BirthGrace:      sparkplug.DefaultRebirthInterval + 3*time.Second,
		NewClient:       paho.NewClient,
		Now:             telem.Now,
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
	c.QueueSize = override.Numeric(c.QueueSize, other.QueueSize)
	c.RebirthInterval = override.Numeric(c.RebirthInterval, other.RebirthInterval)
	c.BirthGrace = override.Numeric(c.BirthGrace, other.BirthGrace)
	c.NewClient = override.Nil(c.NewClient, other.NewClient)
	c.Now = override.Nil(c.Now, other.Now)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("mqtt.factory")
	v.NotNil("db", c.DB)
	v.NotNil("device", c.Device)
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NotNil("status", c.Status)
	v.NotNil("icd", c.ICD)
	v.Positive("queue_size", c.QueueSize)
	v.Positive("rebirth_interval", c.RebirthInterval)
	v.Ternary(
		"birth_grace",
		c.BirthGrace <= c.RebirthInterval,
		"must be longer than rebirth_interval",
	)
	return v.Error()
}

// Factory configures the MQTT task types. Its tasks share one connection for each
// broker device, and an edit of the properties of a broker device reaches that
// connection, whether or not a scan task runs. Safe for concurrent use.
type Factory struct {
	pool     *pool
	reporter driver.ConfigReporter
	cfg      FactoryConfig
	// edited holds the keys of the broker devices that changed since the last
	// reconfigure.
	edited struct {
		keys set.Set[device.Key]
		sync.Mutex
	}
	// wake signals that edited holds a key.
	wake       chan struct{}
	disconnect observe.Disconnect
	shutdown   io.Closer
}

var _ driver.Factory = (*Factory)(nil)

// OpenFactory opens the factory for the MQTT task types. The caller must Close it after
// the last of its tasks stops.
func OpenFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	f := &Factory{
		cfg: cfg,
		pool: newPool(
			cfg.Instrumentation,
			cfg.DB,
			cfg.Device,
			cfg.RebirthInterval,
			cfg.NewClient,
		),
		reporter: driver.ConfigReporter{
			DB:              cfg.DB,
			Status:          cfg.Status,
			Instrumentation: cfg.Instrumentation,
		},
		wake: make(chan struct{}, 1),
	}
	f.edited.keys = make(set.Set[device.Key])
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(f.reconfigureEdited, signal.RecoverWithErrOnPanic())
	f.shutdown = signal.NewHardShutdown(sCtx, cancel)
	f.disconnect = cfg.Device.Observe().OnChange(f.onDeviceChange)
	return f, nil
}

// Close stops following the edits of broker devices.
func (f *Factory) Close() error {
	f.disconnect()
	return f.shutdown.Close()
}

// onDeviceChange runs in the commit of a device change, so it hands the edited brokers
// to reconfigureEdited and never waits.
func (f *Factory) onDeviceChange(
	_ context.Context,
	r gorp.TxReader[device.Key, device.Device],
) {
	for ch := range r {
		if ch.Kind == gorp.ChangeDelete || ch.Value.Make != Make {
			continue
		}
		f.edited.Lock()
		f.edited.keys.Add(ch.Key)
		f.edited.Unlock()
		select {
		case f.wake <- struct{}{}:
		default:
		}
	}
}

// reconfigureEdited points the connection of each edited broker at its new properties
// until ctx is cancelled.
func (f *Factory) reconfigureEdited(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.wake:
		}
		f.edited.Lock()
		keys := f.edited.keys
		f.edited.keys = make(set.Set[device.Key])
		f.edited.Unlock()
		for key := range keys {
			f.pool.reconfigure(ctx, key)
		}
	}
}

// Name implements driver.Factory.
func (f *Factory) Name() string { return Make }

// InitialTasks implements driver.Factory.
func (f *Factory) InitialTasks() []task.Task {
	return []task.Task{{Name: "MQTT Scanner", Type: ScanTaskType}}
}

// autoStarter is a task that a factory starts when its config asks for it.
type autoStarter interface {
	driver.Task
	Start(ctx context.Context, cmdKey string) error
}

// ConfigureTask implements driver.Factory.
func (f *Factory) ConfigureTask(
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
	case EdgeTaskType:
		configured, autoStart, err = f.configureEdge(ctx, t)
	default:
		return nil, driver.ErrTaskNotHandled
	}
	if err != nil {
		f.reporter.Report(ctx, t, cmdKey, err)
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

func (f *Factory) retrieveDevice(
	ctx context.Context,
	key device.Key,
) (device.Device, error) {
	var dev device.Device
	if key == "" {
		return device.Device{}, errors.Wrap(
			validate.ErrValidation,
			"device: a broker is required",
		)
	}
	err := f.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return f.cfg.Device.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&dev).
			Exec(ctx, snap)
	})
	if errors.Is(err, query.ErrNotFound) {
		return device.Device{}, errors.Wrapf(
			validate.ErrValidation, "device: broker %s does not exist", key,
		)
	}
	if err != nil {
		return device.Device{}, err
	}
	if dev.Make != Make {
		return device.Device{}, errors.Wrapf(
			validate.ErrValidation, "device: %s is not an MQTT broker", dev.Name,
		)
	}
	// Bad properties fail the configure, not the first start.
	if _, err = newClientConfig(dev); err != nil {
		return device.Device{}, err
	}
	return dev, nil
}

func (f *Factory) retrieveChannels(
	ctx context.Context,
	keys channel.Keys,
) (map[channel.Key]channel.Channel, error) {
	var channels []channel.Channel
	if len(keys) > 0 {
		if err := f.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			return f.cfg.Channel.NewRetrieve().
				Where(channel.MatchKeys(keys...)).
				Entries(&channels).
				Exec(ctx, snap)
		}); err != nil && !errors.Is(err, query.ErrNotFound) {
			return nil, err
		}
	}
	byKey := make(map[channel.Key]channel.Channel, len(channels))
	for _, ch := range channels {
		byKey[ch.Key()] = ch
	}
	return byKey, nil
}

func (f *Factory) retrieveICD(ctx context.Context, key icd.Key) (icd.ICD, error) {
	var ic icd.ICD
	err := f.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return f.cfg.ICD.NewRetrieve().
			Where(icd.MatchKeys(key)).
			Entry(&ic).
			Exec(ctx, snap)
	})
	if errors.Is(err, query.ErrNotFound) {
		return icd.ICD{}, errors.Wrapf(
			validate.ErrValidation, "icd: ICD %s does not exist", key,
		)
	}
	return ic, err
}

func (f *Factory) configureRead(
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
	enabled, keys, icdKeys, err := enabledReadEntries(cfg.Entries)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	channels, err := f.retrieveChannels(ctx, keys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	icds, err := f.retrieveICDs(ctx, icdKeys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	src := &readSource{
		pool:       f.pool,
		broker:     dev.Key,
		queueSize:  f.cfg.QueueSize,
		birthGrace: f.cfg.BirthGrace,
		topics:     make(map[string]*readTopic),
		tags:       make(map[tagID]readTag),
		lastStamp:  make(map[channel.Key]telem.TimeStamp),
		framer:     f.cfg.Framer,
		subject:    control.Subject{Key: t.Key.String(), Name: t.Name},
		ins:        f.cfg.Child(t.Key.String()),
	}
	writerKeys, indexes, err := src.addEntries(enabled, icds, channels)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	rt, err := f.openRead(t, src, writerKeys, indexes, cfg.DataSavingDisabled)
	return rt, cfg.AutoStart, err
}

func (f *Factory) openRead(
	t task.Task,
	src *readSource,
	writerKeys, indexes channel.Keys,
	dataSavingDisabled bool,
) (autoStarter, error) {
	mode := framer.WriterModePersistStream
	if dataSavingDisabled {
		mode = framer.WriterModeStreamOnly
	} else {
		src.stored = indexes
	}
	rt, err := driver.NewReadTask(driver.ReadTaskConfig{
		Instrumentation: f.cfg.Child(t.Key.String()),
		Source:          src,
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        writerKeys,
		Task:            t,
		Mode:            mode,
	})
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// enabledReadEntries returns the enabled entries, the channels that they name, and the
// ICDs that their messages belong to.
func enabledReadEntries(
	entries []ReadEntry,
) ([]ReadEntryVariant, channel.Keys, []icd.Key, error) {
	var (
		enabled []ReadEntryVariant
		keys    channel.Keys
		icds    []icd.Key
	)
	for _, e := range entries {
		switch entry := e.Variant.(type) {
		case PlainReadEntry:
			if entry.Disabled {
				continue
			}
			icds = append(icds, entry.ICD)
			for _, m := range entry.Messages {
				for _, b := range m.Bindings {
					keys = append(keys, b.Channel)
				}
			}
		case SparkplugReadEntry:
			if entry.Disabled {
				continue
			}
			keys = append(keys, entry.Channel)
		case nil:
			return nil, nil, nil, errors.Wrap(
				validate.ErrValidation, "entries: an entry is null",
			)
		default:
			panic(errors.Newf("mqtt: unknown read entry variant %T", entry))
		}
		enabled = append(enabled, e.Variant)
	}
	if len(enabled) == 0 {
		return nil, nil, nil, errors.Wrap(
			validate.ErrValidation, "entries: the task has no enabled entries",
		)
	}
	return enabled, keys, icds, nil
}

// retrieveICDs returns the ICDs of keys, by key.
func (f *Factory) retrieveICDs(
	ctx context.Context,
	keys []icd.Key,
) (map[icd.Key]icd.ICD, error) {
	icds := make(map[icd.Key]icd.ICD)
	for _, key := range keys {
		if _, ok := icds[key]; ok {
			continue
		}
		ic, err := f.retrieveICD(ctx, key)
		if err != nil {
			return nil, err
		}
		icds[key] = ic
	}
	return icds, nil
}

// jsonMessageOf returns the message of ic with key, and its JSON payload.
func jsonMessageOf(
	ic icd.ICD,
	key icd.MessageKey,
) (icd.Message, icd.JSONPayload, error) {
	for _, m := range ic.Messages {
		if m.Key != key {
			continue
		}
		payload, ok := m.Payload.Variant.(icd.JSONPayload)
		if !ok {
			return icd.Message{}, icd.JSONPayload{}, errors.Wrapf(
				validate.ErrValidation, "message %s is not a JSON message", m.Name,
			)
		}
		return m, payload, nil
	}
	return icd.Message{}, icd.JSONPayload{}, errors.Wrapf(
		validate.ErrValidation, "message %s does not exist in ICD %s", key, ic.Name,
	)
}

// writers holds the channels that the messages and tags of a read task write to.
type writers struct {
	// names names the message or tag that writes to each channel.
	names   map[channel.Key]string
	keys    channel.Keys
	indexes channel.Keys
}

// claim takes the channels of keys and their index for the writer name. Two writers on
// one index would race for the rising order of its timestamps, or write frames that
// lack the channels of the other.
func (w *writers) claim(name string, keys channel.Keys, index channel.Key) error {
	for _, key := range keys {
		if other, ok := w.names[key]; ok {
			return errors.Wrapf(
				validate.ErrValidation,
				"entries: %s and %s write to the same channel %d",
				other, name, key,
			)
		}
		w.names[key] = name
		w.keys = append(w.keys, key)
	}
	if index != 0 {
		w.indexes = append(w.indexes, index)
	}
	return nil
}

// addEntries adds the topics and tags of enabled to s. icds holds the ICD of each plain
// entry. It returns the channels that the entries write to and their index channels.
func (s *readSource) addEntries(
	enabled []ReadEntryVariant,
	icds map[icd.Key]icd.ICD,
	channels map[channel.Key]channel.Channel,
) (writerKeys, indexes channel.Keys, err error) {
	w := writers{names: make(map[channel.Key]string)}
	for _, variant := range enabled {
		if err = s.addEntry(variant, icds, channels, &w); err != nil {
			return nil, nil, err
		}
	}
	return w.keys, w.indexes, nil
}

// addEntry adds the topic or tag of variant to s, and claims its channels in w.
func (s *readSource) addEntry(
	variant ReadEntryVariant,
	icds map[icd.Key]icd.ICD,
	channels map[channel.Key]channel.Channel,
	w *writers,
) error {
	switch entry := variant.(type) {
	case PlainReadEntry:
		topic, err := newReadTopic(entry, icds[entry.ICD], channels)
		if err != nil {
			return err
		}
		if _, ok := s.topics[topic.topic]; ok {
			return errors.Wrapf(
				validate.ErrValidation,
				"entries: topic %s appears more than once", topic.topic,
			)
		}
		s.topics[topic.topic] = topic
		for _, m := range topic.messages {
			name := "message " + m.name + " of topic " + topic.topic
			if err = w.claim(name, m.keys, m.index); err != nil {
				return err
			}
		}
		return nil
	case SparkplugReadEntry:
		tag, err := newReadTag(entry, channels)
		if err != nil {
			return err
		}
		if _, ok := s.tags[tag.tagID]; ok {
			return errors.Wrapf(
				validate.ErrValidation, "entries: %s appears more than once", tag,
			)
		}
		s.tags[tag.tagID] = tag
		return w.claim(tag.String(), tag.keys(), tag.index)
	default:
		panic(errors.Newf("mqtt: unknown read entry variant %T", entry))
	}
}

func (f *Factory) configureWrite(
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
	enabled, keys, icdKeys, err := enabledTargets(cfg.Targets)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	channels, err := f.retrieveChannels(ctx, keys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	icds, err := f.retrieveICDs(ctx, icdKeys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	sink := &writeSink{pool: f.pool, broker: dev.Key}
	if err = sink.addTargets(enabled, icds, channels); err != nil {
		return nil, cfg.AutoStart, err
	}
	wt, err := driver.NewWriteTask(driver.WriteTaskConfig{
		Instrumentation: f.cfg.Child(t.Key.String()),
		Sink:            sink,
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        keys.Unique(),
		Task:            t,
	})
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	return wt, cfg.AutoStart, nil
}

// enabledTargets returns the enabled targets, their command channels, and the ICDs that
// their messages belong to.
func enabledTargets(
	targets []WriteTarget,
) ([]WriteTargetVariant, channel.Keys, []icd.Key, error) {
	var (
		enabled []WriteTargetVariant
		keys    channel.Keys
		icds    []icd.Key
	)
	for _, tg := range targets {
		switch target := tg.Variant.(type) {
		case PlainWriteTarget:
			if target.Disabled {
				continue
			}
			icds = append(icds, target.ICD)
			for _, b := range target.Message.Bindings {
				keys = append(keys, b.Channel)
			}
		case SparkplugWriteTarget:
			if target.Disabled {
				continue
			}
			keys = append(keys, target.Channel)
		case nil:
			return nil, nil, nil, errors.Wrap(
				validate.ErrValidation, "targets: a target is null",
			)
		default:
			panic(errors.Newf("mqtt: unknown write target variant %T", target))
		}
		enabled = append(enabled, tg.Variant)
	}
	if len(enabled) == 0 {
		return nil, nil, nil, errors.Wrap(
			validate.ErrValidation, "targets: the task has no enabled targets",
		)
	}
	return enabled, keys, icds, nil
}

// addTargets adds the targets of enabled to s. icds holds the ICD of each plain target,
// and channels holds the command channels.
func (s *writeSink) addTargets(
	enabled []WriteTargetVariant,
	icds map[icd.Key]icd.ICD,
	channels map[channel.Key]channel.Channel,
) error {
	for _, variant := range enabled {
		var (
			built target
			err   error
		)
		switch target := variant.(type) {
		case PlainWriteTarget:
			built, err = newMessageTarget(target, icds[target.ICD], channels)
		case SparkplugWriteTarget:
			ch, ok := channels[target.Channel]
			if !ok {
				return errors.Wrapf(
					validate.ErrValidation,
					"target %s: channel %d does not exist", target.Tag, target.Channel,
				)
			}
			built, err = newCommandTarget(target, ch)
		default:
			panic(errors.Newf("mqtt: unknown write target variant %T", target))
		}
		if err != nil {
			return err
		}
		s.targets = append(s.targets, built)
	}
	return nil
}

func (f *Factory) configureEdge(
	ctx context.Context,
	t task.Task,
) (autoStarter, bool, error) {
	var cfg EdgeConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	dev, err := f.retrieveDevice(ctx, cfg.Device)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	node := sparkplug.NodeID{Group: cfg.Group, EdgeNode: cfg.EdgeNode}
	if err = node.Validate(); err != nil {
		return nil, cfg.AutoStart, err
	}
	keys, err := enabledTagKeys(cfg.Tags)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	channels, err := f.retrieveChannels(ctx, keys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	if _, err = newClientConfig(dev); err != nil {
		return nil, cfg.AutoStart, err
	}
	edge := &edgeNode{
		ins:       f.cfg.Child(t.Key.String()),
		broker:    dev.Key,
		db:        f.cfg.DB,
		devices:   f.cfg.Device,
		newClient: f.cfg.NewClient,
		command:   sparkplug.CommandTopic(node, "").String(),
		tags:      make(map[channel.Key][]publishedTag),
		byName:    make(map[string]publishedTag),
		framer:    f.cfg.Framer,
		task:      t,
		authority: control.Authority(cfg.Authority),
		clock:     telem.MonoClock{Source: f.cfg.Now},
	}
	declared, streamKeys, err := edge.addTags(node, cfg.Tags, channels)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	edge.edge = sparkplug.NewEdge(node, declared)
	edge.mu.seq = sparkplug.NewSequence(edge.edge)
	wt, err := driver.NewWriteTask(driver.WriteTaskConfig{
		Instrumentation: edge.ins,
		Sink:            edge,
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        streamKeys.Unique(),
		Task:            t,
	})
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	return wt, cfg.AutoStart, nil
}

// enabledTagKeys returns the channels and command channels of the enabled tags.
func enabledTagKeys(tags []EdgeTag) (channel.Keys, error) {
	var keys channel.Keys
	for _, tg := range tags {
		if tg.Disabled {
			continue
		}
		keys = append(keys, tg.Channel)
		if tg.CommandChannel != 0 {
			keys = append(keys, tg.CommandChannel)
		}
	}
	if len(keys) == 0 {
		return nil, errors.Wrap(
			validate.ErrValidation, "tags: the task has no enabled tags",
		)
	}
	return keys, nil
}

// addTags adds the enabled tags to e. It returns the tags to declare in births and the
// channels to stream.
func (e *edgeNode) addTags(
	node sparkplug.NodeID,
	tags []EdgeTag,
	channels map[channel.Key]channel.Channel,
) (declared []sparkplug.Tag, streamKeys channel.Keys, err error) {
	// commanded names the tag that writes to each command channel.
	commanded := make(map[channel.Key]string)
	for _, tg := range tags {
		if tg.Disabled {
			continue
		}
		et, tag, err := newPublishedTag(node, tg, channels)
		if err != nil {
			return nil, nil, err
		}
		if _, ok := e.byName[et.name]; ok {
			return nil, nil, errors.Wrapf(
				validate.ErrValidation, "tags: tag %s appears more than once", et.name,
			)
		}
		if et.command != nil {
			// One command would carry two values for one sample of the channel.
			if other, ok := commanded[et.command.channel]; ok {
				return nil, nil, errors.Wrapf(
					validate.ErrValidation,
					"tags: tag %s and tag %s write to the same command channel %d",
					other, et.name, et.command.channel,
				)
			}
			commanded[et.command.channel] = et.name
		}
		e.byName[et.name] = et
		e.tags[et.channel] = append(e.tags[et.channel], et)
		declared = append(declared, tag)
		streamKeys = append(streamKeys, et.channel)
		if et.index != 0 {
			streamKeys = append(streamKeys, et.index)
		}
	}
	return declared, streamKeys, nil
}

func (f *Factory) configureScan(t task.Task) (autoStarter, bool, error) {
	var cfg ScanConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, true, err
	}
	scanCfg := driver.ScanTaskConfig{
		Instrumentation: f.cfg.Child("scan"),
		Scanner: &scanner{
			ins:    f.cfg.Instrumentation,
			pool:   f.pool,
			db:     f.cfg.DB,
			device: f.cfg.Device,
		},
		DB:                 f.cfg.DB,
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
	if err != nil {
		return nil, !cfg.Disabled, err
	}
	return st, !cfg.Disabled, nil
}
