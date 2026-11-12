// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ingest sends channel data to Sift. It reads every sample from the Core's
// storage, so after a Sift outage or a restart it sends again from the last position
// Sift confirmed instead of keeping its own copy.
package ingest

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv2 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v2"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Positions maps each index channel to the time before which every sample of its
// channels was sent.
type Positions map[channel.Key]telem.TimeStamp

// Checkpoint reports what Sift confirmed.
type Checkpoint struct {
	// Positions holds, for each index, the time before which Sift confirmed every
	// sample.
	Positions Positions
	// Through is the earliest of Positions. Once Stream sends live samples, it is the
	// latest of Positions and the last time Stream had sent every sample it received,
	// so an idle index or an idle stream does not hold it back.
	Through telem.TimeStamp
	// Samples is the number of values Sift confirmed since the previous checkpoint.
	Samples int64
}

// Config configures an Engine.
type Config struct {
	alamos.Instrumentation
	// DB reads the channels in Keys and their indexes.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Framer reads the samples sent to Sift.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Channel resolves the channels in Keys and their indexes.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Conn is the connection to Sift.
	//
	// [REQUIRED]
	Conn grpc.ClientConnInterface
	// Asset is the name of the Sift asset that receives the samples.
	//
	// [REQUIRED]
	Asset string
	// Keys are the channels to send. Index channels in Keys send nothing themselves.
	//
	// [REQUIRED]
	Keys channel.Keys
	// CheckpointInterval is how long the engine sends before it waits for Sift to
	// confirm what it sent.
	//
	// [OPTIONAL] - Defaults to 10 s.
	CheckpointInterval telem.TimeSpan
	// ConfirmTimeout is how long the engine waits for Sift to confirm what it sent. A
	// stopped engine also waits at most this long for a stream that Sift stops reading.
	//
	// [OPTIONAL] - Defaults to 30 s.
	ConfirmTimeout telem.TimeSpan
	// Retry sets the wait between attempts after a temporary failure.
	//
	// [OPTIONAL] - Defaults to 1 s, doubling to at most 30 s.
	Retry breaker.Config
	// OnCheckpoint runs after each time Sift confirms what the engine sent.
	//
	// [REQUIRED]
	OnCheckpoint func(ctx context.Context, cp Checkpoint) error
	// OnRetry runs after a temporary failure, before the engine waits to try again.
	//
	// [OPTIONAL]
	OnRetry func(ctx context.Context, err error)
}

var _ config.Config[Config] = Config{}

// DefaultConfig is the default configuration for an Engine.
var DefaultConfig = Config{
	CheckpointInterval: 10 * telem.Second,
	ConfirmTimeout:     30 * telem.Second,
	Retry: breaker.Config{
		BaseInterval: time.Second,
		Scale:        2,
		MaxInterval:  30 * time.Second,
		MaxRetries:   breaker.InfiniteRetries,
		Clock:        xtime.Real,
	},
	OnRetry: func(context.Context, error) {},
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Conn = override.Nil(c.Conn, other.Conn)
	c.Asset = override.String(c.Asset, other.Asset)
	c.Keys = override.Slice(c.Keys, other.Keys)
	c.CheckpointInterval = override.Numeric(
		c.CheckpointInterval,
		other.CheckpointInterval,
	)
	c.ConfirmTimeout = override.Numeric(c.ConfirmTimeout, other.ConfirmTimeout)
	c.Retry = c.Retry.Override(other.Retry)
	c.OnCheckpoint = override.Nil(c.OnCheckpoint, other.OnCheckpoint)
	c.OnRetry = override.Nil(c.OnRetry, other.OnRetry)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("sift.ingest")
	v.NotNil("db", c.DB)
	v.NotNil("framer", c.Framer)
	v.NotNil("channel", c.Channel)
	v.NotNil("conn", c.Conn)
	v.NotEmptyString("asset", c.Asset)
	v.NotEmptySlice("keys", c.Keys)
	v.Positive("checkpoint_interval", c.CheckpointInterval)
	v.Positive("confirm_timeout", c.ConfirmTimeout)
	v.NotNil("on_checkpoint", c.OnCheckpoint)
	v.NotNil("on_retry", c.OnRetry)
	return errors.Join(v.Error(), c.Retry.Validate())
}

// handoffWindow is how short a catch-up pass must be before a stream starts reading
// live samples. It keeps the live samples held during the last pass few.
const handoffWindow = time.Second

// liveBuffer is how many live frames a stream holds before it falls back to reading
// from storage.
const liveBuffer = 1024

// errBehind means a stream fell behind its live samples and must read from storage.
var errBehind = errors.New("fell behind the live samples")

// Engine sends the samples of a fixed set of channels to one Sift asset. An Engine is
// not safe for concurrent use, except that SetRun may run alongside Upload or Stream.
type Engine struct {
	cfg     Config
	flows   []flow
	ingest  ingestv1.IngestServiceClient
	configs ingestionconfigsv2.IngestionConfigServiceClient
	runs    runsv2.RunServiceClient
	// mu guards configID and assetID.
	mu       sync.Mutex
	configID string
	assetID  string
	// confirms counts the checkpoints Sift has confirmed.
	confirms int
}

// New returns an Engine for the channels in cfg. It contacts no Sift server. It
// returns a validation error when a channel is virtual, has no index, or has a data
// type Sift cannot store, and query.ErrNotFound when a channel does not exist.
func New(ctx context.Context, cfgs ...Config) (*Engine, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	var channels, indexes []channel.Channel
	if err = cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		if err := cfg.Channel.NewRetrieve().
			Where(channel.MatchKeys(cfg.Keys...)).
			Entries(&channels).
			Exec(ctx, snap); err != nil {
			return err
		}
		indexKeys := make(set.Set[channel.Key])
		for _, ch := range channels {
			if !ch.IsIndex && ch.Index() != 0 {
				indexKeys.Add(ch.Index())
			}
		}
		if len(indexKeys) == 0 {
			return nil
		}
		return cfg.Channel.NewRetrieve().
			Where(channel.MatchKeys(indexKeys.Slice()...)).
			Entries(&indexes).
			Exec(ctx, snap)
	}); err != nil {
		return nil, err
	}
	flows, err := buildFlows(channels, indexes)
	if err != nil {
		return nil, err
	}
	return &Engine{
		cfg:     cfg,
		flows:   flows,
		ingest:  ingestv1.NewIngestServiceClient(cfg.Conn),
		configs: ingestionconfigsv2.NewIngestionConfigServiceClient(cfg.Conn),
		runs:    runsv2.NewRunServiceClient(cfg.Conn),
	}, nil
}

// Indexes returns the keys of the index channels the engine tracks positions for.
func (e *Engine) Indexes() channel.Keys {
	keys := make(channel.Keys, len(e.flows))
	for i, f := range e.flows {
		keys[i] = f.index.Key()
	}
	return keys
}

// Upload sends the stored samples in tr. Each index starts at its position in from,
// or at the start of tr when from has none. Upload returns nil once Sift confirms
// every sample, and retries temporary failures from the last confirmed positions. If
// ctx is cancelled first, Upload waits for Sift to confirm what it sent and returns
// ctx.Err().
func (e *Engine) Upload(ctx context.Context, tr telem.TimeRange, from Positions) error {
	confirmed := e.positions(from, tr.Start)
	return e.retry(ctx, func(ctx context.Context, a *attempt) error {
		for _, f := range e.flows {
			key := f.index.Key()
			if a.sent[key] >= tr.End {
				continue
			}
			if err := e.read(
				ctx,
				f,
				f.channels,
				telem.TimeRange{Start: a.sent[key], End: tr.End},
				func(fr framer.Frame) error {
					if err := a.send(ctx, f, fr); err != nil {
						return err
					}
					return a.maybeCheckpoint(ctx)
				},
			); err != nil {
				return err
			}
			a.sent[key] = tr.End
		}
		return a.checkpoint(ctx, true)
	}, confirmed)
}

// Stream sends the stored samples from the positions in from, then each sample as it
// is written, until ctx is cancelled. An index with no position starts at the current
// time. Stream retries temporary failures from the last confirmed positions. When ctx
// is cancelled it waits for Sift to confirm what it sent and returns nil, or the error
// of that confirm.
func (e *Engine) Stream(ctx context.Context, from Positions) error {
	err := e.retry(ctx, e.stream, e.positions(from, telem.Now()))
	if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
		return nil
	}
	return err
}

// stream catches up from storage until a pass is short, then sends live samples.
func (e *Engine) stream(ctx context.Context, a *attempt) (err error) {
	for {
		start := time.Now()
		if err = e.catchUp(ctx, a); err != nil {
			return err
		}
		if time.Since(start) < handoffWindow {
			break
		}
	}
	l, err := e.openLive(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, l.close()) }()
	if err = e.catchUp(ctx, a); err != nil {
		return err
	}
	a.live = true
	ticker := time.NewTicker(e.cfg.CheckpointInterval.Duration())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if len(l.frames) == 0 {
				a.idle = telem.Now()
			}
			if err = a.checkpoint(ctx, false); err != nil {
				return err
			}
		case fr, ok := <-l.frames:
			if !ok {
				return a.fallBehind(ctx, l.err())
			}
			for _, f := range e.flows {
				if err = a.sendLive(ctx, f, fr); err != nil {
					return err
				}
			}
			if err = a.maybeCheckpoint(ctx); err != nil {
				return err
			}
		}
	}
}

func (e *Engine) positions(from Positions, start telem.TimeStamp) Positions {
	p := make(Positions, len(e.flows))
	for _, f := range e.flows {
		pos, ok := from[f.index.Key()]
		if !ok {
			pos = start
		}
		p[f.index.Key()] = pos
	}
	return p
}

// catchUp sends the stored samples of every flow from its sent position.
func (e *Engine) catchUp(ctx context.Context, a *attempt) error {
	for _, f := range e.flows {
		if err := e.read(
			ctx,
			f,
			f.channels,
			telem.TimeRange{Start: a.sent[f.index.Key()], End: telem.TimeStampMax},
			func(fr framer.Frame) error {
				if err := a.send(ctx, f, fr); err != nil {
					return err
				}
				return a.maybeCheckpoint(ctx)
			},
		); err != nil {
			return err
		}
	}
	return nil
}

// read calls fn with the stored samples of f in tr, one chunk of the index at a time.
// Each frame holds every sample of channels in the time span of its chunk, so a data
// sample always arrives with its index sample.
func (e *Engine) read(
	ctx context.Context,
	f flow,
	channels []channel.Channel,
	tr telem.TimeRange,
	fn func(framer.Frame) error,
) (err error) {
	idx, err := e.cfg.Framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   channel.Keys{f.index.Key()},
		Bounds: tr,
	})
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, idx.Close()) }()
	data, err := e.cfg.Framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   channel.KeysFromChannels(channels),
		Bounds: tr,
	})
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, data.Close()) }()
	if !idx.SeekFirst() {
		return nil
	}
	for idx.Next(framer.IteratorAutoSpan) {
		fr := idx.Value()
		span := timeRange(fr.Get(f.index.Key()).Series)
		data.SetBounds(span)
		if data.SeekFirst() && data.Next(span.Span()) {
			fr = fr.Extend(data.Value())
		} else if err = data.Error(); err != nil {
			return err
		}
		if err = fn(fr); err != nil {
			return err
		}
	}
	return nil
}

// timeRange returns the time range from the first to just after the last timestamp in
// series.
func timeRange(series []telem.Series) telem.TimeRange {
	return telem.TimeRange{
		Start: series[0].ValueAt[telem.TimeStamp](0),
		End:   series[len(series)-1].ValueAt[telem.TimeStamp](-1) + 1,
	}
}

// retry runs attempts until one succeeds, ctx is cancelled, or one fails with an error
// that retrying cannot fix. Each attempt starts from confirmed, which attempts update
// as Sift confirms what they send.
func (e *Engine) retry(
	ctx context.Context,
	run func(context.Context, *attempt) error,
	confirmed Positions,
) error {
	brk, err := breaker.NewBreaker(ctx, e.cfg.Retry)
	if err != nil {
		return err
	}
	for {
		id, err := e.ingestionConfig(ctx)
		if err == nil {
			a := &attempt{
				engine: e,
				sender: sender{
					client:  e.ingest,
					ctx:     ctx,
					timeout: e.cfg.ConfirmTimeout.Duration(),
				},
				configID:  id,
				confirmed: confirmed,
				sent:      maps.Clone(confirmed),
				start:     maps.Clone(confirmed),
				ends:      make(map[channel.Key]telem.Alignment),
				values:    make(map[channel.Key]telem.TimeStamp),
				unindexed: make(map[channel.Key]set.Set[channel.Key]),
				last:      time.Now(),
			}
			confirms := e.confirms
			err = run(ctx, a)
			if err != nil && ctx.Err() != nil {
				return a.stop(ctx)
			}
			a.sender.abort()
			if err == nil {
				return nil
			}
			if e.confirms > confirms {
				brk.Reset()
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, errBehind) {
			e.cfg.L.Debug(
				"reading from storage to catch up",
				zap.String("asset", e.cfg.Asset),
			)
			continue
		}
		permanent, err := classify(err)
		if permanent {
			return err
		}
		e.cfg.OnRetry(ctx, err)
		if !brk.Wait() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
	}
}

// asset returns the ID of the Sift asset, creating the ingestion config of the engine
// when it does not exist yet.
func (e *Engine) asset(ctx context.Context) (string, error) {
	if _, err := e.ingestionConfig(ctx); err != nil {
		return "", err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.assetID, nil
}

// ingestionConfig returns the ID of the Sift ingestion config for the engine's flows,
// creating it the first time.
func (e *Engine) ingestionConfig(ctx context.Context) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.configID != "" {
		return e.configID, nil
	}
	key := clientKey(e.cfg.Asset, e.flows)
	find := func() (*ingestionconfigsv2.IngestionConfig, error) {
		res, err := e.configs.ListIngestionConfigs(
			ctx,
			&ingestionconfigsv2.ListIngestionConfigsRequest{
				Filter: fmt.Sprintf("client_key == %q", key),
			},
		)
		if err != nil || len(res.IngestionConfigs) == 0 {
			return nil, err
		}
		return res.IngestionConfigs[0], nil
	}
	cfg, err := find()
	if err != nil {
		return "", err
	}
	if cfg == nil {
		flows := make([]*ingestionconfigsv2.FlowConfig, len(e.flows))
		for i, f := range e.flows {
			flows[i] = f.config()
		}
		res, err := e.configs.CreateIngestionConfig(
			ctx,
			&ingestionconfigsv2.CreateIngestionConfigRequest{
				AssetName: e.cfg.Asset,
				Flows:     flows,
				ClientKey: key,
			},
		)
		switch {
		case status.Code(err) == codes.AlreadyExists:
			if cfg, err = find(); err != nil {
				return "", err
			}
			if cfg == nil {
				return "", errors.New(
					"Sift reported an ingestion config that it does not list",
				)
			}
		case err != nil:
			return "", err
		default:
			cfg = res.IngestionConfig
		}
	}
	e.configID, e.assetID = cfg.IngestionConfigId, cfg.AssetId
	return e.configID, nil
}

// attempt is one try at sending, from the positions Sift last confirmed.
type attempt struct {
	engine    *Engine
	sender    sender
	configID  string
	confirmed Positions
	sent      Positions
	// start holds the positions the attempt started from.
	start Positions
	// pending counts the values sent since the last checkpoint.
	pending int64
	last    time.Time
	// failed is set once a send or a confirm fails, after which Sift confirms nothing
	// more.
	failed bool
	// live is set once the attempt sends live samples.
	live bool
	// idle is the last time a live attempt had sent every sample it received.
	idle telem.TimeStamp
	// reported is the Through of the last checkpoint.
	reported telem.TimeStamp
	// ends holds, for each index, the alignment after the last live sample.
	ends map[channel.Key]telem.Alignment
	// values holds, for each data channel, the time after the last value sent.
	values map[channel.Key]telem.TimeStamp
	// unindexed holds the data channels of each index that arrived live without it.
	unindexed map[channel.Key]set.Set[channel.Key]
}

func (a *attempt) send(ctx context.Context, f flow, fr framer.Frame) error {
	next, err := a.encode(ctx, f, fr, a.sent[f.index.Key()])
	a.sent[f.index.Key()] = next
	return err
}

// encode sends the samples of f in fr at or after from, and returns the timestamp after
// the last one it read.
func (a *attempt) encode(
	ctx context.Context,
	f flow,
	fr framer.Frame,
	from telem.TimeStamp,
) (telem.TimeStamp, error) {
	last := make([]telem.TimeStamp, len(f.channels))
	next, n, err := encode(a.configID, f, fr, from, last, func(
		req *ingestv1.IngestWithConfigDataStreamRequest,
	) error {
		// The stream outlives ctx, so a cancelled ctx stops nothing on its own.
		if err := ctx.Err(); err != nil {
			return err
		}
		return a.sender.send(req)
	})
	for i, ts := range last {
		if ts > a.value(f, f.channels[i]) {
			a.values[f.channels[i].Key()] = ts
		}
	}
	a.pending += n
	a.failed = err != nil && !errors.Is(err, ctx.Err())
	return next, err
}

// value returns the time after the last value of ch sent, or the start of the attempt
// when it sent none.
func (a *attempt) value(f flow, ch channel.Channel) telem.TimeStamp {
	if v, ok := a.values[ch.Key()]; ok {
		return v
	}
	return a.start[f.index.Key()]
}

// stop confirms what the attempt sent after ctx is cancelled, so the next run starts
// after it. It returns ctx.Err() unless the confirm fails.
func (a *attempt) stop(ctx context.Context) error {
	defer a.sender.abort()
	if a.failed {
		return ctx.Err()
	}
	if err := a.checkpoint(context.WithoutCancel(ctx), false); err != nil {
		return err
	}
	return ctx.Err()
}

func (a *attempt) maybeCheckpoint(ctx context.Context) error {
	if time.Since(a.last) < a.engine.cfg.CheckpointInterval.Duration() {
		return nil
	}
	return a.checkpoint(ctx, false)
}

// checkpoint waits for Sift to confirm what the attempt sent, then reports it. Unless
// always is set, it reports nothing when nothing changed.
func (a *attempt) checkpoint(ctx context.Context, always bool) error {
	a.last = time.Now()
	if err := a.sendUnindexed(ctx); err != nil {
		return err
	}
	if err := a.sender.confirm(); err != nil {
		a.failed = true
		return err
	}
	through := a.through()
	if !always && a.pending == 0 && maps.Equal(a.sent, a.confirmed) &&
		through == a.reported {
		return nil
	}
	maps.Copy(a.confirmed, a.sent)
	a.engine.confirms++
	cp := Checkpoint{
		Positions: maps.Clone(a.confirmed),
		Through:   through,
		Samples:   a.pending,
	}
	a.pending = 0
	a.reported = through
	return a.engine.cfg.OnCheckpoint(ctx, cp)
}

// fallBehind confirms what the attempt sent before it returns err. When err is
// errBehind, every sample sent precedes the first one lost, so the next attempt reads
// from storage without sending any value twice.
func (a *attempt) fallBehind(ctx context.Context, err error) error {
	if !errors.Is(err, errBehind) {
		return err
	}
	return errors.Combine(a.checkpoint(ctx, false), err)
}

// sendLive sends the samples of f in a live frame. When the frame does not continue the
// last live samples of f, as after a frame the relay dropped or a write the streamer
// missed while it opened, sendLive first sends the stored samples between them.
func (a *attempt) sendLive(ctx context.Context, f flow, fr framer.Frame) error {
	key := f.index.Key()
	series := fr.Get(key).Series
	if len(series) == 0 {
		a.markUnindexed(f, fr)
		return nil
	}
	var gap telem.TimeStamp
	for _, s := range series {
		end, ok := a.ends[key]
		sameDomain := ok && s.Alignment.DomainIndex() == end.DomainIndex()
		if !sameDomain || s.Alignment.SampleIndex() > end.SampleIndex() {
			gap = max(gap, s.ValueAt[telem.TimeStamp](0))
		}
		if next := s.Alignment.AddSamples(uint32(s.Len())); !sameDomain || next > end {
			a.ends[key] = next
		}
	}
	if gap > a.sent[key] {
		if err := a.engine.read(
			ctx,
			f,
			f.channels,
			telem.TimeRange{Start: a.sent[key], End: gap},
			func(stored framer.Frame) error { return a.send(ctx, f, stored) },
		); err != nil {
			return err
		}
	}
	return a.send(ctx, f, fr)
}

// markUnindexed records the data channels of f in a live frame without the index of f.
// Their timestamps are only in storage.
func (a *attempt) markUnindexed(f flow, fr framer.Frame) {
	for _, ch := range f.channels {
		if len(fr.Get(ch.Key()).Series) == 0 {
			continue
		}
		if a.unindexed[f.index.Key()] == nil {
			a.unindexed[f.index.Key()] = set.New[channel.Key]()
		}
		a.unindexed[f.index.Key()].Add(ch.Key())
	}
}

// sendUnindexed sends the stored values of each channel that arrived without its index,
// from its last value sent to the index position. It sends no other channel, so Sift
// gets no value twice. A channel stays marked while it has a stored value at or after
// the index position.
func (a *attempt) sendUnindexed(ctx context.Context) error {
	for _, f := range a.engine.flows {
		keys := a.unindexed[f.index.Key()]
		end := a.sent[f.index.Key()]
		for _, ch := range f.channels {
			if !keys.Contains(ch.Key()) {
				continue
			}
			if start := a.value(f, ch); start < end {
				if err := a.engine.read(
					ctx,
					f,
					[]channel.Channel{ch},
					telem.TimeRange{Start: start, End: end},
					func(fr framer.Frame) error {
						_, err := a.encode(ctx, f, fr, start)
						return err
					},
				); err != nil {
					return err
				}
			}
			pending, err := a.engine.stored(ctx, ch, end)
			if err != nil {
				return err
			}
			if !pending {
				keys.Remove(ch.Key())
			}
		}
	}
	return nil
}

// stored reports whether storage holds a value of ch at or after ts.
func (e *Engine) stored(
	ctx context.Context,
	ch channel.Channel,
	ts telem.TimeStamp,
) (found bool, err error) {
	it, err := e.cfg.Framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   channel.Keys{ch.Key()},
		Bounds: telem.TimeRange{Start: ts, End: telem.TimeStampMax},
	})
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Combine(err, it.Close()) }()
	found = it.SeekFirst() && it.Next(framer.IteratorAutoSpan)
	return found, it.Error()
}

func (a *attempt) through() telem.TimeStamp {
	positions := slices.Collect(maps.Values(a.sent))
	if a.live {
		return max(slices.Max(positions), a.idle)
	}
	return slices.Min(positions)
}

// live holds the frames a streamer receives until the engine sends them. It never
// blocks the streamer: once the engine falls behind, it closes frames and drops the
// rest.
type live struct {
	frames   <-chan framer.Frame
	shutdown func() error
	once     sync.Once
	closeErr error
	// behind is written before frames closes.
	behind bool
}

// err returns why frames closed. close returns the cause of an unexpected close.
func (l *live) err() error {
	if l.behind {
		return errBehind
	}
	return errors.New("the live stream closed unexpectedly")
}

func (l *live) close() error {
	l.once.Do(func() { l.closeErr = l.shutdown() })
	return l.closeErr
}

func (e *Engine) openLive(ctx context.Context) (*live, error) {
	keys := make(channel.Keys, 0)
	for _, f := range e.flows {
		keys = append(keys, f.keys()...)
	}
	streamer, err := e.cfg.Framer.NewStreamer(
		ctx,
		framer.StreamerConfig{Keys: keys, SendOpenAck: true},
	)
	if err != nil {
		return nil, err
	}
	requests := confluence.NewStream[framer.StreamerRequest]()
	responses := confluence.NewStream[framer.StreamerResponse](1)
	streamer.InFrom(requests)
	streamer.OutTo(responses)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(e.cfg.Instrumentation))
	streamer.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	shutdown := signal.NewGracefulShutdown(sCtx, cancel)
	frames := make(chan framer.Frame, liveBuffer)
	l := &live{
		frames: frames,
		shutdown: func() error {
			requests.Close()
			return shutdown.Close()
		},
	}
	select {
	case <-ctx.Done():
		cancel()
		return nil, errors.Join(ctx.Err(), l.close())
	case _, ok := <-responses.Outlet():
		if !ok {
			return nil, errors.Join(
				errors.New("the live stream closed before opening"),
				l.close(),
			)
		}
	}
	sCtx.Go(func(context.Context) error {
		open := true
		for res := range responses.Outlet() {
			if !open {
				continue
			}
			select {
			case frames <- res.Frame:
			default:
				l.behind = true
				close(frames)
				open = false
			}
		}
		if open {
			close(frames)
		}
		return nil
	}, signal.RecoverWithErrOnPanic())
	return l, nil
}
