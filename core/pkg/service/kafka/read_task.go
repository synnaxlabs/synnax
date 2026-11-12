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
	"encoding/json/jsontext"
	"fmt"
	"io"
	"net"
	"slices"
	"sync/atomic"
	"time"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/http"
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

// readGroup is the set of fields sharing one record key, one index, and one writer.
type readGroup struct {
	// key is the record key the group claims; empty claims every unclaimed record.
	key    string
	fields []ReadField
	// channels[i] is the channel fields[i] writes to.
	channels []channel.Channel
	// extractor resolves the fields' pointers in a record.
	extractor *xjson.Extractor
	// values[i] is scratch space for the JSON value of fields[i] in a record.
	values []jsontext.Value
	// columns[i] holds the current poll's samples of fields[i].
	columns []column
	// rows are the current poll's records, which columns hold in arrival order.
	rows []row
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
	// pos is the record's position in the group's columns.
	pos int
	// partition and offset locate the record in its topic.
	partition int32
	offset    int64
}

// readTask consumes a topic and writes record fields into channels.
type readTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        ReadConfig
	// status is the authority on this instance's current status.
	status *driver.StatusHandler
	groups map[string]*readGroup
	// writer writes every group's channels, nil once the consumer exits.
	writer *framer.Writer
	// client is the consumer, nil while stopped.
	client *kgo.Client
	// closer releases the consumer, nil once released.
	closer io.Closer
	// sCtx is the consumer's context. A consumer that fails cancels it.
	sCtx signal.Context
	// failed is set before the consumer reports its failure. sCtx only cancels once the
	// consumer exits, after the report.
	failed atomic.Bool
	// skips counts the records of the current run that the task reads and skips.
	skips *driver.Skips
	// reassigned is set when the group assigns partitions, which replay from the last
	// committed offset.
	reassigned atomic.Bool
	// disconnected is true from a connection failure until records arrive again.
	disconnected bool
	// marks is the offsets the current poll consumed by partition.
	marks map[string]map[int32]kgo.EpochOffset
}

var _ driver.Task = (*readTask)(nil)

func (f *factory) configureRead(ctx context.Context, t task.Task) (*readTask, error) {
	var cfg ReadConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Topic == "" {
		return nil, errors.Wrap(validate.ErrValidation, "topic: must not be empty")
	}
	if _, err := resolveProperties(
		ctx,
		f.cfg.DB,
		f.cfg.Device,
		cfg.Device,
	); err != nil {
		return nil, err
	}
	groups, err := buildReadGroups(ctx, f.cfg.DB, f.cfg.Channel, cfg)
	if err != nil {
		return nil, err
	}
	return &readTask{
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		status:     driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t),
		groups:     groups,
		marks:      map[string]map[int32]kgo.EpochOffset{cfg.Topic: {}},
	}, nil
}

// buildReadGroups resolves the enabled fields of each record in cfg into a group and
// validates that each group writes one even frame per record.
func buildReadGroups(
	ctx context.Context,
	db *gorp.DB,
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
	channels, err := retrieveChannels(ctx, db, svc, keys)
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
			if ch.DataType == telem.TimestampT && fld.TimeFormat == nil {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"field %q: time_format is required for timestamp channel %s",
					fld.Key, ch.Name,
				)
			}
			if isIndex := fld.Key == rec.Index; isIndex != ch.IsIndex {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"field %q: only the record's index field targets an index channel",
					fld.Key,
				)
			}
			pointer, err := xjson.ParsePointer(fld.Pointer)
			if err != nil {
				return nil, errors.Wrapf(err, "field %q", fld.Key)
			}
			dec, err := newDecoder(fld, ch)
			if err != nil {
				return nil, errors.Wrapf(err, "field %q", fld.Key)
			}
			if fld.Key == rec.Index {
				g.stamp = len(g.fields)
			}
			g.fields = append(g.fields, fld)
			g.channels = append(g.channels, ch)
			pointers = append(pointers, pointer)
			g.columns = append(g.columns, newColumn(dec, ch.DataType))
		}
		if len(g.fields) == 0 {
			continue
		}
		g.extractor = xjson.NewExtractor(pointers)
		g.values = make([]jsontext.Value, len(g.fields))
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
		for _, ch := range g.channels {
			g.keys = append(g.keys, ch.Key())
		}
	}
	return groups, nil
}

// newDecoder returns the decoder of fld, which writes to ch.
func newDecoder(fld ReadField, ch channel.Channel) (*xjson.Decoder, error) {
	f := xjson.Field{DataType: ch.DataType, Enums: enums(fld.EnumValues)}
	if fld.TimeFormat != nil {
		var err error
		if f.TimeFormat, err = xjson.ParseTimeFormat(
			string(*fld.TimeFormat),
		); err != nil {
			return nil, err
		}
	}
	return xjson.NewDecoder(f)
}

// enums returns the labels of entries in the form xjson takes.
func enums(entries []http.EnumEntry) []xjson.Enum {
	out := make([]xjson.Enum, len(entries))
	for i, e := range entries {
		out[i] = xjson.NewEnum(e.Label, e.Value)
	}
	return out
}

// Exec implements driver.Task. Start and stop answer the command with a status,
// failures included, so Exec returns an error only when that status was not written.
func (t *readTask) Exec(ctx context.Context, cmd task.Command) error {
	var err, answerErr error
	switch cmd.Type {
	case "start":
		err, answerErr = t.start(ctx, cmd.Key)
	case "stop":
		err, answerErr = t.stop(ctx, cmd.Key, true)
	default:
		return driver.ErrUnsupportedCommand
	}
	if answerErr != nil {
		return errors.Combine(err, answerErr)
	}
	if err != nil {
		t.factoryCfg.L.Error("failed to execute command",
			zap.Stringer("task", t.task.Key),
			zap.Stringer("command", cmd),
			zap.Error(err),
		)
	}
	return nil
}

// Stop implements driver.Task.
func (t *readTask) Stop(sendStatus bool) error {
	return errors.Combine(t.stop(context.TODO(), driver.NoCommand, sendStatus))
}

// isRunning reports whether the consumer is up. A consumer that failed on its own sets
// failed, and one that panicked cancels sCtx. Neither releases the closer.
func (t *readTask) isRunning() bool {
	return t.closer != nil && !t.failed.Load() && t.sCtx.Err() == nil
}

func (t *readTask) release() error {
	err := t.closer.Close()
	t.closer = nil
	t.client = nil
	return err
}

// start returns the error that kept the consumer from starting, and the error of the
// status write that answers cmdKey.
func (t *readTask) start(ctx context.Context, cmdKey string) (err, answerErr error) {
	if t.isRunning() {
		return nil, t.status.Ack(ctx, cmdKey, true)
	}
	if t.closer != nil {
		if err := t.release(); err != nil {
			t.factoryCfg.L.Warn("kafka consumer failed",
				zap.Stringer("task", t.task.Key),
				zap.Error(err),
			)
		}
	}
	if err = t.open(ctx); err != nil {
		return err, t.status.Send(
			ctx,
			cmdKey,
			status.VariantError,
			false,
			"Failed to start task",
			err.Error(),
		)
	}
	if answerErr = t.status.Send(
		ctx, cmdKey, status.VariantSuccess, true, "Started task", "",
	); answerErr != nil {
		// The caller sees the start fail, so the consumer must not outlive it. The
		// consumer never ran, so the start closes the writer itself.
		return errors.Combine(t.release(), t.closeWriter()), answerErr
	}
	// The consumer writes statuses too, so it starts after this one to not be replaced.
	t.sCtx.Go(
		t.consume,
		signal.RecoverWithErrOnPanic(),
		signal.CancelOnFail(),
		signal.DeferErr(t.closeWriter),
	)
	return nil, nil
}

// stop returns the error that ended the consumer, and the error of the status write
// that answers cmdKey.
func (t *readTask) stop(
	ctx context.Context,
	cmdKey string,
	sendStatus bool,
) (err, answerErr error) {
	if t.closer == nil {
		if sendStatus {
			return nil, t.status.Ack(ctx, cmdKey, false)
		}
		return nil, nil
	}
	err = t.release()
	if !sendStatus {
		return err, nil
	}
	if err != nil {
		return err, t.status.Send(
			ctx, cmdKey, status.VariantError, false, "Failed to stop task", err.Error(),
		)
	}
	return nil, t.status.Send(
		ctx, cmdKey, status.VariantSuccess, false, "Stopped task", "",
	)
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
	props, err := resolveProperties(
		ctx,
		t.factoryCfg.DB,
		t.factoryCfg.Device,
		t.cfg.Device,
	)
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
	// The writer outlives the start command and sCtx: a stop cancels sCtx, and the
	// consumer must finish its write. The consumer closes it on exit.
	wCtx, cancelWriter := context.WithCancel(context.Background())
	if t.writer, err = t.openWriter(wCtx); err != nil {
		cancelWriter()
		cancel()
		return err
	}
	t.client = cl
	t.sCtx = sCtx
	t.failed.Store(false)
	t.skips = driver.NewSkips("record")
	t.disconnected = false
	stopConsumer := signal.NewHardShutdown(sCtx, cancel)
	t.closer = xio.MultiCloser{
		xio.CloserFunc(t.closeClient),
		xio.CloserFunc(func() error {
			defer cancelWriter()
			return stopConsumer.Close()
		}),
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

// closeWriter releases the task's channels. It runs as the consumer exits, for any
// reason, once no write is in progress.
func (t *readTask) closeWriter() error {
	err := t.writer.Close()
	t.writer = nil
	return err
}

// closeClient commits the marked offsets and closes the consumer.
func (t *readTask) closeClient() (err error) {
	ctx, cancel := context.WithTimeout(
		context.Background(), t.factoryCfg.PingTimeout.Duration(),
	)
	defer cancel()
	err = t.client.CommitMarkedOffsets(ctx)
	t.client.Close()
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
		fetches := t.poll(ctx)
		if ctx.Err() != nil || fetches.IsClientClosed() {
			return nil
		}
		var fatal, transient error
		fetches.EachError(func(topic string, partition int32, err error) {
			// The deadline of poll ends a quiet poll, and is no failure.
			if errors.Is(err, context.DeadlineExceeded) {
				return
			}
			err = errors.Wrapf(err, "fetching %s/%d", topic, partition)
			if isConnectionErr(err) {
				transient = cmp.Or(transient, err)
			} else {
				fatal = cmp.Or(fatal, err)
			}
		})
		if fatal != nil {
			return t.fail(ctx, "Failed to fetch records", fatal)
		}
		// Recovery is reported before the batch, so a skip in it stays visible.
		if transient != nil && !t.disconnected {
			t.disconnected = true
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
			return t.fail(ctx, "Failed to process records", err)
		}
	}
}

// poll returns the next records. It returns early with a deadline error when the skip
// warning must change.
func (t *readTask) poll(ctx context.Context) kgo.Fetches {
	if deadline, ok := t.skips.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	return t.client.PollFetches(ctx)
}

// fail reports the error that stops the consumer and returns it. A start that follows
// the status replaces the consumer instead of acknowledging the failure.
func (t *readTask) fail(ctx context.Context, message string, err error) error {
	t.failed.Store(true)
	t.setStatus(ctx, driver.NoCommand, status.VariantError, false, message, err.Error())
	return err
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
	for _, g := range t.groups {
		g.reset()
	}
	marks := t.marks[t.cfg.Topic]
	clear(marks)
	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		for _, rec := range p.Records {
			t.convert(rec)
		}
		if len(p.Records) > 0 {
			last := p.Records[len(p.Records)-1]
			head := kgo.EpochOffset{Epoch: last.LeaderEpoch, Offset: last.Offset + 1}
			if cur, ok := marks[p.Partition]; !ok || cur.Less(head) {
				marks[p.Partition] = head
			}
		}
	})
	var keys, size int
	for _, g := range t.groups {
		t.keep(g)
		if len(g.rows) > 0 {
			keys += len(g.keys)
			size += g.size()
		}
	}
	t.reportSkips(ctx, time.Now())
	if keys > 0 {
		if _, err := t.writer.Write(t.frame(keys, size)); err != nil {
			return errors.Wrap(err, "writing records")
		}
	}
	if len(marks) > 0 {
		t.client.MarkCommitOffsets(t.marks)
	}
	return nil
}

// frame builds the frame of the current poll out of one allocation of series data. The
// relay reads a frame after Write returns, so nothing in it can be reused.
func (t *readTask) frame(keyCount, size int) frame.Frame {
	keys := make(channel.Keys, 0, keyCount)
	series := make([]telem.Series, 0, keyCount)
	data := make([]byte, 0, size)
	for _, g := range t.groups {
		if len(g.rows) > 0 {
			keys = append(keys, g.keys...)
			series, data = g.appendSeries(series, data)
		}
	}
	return frame.NewMulti(keys, series)
}

// convert parses rec into its group's samples. A record that is not addressed to this
// task leaves nothing, and neither does one that is skipped.
func (t *readTask) convert(rec *kgo.Record) {
	g, ok := t.groups[string(rec.Key)]
	if !ok {
		if g, ok = t.groups[""]; !ok {
			return
		}
	}
	t.skips.Read()
	r := row{
		ts:        telem.NewTimeStamp(rec.Timestamp),
		pos:       len(g.rows),
		partition: rec.Partition,
		offset:    rec.Offset,
	}
	if err := g.extractor.Extract(rec.Value, g.values); err != nil {
		t.skip(r, "Record is not JSON: "+err.Error())
		return
	}
	for i, fld := range g.fields {
		if g.values[i] == nil {
			t.skip(r, fmt.Sprintf("Record has no value at %s", fld.Pointer))
			g.truncate(r.pos)
			return
		}
		col := &g.columns[i]
		if err := col.append(g.values[i]); err != nil {
			t.skip(r, fmt.Sprintf("Value at %s: %s", fld.Pointer, err))
			g.truncate(r.pos)
			return
		}
		if i == g.stamp {
			r.ts = telem.TimeStamp(telem.ByteOrder.Uint64(col.sample(r.pos)))
		}
	}
	g.rows = append(g.rows, r)
}

// keep sorts the group's rows by timestamp and drops the replay. An indexed group drops
// a regression with a status. Kafka timestamps only count milliseconds, so a row in the
// millisecond of the last Kafka stamp moves to a nanosecond past the last written one.
func (t *readTask) keep(g *readGroup) {
	slices.SortStableFunc(g.rows, func(a, b row) int { return cmp.Compare(a.ts, b.ts) })
	kept := g.rows[:0]
	for _, r := range g.rows {
		if r.ts <= g.last {
			if g.replaying {
				continue
			}
			tie := g.stamp == -1 &&
				r.ts/telem.MillisecondTS == g.last/telem.MillisecondTS
			if g.index != 0 && !tie {
				t.skip(r, fmt.Sprintf(
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
	g.rows = kept
}

func (g *readGroup) reset() {
	g.rows = g.rows[:0]
	g.truncate(0)
}

func (g *readGroup) truncate(n int) {
	for i := range g.columns {
		g.columns[i].truncate(n)
	}
}

// stamped returns true when the group's index samples come from Kafka timestamps.
func (g *readGroup) stamped() bool { return g.index != 0 && g.stamp == -1 }

func (g *readGroup) size() int {
	var n int
	if g.stamped() {
		n = len(g.rows) * int(telem.TimestampT.Density())
	}
	for i := range g.columns {
		n += g.columns[i].size(g.rows)
	}
	return n
}

// appendSeries appends a series per group key to series. Their data is appended to
// data, and each series is capped so that an append to it cannot reach the next one.
func (g *readGroup) appendSeries(
	series []telem.Series,
	data []byte,
) ([]telem.Series, []byte) {
	if g.stamped() {
		start := len(data)
		for _, r := range g.rows {
			data = telem.ByteOrder.AppendUint64(data, uint64(r.ts))
		}
		series = append(series, telem.Series{
			DataType: telem.TimestampT,
			Data:     data[start:len(data):len(data)],
		})
	}
	for i := range g.columns {
		c := &g.columns[i]
		start := len(data)
		for _, r := range g.rows {
			data = append(data, c.sample(r.pos)...)
		}
		series = append(series, telem.Series{
			DataType: c.dataType,
			Data:     data[start:len(data):len(data)],
		})
	}
	return series, data
}

// skip counts the record of r as skipped. reason says why.
func (t *readTask) skip(r row, reason string) {
	t.skips.Skip(time.Now(), fmt.Sprintf(
		"%s (partition %d, offset %d).", reason, r.partition, r.offset,
	))
}

// reportSkips writes the skip warning at now, or clears it once it expires. It writes
// nothing while the connection warning shows.
func (t *readTask) reportSkips(ctx context.Context, now time.Time) {
	w := t.skips.Warning(now)
	if t.disconnected {
		return
	}
	var err error
	if w != nil {
		err = t.status.Warn(ctx, *w)
	} else {
		err = t.status.ClearWarning(ctx)
	}
	if err != nil {
		t.factoryCfg.L.Error("failed to write task warning",
			zap.Stringer("task", t.task.Key),
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
