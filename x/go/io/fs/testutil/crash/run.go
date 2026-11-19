// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

// snapshot is the crash image behind one event, with the rules for an engine that
// recovers from it.
type snapshot struct {
	image *fault.FS
	rules rules
	event int
}

// run is the state of one Run.
type run struct {
	ctx     context.Context
	disk    *fault.FS
	counter *counter
	// pending is the newest crash image. It waits for the acks that the engine gives
	// while the disk holds its state.
	pending *snapshot
	// broken is the first rule that the engine broke, at the event brokenAt.
	broken *broken
	// err is the first error of the harness itself.
	err  error
	plan plan
	// attempts holds each commit so far, and acked holds the attempts as they were when
	// the newest disk change began.
	attempts, acked []attempt
	cfg             Config
	brokenAt        int
	// mu guards every field below plan.
	mu sync.Mutex
}

// Run runs the workload of cfg on an engine. After each Stride calls that change the
// disk it takes a crash image. It opens the engine on each image two times, and applies
// the oracle to what the engine recovers, with the acks that the engine gave while the
// disk held that state. It returns a *Violation for the first rule that the engine
// breaks, an error that matches validate.ErrValidation for a bad cfg, and nil when the
// engine holds.
func Run(ctx context.Context, cfg Config) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	r := &run{ctx: ctx, cfg: cfg, plan: newPlan(cfg)}
	disk, err := fault.New(r.plan.workload)
	if err != nil {
		return err
	}
	r.disk = disk
	r.counter = &counter{
		grown:  map[fault.ID]growth{},
		before: r.beforeEvent,
		after:  r.afterEvent,
	}
	r.pending = r.snapshot(0)
	if engine, err := cfg.Open(ctx, &countedFS{inner: disk, c: r.counter}); err != nil {
		r.worksf(r.faulted(), "open on an empty disk: %v", err)
	} else {
		r.work(engine)
		if err = engine.Close(); err != nil {
			r.worksf(r.faulted(), "close: %v", err)
		}
	}
	r.counter.mu.Lock()
	events := r.counter.events
	r.counter.mu.Unlock()
	if r.pending == nil || r.pending.event != events {
		r.pending = r.snapshot(events)
	}
	r.check(r.pending, r.attempts)
	if r.err != nil {
		return errors.Wrap(r.err, "crash: harness")
	}
	if r.broken == nil {
		return nil
	}
	return &Violation{
		Rule:    r.broken.rule,
		Detail:  r.broken.detail,
		RunKey:  cfg.RunKey,
		Profile: cfg.Profile,
		Event:   r.brokenAt,
	}
}

// work runs the commits of the workload on engine, then reads them back.
func (r *run) work(engine Engine) {
	rng := rand.New(rand.NewPCG(r.cfg.RunKey, 0))
	for id := uint64(1); id <= uint64(r.cfg.Commits) && !r.stopped(); id++ {
		rec, want := Record{ID: id, Payload: payload(r.cfg.RunKey, id)}, Durable
		if rng.IntN(5) < 2 {
			want = Visible
		}
		r.mu.Lock()
		r.attempts = append(r.attempts, attempt{record: rec})
		r.mu.Unlock()
		ack, err := engine.Commit(r.ctx, rec, want)
		if err != nil {
			r.worksf(
				r.faulted(),
				"commit %d failed on a disk with no fault: %v",
				id,
				err,
			)
			continue
		}
		r.worksf(
			ack == Visible || ack == Durable,
			"commit %d got the ack %d",
			id,
			ack,
		)
		r.counter.mu.Lock()
		events := r.counter.events
		r.counter.mu.Unlock()
		r.mu.Lock()
		r.attempts[id-1].ack, r.attempts[id-1].ackedAt = ack, events
		r.mu.Unlock()
	}
	if r.stopped() {
		return
	}
	records, err := engine.Recovered(r.ctx)
	if err != nil {
		r.worksf(r.faulted(), "read before a crash: %v", err)
		return
	}
	if b := checkLive(r.attempts, records); b != nil {
		r.fail(b, 0)
	}
}

// faulted reports if the disk of the workload failed a call.
func (r *run) faulted() bool {
	r.counter.mu.Lock()
	defer r.counter.mu.Unlock()
	return r.counter.faults > 0
}

// worksf breaks R0, before any crash, when ok is false.
func (r *run) worksf(ok bool, format string, args ...any) {
	if !ok {
		r.fail(&broken{R0, fmt.Sprintf(format, args...)}, 0)
	}
}

func (r *run) stopped() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.broken != nil || r.err != nil
}

func (r *run) fail(b *broken, event int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.broken == nil {
		r.broken, r.brokenAt = b, event
	}
}

// beforeEvent notes the acks that the engine gave before the disk changes.
func (r *run) beforeEvent() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.acked = append(r.acked[:0], r.attempts...)
}

// afterEvent checks the image of the state that the disk just left, against the acks
// that the engine gave in that state. Then it takes the image of the new state. The
// counter holds its mutex, so the disk does not change.
func (r *run) afterEvent(event int) {
	if r.stopped() {
		return
	}
	if r.pending != nil {
		r.check(r.pending, r.acked)
	}
	r.pending = nil
	if event%r.cfg.Stride == 0 {
		r.pending = r.snapshot(event)
	}
}

// snapshot takes the crash image behind the event. The counter does not change while it
// runs.
func (r *run) snapshot(event int) *snapshot {
	image, err := r.disk.Crash(r.plan.crash(event))
	if err == nil {
		var rules rules
		if rules, err = r.plan.spoil(image, r.counter, event); err == nil {
			return &snapshot{image: image, rules: rules, event: event}
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		r.err = err
	}
	return nil
}

// check opens the engine on the image of s two times, and applies the oracle to what it
// recovers each time. The second time finds a recovery that harms the disk.
func (r *run) check(s *snapshot, attempts []attempt) {
	if s == nil || r.stopped() {
		return
	}
	for range 2 {
		records, err := r.recover(s.image)
		if err != nil {
			refused := s.rules.refusalAllowed && r.cfg.Corrupt != nil &&
				r.cfg.Corrupt(err)
			r.unlessAtf(refused, s.event, R4, "the engine did not recover: %v", err)
			return
		}
		if b := checkRecovered(attempts, records, s.rules); b != nil {
			r.fail(b, s.event)
			return
		}
	}
}

// unlessAtf breaks the rule at the event when ok is false.
func (r *run) unlessAtf(ok bool, event int, rule Rule, format string, args ...any) {
	if !ok {
		r.fail(&broken{rule, fmt.Sprintf(format, args...)}, event)
	}
}

// recover opens the engine on image and returns the commits that it holds.
func (r *run) recover(image *fault.FS) (records []Record, err error) {
	engine, err := r.cfg.Open(r.ctx, image)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, engine.Close()) }()
	return engine.Recovered(r.ctx)
}

// payload returns the bytes of the commit id in the run of runKey.
func payload(runKey, id uint64) []byte {
	rng := rand.New(rand.NewPCG(runKey, id))
	out := make([]byte, 1+rng.IntN(1400))
	for i := range out {
		out[i] = byte(rng.Uint32())
	}
	return out
}
