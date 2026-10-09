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
	"encoding/json/v2"
	"io"
	"maps"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// writeChannel pairs a configured channel with its record.
type writeChannel struct {
	cfg WriteChannel
	ch  channel.Channel
}

// writeTask produces one record per sample of each channel to a topic.
type writeTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        WriteConfig
	// status is the authority on this instance's current status.
	status   *driver.StatusHandler
	channels map[channel.Key]writeChannel
	// keys are the streamed channels: every enabled channel and its index.
	keys channel.Keys
	// client is the producer, nil while stopped.
	client *kgo.Client
	// closer releases the streamer and producer, nil once released.
	closer io.Closer
	// sCtx is the streamer's context. A streamer that fails cancels it.
	sCtx signal.Context
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
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Topic == "" {
		return nil, errors.Wrap(validate.ErrValidation, "topic: must not be empty")
	}
	if cfg.Record.ValuePointer == "" {
		return nil, errors.Wrap(
			validate.ErrValidation, "record.value_pointer: must not be empty",
		)
	}
	if _, err := resolveProperties(ctx, f.cfg.Device, cfg.Device); err != nil {
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
	channels, err := retrieveChannels(ctx, f.cfg.Channel, keys)
	if err != nil {
		return nil, err
	}
	wt := &writeTask{
		factoryCfg: f.cfg,
		task:       t,
		cfg:        cfg,
		status:     driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t),
		channels:   make(map[channel.Key]writeChannel, len(keys)),
	}
	streamed := make(set.Set[channel.Key], len(keys))
	for _, wc := range cfg.Channels {
		if wc.Disabled {
			continue
		}
		ch := channels[wc.Channel]
		if ch.DataType == telem.TimestampT && wc.TimeFormat == nil {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"channel %q: time_format is required for timestamp channel %s",
				wc.Key, ch.Name,
			)
		}
		wt.channels[wc.Channel] = writeChannel{cfg: wc, ch: ch}
		for _, k := range []channel.Key{ch.Key(), ch.Index()} {
			if k != 0 && !streamed.Contains(k) {
				streamed.Add(k)
				wt.keys = append(wt.keys, k)
			}
		}
	}
	return wt, nil
}

// Exec implements driver.Task. Every command it handles ends in a status carrying the
// command key, so a caller waiting on the command always resolves.
func (t *writeTask) Exec(ctx context.Context, cmd task.Command) error {
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
func (t *writeTask) Stop(sendStatus bool) error {
	return t.stop(context.TODO(), driver.NoCommand, sendStatus)
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

func (t *writeTask) start(ctx context.Context, cmdKey string) error {
	if t.isRunning() {
		t.ackCurrent(ctx, cmdKey, true)
		return nil
	}
	if t.closer != nil {
		if err := t.release(); err != nil {
			t.factoryCfg.L.Warn("kafka producer failed",
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

func (t *writeTask) stop(ctx context.Context, cmdKey string, sendStatus bool) error {
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

// open connects the producer and streams the channels into it. It writes no
// statuses: start owns them.
func (t *writeTask) open(ctx context.Context) (err error) {
	props, err := resolveProperties(ctx, t.factoryCfg.Device, t.cfg.Device)
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
		ctx, framer.StreamerConfig{Keys: t.keys},
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
	t.client = cl
	t.sCtx = sCtx
	t.lastErr = ""
	t.closer = xio.MultiCloser{
		xio.CloserFunc(t.closeClient),
		signal.NewGracefulShutdown(sCtx, cancel),
		xio.NoFailCloserFunc(requests.Close),
	}
	streamer.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	sCtx.Go(func(ctx context.Context) error {
		for res := range responses.Outlet() {
			t.produce(ctx, res.Frame)
		}
		return nil
	}, signal.RecoverWithErrOnPanic())
	return nil
}

// closeClient flushes the buffered records and closes the producer.
func (t *writeTask) closeClient() error {
	ctx, cancel := context.WithTimeout(
		context.Background(), t.factoryCfg.PingTimeout.Duration(),
	)
	defer cancel()
	err := t.client.Flush(ctx)
	t.client.Close()
	return err
}

// produce emits one record per sample of every configured channel in fr.
func (t *writeTask) produce(ctx context.Context, fr frame.Frame) {
	series := maps.Collect(fr.Entries())
	now := telem.Now()
	for key, wc := range t.channels {
		s, ok := series[key]
		if !ok {
			continue
		}
		index, hasIndex := series[wc.ch.Index()]
		for i := range int(s.Len()) {
			ts := now
			if hasIndex && int64(i) < index.Len() {
				ts = index.ValueAt[telem.TimeStamp](i)
			}
			rec, err := t.record(wc, s, i, ts)
			if err != nil {
				t.reportError(ctx, "Record build failed: "+err.Error())
				continue
			}
			t.client.Produce(ctx, rec, func(_ *kgo.Record, err error) {
				if err != nil {
					t.reportError(ctx, "Produce failed: "+err.Error())
				}
			})
		}
	}
}

// record builds the record for sample i of s, stamped ts.
func (t *writeTask) record(
	wc writeChannel,
	s telem.Series,
	i int,
	ts telem.TimeStamp,
) (*kgo.Record, error) {
	doc := make(map[string]any)
	value, err := writeSample(s, i, wc.cfg)
	if err != nil {
		return nil, err
	}
	if err := pointerSet(doc, t.cfg.Record.ValuePointer, value); err != nil {
		return nil, err
	}
	if p := t.cfg.Record.ChannelPointer; p != nil {
		if err := pointerSet(doc, *p, wc.ch.Name); err != nil {
			return nil, err
		}
	}
	if p := t.cfg.Record.TimestampPointer; p != nil {
		stamp, err := formatTimestamp(ts, t.cfg.Record.TimeFormat)
		if err != nil {
			return nil, err
		}
		if err := pointerSet(doc, *p, stamp); err != nil {
			return nil, err
		}
	}
	for _, f := range t.cfg.Record.Fields {
		pointer, v, err := extraFieldValue(f, ts)
		if err != nil {
			return nil, err
		}
		if err := pointerSet(doc, pointer, v); err != nil {
			return nil, err
		}
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	rec := &kgo.Record{Value: body, Timestamp: ts.Time()}
	if t.cfg.RecordKey == RecordKeyChannelName {
		rec.Key = []byte(wc.ch.Name)
	}
	return rec, nil
}

// reportError writes an error status that keeps the task running. A repeat of the
// last report is not rewritten.
func (t *writeTask) reportError(ctx context.Context, message string) {
	t.mu.Lock()
	repeat := message == t.lastErr
	t.lastErr = message
	t.mu.Unlock()
	if repeat {
		return
	}
	t.setStatus(ctx, driver.NoCommand, status.VariantError, true, message)
}

// ackCurrent answers cmdKey with the task's current status, for a command that needs
// no work.
func (t *writeTask) ackCurrent(ctx context.Context, cmdKey string, running bool) {
	if err := t.status.Ack(ctx, cmdKey, running); err != nil {
		t.factoryCfg.L.Error("failed to acknowledge command",
			zap.Stringer("task", t.task.Key),
			zap.String("cmd", cmdKey),
			zap.Error(err),
		)
	}
}

func (t *writeTask) setStatus(
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
