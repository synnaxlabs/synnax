// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/sift/ingest"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// StreamTaskType is the type of a task that streams live data to Sift.
const StreamTaskType = "sift_stream"

// streamKeyPrefix starts the key of the progress status of every stream task.
const streamKeyPrefix = "sift_stream:"

// ProgressStatus is the status that holds how far a stream task has sent data to
// Sift.
type ProgressStatus = status.Status[StreamDetails]

// ProgressKey returns the key of the progress status of the stream task with key.
// Deleting the status makes the task start again from the current time.
func ProgressKey(key task.Key) status.Key { return streamKeyPrefix + key.String() }

func validateStreamConfig(c StreamConfig) error {
	v := validate.New("sift.stream_config")
	v.NotEmptyString("device", c.Device)
	v.NotEmptyString("asset", c.Asset)
	v.NotEmptySlice("channels", c.Channels)
	return v.Error()
}

// streamTask sends the live data of its channels to a Sift asset. It catches up on
// the data written while it was stopped, and makes each range with one of its labels
// a Sift run once Sift has the data of the range.
type streamTask struct {
	*driver.Runner
	factoryCfg FactoryConfig
	task       task.Task
	cfg        StreamConfig
}

func newStreamTask(
	factoryCfg FactoryConfig,
	t task.Task,
	cfg StreamConfig,
) (*streamTask, error) {
	st := &streamTask{factoryCfg: factoryCfg, task: t, cfg: cfg}
	var err error
	st.Runner, err = driver.NewRunner(driver.RunnerConfig{
		DB:              factoryCfg.DB,
		Status:          factoryCfg.Status,
		Instrumentation: factoryCfg.Instrumentation,
		Task:            t,
		Open:            st.open,
	})
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (t *streamTask) open(ctx context.Context) (func(context.Context) error, error) {
	// Data written once the task reports that it started must reach Sift.
	start := telem.Now()
	var dev device.Device
	if err := t.factoryCfg.Device.NewRetrieve().
		Where(device.MatchKeys(t.cfg.Device)).
		Entry(&dev).
		Exec(ctx, t.factoryCfg.DB); err != nil {
		return nil, err
	}
	if dev.Rack != t.task.Rack {
		return nil, validate.Errorf("device %s belongs to another rack", dev.Name)
	}
	props, err := decodeProperties(dev)
	if err != nil {
		return nil, err
	}
	p, err := t.loadProgress(ctx, start)
	if err != nil {
		return nil, err
	}
	conn, err := connect(ctx, t.factoryCfg.Secret, t.factoryCfg.DB, props)
	if err != nil {
		return nil, err
	}
	closeConn := func() {
		if err := conn.Close(); err != nil {
			t.factoryCfg.L.Warn("failed to close connection", zap.Error(err))
		}
	}
	h := &health{task: t}
	runs := &runSync{
		task:     t,
		uri:      props.URI,
		progress: p,
		health:   h,
		wake:     make(chan struct{}, 1),
		changed:  make(set.Set[ranger.Key]),
		labels:   make(set.Set[label.Key]),
		rejected: make(map[ranger.Key]string),
	}
	e, err := ingest.New(ctx, ingest.Config{
		DB:                 t.factoryCfg.DB,
		Instrumentation:    t.factoryCfg.Child("stream"),
		Framer:             t.factoryCfg.Framer,
		Channel:            t.factoryCfg.Channel,
		Conn:               conn,
		Asset:              t.cfg.Asset,
		Keys:               t.cfg.Channels,
		CheckpointInterval: t.factoryCfg.CheckpointInterval,
		OnCheckpoint: func(ctx context.Context, cp ingest.Checkpoint) error {
			if p.advance(ctx, cp) {
				h.setSend(ctx, nil)
			}
			runs.signal()
			return nil
		},
		OnRetry:    func(ctx context.Context, err error) { h.setSend(ctx, err) },
		OnRunRetry: func(ctx context.Context, err error) { h.setRun(ctx, err.Error()) },
	})
	if err != nil {
		closeConn()
		return nil, err
	}
	runs.engine = e
	saved, behind := p.from()
	from := make(ingest.Positions)
	for _, index := range e.Indexes() {
		from[index] = start
		if pos, ok := saved[index]; ok {
			from[index] = pos
		}
	}
	p.begin(ctx, from)
	return func(ctx context.Context) error {
		defer closeConn()
		sCtx, cancel := signal.WithCancel(
			ctx,
			signal.WithInstrumentation(t.factoryCfg.Instrumentation),
		)
		defer cancel()
		if len(t.cfg.Labels) > 0 {
			disconnectRanges := t.factoryCfg.Ranger.OnAction(runs.observe)
			defer disconnectRanges()
			disconnectLabels := t.factoryCfg.Label.Actions().OnChange(runs.observeLabel)
			defer disconnectLabels()
			runs.signal()
			sCtx.Go(runs.run, signal.RecoverWithErrOnPanic(), signal.CancelOnFail())
		}
		sCtx.Go(func(ctx context.Context) error {
			return e.Stream(ctx, from, behind)
		}, signal.RecoverWithErrOnPanic(), signal.CancelOnFail())
		return errors.Skip(sCtx.Wait(), context.Canceled)
	}, nil
}

// loadProgress reads the progress status of the task. A task without one starts at
// start.
func (t *streamTask) loadProgress(
	ctx context.Context,
	start telem.TimeStamp,
) (*progress, error) {
	p := &progress{cfg: t.factoryCfg, task: t.task, asset: t.cfg.Asset}
	err := t.factoryCfg.Status.NewRetrieve[StreamDetails]().
		Where(status.MatchKeys[StreamDetails](ProgressKey(t.task.Key))).
		Entry(&p.status).
		Exec(ctx, t.factoryCfg.DB)
	switch {
	case errors.Is(err, query.ErrNotFound):
		p.status = ProgressStatus{
			Key:     ProgressKey(t.task.Key),
			Name:    t.task.Name,
			Variant: status.VariantInfo,
			Details: StreamDetails{Start: start, RunsThrough: start},
		}
	case err != nil:
		return nil, err
	}
	return p, nil
}

// progress is the progress status of a stream task. It is safe for concurrent use.
type progress struct {
	cfg   FactoryConfig
	task  task.Task
	asset string
	mu    sync.Mutex
	// status, sent, and samples are guarded by mu.
	status ProgressStatus
	// sent is the time before which Sift has every sample of the task.
	sent telem.TimeStamp
	// samples counts the values Sift confirmed since the task started.
	samples int64
}

// from returns the saved positions of the task, and the channels that are behind them.
func (p *progress) from() (ingest.Positions, ingest.ChannelPositions) {
	p.mu.Lock()
	defer p.mu.Unlock()
	d := p.status.Details
	return fromPositions(d.Positions), fromChannelPositions(d.Behind)
}

// through returns the time the task started to send from, the time before which Sift
// has every sample of the task, and the time before which every labeled range that
// ended has a run.
func (p *progress) through() (start, sent, runs telem.TimeStamp) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status.Details.Start, p.sent, p.status.Details.RunsThrough
}

// begin saves the positions the task starts from, so that a restart before the first
// checkpoint starts there too.
func (p *progress) begin(ctx context.Context, from ingest.Positions) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.Details.Positions = toPositions(from)
	p.sent = slices.Min(slices.Collect(maps.Values(from)))
	p.write(ctx)
}

// advance saves cp, and reports whether Sift confirmed more of the task than the saved
// progress showed.
func (p *progress) advance(ctx context.Context, cp ingest.Checkpoint) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	positions := toPositions(cp.Positions)
	confirmed := cp.Samples > 0 || cp.Through > p.sent ||
		!slices.Equal(positions, p.status.Details.Positions)
	p.status.Details.Positions = positions
	p.status.Details.Behind = toChannelPositions(cp.Behind)
	p.sent = cp.Through
	p.samples += cp.Samples
	p.write(ctx)
	return confirmed
}

func (p *progress) setRunsThrough(ctx context.Context, ts telem.TimeStamp) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.Details.RunsThrough = ts
	p.write(ctx)
}

func toChannelPositions(p ingest.ChannelPositions) []ChannelPosition {
	positions := make([]ChannelPosition, 0, len(p))
	for ch, t := range p {
		positions = append(positions, ChannelPosition{Channel: ch, Time: t})
	}
	slices.SortFunc(positions, func(a, b ChannelPosition) int {
		return cmp.Compare(a.Channel, b.Channel)
	})
	return positions
}

func fromChannelPositions(positions []ChannelPosition) ingest.ChannelPositions {
	p := make(ingest.ChannelPositions, len(positions))
	for _, pos := range positions {
		p[pos.Channel] = pos.Time
	}
	return p
}

// write saves the status under the name of the task, unless the task was deleted. p.mu
// must be held.
func (p *progress) write(ctx context.Context) {
	p.status.Message = "Sent to Sift"
	p.status.Description = fmt.Sprintf(
		"Sent %s to asset %s since the task started.",
		sampleCount(p.samples),
		p.asset,
	)
	p.status.Time = telem.Now()
	ctx = context.WithoutCancel(ctx)
	if err := p.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		// Transactions run one at a time, so no delete of the task lands between the
		// check and the set. The factory deletes a status saved before the delete.
		var t task.Task
		err := p.cfg.Task.NewRetrieve().
			Where(task.MatchKeys(p.task.Key)).
			Entry(&t).
			Exec(ctx, tx)
		if errors.Is(err, query.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		p.status.Name = t.Name
		return p.cfg.Status.NewWriter(tx).Set(ctx, &p.status)
	}); err != nil {
		p.cfg.L.Error(
			"failed to save stream progress",
			zap.Stringer("task", p.task),
			zap.Error(err),
		)
	}
}

// runSync makes each range with one of the labels of a stream task a Sift run once
// Sift has every sample before the end of the range, and updates the run when the
// range or one of its labels changes.
type runSync struct {
	task     *streamTask
	uri      string
	engine   *ingest.Engine
	progress *progress
	health   *health
	wake     chan struct{}
	// synced is set once a sync succeeds. Until then, every sync checks the run of each
	// ended range, since a range can change while the task is stopped.
	synced bool
	// rejected holds why Sift rejected the run of each range that waits until it
	// changes.
	rejected map[ranger.Key]string
	mu       sync.Mutex
	// changed holds the ranges that changed since the last sync, and labels holds the
	// labels that changed since then. They are guarded by mu.
	changed set.Set[ranger.Key]
	labels  set.Set[label.Key]
}

func (r *runSync) signal() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *runSync) observe(
	_ context.Context,
	a actions.Scoped[ranger.Key, ranger.Action],
) {
	r.mu.Lock()
	r.changed.Add(a.Key)
	r.mu.Unlock()
	r.signal()
}

func (r *runSync) observeLabel(
	_ context.Context,
	a actions.Scoped[label.Key, label.Action],
) {
	r.mu.Lock()
	r.labels.Add(a.Key)
	r.mu.Unlock()
	r.signal()
}

func (r *runSync) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.wake:
		}
		err := r.sync(ctx)
		if err == nil || ctx.Err() != nil {
			continue
		}
		// The API key fails every run and every send, so the task stops.
		if st, ok := statusOf(err); ok && denied(st) {
			return failure(r.uri, st.Err())
		}
		r.health.setRun(ctx, err.Error())
	}
}

func (r *runSync) sync(ctx context.Context) (err error) {
	start, sent, since := r.progress.through()
	r.mu.Lock()
	changed, labels := r.changed, r.labels
	r.changed, r.labels = make(set.Set[ranger.Key]), make(set.Set[label.Key])
	r.mu.Unlock()
	defer func() {
		if err != nil {
			r.mu.Lock()
			r.changed.Add(changed.Slice()...)
			r.labels.Add(labels.Slice()...)
			r.mu.Unlock()
		}
	}()
	var ranges []ranger.Range
	if err = r.task.factoryCfg.Ranger.NewRetrieve().
		Where(ranger.MatchLabels(r.task.cfg.Labels...)).
		Entries(&ranges).
		Exec(ctx, r.task.factoryCfg.DB); err != nil {
		return err
	}
	var rejection string
	for _, rng := range ranges {
		// The names of the labels of a range are the tags of its run.
		if slices.ContainsFunc(rng.Labels, labels.Contains) {
			changed.Add(rng.Key)
		}
		if changed.Contains(rng.Key) {
			delete(r.rejected, rng.Key)
		}
		if why, ok := r.rejected[rng.Key]; ok {
			rejection = cmp.Or(rejection, why)
			continue
		}
		end := rng.TimeRange.End
		// Sift has every sample before since, also when sent falls behind it after a
		// restart or a catch-up. So a range that ends by since does not wait for sent.
		ready := end <= sent
		if end <= since {
			ready = !r.synced || changed.Contains(rng.Key)
		}
		// Sift has no data of a range that ends by start.
		if end <= start || !ready {
			continue
		}
		tags, err := rangeTags(ctx, r.task.factoryCfg, rng)
		if err != nil {
			return err
		}
		if _, err = r.engine.SetRun(ctx, ingest.Run{
			Key:       rng.Key.String(),
			Name:      rng.Name,
			TimeRange: rng.TimeRange,
			Tags:      tags,
		}); err != nil {
			st, ok := statusOf(err)
			if !ok || denied(st) {
				return err
			}
			why := fmt.Sprintf(
				"Sift rejected the run of %s: %s. Change the range to try again.",
				rng.Name,
				st.Message(),
			)
			r.rejected[rng.Key] = why
			rejection = cmp.Or(rejection, why)
		}
	}
	if sent > since {
		r.progress.setRunsThrough(ctx, sent)
	}
	r.synced = true
	r.health.setRun(ctx, rejection)
	return nil
}

// health reports the problems of a running stream task as its one warning. A failed
// send shows before a failed run, since Sift then gets no data at all.
type health struct {
	task *streamTask
	mu   sync.Mutex
	// send is the error of the last attempt to send data, and is nil once Sift confirms
	// data. It is guarded by mu.
	send error
	// run says why the last sync left a run unset, and is empty when it set each run.
	// It is guarded by mu.
	run string
}

func (h *health) setSend(ctx context.Context, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.send = err
	h.report(ctx)
}

func (h *health) setRun(ctx context.Context, why string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.run = why
	h.report(ctx)
}

// report writes the warning of the first problem, or clears the warning when there is
// none. h.mu must be held.
func (h *health) report(ctx context.Context) {
	switch {
	case h.send != nil:
		h.task.Report(ctx, "Failed to send data to Sift", h.send)
	case h.run != "":
		h.task.Report(ctx, "", driver.Warning{
			Message:     "Failed to set Sift run",
			Description: h.run,
		}.Degraded())
	default:
		h.task.Report(ctx, "", nil)
	}
}

// statusOf returns the gRPC status that err wraps.
func statusOf(err error) (*grpcstatus.Status, bool) {
	var gs interface{ GRPCStatus() *grpcstatus.Status }
	if !errors.As(err, &gs) {
		return nil, false
	}
	return gs.GRPCStatus(), true
}

// denied reports whether st rejects the API key rather than one request.
func denied(st *grpcstatus.Status) bool {
	return st.Code() == codes.Unauthenticated || st.Code() == codes.PermissionDenied
}

// rangeTags returns the names of the labels of rng, which become the tags of its Sift
// run.
func rangeTags(
	ctx context.Context,
	cfg FactoryConfig,
	rng ranger.Range,
) ([]string, error) {
	var ls []label.Label
	if len(rng.Labels) > 0 {
		if err := cfg.Label.NewRetrieve().
			Where(label.MatchKeys(rng.Labels...)).
			Entries(&ls).
			Exec(ctx, cfg.DB); err != nil {
			return nil, err
		}
	}
	tags := make([]string, len(ls))
	for i, l := range ls {
		tags[i] = l.Name
	}
	return tags, nil
}
