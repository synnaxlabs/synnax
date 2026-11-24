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
	"fmt"
	"iter"
	"slices"
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kgo"
)

// target produces the documents of one ICD message as records of one topic.
type target struct {
	name  string
	topic string
	// key is the key of each record, nil for none.
	key  []byte
	docs *write.Documents
	// indexes holds the index of each command channel of docs, in binding order, 0 for
	// a channel with no index.
	indexes channel.Keys
}

// newTarget returns the target of wt, whose message is in ic. channels holds the
// command channels of wt.
func newTarget(
	wt WriteTarget,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
) (*target, error) {
	name := "target " + wt.Topic
	docs, err := write.NewDocuments(wt.Message, ic, channels, name)
	if err != nil {
		return nil, errors.Wrap(err, name)
	}
	t := &target{name: name, topic: wt.Topic, docs: docs}
	if wt.RecordKey != "" {
		t.key = []byte(wt.RecordKey)
	}
	for _, key := range docs.Channels() {
		t.indexes = append(t.indexes, channels[key].Index())
	}
	return t, nil
}

// records returns the record of each document that the commands in fr build. now is
// the time of a record whose command channel has no index. records counts each command
// in skips as read, and each command that it drops as skipped.
func (t *target) records(
	fr framer.Frame,
	now telem.TimeStamp,
	skips *driver.Skips,
) iter.Seq[*kgo.Record] {
	return func(yield func(*kgo.Record) bool) {
		for row, doc := range t.docs.From(fr, skips) {
			ts, ok := t.stamp(fr, row, now)
			if !ok {
				skips.Skip(time.Now(), fmt.Sprintf(
					"Command for %s: the command arrived without its index", t.name,
				))
				continue
			}
			if !yield(&kgo.Record{
				Topic:     t.topic,
				Key:       t.key,
				Value:     doc,
				Timestamp: ts.Time(),
			}) {
				return
			}
		}
	}
}

// stamp returns the time of the record of row: the time in the index of the first
// command channel with a command at row, or now when that channel has no index. It
// returns false when fr holds no such time.
func (t *target) stamp(
	fr framer.Frame,
	row int,
	now telem.TimeStamp,
) (telem.TimeStamp, bool) {
	for b, key := range t.docs.Channels() {
		if fr.Get(key).Len() <= int64(row) {
			continue
		}
		if t.indexes[b] == 0 {
			return now, true
		}
		return sampleAt(fr, t.indexes[b], row)
	}
	return 0, false
}

// sampleAt returns sample i of the timestamps of key in fr, and false when fr holds
// fewer.
func sampleAt(fr framer.Frame, key channel.Key, i int) (telem.TimeStamp, bool) {
	for k, s := range fr.Entries() {
		if k != key {
			continue
		}
		if n := int(s.Len()); i >= n {
			i -= n
			continue
		}
		return s.ValueAt[telem.TimeStamp](i), true
	}
	return 0, false
}

// writeSink is the driver.Sink of a Kafka write task with targets.
type writeSink struct {
	cfg     FactoryConfig
	device  device.Key
	targets []*target
}

var _ driver.Sink = (*writeSink)(nil)

// Open implements driver.Sink. It connects to the cluster of the task.
func (s *writeSink) Open(
	ctx context.Context,
	skips *driver.Skips,
) (driver.Writer, error) {
	props, err := resolveProperties(ctx, s.cfg.DB, s.cfg.Device, s.device)
	if err != nil {
		return nil, err
	}
	cl, err := openClient(
		props,
		kgo.AllowAutoTopicCreation(),
		// Without it, the client waits out a lost cluster for a record in flight.
		kgo.AllowIdempotentProduceCancellation(),
	)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, s.cfg.PingTimeout.Duration())
	defer cancel()
	if err = cl.Ping(pingCtx); err != nil {
		cl.Close()
		return nil, errors.Wrap(err, "connecting to cluster")
	}
	for _, t := range s.targets {
		t.docs.Reset()
	}
	return &writer{writeSink: s, client: cl, skips: skips}, nil
}

// writer is the driver.Writer of one run of a Kafka write task with targets.
type writer struct {
	*writeSink
	client *kgo.Client
	// skips counts the commands that the run sends and drops.
	skips *driver.Skips
	// records holds the records of one Write.
	records []*kgo.Record
}

var _ driver.Writer = (*writer)(nil)

// Write implements driver.Writer. It produces the records of fr and waits for the
// cluster to take them. A record that the cluster has not taken by the ping timeout of
// the factory fails at the next retry of the client, and is never sent after that. A
// failed record returns an error that matches driver.ErrTemporary.
func (w *writer) Write(ctx context.Context, fr framer.Frame) error {
	now := telem.Now()
	w.records = w.records[:0]
	for _, t := range w.targets {
		w.records = slices.AppendSeq(w.records, t.records(fr, now, w.skips))
	}
	if len(w.records) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, w.cfg.PingTimeout.Duration())
	defer cancel()
	for _, res := range w.client.ProduceSync(ctx, w.records...) {
		if res.Err != nil {
			return errors.Mark(
				errors.Wrapf(res.Err, "producing to topic %s", res.Record.Topic),
				driver.ErrTemporary,
			)
		}
	}
	return nil
}

// Health implements driver.Writer. The client reports no change to its connections, so
// Health blocks until ctx is done.
func (w *writer) Health(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// Close implements driver.Writer.
func (w *writer) Close() error {
	w.client.Close()
	return nil
}

// validateTopics returns an error that names the first enabled target with no topic.
func validateTopics(targets []WriteTarget) error {
	for i, wt := range targets {
		if !wt.Disabled && wt.Topic == "" {
			return validate.PathedError(
				validate.Errorf("required"), "targets", strconv.Itoa(i), "topic",
			)
		}
	}
	return nil
}

// configureTargets returns the write task of t, which produces the records of the
// enabled targets of cfg.
func (f *factory) configureTargets(
	ctx context.Context,
	t task.Task,
	cfg WriteConfig,
) (*driver.WriteTask, error) {
	if _, err := resolveProperties(
		ctx,
		f.cfg.DB,
		f.cfg.Device,
		cfg.Device,
	); err != nil {
		return nil, err
	}
	if err := validateTopics(cfg.Targets); err != nil {
		return nil, err
	}
	enabled := lo.Filter(cfg.Targets, func(wt WriteTarget, _ int) bool {
		return !wt.Disabled
	})
	if len(enabled) == 0 {
		return nil, errors.Wrap(
			validate.ErrValidation, "targets: the task has no enabled targets",
		)
	}
	var (
		keys    channel.Keys
		icdKeys []icd.Key
	)
	for _, wt := range enabled {
		icdKeys = append(icdKeys, wt.ICD)
		for _, b := range write.ChannelBindings(wt.Message) {
			keys = append(keys, b.Channel)
		}
	}
	channels, err := retrieveChannels(ctx, f.cfg.DB, f.cfg.Channel, keys.Unique())
	if err != nil {
		return nil, err
	}
	icds, err := retrieveICDs(ctx, f.cfg.DB, f.cfg.ICD, lo.Uniq(icdKeys))
	if err != nil {
		return nil, err
	}
	sink := &writeSink{cfg: f.cfg, device: cfg.Device}
	streamed := keys
	for _, wt := range enabled {
		built, err := newTarget(wt, icds[wt.ICD], channels)
		if err != nil {
			return nil, err
		}
		sink.targets = append(sink.targets, built)
		streamed = append(streamed, lo.Compact(built.indexes)...)
	}
	return driver.NewWriteTask(driver.WriteTaskConfig{
		Instrumentation: f.cfg.Child(t.Key.String()),
		Sink:            sink,
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        streamed.Unique(),
		Task:            t,
	})
}
