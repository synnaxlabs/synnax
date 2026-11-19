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
	result          Result
	// brokenAt is the event of broken, and brokenFaults holds the read faults of the
	// recovery that broke it.
	brokenAt     int
	brokenFaults []fault.Fault
	// mu guards every field below plan.
	mu sync.Mutex
}

// Run runs the workload of cfg on an engine. After each Stride calls that change the
// disk it takes a crash image. It opens the engine on each image, and again after a
// crash in that recovery, and applies the oracle to what the engine recovers, with the
// acks that the engine gave while the disk held that state. It returns the counts of
// what it checked, and a *Violation for the first rule that the engine breaks, an error
// that matches validate.ErrValidation for a bad cfg, or nil when the engine holds.
func Run(ctx context.Context, cfg Config) (Result, error) {
	if err := cfg.validate(); err != nil {
		return Result{}, err
	}
	r := &run{ctx: ctx, cfg: cfg, plan: newPlan(cfg)}
	disk, err := fault.New(r.plan.workload)
	if err != nil {
		return Result{}, err
	}
	r.disk = disk
	r.counter = &counter{
		files:  map[fault.ID]*syncs{},
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
		return r.result, errors.Wrap(r.err, "crash: harness")
	}
	if r.broken == nil {
		return r.result, nil
	}
	return r.result, &Violation{
		Rule:       r.broken.rule,
		Detail:     r.broken.detail,
		RunKey:     cfg.RunKey,
		Profile:    cfg.Profile,
		Event:      r.brokenAt,
		ReadFaults: len(r.brokenFaults),
		Clustered:  largest(r.brokenFaults),
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
		r.fail(b, 0, nil)
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
		r.fail(&broken{R0, fmt.Sprintf(format, args...)}, 0, nil)
	}
}

func (r *run) stopped() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.broken != nil || r.err != nil
}

func (r *run) fail(b *broken, event int, faults []fault.Fault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.broken == nil {
		r.broken, r.brokenAt, r.brokenFaults = b, event, faults
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
	r.abort(err)
	return nil
}

// check opens the engine on the image of s, and applies the oracle to what it recovers.
// Then it crashes the disk of that recovery and opens the engine again, which finds a
// recovery that harms the disk. The second recovery must keep what the first held, and
// can refuse only after read faults.
func (r *run) check(s *snapshot, attempts []attempt) {
	if s == nil || r.stopped() {
		return
	}
	first, faults, audit, err := r.recover(s.image)
	rules, _ := r.underFaults(s.rules, faults)
	if err != nil {
		r.refused(err, rules, s.event, faults, "the engine did not recover")
		return
	}
	if b, lost := checkRecovered(
		attempts,
		first,
		rules,
	); !r.held(
		b,
		lost,
		s.event,
		faults,
	) || !r.audited(audit, s.event, faults) {
		return
	}
	again, err := s.image.Crash(r.plan.crash(s.event))
	if err != nil {
		r.abort(err)
		return
	}
	// A loss that the faults of the first recovery freed stays after the crash.
	second, faults, audit, err := r.recover(again)
	rules.refusalAllowed = false
	rules, faulted := r.underFaults(rules, faults)
	if err != nil {
		r.refused(
			err,
			rules,
			s.event,
			faults,
			"the engine did not recover after a crash "+
				"in recovery",
		)
		return
	}
	b, lost := checkSecond(attempts, first, second, rules, faulted)
	if r.held(b, lost, s.event, faults) {
		r.audited(audit, s.event, faults)
	}
}

// audited applies R5 to the error of an Audit, and reports if it held.
func (r *run) audited(audit error, event int, faults []fault.Fault) bool {
	if audit == nil {
		return true
	}
	r.fail(
		&broken{rule: R5, detail: "the audit failed: " + audit.Error()},
		event,
		faults,
	)
	return false
}

// underFaults returns base for a recovery on a disk that gave the read faults, and if
// they relaxed it. Under RepeatedReadFaults, a cluster of two or more faults lets the
// engine refuse, return unreadable records, and lose the commits from the first Sync
// that changed the earliest block that a fault of the cluster hit.
func (r *run) underFaults(base rules, faults []fault.Fault) (rules, bool) {
	if r.cfg.Profile != RepeatedReadFaults {
		return base, false
	}
	relaxed := false
	for _, cluster := range clusters(faults) {
		if len(cluster) < 2 {
			continue
		}
		relaxed = true
		for _, f := range cluster {
			if file, ok := r.counter.files[f.File]; ok {
				if event, ok := file.first[f.Block]; ok {
					base.exemptFrom = min(base.exemptFrom, event)
				}
			}
		}
	}
	if relaxed {
		base.refusalAllowed, base.unreadableAllowed = true, true
	}
	return base, relaxed
}

// clusters groups the read faults whose reads overlap in one file, also through other
// faults, in the order of their first faults. A fault with a good retry is alone in its
// cluster.
func clusters(faults []fault.Fault) [][]fault.Fault {
	root := make([]int, len(faults))
	for i := range root {
		root[i] = i
	}
	find := func(i int) int {
		for root[i] != i {
			i = root[i]
		}
		return i
	}
	for i, a := range faults {
		for j, b := range faults[:i] {
			if a.File == b.File && a.Read.From < b.Read.To && b.Read.From < a.Read.To {
				root[find(i)] = find(j)
			}
		}
	}
	var (
		groups [][]fault.Fault
		group  = map[int]int{}
	)
	for i, f := range faults {
		g, ok := group[find(i)]
		if !ok {
			g = len(groups)
			group[find(i)] = g
			groups = append(groups, nil)
		}
		groups[g] = append(groups[g], f)
	}
	return groups
}

// largest returns the count of faults in the largest cluster.
func largest(faults []fault.Fault) int {
	most := 0
	for _, cluster := range clusters(faults) {
		most = max(most, len(cluster))
	}
	return most
}

// refused counts the refusal err of a recovery that the rules allow, or breaks R4.
func (r *run) refused(
	err error,
	rules rules,
	event int,
	faults []fault.Fault,
	what string,
) {
	if rules.refusalAllowed && r.cfg.Corrupt != nil && r.cfg.Corrupt(err) {
		r.result.Refusals++
		return
	}
	r.fail(&broken{R4, fmt.Sprintf("%s: %v", what, err)}, event, faults)
}

// held breaks the rule of b, or counts a recovery that lost a commit that the rules
// free. It reports if the recovery held.
func (r *run) held(b *broken, lost bool, event int, faults []fault.Fault) bool {
	if b != nil {
		r.fail(b, event, faults)
		return false
	}
	if lost {
		r.result.Losses++
	}
	return true
}

// abort ends the run with err, an error of the harness.
func (r *run) abort(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		r.err = err
	}
}

// recover opens the engine on image and returns the commits that it holds, the read
// faults that the disk gave, and the error of its Audit.
func (r *run) recover(
	image *fault.FS,
) (records []Record, faults []fault.Fault, audit, err error) {
	r.result.Recoveries++
	defer func() {
		faults = image.ReadFaults()
		r.result.ReadFaults += len(faults)
	}()
	engine, err := r.cfg.Open(r.ctx, image)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { err = errors.Combine(err, engine.Close()) }()
	if records, err = engine.Recovered(r.ctx); err != nil {
		return nil, nil, nil, err
	}
	if auditor, ok := engine.(Auditor); ok {
		audit = auditor.Audit(r.ctx)
	}
	return records, nil, audit, nil
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
