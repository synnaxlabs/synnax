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
	"context"
	"fmt"
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
	runs := &runSync{
		task:     t,
		uri:      props.URI,
		progress: p,
		wake:     make(chan struct{}, 1),
		changed:  make(set.Set[ranger.Key]),
		rejected: make(set.Set[ranger.Key]),
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
			t.Report(ctx, "", nil)
			p.advance(ctx, cp)
			runs.signal()
			return nil
		},
		OnRetry: func(ctx context.Context, err error) {
			t.Report(ctx, "Failed to send data to Sift", err)
		},
	})
	if err != nil {
		closeConn()
		return nil, err
	}
	runs.engine = e
	from := fromPositions(p.positions())
	for _, index := range e.Indexes() {
		if _, ok := from[index]; !ok {
			from[index] = start
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
			disconnect := t.factoryCfg.Ranger.OnAction(runs.observe)
			defer disconnect()
			runs.signal()
			sCtx.Go(runs.run, signal.RecoverWithErrOnPanic(), signal.CancelOnFail())
		}
		sCtx.Go(func(ctx context.Context) error {
			return e.Stream(ctx, from)
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
			Details: StreamDetails{RunsThrough: start},
		}
	case err != nil:
		return nil, err
	}
	p.sent = p.status.Details.RunsThrough
	if len(p.status.Details.Positions) > 0 {
		p.sent = telem.TimeStampMax
		for _, pos := range p.status.Details.Positions {
			p.sent = min(p.sent, pos.Time)
		}
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

func (p *progress) positions() []Position {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status.Details.Positions
}

// through returns the time before which Sift has every sample of the task, and the
// time before which every labeled range that ended has a run.
func (p *progress) through() (sent, runs telem.TimeStamp) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sent, p.status.Details.RunsThrough
}

// begin saves the positions the task starts from, so that a restart before the first
// checkpoint starts there too.
func (p *progress) begin(ctx context.Context, from ingest.Positions) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.Details.Positions = toPositions(from)
	p.write(ctx)
}

func (p *progress) advance(ctx context.Context, cp ingest.Checkpoint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.Details.Positions = toPositions(cp.Positions)
	p.sent = cp.Through
	p.samples += cp.Samples
	p.write(ctx)
}

func (p *progress) setRunsThrough(ctx context.Context, ts telem.TimeStamp) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.Details.RunsThrough = ts
	p.write(ctx)
}

// write saves the status. p.mu must be held.
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
// range changes.
type runSync struct {
	task     *streamTask
	uri      string
	engine   *ingest.Engine
	progress *progress
	wake     chan struct{}
	// synced is set once a sync succeeds. Until then, every sync checks the run of each
	// ended range, since a range can change while the task is stopped.
	synced bool
	// rejected holds the ranges whose run Sift rejected, which wait until they change.
	rejected set.Set[ranger.Key]
	mu       sync.Mutex
	// changed holds the ranges that changed since the last sync. It is guarded by mu.
	changed set.Set[ranger.Key]
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
		r.task.ReportError(ctx, "Failed to set Sift run", err.Error())
	}
}

func (r *runSync) sync(ctx context.Context) (err error) {
	sent, since := r.progress.through()
	r.mu.Lock()
	changed := r.changed
	r.changed = make(set.Set[ranger.Key])
	r.mu.Unlock()
	defer func() {
		if err != nil {
			r.mu.Lock()
			r.changed.Add(changed.Slice()...)
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
	for _, rng := range ranges {
		if changed.Contains(rng.Key) {
			r.rejected.Remove(rng.Key)
		}
		end := rng.TimeRange.End
		// Sift has every sample before since, also when sent falls behind it after a
		// restart or a catch-up. So a range that ends by since does not wait for sent.
		ready := end <= sent
		if end <= since {
			ready = !r.synced || changed.Contains(rng.Key)
		}
		if !ready || r.rejected.Contains(rng.Key) {
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
			r.rejected.Add(rng.Key)
			r.task.ReportError(ctx, "Failed to set Sift run", fmt.Sprintf(
				"Sift rejected the run of %s: %s. Change the range to try again.",
				rng.Name,
				st.Message(),
			))
		}
	}
	if sent > since {
		r.progress.setRunsThrough(ctx, sent)
	}
	r.synced = true
	return nil
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
