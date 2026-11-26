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
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
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
	// armed is true once the engine of the workload opened and Run armed the faults of
	// its disk.
	armed atomic.Bool
	// mu guards every field below plan.
	mu sync.Mutex
}

// Run runs the workload of cfg on an engine. After each Stride calls that change the
// disk it takes a crash image. It opens the engine on each image, and again after a
// crash in that recovery, and applies the oracle to what the engine recovers, with the
// acks that the engine gave while the disk held that state. The faults of the workload
// disk start once the engine opens. It returns the counts of what it checked, and a
// *Violation for the first rule that the engine breaks, an error that matches
// validate.ErrValidation for a bad cfg, an error when no fault of the profile fired
// after the open, so the run checked nothing that the profile names, or nil when the
// engine holds.
func Run(ctx context.Context, cfg Config) (Result, error) {
	if err := cfg.validate(); err != nil {
		return Result{}, err
	}
	r := &run{ctx: ctx, cfg: cfg, plan: newPlan(cfg)}
	disk, err := fault.New(fault.Config{RunKey: r.plan.workload.RunKey})
	if err != nil {
		return Result{}, err
	}
	r.disk = disk
	r.counter = &counter{
		files:  map[fault.ID]*syncs{},
		before: r.beforeEvent,
		after:  r.afterEvent,
		writebackRefused: cfg.Profile == OldCorruption ||
			cfg.Profile == NewCorruption || cfg.Profile == RepeatedReadFaults,
	}
	r.pending = r.snapshot(0)
	if engine, err := cfg.Open(ctx, &countedFS{inner: disk, c: r.counter}); err != nil {
		r.worksf(r.faulted(), "open on an empty disk: %v", err)
	} else {
		if err = disk.Arm(r.plan.workload); err != nil {
			r.abort(err)
		}
		r.fill(0)
		r.armed.Store(true)
		r.work(engine)
		if err = engine.Close(); err != nil {
			r.worksf(r.faulted(), "close: %v", err)
		}
	}
	r.finish()
	if r.err != nil {
		return r.result, errors.Wrap(r.err, "crash: harness")
	}
	if r.broken == nil {
		if unfired := r.unfired(); r.armed.Load() && len(unfired) > 0 {
			return r.result, errors.Wrapf(
				errNoFault,
				"profile %d: no %s",
				cfg.Profile,
				strings.Join(unfired, ", no "),
			)
		}
		return r.result, nil
	}
	if cfg.Exhaustive {
		// The live check of the workload can find its break after the image checks.
		slices.SortStableFunc(r.result.Violations, func(a, b Violation) int {
			return a.Event - b.Event
		})
		first := r.result.Violations[0]
		return r.result, &first
	}
	first := r.violation(r.broken, r.brokenAt, r.brokenFaults)
	return r.result, &first
}

// violation returns the Violation of b, at the event with the read faults.
func (r *run) violation(b *broken, event int, faults []fault.Fault) Violation {
	return Violation{
		Rule:       b.rule,
		Detail:     b.detail,
		RunKey:     r.cfg.RunKey,
		Profile:    r.cfg.Profile,
		Event:      event,
		ReadFaults: len(faults),
		Clustered:  largest(faults),
	}
}

// work runs the commits of the workload on engine, then reads them back.
func (r *run) work(engine Engine) {
	acks := wants(r.cfg.RunKey, r.cfg.Commits)
	for id := uint64(1); id <= uint64(r.cfg.Commits) && !r.stopped(); id++ {
		rec, want := Record{ID: id, Payload: payload(r.cfg.RunKey, id)}, acks[id-1]
		r.mu.Lock()
		r.attempts = append(r.attempts, attempt{record: rec})
		r.mu.Unlock()
		ack, err := engine.Commit(r.ctx, rec, want)
		r.fill(int(id))
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

// finish checks the image of the last state of the workload, with one forced read fault
// when the profile has read faults and no recovery got one, and counts the faults of
// the workload.
func (r *run) finish() {
	r.counter.mu.Lock()
	events := r.counter.events
	r.counter.mu.Unlock()
	if r.pending == nil || r.pending.event != events {
		r.pending = r.snapshot(events)
	}
	r.check(r.pending, r.attempts)
	if r.armed.Load() && r.needsReadFault() {
		r.check(r.forcedRead(events), r.attempts)
	}
	r.counter.mu.Lock()
	defer r.counter.mu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.result.SyncErrors, r.result.FullDisk = r.counter.syncErrors, r.counter.fullDisk
}

// fill fills the disk of the workload when the plan fills it after commit id.
func (r *run) fill(id int) {
	if r.plan.fills && r.plan.fillAfter == id {
		r.disk.Fill(r.plan.room)
	}
}

// needsReadFault reports if the profile is one of read faults and no recovery got one.
func (r *run) needsReadFault() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	read := r.cfg.Profile == OneTimeReadFaults || r.cfg.Profile == RepeatedReadFaults
	return read && r.result.ReadFaults == 0
}

// forcedRead takes the crash image behind the event with a read fault on the first
// block read of its recovery, so that a run of a read fault profile gets one.
func (r *run) forcedRead(event int) *snapshot {
	cfg := r.plan.crash(event)
	cfg.Image.ReadFaultAt = 1
	image, err := r.disk.Crash(cfg)
	if err != nil {
		r.abort(err)
		return nil
	}
	return &snapshot{image: image, rules: strictRules, event: event}
}

// unfired returns the kinds of fault of the profile that never fired after the engine
// of the workload opened. Mixed needs one fault of the workload, since an engine that
// stops on its first fault never meets a second kind.
func (r *run) unfired() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, fired := r.result, set.New[string]()
	add := func(kind string, ok bool) {
		if ok {
			fired.Add(kind)
		}
	}
	add("unsynced image", res.Unsynced > 0)
	add("sync error", res.SyncErrors > 0)
	add("full disk", res.FullDisk > 0)
	add("sync error or full disk", res.SyncErrors+res.FullDisk > 0)
	add("read fault", res.ReadFaults > 0)
	add("spoiled image", res.Spoiled > 0)
	var needs []string
	switch r.cfg.Profile {
	case Strict:
	case Partial:
		needs = []string{"unsynced image"}
	case SyncErrors:
		needs = []string{"sync error"}
	case FullDisk:
		needs = []string{"full disk"}
	case Mixed:
		needs = []string{"sync error or full disk"}
	case OneTimeReadFaults, RepeatedReadFaults:
		needs = []string{"read fault"}
	case OldCorruption, NewCorruption:
		needs = []string{"spoiled image"}
	}
	return slices.DeleteFunc(needs, fired.Contains)
}

// faulted reports if the disk of the workload failed a call.
func (r *run) faulted() bool {
	r.counter.mu.Lock()
	defer r.counter.mu.Unlock()
	return r.counter.syncErrors+r.counter.fullDisk > 0
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
	return (r.broken != nil && !r.cfg.Exhaustive) || r.err != nil
}

func (r *run) fail(b *broken, event int, faults []fault.Fault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.broken == nil {
		r.broken, r.brokenAt, r.brokenFaults = b, event, faults
	}
	if r.cfg.Exhaustive {
		r.result.Violations = append(r.result.Violations, r.violation(b, event, faults))
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
	unsynced := r.disk.Unsynced()
	image, err := r.disk.Crash(r.plan.crash(event))
	if err == nil {
		var rules rules
		if rules, err = r.plan.spoil(image, r.counter, event); err == nil {
			if r.armed.Load() {
				r.mu.Lock()
				r.result.Unsynced += btoi(unsynced)
				r.result.Spoiled += btoi(rules != strictRules)
				r.mu.Unlock()
			}
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
	defer func() {
		if err = errors.Combine(err, engine.Close()); err != nil {
			records, audit = nil, nil
		}
	}()
	if records, err = engine.Recovered(r.ctx); err != nil {
		return nil, nil, nil, err
	}
	if a, ok := engine.(auditor); ok {
		audit = a.Audit(r.ctx)
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

// wants returns the ack that each commit of a workload asks for, in commit order.
func wants(runKey uint64, commits int) []Ack {
	rng := rand.New(rand.NewPCG(runKey, 0))
	out := make([]Ack, commits)
	for i := range out {
		out[i] = Durable
		if rng.IntN(5) < 2 {
			out[i] = Visible
		}
	}
	return out
}

// payloadSize returns the length of payload(runKey, id).
func payloadSize(runKey, id uint64) int {
	return 1 + rand.New(rand.NewPCG(runKey, id)).IntN(1400)
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
