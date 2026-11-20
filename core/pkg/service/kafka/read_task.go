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
	"maps"
	"net"
	"slices"
	"sync/atomic"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
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

// readGroup holds the samples of one ICD message in the records of one key. It has one
// index and one set of channels in the writer.
type readGroup struct {
	// name names the message and its record key in errors.
	name string
	// data[i] holds the current poll's samples of binding i. The decoder of the record
	// appends to it.
	data [][]byte
	// columns[i] places the samples of binding i in data[i].
	columns []column
	// rows are the current poll's records, which columns hold in arrival order.
	rows []row
	// stamp is the position in columns of the binding of the index field, or -1 when
	// records are stamped with their Kafka timestamp.
	stamp int
	// index is the group's index channel, zero when its channels are virtual.
	index channel.Key
	// keys are the group's channels in the writer: the index first when no binding
	// writes it.
	keys channel.Keys
	// last is the last timestamp written, seeded with the last stored one at start.
	last telem.TimeStamp
	// replaying is true until the first record past last arrives.
	replaying bool
}

// readRecord reads the records of one Kafka key as the messages of an ICD.
type readRecord struct {
	decoder *codec.JSON
	// groups[i] holds the samples of message i of decoder.
	groups []*readGroup
	// samples[i] is the data of groups[i], which decoder appends to.
	samples [][][]byte
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

// readTask consumes a topic and writes the fields of ICD messages into channels.
type readTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        ReadConfig
	// status is the authority on this instance's current status.
	status *driver.StatusHandler
	// records holds the record of each record key.
	records map[string]*readRecord
	// groups holds the groups of every record.
	groups []*readGroup
	// writer writes every group's channels, nil once the consumer exits.
	writer *framer.Writer
	// client is the consumer, nil while stopped.
	client *kgo.Client
	// closer releases the consumer, nil once released.
	closer io.Closer
	// sCtx is the consumer's context. A consumer that panics or fails without a report
	// cancels it.
	sCtx signal.Context
	// failed is set before the consumer reports its failure.
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
	records, groups, err := f.buildReadRecords(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &readTask{
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		status:     driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t),
		records:    records,
		groups:     groups,
		marks:      map[string]map[int32]kgo.EpochOffset{cfg.Topic: {}},
	}, nil
}

// buildReadRecords resolves the enabled records of cfg against their ICDs and channels.
// It returns the records by record key and the groups of every record, and validates
// that each group writes one even frame per record.
func (f *factory) buildReadRecords(
	ctx context.Context,
	cfg ReadConfig,
) (map[string]*readRecord, []*readGroup, error) {
	enabled := lo.Filter(cfg.Records, func(r ReadRecord, _ int) bool {
		return !r.Disabled
	})
	if len(enabled) == 0 {
		return nil, nil, errors.Wrap(
			validate.ErrValidation, "records: at least one record must be enabled",
		)
	}
	icds, err := retrieveICDs(ctx, f.cfg.DB, f.cfg.ICD, lo.Uniq(lo.Map(
		enabled, func(r ReadRecord, _ int) icd.Key { return r.ICD },
	)))
	if err != nil {
		return nil, nil, err
	}
	var keys channel.Keys
	for _, rec := range enabled {
		for _, m := range rec.Messages {
			for _, b := range m.Bindings {
				keys = append(keys, b.Channel)
			}
		}
	}
	channels, err := f.readChannels(ctx, lo.Uniq(keys))
	if err != nil {
		return nil, nil, err
	}
	var (
		records = make(map[string]*readRecord, len(enabled))
		groups  []*readGroup
		w       = writers{channels: channels, names: make(map[channel.Key]string)}
	)
	for _, rec := range enabled {
		if _, ok := records[rec.RecordKey]; ok {
			return nil, nil, errors.Wrapf(
				validate.ErrValidation,
				"record key %q: claimed by more than one record",
				rec.RecordKey,
			)
		}
		rd, err := newReadRecord(rec, icds[rec.ICD], channels)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "record key %q", rec.RecordKey)
		}
		for _, g := range rd.groups {
			if err = w.claim(g); err != nil {
				return nil, nil, err
			}
		}
		records[rec.RecordKey] = rd
		groups = append(groups, rd.groups...)
	}
	return records, groups, nil
}

// readChannels returns the channels of keys and the index channel of each, so that an
// error can name an index channel that no binding names.
func (f *factory) readChannels(
	ctx context.Context,
	keys channel.Keys,
) (map[channel.Key]channel.Channel, error) {
	channels, err := retrieveChannels(ctx, f.cfg.DB, f.cfg.Channel, keys)
	if err != nil {
		return nil, err
	}
	var indexes channel.Keys
	for _, ch := range channels {
		if _, ok := channels[ch.Index()]; !ok && !ch.IsIndex && ch.Index() != 0 {
			indexes = append(indexes, ch.Index())
		}
	}
	if len(indexes) == 0 {
		return channels, nil
	}
	found, err := retrieveChannels(ctx, f.cfg.DB, f.cfg.Channel, lo.Uniq(indexes))
	if err != nil {
		return nil, err
	}
	maps.Copy(channels, found)
	return channels, nil
}

// writers holds the channels that the groups of a read task write to.
type writers struct {
	channels map[channel.Key]channel.Channel
	// names names the group that writes to each channel.
	names map[channel.Key]string
}

// claim takes the channels of g. Two groups on one index would write frames that lack
// the channels of the other.
func (w writers) claim(g *readGroup) error {
	for _, key := range g.keys {
		other, ok := w.names[key]
		if !ok {
			w.names[key] = g.name
			continue
		}
		what := "channel"
		if key == g.index {
			what = "index channel"
		}
		return errors.Wrapf(
			validate.ErrValidation,
			"records: %s and %s write to the same %s %s",
			other, g.name, what, w.channels[key].Name,
		)
	}
	return nil
}

// newReadRecord validates rec against ic, the ICD it names, and channels, which holds
// every channel that a binding of rec names.
func newReadRecord(
	rec ReadRecord,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
) (*readRecord, error) {
	if len(rec.Messages) == 0 {
		return nil, errors.Wrap(validate.ErrValidation, "reads no messages")
	}
	var (
		decoded = make([]codec.JSONMessage, len(rec.Messages))
		rd      = &readRecord{
			groups:  make([]*readGroup, len(rec.Messages)),
			samples: make([][][]byte, len(rec.Messages)),
		}
		err error
	)
	for i, rm := range rec.Messages {
		if rd.groups[i], err = newReadGroup(
			rec.RecordKey, rm, ic, channels, &decoded[i],
		); err != nil {
			return nil, err
		}
		rd.samples[i] = rd.groups[i].data
	}
	if rd.decoder, err = codec.NewJSON(decoded); err != nil {
		return nil, err
	}
	return rd, nil
}

// newReadGroup validates rm, a message of the record with recordKey, and fills decoded
// with what the decoder needs to read it.
func newReadGroup(
	recordKey string,
	rm ReadMessage,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
	decoded *codec.JSONMessage,
) (*readGroup, error) {
	i := slices.IndexFunc(ic.Messages, func(m icd.Message) bool {
		return m.Key == rm.Message
	})
	if i < 0 {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"message %s does not exist in ICD %s", rm.Message, ic.Name,
		)
	}
	msg := ic.Messages[i]
	payload, ok := msg.Payload.Variant.(icd.JSONPayload)
	if !ok {
		return nil, errors.Wrapf(
			validate.ErrValidation, "message %s is not a JSON message", msg.Name,
		)
	}
	if len(rm.Bindings) == 0 {
		return nil, errors.Wrapf(
			validate.ErrValidation, "message %s binds no fields", msg.Name,
		)
	}
	g := &readGroup{
		name:      fmt.Sprintf("message %s of key %q", msg.Name, recordKey),
		data:      make([][]byte, len(rm.Bindings)),
		columns:   make([]column, len(rm.Bindings)),
		stamp:     -1,
		replaying: true,
	}
	bindings, err := g.bind(rm, channels)
	if err != nil {
		return nil, errors.Wrapf(err, "message %s", msg.Name)
	}
	*decoded = codec.JSONMessage{Name: msg.Name, Payload: payload, Bindings: bindings}
	if g.index != 0 && g.stamp == -1 {
		g.keys = append(g.keys, g.index)
	}
	for _, b := range rm.Bindings {
		g.keys = append(g.keys, b.Channel)
	}
	return g, nil
}

// bind sets the columns, the index channel, and the stamp of g from the bindings of rm
// and their channels. It returns the codec binding of each.
func (g *readGroup) bind(
	rm ReadMessage,
	channels map[channel.Key]channel.Channel,
) ([]codec.Binding, error) {
	bindings := make([]codec.Binding, len(rm.Bindings))
	for i, b := range rm.Bindings {
		ch, ok := channels[b.Channel]
		switch {
		case b.Channel == 0:
			return nil, errors.Wrapf(
				validate.ErrValidation, "binding %d has no channel", i,
			)
		case !ok:
			return nil, errors.Wrapf(
				validate.ErrValidation, "channel %d does not exist", b.Channel,
			)
		}
		bindings[i] = codec.Binding{Field: b.Field, DataType: ch.DataType}
		g.columns[i] = newColumn(ch.DataType)
		index := ch.Index()
		if ch.IsIndex {
			index = ch.Key()
		}
		if i == 0 {
			g.index = index
		} else if index != g.index {
			return nil, errors.Wrap(
				validate.ErrValidation, "every channel must share one index channel",
			)
		}
		isIndexField := rm.Index != nil && *rm.Index == b.Field
		switch {
		case isIndexField && !ch.IsIndex:
			return nil, errors.Wrap(
				validate.ErrValidation,
				"the index field must write to an index channel",
			)
		case !isIndexField && ch.IsIndex:
			return nil, errors.Wrap(
				validate.ErrValidation,
				"only the index field can write to an index channel",
			)
		case isIndexField:
			g.stamp = i
		}
	}
	if rm.Index != nil && g.stamp == -1 {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"the index field has no binding",
		)
	}
	return bindings, nil
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

// stop returns the error that ended the consumer and that no status reports, and the
// error of the status write that answers cmdKey.
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
	case StartOffsetLatest:
		return kgo.NewOffset().AtEnd()
	default:
		panic(fmt.Sprintf("unknown start offset %q", start))
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
	cl, err := openClient(props, t.consumerOpts()...)
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
	if err = t.resetGroups(ctx); err != nil {
		return err
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

// consumerOpts returns the client options of the consumer.
func (t *readTask) consumerOpts() []kgo.Opt {
	return []kgo.Opt{
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
	}
}

// resetGroups loads the last stored timestamp of each group with an index, which
// replays until a record passes it.
func (t *readTask) resetGroups(ctx context.Context) error {
	for _, g := range t.groups {
		g.last = 0
		g.replaying = false
		if g.index == 0 {
			continue
		}
		var err error
		if g.last, err = t.lastStored(ctx, g.index); err != nil {
			return err
		}
		g.replaying = true
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
	if t.writer == nil {
		return nil
	}
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
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if fetches.IsClientClosed() {
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

// fail reports the error that stops the consumer. It returns the error only when the
// report fails, so a stop does not report a failure the status already shows. A start
// that follows the status replaces the consumer instead of acknowledging the failure.
func (t *readTask) fail(ctx context.Context, message string, err error) error {
	t.failed.Store(true)
	if sendErr := t.status.Send(
		ctx, driver.NoCommand, status.VariantError, false, message, err.Error(),
	); sendErr != nil {
		return errors.Combine(err, sendErr)
	}
	return nil
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
	_, op := errors.AsType[*net.OpError](err)
	return op ||
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
			// A failed write closes the writer, and err holds its close error.
			t.writer = nil
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

// convert decodes rec into the samples of the group of its message. A record that is
// not addressed to this task leaves nothing, and neither does one that is skipped.
func (t *readTask) convert(rec *kgo.Record) {
	rd, ok := t.records[string(rec.Key)]
	if !ok {
		if rd, ok = t.records[""]; !ok {
			return
		}
	}
	t.skips.Read()
	r := row{
		ts:        telem.NewTimeStamp(rec.Timestamp),
		partition: rec.Partition,
		offset:    rec.Offset,
	}
	m, ok, err := rd.decoder.Decode(rec.Value, rd.samples)
	if err != nil {
		t.skip(r, "Record value: "+err.Error())
		return
	}
	if !ok {
		t.skip(r, "Record value matches no message")
		return
	}
	g := rd.groups[m]
	r.pos = len(g.rows)
	for i := range g.columns {
		g.columns[i].appended(g.data[i])
	}
	if g.stamp >= 0 {
		r.ts = telem.TimeStamp(telem.ByteOrder.Uint64(g.sample(g.stamp, r.pos)))
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
					"Record stamped %s is not after the last written %s for %s",
					r.ts, g.last, g.name,
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
	for i := range g.columns {
		g.data[i] = g.data[i][:0]
		g.columns[i].ends = g.columns[i].ends[:0]
	}
}

// sample returns the sample of binding i at pos.
func (g *readGroup) sample(i, pos int) []byte {
	return g.columns[i].sample(g.data[i], pos)
}

// stamped returns true when the group's index samples come from Kafka timestamps.
func (g *readGroup) stamped() bool { return g.index != 0 && g.stamp == -1 }

func (g *readGroup) size() int {
	var n int
	if g.stamped() {
		n = len(g.rows) * int(telem.TimestampT.Density())
	}
	for i := range g.columns {
		n += g.columns[i].size(g.data[i], g.rows)
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
		start := len(data)
		for _, r := range g.rows {
			data = append(data, g.sample(i, r.pos)...)
		}
		series = append(series, telem.Series{
			DataType: g.columns[i].dataType,
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
