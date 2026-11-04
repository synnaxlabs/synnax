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
	"fmt"
	"io"
	"net"
	"slices"
	"sync/atomic"

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
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// readField is an enabled field of a group, parsed for conversion.
type readField struct {
	// pointer is the text of the field's pointer, for skip reasons.
	pointer string
	channel channel.Channel
	format  xjson.TimeFormat
	enums   xjson.EnumMap
}

// readGroup is the set of fields sharing one record key, one index, and one writer.
type readGroup struct {
	// key is the record key the group claims; empty claims every unclaimed record.
	key    string
	fields []readField
	// extractor resolves the fields' pointers in a record.
	extractor *xjson.Extractor
	// values and found are scratch space for the values of a record's fields.
	values []any
	found  []bool
	// stamp is the position in fields of the field targeting the index channel, or
	// -1 when records are stamped with their Kafka timestamp.
	stamp int
	// index is the group's index channel, zero when its channels are virtual.
	index channel.Key
	// keys are the group's channels in the writer: the index first when it is not a
	// field.
	keys channel.Keys
	// last is the last timestamp written, seeded with the last stored one at start.
	last telem.TimeStamp
	// replaying is true until the first record past last arrives.
	replaying bool
}

// row is one record converted into a group's samples.
type row struct {
	ts telem.TimeStamp
	// samples[i] is the series encoding of the sample for fields[i].
	samples [][]byte
}

// readTask consumes a topic and writes record fields into channels.
type readTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        ReadConfig
	// status is the authority on this instance's current status.
	status *driver.StatusHandler
	groups map[string]*readGroup
	// writer writes every group's channels, nil while stopped.
	writer *framer.Writer
	// client is the consumer, nil while stopped.
	client *kgo.Client
	// closer releases the consumer, nil once released.
	closer io.Closer
	// sCtx is the consumer's context. A consumer that fails cancels it.
	sCtx signal.Context
	// lastSkip is the last skip reported, so a repeated one is not rewritten.
	lastSkip string
	// skipped counts the records the current poll skipped, and skipReason is why the
	// first was.
	skipped    int
	skipReason string
	// reassigned is set when the group assigns partitions, which replay from the last
	// committed offset.
	reassigned atomic.Bool
	// disconnected is true from a connection failure until records arrive again.
	disconnected bool
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

// buildReadGroups resolves the enabled fields of each record in cfg into a group and
// validates that each group writes one even frame per record.
func buildReadGroups(
	ctx context.Context,
	svc *channel.Service,
	cfg ReadConfig,
) (map[string]*readGroup, error) {
	var keys channel.Keys
	for _, rec := range cfg.Records {
		for _, fld := range rec.Fields {
			if fld.Disabled {
				continue
			}
			if fld.Pointer == "" {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"field %q: pointer must not be empty",
					fld.Key,
				)
			}
			if fld.Channel == 0 {
				return nil, errors.Wrapf(
					validate.ErrValidation, "field %q: channel must be set", fld.Key,
				)
			}
			keys = append(keys, fld.Channel)
		}
	}
	if len(keys) == 0 {
		return nil, errors.Wrap(
			validate.ErrValidation, "records: at least one field must be enabled",
		)
	}
	channels, err := retrieveChannels(ctx, svc, keys)
	if err != nil {
		return nil, err
	}
	claimed := make(set.Set[string])
	groups := make(map[string]*readGroup)
	for _, rec := range cfg.Records {
		if claimed.Contains(rec.RecordKey) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"record key %q: claimed by more than one record",
				rec.RecordKey,
			)
		}
		claimed.Add(rec.RecordKey)
		g := &readGroup{key: rec.RecordKey, stamp: -1, replaying: true}
		var pointers []xjson.Pointer
		for _, fld := range rec.Fields {
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
			if err := xjson.CheckToSample(ch.DataType); err != nil {
				return nil, errors.Wrapf(err, "field %q", fld.Key)
			}
			pointer, err := xjson.ParsePointer(fld.Pointer)
			if err != nil {
				return nil, errors.Wrapf(err, "field %q", fld.Key)
			}
			rf := readField{
				pointer: fld.Pointer,
				channel: ch,
				enums:   enumMap(fld.EnumValues),
			}
			if ch.DataType == telem.TimestampT {
				if fld.TimeFormat == nil {
					return nil, errors.Wrapf(
						validate.ErrValidation,
						"field %q: time_format is required for timestamp channel %s",
						fld.Key, ch.Name,
					)
				}
				if rf.format, err = xjson.ParseTimeFormat(
					string(*fld.TimeFormat),
				); err != nil {
					return nil, errors.Wrapf(err, "field %q", fld.Key)
				}
			}
			if isIndex := fld.Key == rec.Index; isIndex != ch.IsIndex {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"field %q: only the record's index field targets an index channel",
					fld.Key,
				)
			}
			if fld.Key == rec.Index {
				g.stamp = len(g.fields)
			}
			g.fields = append(g.fields, rf)
			pointers = append(pointers, pointer)
		}
		if len(g.fields) == 0 {
			continue
		}
		g.extractor = xjson.NewExtractor(pointers)
		g.values = make([]any, len(g.fields))
		g.found = make([]bool, len(g.fields))
		if rec.Index != "" && g.stamp == -1 {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"record key %q: index field %q is not an enabled field",
				rec.RecordKey, rec.Index,
			)
		}
		groups[rec.RecordKey] = g
	}
	indexes := make(map[channel.Key]string, len(groups))
	for _, g := range groups {
		for i, f := range g.fields {
			ch := f.channel
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
		// The writer wants every data channel on an index in each frame, so one
		// index cannot be split across records that arrive independently.
		if other, ok := indexes[g.index]; ok && g.index != 0 {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"record keys %q and %q: channels on one index belong in one record",
				other, g.key,
			)
		}
		indexes[g.index] = g.key
		if g.index != 0 && g.stamp == -1 {
			g.keys = append(g.keys, g.index)
		}
		for _, f := range g.fields {
			g.keys = append(g.keys, f.channel.Key())
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
		t.setStatus(
			ctx,
			cmdKey,
			status.VariantError,
			false,
			"Failed to start task",
			err.Error(),
		)
		return err
	}
	t.setStatus(ctx, cmdKey, status.VariantSuccess, true, "Started task", "")
	// The consumer writes statuses too, so it starts after this one to not be replaced.
	t.sCtx.Go(t.consume, signal.RecoverWithErrOnPanic(), signal.CancelOnFail())
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
		t.setStatus(
			ctx,
			cmdKey,
			status.VariantError,
			false,
			"Failed to stop task",
			err.Error(),
		)
		return err
	}
	t.setStatus(ctx, cmdKey, status.VariantSuccess, false, "Stopped task", "")
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

// open connects the consumer. It writes no statuses and does not start the fetch loop:
// start owns both.
func (t *readTask) open(ctx context.Context) (err error) {
	props, err := resolveProperties(ctx, t.factoryCfg.Device, t.cfg.Device)
	if err != nil {
		return err
	}
	cl, err := openClient(
		props,
		kgo.ConsumerGroup(t.group()),
		kgo.ConsumeTopics(t.cfg.Topic),
		kgo.ConsumeResetOffset(resetOffset(t.cfg.StartOffset)),
		kgo.AutoCommitMarks(),
		kgo.FetchMaxWait(t.factoryCfg.FetchMaxWait.Duration()),
		kgo.OnPartitionsAssigned(
			func(context.Context, *kgo.Client, map[string][]int32) {
				t.reassigned.Store(true)
			},
		),
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
		g.last = 0
		g.replaying = false
		if g.index == 0 {
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
	// The writer outlives the start command, so it opens in the consumer's context.
	if t.writer, err = t.openWriter(sCtx); err != nil {
		cancel()
		return err
	}
	t.client = cl
	t.sCtx = sCtx
	t.lastSkip = ""
	t.disconnected = false
	t.closer = xio.MultiCloser{
		xio.CloserFunc(t.closeClient),
		signal.NewHardShutdown(sCtx, cancel),
	}
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
	err = errors.Join(err, t.writer.Close())
	t.writer = nil
	return err
}

// openWriter opens one writer over the channels of every group.
func (t *readTask) openWriter(ctx context.Context) (*framer.Writer, error) {
	var keys channel.Keys
	for _, g := range t.groups {
		keys = append(keys, g.keys...)
	}
	mode := ts.WriterModePersistStream
	if t.cfg.DataSavingDisabled {
		mode = ts.WriterModeStreamOnly
	}
	return t.factoryCfg.Framer.OpenWriter(ctx, framer.WriterConfig{
		ControlSubject: control.Subject{
			Name: t.task.Name,
			Key:  t.task.Key.String(),
		},
		Keys:              keys,
		Sync:              new(true),
		ErrOnUnauthorized: new(true),
		Mode:              mode,
		// The offsets commit once Write returns, so the index must persist every
		// commit before then or a crash loses samples Kafka will not replay.
		AutoIndexPersistInterval: ts.AlwaysIndexPersistOnAutoCommit,
	})
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
	defer func() {
		if cErr := it.Close(); cErr != nil {
			last, err = 0, errors.Join(err, cErr)
		}
	}()
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

// consume polls the topic until ctx ends. A connection failure sets a warning status
// while the consumer retries. Any other fetch failure or a write failure stops it with
// an error status.
func (t *readTask) consume(ctx context.Context) error {
	for {
		fetches := t.client.PollFetches(ctx)
		if ctx.Err() != nil || fetches.IsClientClosed() {
			return nil
		}
		var fatal, transient error
		fetches.EachError(func(topic string, partition int32, err error) {
			err = errors.Wrapf(err, "fetching %s/%d", topic, partition)
			if isConnectionErr(err) {
				transient = cmp.Or(transient, err)
			} else {
				fatal = cmp.Or(fatal, err)
			}
		})
		if fatal != nil {
			t.setStatus(
				ctx,
				driver.NoCommand,
				status.VariantError,
				false,
				"Failed to fetch records",
				fatal.Error(),
			)
			return fatal
		}
		// A connection status replaces the last skip, so the next skip is reported
		// again. Recovery is reported before the batch, so a skip in it stays visible.
		if transient != nil && !t.disconnected {
			t.disconnected = true
			t.lastSkip = ""
			t.setStatus(
				ctx,
				driver.NoCommand,
				status.VariantWarning,
				true,
				"Lost connection to cluster",
				transient.Error()+". The task retries until the cluster is reachable.",
			)
		}
		if transient == nil && t.disconnected && fetches.NumRecords() > 0 {
			t.disconnected = false
			t.lastSkip = ""
			t.setStatus(
				ctx,
				driver.NoCommand,
				status.VariantSuccess,
				true,
				"Reconnected to cluster",
				"",
			)
		}
		if err := t.process(ctx, fetches); err != nil {
			t.setStatus(
				ctx,
				driver.NoCommand,
				status.VariantError,
				false,
				"Failed to process records",
				err.Error(),
			)
			return err
		}
	}
}

// unavailable are the broker errors for a broker or coordinator that is down or
// moving.
var unavailable = []error{
	kerr.BrokerNotAvailable,
	kerr.CoordinatorLoadInProgress,
	kerr.CoordinatorNotAvailable,
	kerr.LeaderNotAvailable,
	kerr.NetworkException,
	kerr.NotCoordinator,
	kerr.NotLeaderForPartition,
	kerr.RequestTimedOut,
}

// isConnectionErr reports whether err is a failure to reach the cluster, which the
// consumer retries on its own.
func isConnectionErr(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) ||
		errors.Is(err, io.EOF) ||
		kgo.IsRetryableBrokerErr(err) ||
		slices.ContainsFunc(
			unavailable,
			func(u error) bool { return errors.Is(err, u) },
		)
}

// process writes the records of one poll and marks them for commit once every
// group's write is acknowledged.
func (t *readTask) process(ctx context.Context, fetches kgo.Fetches) error {
	if t.reassigned.Swap(false) {
		for _, g := range t.groups {
			g.replaying = g.index != 0
		}
	}
	rows := make(map[*readGroup][]row)
	fetches.EachRecord(func(rec *kgo.Record) {
		if g, r, ok := t.convert(rec); ok {
			rows[g] = append(rows[g], r)
		}
	})
	var fr frame.Frame
	for g, rs := range rows {
		if kept := t.keep(g, rs); len(kept) > 0 {
			fr = g.appendFrame(fr, kept)
		}
	}
	t.reportSkips(ctx)
	if !fr.Empty() {
		if _, err := t.writer.Write(fr); err != nil {
			return errors.Wrap(err, "writing records")
		}
	}
	t.client.MarkCommitRecords(fetches.Records()...)
	return nil
}

// convert parses rec into its group's samples. ok is false when the record is not
// addressed to this task or was skipped with a status.
func (t *readTask) convert(rec *kgo.Record) (g *readGroup, r row, ok bool) {
	g, ok = t.groups[string(rec.Key)]
	if !ok {
		if g, ok = t.groups[""]; !ok {
			return nil, row{}, false
		}
	}
	if err := g.extractor.Extract(rec.Value, g.values, g.found); err != nil {
		t.skip("Record is not JSON: " + err.Error())
		return nil, row{}, false
	}
	r = row{samples: make([][]byte, len(g.fields))}
	for i, f := range g.fields {
		if !g.found[i] {
			t.skip(fmt.Sprintf("Record has no value at %s", f.pointer))
			return nil, row{}, false
		}
		sample, err := xjson.AppendSample(
			nil, f.channel.DataType, g.values[i], f.format, f.enums,
		)
		if err != nil {
			t.skip(fmt.Sprintf("Value at %s: %s", f.pointer, err))
			return nil, row{}, false
		}
		r.samples[i] = sample
	}
	if g.stamp == -1 {
		r.ts = telem.NewTimeStamp(rec.Timestamp)
	} else {
		r.ts = telem.TimeStamp(telem.ByteOrder.Uint64(r.samples[g.stamp]))
	}
	return g, r, true
}

// keep sorts rows by timestamp and drops the replay. A body-stamped regression is
// dropped with a status. Kafka timestamps only count milliseconds, so a regression of
// one moves to a nanosecond past the last written stamp.
func (t *readTask) keep(g *readGroup, rows []row) []row {
	slices.SortStableFunc(rows, func(a, b row) int { return cmp.Compare(a.ts, b.ts) })
	kept := rows[:0]
	for _, r := range rows {
		if r.ts <= g.last {
			if g.replaying {
				continue
			}
			if g.stamp != -1 {
				t.skip(fmt.Sprintf(
					"Record stamped %s is not after the last written %s for key %q",
					r.ts, g.last, g.key,
				))
				continue
			}
			r.ts = g.last + 1
		}
		g.replaying = false
		g.last = r.ts
		kept = append(kept, r)
	}
	return kept
}

// appendFrame appends the series rows make for the group's keys to fr.
func (g *readGroup) appendFrame(fr frame.Frame, rows []row) frame.Frame {
	series := make([]telem.Series, 0, len(g.keys))
	if g.index != 0 && g.stamp == -1 {
		stamps := make([]telem.TimeStamp, len(rows))
		for i, r := range rows {
			stamps[i] = r.ts
		}
		series = append(series, telem.NewSeries(stamps))
	}
	for i, f := range g.fields {
		s := telem.Series{DataType: f.channel.DataType}
		for _, r := range rows {
			s.Data = append(s.Data, r.samples[i]...)
		}
		series = append(series, s)
	}
	return fr.Extend(frame.NewMulti(g.keys, series))
}

// skip counts a skipped record and why for reportSkips.
func (t *readTask) skip(reason string) {
	if t.skipped == 0 {
		t.skipReason = reason
	}
	t.skipped++
}

// reportSkips writes one status for the records the poll skipped, unless its reason
// repeats the last one.
func (t *readTask) reportSkips(ctx context.Context) {
	n := t.skipped
	if n == 0 {
		return
	}
	t.skipped = 0
	if t.skipReason == t.lastSkip {
		return
	}
	t.lastSkip = t.skipReason
	message := "Skipped record"
	if n > 1 {
		message = fmt.Sprintf("Skipped %d records", n)
	}
	t.setStatus(
		ctx,
		driver.NoCommand,
		status.VariantError,
		true,
		message,
		t.skipReason,
	)
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
	description string,
) {
	if err := t.status.Send(
		ctx,
		cmdKey,
		variant,
		running,
		message,
		description,
	); err != nil {
		t.factoryCfg.L.Error("failed to set task status",
			zap.Stringer("task", t.task.Key),
			zap.Error(err),
		)
	}
}
