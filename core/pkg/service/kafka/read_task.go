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
	"cmp"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// readGroup is the set of fields sharing one record key, one index, and one writer.
type readGroup struct {
	// key is the record key the group claims; empty claims every unclaimed record.
	key    string
	fields []ReadField
	// channels[i] is the channel fields[i] writes to.
	channels []channel.Channel
	// stamp is the position in fields of the field targeting the index channel, or
	// -1 when records are stamped with their receive time.
	stamp int
	// index is the group's index channel, zero when its channels are virtual.
	index channel.Key
	// keys are the writer's channels: the index first when it is not a field.
	keys channel.Keys
	// writer opens at the first write, so its start is the first timestamp written.
	writer *framer.Writer
	// last is the last timestamp written, seeded with the last stored one at start.
	last telem.TimeStamp
	// replaying is true until the first record past last arrives.
	replaying bool
}

// row is one record converted into a group's samples.
type row struct {
	ts telem.TimeStamp
	// values[i] is the sample for fields[i].
	values []any
}

// readTask consumes a topic and writes record fields into channels.
type readTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        ReadConfig
	// status is the authority on this instance's current status.
	status *driver.StatusHandler
	groups map[string]*readGroup
	// client is the consumer, nil while stopped.
	client *kgo.Client
	// closer releases the consumer, nil once released.
	closer io.Closer
	// sCtx is the consumer's context. A consumer that fails cancels it.
	sCtx signal.Context
	// lastSkip is the last skip reported, so a repeated one is not rewritten.
	lastSkip string
}

var _ driver.Task = (*readTask)(nil)

func (f *factory) configureRead(ctx context.Context, t task.Task) (*readTask, error) {
	var cfg ReadConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Topic == "" {
		return nil, errors.Wrap(validate.ErrValidation, "topic: must not be empty")
	}
	if _, err := resolveProperties(ctx, f.cfg.Device, cfg.Device); err != nil {
		return nil, err
	}
	groups, err := buildReadGroups(ctx, f.cfg.Channel, cfg)
	if err != nil {
		return nil, err
	}
	return &readTask{
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		status:     driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t),
		groups:     groups,
	}, nil
}

// buildReadGroups resolves the enabled fields of cfg into groups and validates that
// each group writes one even frame per record.
func buildReadGroups(
	ctx context.Context,
	svc *channel.Service,
	cfg ReadConfig,
) (map[string]*readGroup, error) {
	var keys channel.Keys
	for _, fld := range cfg.Fields {
		if fld.Disabled {
			continue
		}
		if fld.Pointer == "" {
			return nil, errors.Wrapf(
				validate.ErrValidation, "field %q: pointer must not be empty", fld.Key,
			)
		}
		if fld.Channel == 0 {
			return nil, errors.Wrapf(
				validate.ErrValidation, "field %q: channel must be set", fld.Key,
			)
		}
		keys = append(keys, fld.Channel)
	}
	if len(keys) == 0 {
		return nil, errors.Wrap(
			validate.ErrValidation, "fields: at least one must be enabled",
		)
	}
	channels, err := retrieveChannels(ctx, svc, keys)
	if err != nil {
		return nil, err
	}
	groups := make(map[string]*readGroup)
	for _, fld := range cfg.Fields {
		if fld.Disabled {
			continue
		}
		ch := channels[fld.Channel]
		if ch.DataType != fld.DataType {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"field %q: data type %s does not match channel %s (%s)",
				fld.Key, fld.DataType, ch.Name, ch.DataType,
			)
		}
		if ch.DataType == telem.TimestampT && fld.TimeFormat == nil {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"field %q: time_format is required for timestamp channel %s",
				fld.Key, ch.Name,
			)
		}
		g, ok := groups[fld.RecordKey]
		if !ok {
			g = &readGroup{key: fld.RecordKey, stamp: -1, replaying: true}
			groups[fld.RecordKey] = g
		}
		if ch.IsIndex {
			if g.stamp != -1 {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"record key %q: more than one field targets an index channel",
					g.key,
				)
			}
			g.stamp = len(g.fields)
		}
		g.fields = append(g.fields, fld)
		g.channels = append(g.channels, ch)
	}
	for _, g := range groups {
		for i, ch := range g.channels {
			index := ch.Index()
			if ch.IsIndex {
				index = ch.Key()
			}
			if i == 0 {
				g.index = index
			} else if index != g.index {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"record key %q: channels must share one index",
					g.key,
				)
			}
		}
		if g.index != 0 && g.stamp == -1 {
			g.keys = append(g.keys, g.index)
		}
		for _, ch := range g.channels {
			g.keys = append(g.keys, ch.Key())
		}
	}
	return groups, nil
}

// Exec implements driver.Task. Every command it handles ends in a status carrying the
// command key, so a caller waiting on the command always resolves.
func (t *readTask) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return t.start(ctx, cmd.Key)
	case "stop":
		return t.stop(ctx, cmd.Key, true)
	default:
		return driver.ErrUnsupportedCommand
	}
}

// Stop implements driver.Task.
func (t *readTask) Stop(sendStatus bool) error {
	return t.stop(context.TODO(), driver.NoCommand, sendStatus)
}

// isRunning reports whether the consumer is up. A consumer that failed on its own
// cancels sCtx without releasing the closer.
func (t *readTask) isRunning() bool { return t.closer != nil && t.sCtx.Err() == nil }

func (t *readTask) release() error {
	err := t.closer.Close()
	t.closer = nil
	t.client = nil
	return err
}

func (t *readTask) start(ctx context.Context, cmdKey string) error {
	if t.isRunning() {
		t.ackCurrent(ctx, cmdKey, true)
		return nil
	}
	if t.closer != nil {
		if err := t.release(); err != nil {
			t.factoryCfg.L.Warn("kafka consumer failed",
				zap.Stringer("task", t.task.Key),
				zap.Error(err),
			)
		}
	}
	if err := t.open(ctx); err != nil {
		t.setStatus(ctx, cmdKey, status.VariantError, false, err.Error())
		return err
	}
	t.setStatus(ctx, cmdKey, status.VariantSuccess, true, "Task started successfully")
	return nil
}

func (t *readTask) stop(ctx context.Context, cmdKey string, sendStatus bool) error {
	if t.closer == nil {
		if sendStatus {
			t.ackCurrent(ctx, cmdKey, false)
		}
		return nil
	}
	err := t.release()
	if !sendStatus {
		return err
	}
	if err != nil {
		t.setStatus(ctx, cmdKey, status.VariantError, false, err.Error())
		return err
	}
	t.setStatus(ctx, cmdKey, status.VariantSuccess, false, "Task stopped successfully")
	return nil
}

// resetOffset maps the configured start offset onto the consumer's reset offset. A
// group without a committed offset starts there.
func resetOffset(start StartOffset) kgo.Offset {
	switch start {
	case StartOffsetEarliest:
		return kgo.NewOffset().AtStart()
	case StartOffsetNone:
		return kgo.NoResetOffset()
	default:
		return kgo.NewOffset().AtEnd()
	}
}

// group returns the consumer group the task joins.
func (t *readTask) group() string {
	if t.cfg.Group != "" {
		return t.cfg.Group
	}
	return "kafka-" + t.task.Key.String()
}

// open connects the consumer and starts the fetch loop. It writes no statuses: start
// owns them.
func (t *readTask) open(ctx context.Context) (err error) {
	props, err := resolveProperties(ctx, t.factoryCfg.Device, t.cfg.Device)
	if err != nil {
		return err
	}
	cl, err := openClient(props,
		kgo.ConsumerGroup(t.group()),
		kgo.ConsumeTopics(t.cfg.Topic),
		kgo.ConsumeResetOffset(resetOffset(t.cfg.StartOffset)),
		kgo.AutoCommitMarks(),
		kgo.FetchMaxWait(t.factoryCfg.FetchMaxWait.Duration()),
	)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			cl.Close()
		}
	}()
	pingCtx, cancelPing := context.WithTimeout(ctx, t.factoryCfg.PingTimeout.Duration())
	defer cancelPing()
	if err = cl.Ping(pingCtx); err != nil {
		return errors.Wrap(err, "connecting to cluster")
	}
	if t.cfg.StartOffset == StartOffsetNone {
		if err = t.checkCommitted(pingCtx, cl); err != nil {
			return err
		}
	}
	for _, g := range t.groups {
		g.writer = nil
		g.last = 0
		g.replaying = false
		if g.index == 0 || g.stamp == -1 {
			continue
		}
		if g.last, err = t.lastStored(ctx, g.index); err != nil {
			return err
		}
		g.replaying = true
	}
	sCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(t.factoryCfg.Instrumentation),
	)
	t.client = cl
	t.sCtx = sCtx
	t.lastSkip = ""
	t.closer = xio.MultiCloser{
		xio.CloserFunc(t.closeClient),
		signal.NewHardShutdown(sCtx, cancel),
	}
	sCtx.Go(t.consume, signal.RecoverWithErrOnPanic())
	return nil
}

// checkCommitted fails when the consumer group has no committed offset on a partition
// of the topic. A group the cluster has never seen has none on any partition.
func (t *readTask) checkCommitted(ctx context.Context, cl *kgo.Client) error {
	resp, err := kadm.NewClient(cl).FetchOffsetsForTopics(ctx, t.group(), t.cfg.Topic)
	if errors.Is(err, kerr.GroupIDNotFound) {
		return errors.Newf(
			"consumer group %q has no committed offset on topic %q",
			t.group(), t.cfg.Topic,
		)
	}
	if err != nil {
		return errors.Wrap(err, "fetching committed offsets")
	}
	var missing []int32
	resp.Each(func(o kadm.OffsetResponse) {
		if o.At < 0 {
			missing = append(missing, o.Partition)
		}
	})
	if len(missing) > 0 {
		return errors.Newf(
			"consumer group %q has no committed offset on partitions %v of topic %q",
			t.group(), missing, t.cfg.Topic,
		)
	}
	return nil
}

// closeClient commits the marked offsets, closes the consumer, and closes the writers.
func (t *readTask) closeClient() (err error) {
	ctx, cancel := context.WithTimeout(
		context.Background(), t.factoryCfg.PingTimeout.Duration(),
	)
	defer cancel()
	err = t.client.CommitMarkedOffsets(ctx)
	t.client.Close()
	for _, g := range t.groups {
		if g.writer != nil {
			err = errors.Join(err, g.writer.Close())
			g.writer = nil
		}
	}
	return err
}

// lastStored returns the last timestamp stored in index, or zero when it is empty.
func (t *readTask) lastStored(
	ctx context.Context,
	index channel.Key,
) (last telem.TimeStamp, err error) {
	it, err := t.factoryCfg.Framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   channel.Keys{index},
		Bounds: telem.TimeRangeMax,
	})
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, it.Close()) }()
	if !it.SeekLast() || !it.Prev(iterator.AutoSpan) {
		return 0, it.Error()
	}
	ms := it.Value().Get(index)
	if len(ms.Series) == 0 {
		return 0, nil
	}
	s := ms.Series[len(ms.Series)-1]
	if s.Len() == 0 {
		return 0, nil
	}
	return s.ValueAt[telem.TimeStamp](int(s.Len()) - 1), nil
}

// consume polls the topic until ctx ends. A fetch or write failure stops it with an
// error status.
func (t *readTask) consume(ctx context.Context) error {
	for {
		fetches := t.client.PollFetches(ctx)
		if ctx.Err() != nil || fetches.IsClientClosed() {
			return nil
		}
		var fetchErr error
		fetches.EachError(func(topic string, partition int32, err error) {
			if fetchErr == nil {
				fetchErr = errors.Wrapf(err, "fetching %s/%d", topic, partition)
			}
		})
		if fetchErr != nil {
			t.setStatus(
				ctx, driver.NoCommand, status.VariantError, false, fetchErr.Error(),
			)
			return fetchErr
		}
		if err := t.process(ctx, fetches); err != nil {
			t.setStatus(ctx, driver.NoCommand, status.VariantError, false, err.Error())
			return err
		}
	}
}

// process writes the records of one poll and marks them for commit once every
// group's write is acknowledged.
func (t *readTask) process(ctx context.Context, fetches kgo.Fetches) error {
	rows := make(map[*readGroup][]row)
	fetches.EachRecord(func(rec *kgo.Record) {
		if g, r, ok := t.convert(ctx, rec); ok {
			rows[g] = append(rows[g], r)
		}
	})
	for g, rs := range rows {
		if err := t.write(ctx, g, rs); err != nil {
			return err
		}
	}
	t.client.MarkCommitRecords(fetches.Records()...)
	return nil
}

// convert parses rec into its group's samples. ok is false when the record is not
// addressed to this task or was skipped with a status.
func (t *readTask) convert(
	ctx context.Context,
	rec *kgo.Record,
) (g *readGroup, r row, ok bool) {
	g, ok = t.groups[string(rec.Key)]
	if !ok {
		if g, ok = t.groups[""]; !ok {
			return nil, row{}, false
		}
	}
	var doc any
	if err := json.Unmarshal(rec.Value, &doc); err != nil {
		t.skip(ctx, "Record is not JSON: "+err.Error())
		return nil, row{}, false
	}
	r = row{ts: telem.Now(), values: make([]any, len(g.fields))}
	for i, fld := range g.fields {
		raw, found, err := pointerGet(doc, fld.Pointer)
		if err != nil {
			t.skip(ctx, err.Error())
			return nil, row{}, false
		}
		if !found {
			t.skip(ctx, fmt.Sprintf("Record has no value at %s", fld.Pointer))
			return nil, row{}, false
		}
		v, err := readSample(
			raw, g.channels[i].DataType, fld.TimeFormat, fld.EnumValues,
		)
		if err != nil {
			t.skip(ctx, fmt.Sprintf("Value at %s: %s", fld.Pointer, err))
			return nil, row{}, false
		}
		r.values[i] = v
	}
	if g.stamp != -1 {
		r.ts = r.values[g.stamp].(telem.TimeStamp)
	}
	return g, r, true
}

// write sorts rows by timestamp, drops the replay and any regression, and writes the
// rest as one frame.
func (t *readTask) write(ctx context.Context, g *readGroup, rows []row) error {
	slices.SortStableFunc(rows, func(a, b row) int { return cmp.Compare(a.ts, b.ts) })
	kept := rows[:0]
	for _, r := range rows {
		if g.stamp == -1 {
			r.ts = max(r.ts, g.last+1)
		} else if r.ts <= g.last {
			if !g.replaying {
				t.skip(ctx, fmt.Sprintf(
					"Record stamped %s is not after the last written %s for key %q",
					r.ts, g.last, g.key,
				))
			}
			continue
		}
		g.replaying = false
		g.last = r.ts
		kept = append(kept, r)
	}
	if len(kept) == 0 {
		return nil
	}
	if g.writer == nil {
		mode := ts.WriterModePersistStream
		if t.cfg.DataSavingDisabled {
			mode = ts.WriterModeStreamOnly
		}
		w, err := t.factoryCfg.Framer.OpenWriter(ctx, framer.WriterConfig{
			ControlSubject: control.Subject{
				Name: t.task.Name,
				Key:  t.task.Key.String(),
			},
			Start:             kept[0].ts,
			Keys:              g.keys,
			Sync:              new(true),
			ErrOnUnauthorized: new(true),
			Mode:              mode,
		})
		if err != nil {
			return errors.Wrapf(err, "opening writer for key %q", g.key)
		}
		g.writer = w
	}
	if _, err := g.writer.Write(g.frame(kept)); err != nil {
		return errors.Wrapf(err, "writing records for key %q", g.key)
	}
	return nil
}

// frame builds the even frame rows make for the group's writer keys.
func (g *readGroup) frame(rows []row) frame.Frame {
	series := make([]telem.Series, 0, len(g.keys))
	if g.index != 0 && g.stamp == -1 {
		stamps := make([]telem.TimeStamp, len(rows))
		for i, r := range rows {
			stamps[i] = r.ts
		}
		series = append(series, telem.NewSeries(stamps))
	}
	for i, ch := range g.channels {
		s := telem.Series{DataType: ch.DataType}
		for _, r := range rows {
			sample := telem.NewSeriesFromAny(r.values[i], ch.DataType)
			s.Data = append(s.Data, sample.Data...)
		}
		series = append(series, s)
	}
	return frame.NewMulti(g.keys, series)
}

// skip reports a skipped record. A repeat of the last report is not rewritten.
func (t *readTask) skip(ctx context.Context, message string) {
	if message == t.lastSkip {
		return
	}
	t.lastSkip = message
	t.setStatus(ctx, driver.NoCommand, status.VariantError, true, message)
}

// ackCurrent answers cmdKey with the task's current status, for a command that needs
// no work.
func (t *readTask) ackCurrent(ctx context.Context, cmdKey string, running bool) {
	if err := t.status.Ack(ctx, cmdKey, running); err != nil {
		t.factoryCfg.L.Error("failed to acknowledge command",
			zap.Stringer("task", t.task.Key),
			zap.String("cmd", cmdKey),
			zap.Error(err),
		)
	}
}

func (t *readTask) setStatus(
	ctx context.Context,
	cmdKey string,
	variant status.Variant,
	running bool,
	message string,
) {
	if err := t.status.Send(ctx, cmdKey, variant, running, message); err != nil {
		t.factoryCfg.L.Error("failed to set task status",
			zap.Stringer("task", t.task.Key),
			zap.Error(err),
		)
	}
}
