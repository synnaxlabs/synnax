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
	"encoding/json/jsontext"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// recordCapacity is the size a new record's value starts with, which holds a typical
// value without growing.
const recordCapacity = 256

// writeChannel pairs a configured channel with its record.
type writeChannel struct {
	cfg     WriteChannel
	ch      channel.Channel
	encoder *xjson.Encoder
	// name is the channel's name as a JSON string.
	name []byte
	// key is the key of the channel's records, nil when they have none.
	key []byte
}

type slotKind uint8

const (
	slotValue slotKind = iota
	slotChannel
	slotTimestamp
	slotUUID
)

// slot is how a record fills one slot of the template.
type slot struct {
	kind   slotKind
	format xjson.TimeFormat
}

// recordSlots fills the template slots of one record.
type recordSlots struct {
	// slots[i] fills the slot of template field i.
	slots  []slot
	wc     *writeChannel
	sample []byte
	ts     telem.TimeStamp
}

var _ xjson.Slots = (*recordSlots)(nil)

// AppendSlot implements xjson.Slots.
func (r *recordSlots) AppendSlot(dst []byte, field int) ([]byte, error) {
	s := r.slots[field]
	switch s.kind {
	case slotValue:
		return r.wc.encoder.Append(dst, r.sample)
	case slotChannel:
		return append(dst, r.wc.name...), nil
	case slotTimestamp:
		return xjson.AppendTimeStamp(dst, r.ts, s.format), nil
	default:
		dst = append(dst, '"')
		dst, err := uuid.New().AppendText(dst)
		return append(dst, '"'), err
	}
}

// writeTask produces one record per sample of each channel to a topic.
type writeTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        WriteConfig
	// status is the authority on this instance's current status.
	status   *driver.StatusHandler
	channels map[channel.Key]*writeChannel
	// keys are the streamed channels: every enabled channel and its index.
	keys channel.Keys
	// template renders the value of every record.
	template xjson.Template
	// slots fills the template for the record being built.
	slots recordSlots
	// records recycles the *kgo.Record of each produced sample and its value's buffer.
	records sync.Pool
	// client is the producer, nil while stopped.
	client *kgo.Client
	// closer releases the streamer and producer, nil once released.
	closer io.Closer
	// sCtx is the streamer's context. A streamer that fails cancels it.
	sCtx signal.Context
	// promises counts produced records whose promise has not returned. The client
	// calls promises from its own goroutines, even after it closes.
	promises sync.WaitGroup
	// mu guards lastErr, which produce callbacks update from client goroutines.
	mu sync.Mutex
	// lastErr is the last produce error reported, so a repeat is not rewritten.
	lastErr string
}

var _ driver.Task = (*writeTask)(nil)

func (f *factory) configureWrite(
	ctx context.Context,
	t task.Task,
) (*writeTask, error) {
	var cfg WriteConfig
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
	var keys channel.Keys
	for _, wc := range cfg.Channels {
		if wc.Disabled {
			continue
		}
		if wc.Channel == 0 {
			return nil, errors.Wrapf(
				validate.ErrValidation, "channel %q: channel must be set", wc.Key,
			)
		}
		keys = append(keys, wc.Channel)
	}
	if len(keys) == 0 {
		return nil, errors.Wrap(
			validate.ErrValidation, "channels: at least one must be enabled",
		)
	}
	channels, err := retrieveChannels(ctx, f.cfg.DB, f.cfg.Channel, keys)
	if err != nil {
		return nil, err
	}
	fields, slots, err := templateFields(cfg.Record)
	if err != nil {
		return nil, err
	}
	tmpl, err := xjson.NewTemplate(fields)
	if err != nil {
		return nil, err
	}
	wt := &writeTask{
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		status:     driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t),
		channels:   make(map[channel.Key]*writeChannel, len(keys)),
		template:   tmpl,
		slots:      recordSlots{slots: slots},
	}
	streamed := make(set.Set[channel.Key], len(keys))
	for _, wc := range cfg.Channels {
		if wc.Disabled {
			continue
		}
		ch := channels[wc.Channel]
		if ch.DataType == telem.TimestampT {
			if wc.TimeFormat == nil {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"channel %q: time_format is required for timestamp channel %s",
					wc.Key, ch.Name,
				)
			}
			if wc.JSONType != payload.JSONTypeNumber || len(wc.EnumValues) > 0 {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"channel %q: time_format alone encodes timestamp channel %s",
					wc.Key, ch.Name,
				)
			}
		}
		if wt.channels[wc.Channel], err = newWriteChannel(cfg, wc, ch); err != nil {
			return nil, err
		}
		for _, k := range []channel.Key{ch.Key(), ch.Index()} {
			if k != 0 && !streamed.Contains(k) {
				streamed.Add(k)
				wt.keys = append(wt.keys, k)
			}
		}
	}
	return wt, nil
}

func newWriteChannel(
	cfg WriteConfig,
	wc WriteChannel,
	ch channel.Channel,
) (*writeChannel, error) {
	name, err := jsontext.AppendQuote(nil, ch.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "channel %q", wc.Key)
	}
	f := xjson.Field{DataType: ch.DataType, Enums: enums(wc.EnumValues)}
	if f.Type, err = xjson.ParseType(string(wc.JSONType)); err != nil {
		return nil, errors.Wrapf(err, "channel %q", wc.Key)
	}
	if len(f.Enums) > 0 && f.Type != xjson.String {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"channel %q: enum values require the string JSON type", wc.Key,
		)
	}
	if wc.TimeFormat != nil {
		if f.TimeFormat, err = xjson.ParseTimeFormat(
			string(*wc.TimeFormat),
		); err != nil {
			return nil, errors.Wrapf(err, "channel %q", wc.Key)
		}
	}
	out := &writeChannel{cfg: wc, ch: ch, name: name}
	if out.encoder, err = xjson.NewEncoder(f); err != nil {
		return nil, errors.Wrapf(err, "channel %q", wc.Key)
	}
	if cfg.RecordKey == RecordKeyChannelName {
		out.key = []byte(ch.Name)
	}
	return out, nil
}

// templateFields returns the template fields of rec and how a record fills the slot of
// each.
func templateFields(rec Record) ([]xjson.TemplateField, []slot, error) {
	var (
		fields []xjson.TemplateField
		slots  []slot
	)
	add := func(label, pointer string, literal []byte, s slot) error {
		p, err := xjson.ParsePointer(pointer)
		if err != nil {
			return errors.Wrap(err, label)
		}
		if len(p) == 0 {
			return errors.Wrapf(validate.ErrValidation, "%s: must not be empty", label)
		}
		fields = append(
			fields,
			xjson.TemplateField{Label: label, Pointer: p, Literal: literal},
		)
		slots = append(slots, s)
		return nil
	}
	if err := add(
		"record.value_pointer", rec.ValuePointer, nil, slot{kind: slotValue},
	); err != nil {
		return nil, nil, err
	}
	if p := rec.ChannelPointer; p != nil {
		if err := add(
			"record.channel_pointer", *p, nil, slot{kind: slotChannel},
		); err != nil {
			return nil, nil, err
		}
	}
	if p := rec.TimestampPointer; p != nil {
		format, err := xjson.ParseTimeFormat(string(rec.TimeFormat))
		if err != nil {
			return nil, nil, errors.Wrap(err, "record.time_format")
		}
		if err := add(
			"record.timestamp_pointer",
			*p,
			nil,
			slot{kind: slotTimestamp, format: format},
		); err != nil {
			return nil, nil, err
		}
	}
	for _, field := range rec.Fields {
		var (
			base    http.BaseWriteField
			literal []byte
			s       slot
			err     error
		)
		switch v := field.Variant.(type) {
		case http.StaticWriteField:
			base = v.BaseWriteField
			literal, err = xjson.Marshal(v.Value)
		case http.GeneratedWriteField:
			base = v.BaseWriteField
			s, err = generatorSlot(v)
		default:
			return nil, nil, errors.Newf("unknown write field variant %T", v)
		}
		label := fmt.Sprintf("field %q", base.Key)
		if err != nil {
			return nil, nil, errors.Wrap(err, label)
		}
		if err = add(label, base.Pointer, literal, s); err != nil {
			return nil, nil, err
		}
	}
	return fields, slots, nil
}

// generatorSlot returns the slot that fills a generated field.
func generatorSlot(field http.GeneratedWriteField) (slot, error) {
	switch field.Generator {
	case payload.GeneratorTypeUUID:
		return slot{kind: slotUUID}, nil
	case payload.GeneratorTypeTimestamp:
		s := slot{kind: slotTimestamp, format: xjson.ISO8601}
		if field.TimeFormat != nil {
			var err error
			if s.format, err = xjson.ParseTimeFormat(
				string(*field.TimeFormat),
			); err != nil {
				return slot{}, err
			}
		}
		return s, nil
	default:
		return slot{}, errors.Wrapf(
			validate.ErrValidation, "unknown generator %q", field.Generator,
		)
	}
}

// Exec implements driver.Task. Start and stop answer the command with a status,
// failures included, so Exec returns an error only when that status was not written.
func (t *writeTask) Exec(ctx context.Context, cmd task.Command) error {
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
func (t *writeTask) Stop(sendStatus bool) error {
	return errors.Combine(t.stop(context.TODO(), driver.NoCommand, sendStatus))
}

// isRunning reports whether the streamer is up. A streamer that failed on its own
// cancels sCtx without releasing the closer.
func (t *writeTask) isRunning() bool { return t.closer != nil && t.sCtx.Err() == nil }

func (t *writeTask) release() error {
	err := t.closer.Close()
	t.closer = nil
	t.client = nil
	return err
}

// start returns the error that kept the producer from starting, and the error of the
// status write that answers cmdKey.
func (t *writeTask) start(ctx context.Context, cmdKey string) (err, answerErr error) {
	if t.isRunning() {
		return nil, t.status.Ack(ctx, cmdKey, true)
	}
	if t.closer != nil {
		if err := t.release(); err != nil {
			t.factoryCfg.L.Warn("kafka producer failed",
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
		// The caller sees the start fail, so the producer must not outlive it.
		return t.release(), answerErr
	}
	return nil, nil
}

// stop returns the error that ended the producer, and the error of the status write
// that answers cmdKey.
func (t *writeTask) stop(
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

// open connects the producer and streams the channels into it. It writes no
// statuses: start owns them.
func (t *writeTask) open(ctx context.Context) (err error) {
	props, err := resolveProperties(
		ctx,
		t.factoryCfg.DB,
		t.factoryCfg.Device,
		t.cfg.Device,
	)
	if err != nil {
		return err
	}
	cl, err := openClient(props,
		kgo.DefaultProduceTopic(t.cfg.Topic),
		kgo.AllowAutoTopicCreation(),
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
	streamer, err := t.factoryCfg.Framer.NewStreamer(
		ctx, framer.StreamerConfig{Keys: t.keys, SendOpenAck: true},
	)
	if err != nil {
		return err
	}
	requests := confluence.NewStream[framer.StreamerRequest]()
	responses := confluence.NewStream[framer.StreamerResponse](10)
	streamer.InFrom(requests)
	streamer.OutTo(responses)
	sCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(t.factoryCfg.Instrumentation),
	)
	shutdown := xio.MultiCloser{
		signal.NewGracefulShutdown(sCtx, cancel),
		xio.NoFailCloserFunc(requests.Close),
	}
	streamer.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	// Start succeeds only once the relay streams the channels, so no early sample is
	// missed.
	select {
	case <-ctx.Done():
		// Nothing drains the responses yet, so the streamer only exits on cancel.
		cancel()
		return errors.Join(ctx.Err(), shutdown.Close())
	case _, ok := <-responses.Outlet():
		if !ok {
			return errors.Join(
				errors.New("streamer closed before opening"),
				shutdown.Close(),
			)
		}
	}
	t.client = cl
	t.sCtx = sCtx
	t.lastErr = ""
	t.closer = xio.MultiCloser{
		xio.CloserFunc(t.closeClient),
		xio.CloserFunc(func() error {
			// Produce blocks on a full buffer until its context ends, so a lost cluster
			// would hold the graceful wait forever. The first of the drain and the
			// deadline to settle decides the outcome.
			timeout := t.factoryCfg.PingTimeout.Duration()
			var settled atomic.Bool
			deadline := time.AfterFunc(timeout, func() {
				if settled.CompareAndSwap(false, true) {
					cancel()
				}
			})
			err := shutdown.Close()
			deadline.Stop()
			if settled.CompareAndSwap(false, true) {
				return err
			}
			return errors.Join(
				errors.Newf("timed out producing records after %s", timeout),
				errors.Skip(err, context.Canceled),
			)
		}),
	}
	sCtx.Go(func(ctx context.Context) error {
		// The client has finished with a record by the time it calls the promise.
		promise := func(rec *kgo.Record, err error) {
			defer t.promises.Done()
			if err != nil {
				t.reportError(ctx, "Produce failed: "+err.Error())
			}
			t.records.Put(rec)
		}
		for res := range responses.Outlet() {
			t.produce(ctx, res.Frame, promise)
		}
		return nil
	}, signal.RecoverWithErrOnPanic())
	return nil
}

// closeClient flushes the buffered records, closes the producer, and waits for every
// promise to return.
func (t *writeTask) closeClient() error {
	ctx, cancel := context.WithTimeout(
		context.Background(), t.factoryCfg.PingTimeout.Duration(),
	)
	defer cancel()
	err := t.client.Flush(ctx)
	t.client.Close()
	// Close fails the buffered records without waiting for their promises, and a
	// promise that reports after the stop status would overwrite it. The closer runs
	// this after the relay exits, so nothing adds to promises during the wait.
	t.promises.Wait()
	return err
}

// produce emits one record per sample of every configured channel in fr. An indexed
// channel takes its timestamps from its index in the same frame; a free channel takes
// the receive time. A free channel may carry several series in one frame.
func (t *writeTask) produce(
	ctx context.Context,
	fr frame.Frame,
	promise func(*kgo.Record, error),
) {
	now := telem.Now()
	for key, s := range fr.Entries() {
		wc, ok := t.channels[key]
		if !ok {
			continue
		}
		var index telem.Series
		hasIndex := wc.ch.Index() != 0
		if hasIndex {
			if index, ok = firstSeries(fr, wc.ch.Index()); !ok {
				t.reportError(ctx, "Channel "+wc.ch.Name+" arrived without its index")
				continue
			}
		}
		i := -1
		for sample := range s.Samples() {
			i++
			ts := now
			if hasIndex {
				ts = index.ValueAt[telem.TimeStamp](i)
			}
			rec, err := t.record(wc, sample, ts)
			if err != nil {
				t.reportError(ctx, "Record build failed: "+err.Error())
				continue
			}
			t.promises.Add(1)
			t.client.Produce(ctx, rec, promise)
		}
	}
}

func firstSeries(fr frame.Frame, key channel.Key) (s telem.Series, ok bool) {
	for k, s := range fr.Entries() {
		if k == key {
			return s, true
		}
	}
	return telem.Series{}, false
}

func (t *writeTask) record(
	wc *writeChannel,
	sample []byte,
	ts telem.TimeStamp,
) (*kgo.Record, error) {
	rec, ok := t.records.Get().(*kgo.Record)
	if !ok {
		rec = &kgo.Record{Value: make([]byte, 0, recordCapacity)}
	}
	*rec = kgo.Record{Key: wc.key, Timestamp: ts.Time(), Value: rec.Value[:0]}
	t.slots.wc, t.slots.sample, t.slots.ts = wc, sample, ts
	var err error
	if rec.Value, err = t.template.Append(rec.Value, &t.slots); err != nil {
		t.records.Put(rec)
		return nil, err
	}
	return rec, nil
}

// reportError writes an error status that keeps the task running. A repeat of the
// last report is not rewritten.
func (t *writeTask) reportError(ctx context.Context, description string) {
	t.mu.Lock()
	repeat := description == t.lastErr
	t.lastErr = description
	t.mu.Unlock()
	if repeat {
		return
	}
	t.setStatus(
		ctx,
		driver.NoCommand,
		status.VariantError,
		true,
		"Failed to produce record",
		description,
	)
}

func (t *writeTask) setStatus(
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

// enums returns the labels of entries in the form xjson takes.
func enums(entries []payload.EnumEntry) []xjson.Enum {
	out := make([]xjson.Enum, len(entries))
	for i, e := range entries {
		out[i] = xjson.NewEnum(e.Label, e.Value)
	}
	return out
}
