// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// clockEntry is the Cesium writer of one live clock, which the session that holds the
// clock's control writes through. Each transfer of the clock hands the writer to the
// new holder as one event, under mu. The new holder keeps the uncommitted samples of
// the clock and of the channels that both sessions write. The other uncommitted samples
// of the old holder wait on its member until the clock holds their stamps. A transfer
// takes mu to write, and a call of a session takes it to read, so a session that is
// refused does not wait for the holder.
//
// Lock order: mu, then a Cesium claim, then a gate region. The skip predicate of a
// session never takes mu, and no path takes mu while it holds a claim or a region lock.
type clockEntry struct {
	mu    sync.RWMutex
	clock cesium.ChannelKey
	ctl   *controlled
	// members holds the sessions that write the clock, by control subject key.
	members map[string]*clockMember
	holder  *clockMember
	// owner is the session that w was opened or handed to. It is the holder, or the
	// last holder while no session leads the clock.
	owner *clockMember
	w     *cesium.Writer
	// keys are the keys that w claims: the owner's keys, less the keys that the own
	// writer of another session claims.
	keys []cesium.ChannelKey
	// owns holds, for each key that the own writer of a persist and stream session
	// claims, that session's handle. A key holds one live claim: w's or one in owns.
	owns map[cesium.ChannelKey]*clockWriter
	// end is the end of the last commit of a writer of the entry, as of the last
	// transfer, which folds it in from the old holder.
	end telem.TimeStamp
	// handing is true when the holder left in the event that follow ends.
	handing bool
	ins     alamos.Instrumentation
}

// clockMember is one session that writes the clock of an entry.
type clockMember struct {
	subject string
	// keys are the keys of the session on the clock, and start the start of its writer.
	keys  []cesium.ChannelKey
	start telem.TimeStamp
	// end is the end of the last commit of the session, or start, which a commit
	// without the clock returns, as a writer with nothing to commit does. last is the
	// end of its last commit through the entry's writer, or zero.
	end, last telem.TimeStamp
	// pending is true when the session wrote samples since its last commit, and dropped
	// is true when a handoff took them. The next commit then reports a refusal.
	pending, dropped bool
	// parked holds the uncommitted samples that a handoff of the clock or a move of a
	// key took out of a writer of the session.
	parked []parkedRun
	// err is the error of the writer that the entry opened or closed for the member,
	// which its next call returns.
	err error
}

// parkedRun is the uncommitted samples of keys that a handoff or a move took out of a
// writer of a session.
type parkedRun struct {
	pending *cesium.Pending
	keys    []cesium.ChannelKey
	// moved is true for the samples of a key that moved to another writer.
	moved bool
}

// park adds the samples of p, which are of keys, to m.
func (m *clockMember) park(p *cesium.Pending, keys []cesium.ChannelKey) {
	if p != nil {
		m.parked = append(m.parked, parkedRun{pending: p, keys: keys})
	}
}

// keep adds the samples of p, which are of k after k moved to another writer, to m.
// They commit once the clock holds their stamps, whether or not the session still
// controls k.
func (m *clockMember) keep(p *cesium.Pending, k cesium.ChannelKey) {
	if p != nil {
		m.parked = append(m.parked, parkedRun{
			pending: p,
			keys:    []cesium.ChannelKey{k},
			moved:   true,
		})
	}
}

// adopt removes and returns the first run parked on m whose keys all lie in keys, or
// nil.
func (m *clockMember) adopt(keys []cesium.ChannelKey) *cesium.Pending {
	for i, r := range m.parked {
		if subset(r.keys, keys) {
			m.parked = slices.Delete(m.parked, i, i+1)
			return r.pending
		}
	}
	return nil
}

// dropParked drops the samples parked on m.
func (m *clockMember) dropParked() error {
	var err error
	for _, r := range m.parked {
		err = errors.Join(err, r.pending.Close())
	}
	m.parked = nil
	return err
}

// subset reports whether each of a lies in b.
func subset(a, b []cesium.ChannelKey) bool {
	for _, k := range a {
		if !slices.Contains(b, k) {
			return false
		}
	}
	return true
}

// without returns keys without k.
func without(keys []cesium.ChannelKey, k cesium.ChannelKey) []cesium.ChannelKey {
	return slices.DeleteFunc(slices.Clone(keys), func(o cesium.ChannelKey) bool {
		return o == k
	})
}

// unclaimed returns the keys of keys that no writer of e claims. The caller holds e.mu.
func (e *clockEntry) unclaimed(keys []cesium.ChannelKey) []cesium.ChannelKey {
	return slices.DeleteFunc(e.free(keys), func(k cesium.ChannelKey) bool {
		return slices.Contains(e.keys, k)
	})
}

// leave closes the own writer of c, which releases its claims, and drops the samples
// parked on its session. The caller holds the entry's lock to write.
func (c *clockWriter) leave() error {
	for _, k := range c.claimed {
		if c.entry.owns[k] == c {
			delete(c.entry.owns, k)
		}
	}
	err := c.member.dropParked()
	if c.own != nil {
		err = errors.Join(err, c.own.Close())
	}
	c.own, c.claimed = nil, nil
	return err
}

// free returns the keys of keys that the own writer of no session claims.
func (e *clockEntry) free(keys []cesium.ChannelKey) []cesium.ChannelKey {
	return slices.DeleteFunc(slices.Clone(keys), func(k cesium.ChannelKey) bool {
		_, owned := e.owns[k]
		return owned
	})
}

// follow moves the writer of e to the session that leads the clock's control, when it
// is not the holder. While no session leads, the writer keeps the samples of its owner.
// The caller holds e.mu.
func (e *clockEntry) follow(ctx context.Context, ts *cesium.DB) {
	var leader *clockMember
	if st := e.ctl.c.LeadingState(); st != nil {
		leader = e.members[st.Subject.Key]
	}
	if leader != nil && leader == e.holder {
		return
	}
	handoff := e.holder != nil || e.handing
	e.handing = false
	if e.holder != nil {
		e.end = max(e.end, e.holder.last)
		e.holder = nil
	}
	if leader == nil {
		if e.w != nil && !e.member(e.owner) {
			e.warn(e.w.Close(), "failed to close the writer of a clock with no leader")
			e.w, e.owner, e.keys = nil, nil, nil
		}
		return
	}
	e.holder = leader
	if leader.err != nil || (e.w != nil && e.owner == leader) {
		return
	}
	// A writer opens after the session's own last commit. A handoff also starts it at
	// the old writer's last commit end, and a first open at the session's own start,
	// which refuses stored data there.
	start := leader.end
	if handoff {
		start = max(e.end, start)
	}
	cfg := cesium.WriterConfig{
		Channels: e.free(leader.keys),
		Clock:    e.clock,
		Start:    start,
	}
	if e.w != nil {
		e.handTo(ctx, ts, leader, cfg)
		return
	}
	e.open(ctx, ts, leader, cfg)
}

// open opens the writer of e for leader with cfg. The caller holds e.mu.
func (e *clockEntry) open(
	ctx context.Context,
	ts *cesium.DB,
	leader *clockMember,
	cfg cesium.WriterConfig,
) {
	e.w, leader.err = ts.OpenWriter(ctx, cfg)
	e.owner, e.keys = leader, nil
	if leader.err == nil {
		e.keys = cfg.Channels
	}
}

// handTo hands the writer of e to leader through a writer of cfg, with the samples that
// a handoff parked on leader. A handoff that fails drops the samples it held and logs a
// warning, and leader then opens a writer of its own. The caller holds e.mu.
func (e *clockEntry) handTo(
	ctx context.Context,
	ts *cesium.DB,
	leader *clockMember,
	cfg cesium.WriterConfig,
) {
	owner, adopt, before := e.owner, leader.adopt(cfg.Channels), e.keys
	w, parked, err := ts.Handoff(ctx, e.w, cfg, adopt, cesium.NewRun)
	e.w, e.owner, e.keys = w, leader, cfg.Channels
	if err != nil {
		owner.dropped, owner.pending = owner.pending, false
		leader.dropped = leader.dropped || adopt != nil
		e.ins.L.Warn(
			"dropped the uncommitted samples that the next holder of a clock could not take",
			zap.Uint32("clock", e.clock),
			zap.Error(err),
		)
		e.open(ctx, ts, leader, cfg)
		return
	}
	if parked == nil {
		return
	}
	if e.member(owner) {
		owner.park(
			parked,
			slices.DeleteFunc(slices.Clone(before), func(k cesium.ChannelKey) bool {
				return slices.Contains(cfg.Channels, k)
			}),
		)
		return
	}
	e.warn(parked.Close(), "failed to drop the samples of a session that closed")
}

// member reports whether m is a member of e.
func (e *clockEntry) member(m *clockMember) bool {
	return m != nil && e.members[m.subject] == m
}

// warn logs err with msg when err is not nil.
func (e *clockEntry) warn(err error, msg string) {
	if err != nil {
		e.ins.L.Warn(msg, zap.Uint32("clock", e.clock), zap.Error(err))
	}
}

// leave removes m from e and drops the samples parked on it. When m holds the clock,
// leave makes the commits of its writer durable and keeps the writer for follow to hand
// to the next holder. The caller holds e.mu.
func (e *clockEntry) leave(ctx context.Context, m *clockMember) error {
	delete(e.members, m.subject)
	err := m.dropParked()
	if e.holder != m {
		return err
	}
	e.end = max(e.end, m.last)
	e.holder, e.handing = nil, true
	if e.w != nil {
		err = errors.Join(err, e.w.Sync(ctx))
	}
	return err
}

// clockWriter is the handle of a session on the Cesium writer of one clock: its own, or
// the writer of the clock's entry, which only the holder of the clock writes through. A
// session that holds a gate on the clock and keeps its own writer still follows the
// entry, since its transfers move the shared writer.
type clockWriter struct {
	ts     *cesium.DB
	clock  cesium.ChannelKey
	own    *cesium.Writer
	entry  *clockEntry
	member *clockMember
	// shares is true when the session writes through the entry's writer.
	shares bool
	// claims is true for a persist and stream session that writes the clock's data and
	// not the clock: its own writer takes part in the entry's one live claim per key,
	// and claimed are the keys that it claims.
	claims  bool
	claimed []cesium.ChannelKey
}

// held returns the writer that the session may use, or nil when another session holds
// the clock. The caller holds the entry's lock for a shared writer.
func (c *clockWriter) held() (*cesium.Writer, error) {
	if c.own != nil {
		return c.own, nil
	}
	if c.member.err != nil {
		return nil, c.member.err
	}
	if c.entry.holder != c.member {
		return nil, nil
	}
	return c.entry.w, nil
}

// lock locks the entry of the clock to read. A session changes only its own member and
// writer under it, and only the holder writes through the entry's writer. A move of a
// key takes the lock to write.
func (c *clockWriter) lock() {
	if c.shares || c.claims {
		c.entry.mu.RLock()
	}
}

func (c *clockWriter) unlock() {
	if c.shares || c.claims {
		c.entry.mu.RUnlock()
	}
}

// Write writes fr through the writer of the clock, and leaves out the keys of the clock
// when the session does not hold it. It first moves into the writer the live claim of
// each key that the session controls and writes in fr, and that another writer holds.
func (c *clockWriter) Write(
	ctx context.Context,
	fr cesium.Frame,
	skip func(cesium.ChannelKey) bool,
) (cesium.Frame, bool, error) {
	c.lock()
	if missing := c.unclaimed(fr, skip); len(missing) > 0 {
		c.unlock()
		c.entry.mu.Lock()
		defer c.entry.mu.Unlock()
		if missing = c.unclaimed(fr, skip); len(missing) > 0 {
			if err := c.take(ctx, missing); err != nil {
				return fr, false, err
			}
		}
		return c.write(ctx, fr, skip)
	}
	defer c.unlock()
	return c.write(ctx, fr, skip)
}

// write writes fr through the writer that the session may use. The caller holds the
// entry's lock.
func (c *clockWriter) write(
	ctx context.Context,
	fr cesium.Frame,
	skip func(cesium.ChannelKey) bool,
) (cesium.Frame, bool, error) {
	w, err := c.held()
	if err != nil {
		return fr, false, err
	}
	if w == nil {
		return refuse(fr, c.member.keys)
	}
	lost := c.lost(fr, skip)
	if len(lost) > 0 {
		fr = fr.ExcludeKeys(lost)
	}
	written, skipped, err := w.Write(ctx, fr, skip)
	if err == nil && c.own == nil && holds(written, c.member.keys) {
		c.member.pending = true
	}
	return written, skipped || len(lost) > 0, err
}

// lost returns the keys of the session in fr that skip refuses and that the writer it
// uses lacks, since that writer does not refuse them. The caller holds the entry's
// lock.
func (c *clockWriter) lost(
	fr cesium.Frame,
	skip func(cesium.ChannelKey) bool,
) []cesium.ChannelKey {
	if skip == nil || (!c.shares && !c.claims) {
		return nil
	}
	held := c.claimed
	if c.shares {
		held = c.entry.keys
	}
	var lost []cesium.ChannelKey
	for i, k := range fr.RawKeys() {
		if !fr.ShouldExcludeRaw(i) && skip(k) && slices.Contains(c.member.keys, k) &&
			!slices.Contains(held, k) && !slices.Contains(lost, k) {
			lost = append(lost, k)
		}
	}
	return lost
}

// unclaimed returns the keys of the session in fr that it controls and whose live claim
// the writer it uses lacks. A session that shares the clock's writer and does not hold
// it writes nothing, so it moves no key. The caller holds the entry's lock.
func (c *clockWriter) unclaimed(
	fr cesium.Frame,
	skip func(cesium.ChannelKey) bool,
) []cesium.ChannelKey {
	e := c.entry
	if (!c.shares && !c.claims) || c.member.err != nil {
		return nil
	}
	claimed := c.claimed
	if c.shares {
		if e.holder != c.member || e.w == nil {
			return nil
		}
		claimed = e.keys
	}
	var missing []cesium.ChannelKey
	for i, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(i) || !slices.Contains(c.member.keys, k) ||
			slices.Contains(claimed, k) || slices.Contains(missing, k) ||
			(skip != nil && skip(k)) {
			continue
		}
		missing = append(missing, k)
	}
	return missing
}

// take moves the live claim of each of keys into the writer of c, with the samples that
// an earlier move parked on the session for them. The caller holds the entry's lock to
// write.
func (c *clockWriter) take(ctx context.Context, keys []cesium.ChannelKey) error {
	e := c.entry
	for _, k := range keys {
		if err := e.release(ctx, c.ts, k); err != nil {
			return err
		}
	}
	if c.shares {
		next := append(slices.Clone(e.keys), keys...)
		start, err := e.start(c.ts, next)
		if err != nil {
			return err
		}
		cfg := cesium.WriterConfig{Channels: next, Clock: e.clock, Start: start}
		w, parked, err := c.ts.Handoff(
			ctx,
			e.w,
			cfg,
			c.member.adopt(next),
			cesium.SameRun,
		)
		e.w, e.keys = w, next
		c.member.park(parked, nil)
		return err
	}
	// A writer of data against stored stamps keeps its start, since its samples go on
	// the stored stamps from there. The stamps that the clock's holder has written must
	// be stored before that writer writes on them.
	if err := e.commitStamps(ctx); err != nil {
		return err
	}
	next := append(slices.Clone(c.claimed), keys...)
	cfg := cesium.WriterConfig{Channels: next, Clock: c.clock, Start: c.member.start}
	opened := c.own == nil
	if opened {
		var err error
		if c.own, err = c.ts.OpenWriter(ctx, cfg); err != nil {
			return err
		}
	}
	if adopt := c.member.adopt(next); !opened || adopt != nil {
		var (
			parked *cesium.Pending
			err    error
		)
		c.own, parked, err = c.ts.Handoff(ctx, c.own, cfg, adopt, cesium.NewRun)
		c.member.park(parked, nil)
		if err != nil {
			return err
		}
	}
	c.claim(keys, next)
	return nil
}

// claim records that the own writer of c claims next, which adds keys. The caller holds
// the entry's lock to write.
func (c *clockWriter) claim(keys, next []cesium.ChannelKey) {
	c.claimed = next
	for _, k := range keys {
		c.entry.owns[k] = c
	}
}

// commitStamps commits the uncommitted stamps of the clock's writer and makes them
// durable when it holds no uncommitted data, and commits nothing when it does. The
// caller holds e.mu to write.
func (e *clockEntry) commitStamps(ctx context.Context) error {
	if e.w == nil || e.owner == nil {
		return nil
	}
	end, skipped, err := e.w.Commit(ctx, func(k cesium.ChannelKey) bool {
		return k != e.clock
	})
	if err != nil || skipped {
		return err
	}
	e.owner.end, e.owner.last = end, end
	return e.w.Sync(ctx)
}

// release takes the live claim of k out of the writer that holds it, and parks the
// uncommitted samples of k on that writer's session. The caller holds e.mu to write.
func (e *clockEntry) release(
	ctx context.Context,
	ts *cesium.DB,
	k cesium.ChannelKey,
) error {
	if x, ok := e.owns[k]; ok {
		delete(e.owns, k)
		rest := without(x.claimed, k)
		cfg := cesium.WriterConfig{
			Channels: rest,
			Clock:    e.clock,
			Start:    x.member.start,
		}
		w, parked, err := ts.Handoff(ctx, x.own, cfg, nil, cesium.NewRun)
		x.own, x.claimed = w, rest
		x.member.keep(parked, k)
		return err
	}
	if e.w == nil || !slices.Contains(e.keys, k) {
		return nil
	}
	rest := without(e.keys, k)
	start, err := e.start(ts, rest)
	if err != nil {
		return err
	}
	cfg := cesium.WriterConfig{Channels: rest, Clock: e.clock, Start: start}
	w, parked, err := ts.Handoff(ctx, e.w, cfg, nil, cesium.SameRun)
	e.w, e.keys = w, rest
	e.owner.keep(parked, k)
	return err
}

// start returns the start of a writer of the entry over keys: the later of the owner's
// start and the end of the samples that keys have stored. The caller holds e.mu.
func (e *clockEntry) start(
	ts *cesium.DB,
	keys []cesium.ChannelKey,
) (telem.TimeStamp, error) {
	stored, err := committedEnd(ts, keys)
	if e.owner == nil {
		return stored, err
	}
	return max(e.owner.start, stored), err
}

// committedEnd returns the end of the samples that keys have stored, or zero when they
// hold none.
func committedEnd(ts *cesium.DB, keys []cesium.ChannelKey) (telem.TimeStamp, error) {
	it, err := ts.OpenIterator(cesium.IteratorConfig{
		Channels: keys,
		Bounds:   telem.TimeRangeMax,
	})
	if err != nil {
		return 0, err
	}
	var end telem.TimeStamp
	if it.SeekLast() && it.Prev(cesium.AutoSpan) {
		for _, s := range it.Value().RawSeries() {
			end = max(end, s.TimeRange.End)
		}
	}
	return end, errors.Join(it.Error(), it.Close())
}

// holds reports whether fr holds a series of one of keys.
func holds(fr cesium.Frame, keys []cesium.ChannelKey) bool {
	for i, k := range fr.RawKeys() {
		if !fr.ShouldExcludeRaw(i) && slices.Contains(keys, k) {
			return true
		}
	}
	return false
}

// refuse returns fr without keys, and true when fr held a series of one of them.
func refuse(fr cesium.Frame, keys []cesium.ChannelKey) (cesium.Frame, bool, error) {
	if !holds(fr, keys) {
		return fr, false, nil
	}
	return fr.ExcludeKeys(keys), true, nil
}

// Commit commits the writer of the clock. It commits nothing and returns a refusal once
// after a handoff dropped samples of the session. A session that does not hold the
// clock commits nothing and returns the end of its last commit.
func (c *clockWriter) Commit(
	ctx context.Context,
	skip func(cesium.ChannelKey) bool,
) (telem.TimeStamp, bool, error) {
	c.lock()
	defer c.unlock()
	w, err := c.held()
	if err != nil {
		return 0, false, err
	}
	if c.own == nil && c.member.dropped {
		c.member.dropped = false
		return 0, true, nil
	}
	if w == nil {
		return c.commitParked(ctx, skip)
	}
	end, skipped, err := w.Commit(ctx, skip)
	if err == nil && !skipped && c.own == nil {
		c.member.end, c.member.last = end, end
		c.member.pending, c.member.dropped = false, false
	}
	// The commit stored the stamps of the samples parked on the session, so they commit
	// after it.
	if err == nil && len(c.member.parked) > 0 {
		dropped, _, runErr := c.commitRuns(ctx, skip)
		if runErr != nil {
			return 0, false, runErr
		}
		skipped = skipped || dropped
	}
	return end, skipped, err
}

// commitParked commits the samples that a handoff or a move parked on the session, once
// the clock holds their stamps, and drops them when the session lacks one of their
// channels. It returns true when the session has samples that it cannot commit. The
// caller holds the entry's lock.
func (c *clockWriter) commitParked(
	ctx context.Context,
	skip func(cesium.ChannelKey) bool,
) (telem.TimeStamp, bool, error) {
	m := c.member
	if len(m.parked) == 0 {
		if m.pending {
			return 0, true, nil
		}
		return m.end, false, nil
	}
	dropped, waiting, err := c.commitRuns(ctx, skip)
	if err != nil {
		return 0, false, err
	}
	if dropped || waiting {
		return 0, true, nil
	}
	m.pending = false
	return m.end, false, nil
}

// commitRuns commits each run parked on the session once the clock holds its stamps,
// and drops each run that a handoff parked for a channel that the session lacks. It
// returns whether it dropped a run, and whether a run still waits for its stamps. The
// caller holds the entry's lock.
func (c *clockWriter) commitRuns(
	ctx context.Context,
	skip func(cesium.ChannelKey) bool,
) (dropped, waiting bool, err error) {
	m := c.member
	kept := m.parked[:0]
	for _, r := range m.parked {
		if !r.moved && c.lacksAny(r.keys, skip) {
			err, dropped = errors.Join(err, r.pending.Close()), true
			continue
		}
		end, held, commitErr := r.pending.Commit(ctx)
		switch {
		case commitErr != nil:
			err = errors.Join(err, commitErr, r.pending.Close())
		case !held:
			kept, waiting = append(kept, r), true
		default:
			m.end = max(m.end, end)
		}
	}
	m.parked = kept
	return dropped, waiting, err
}

// lacksAny reports whether skip refuses one of keys other than the clock.
func (c *clockWriter) lacksAny(
	keys []cesium.ChannelKey,
	skip func(cesium.ChannelKey) bool,
) bool {
	return skip != nil && slices.ContainsFunc(keys, func(k cesium.ChannelKey) bool {
		return k != c.clock && skip(k)
	})
}

// writesClock reports whether the session writes the clock of c.
func (c *clockWriter) writesClock() bool { return slices.Contains(c.member.keys, c.clock) }

// Sync syncs the writer of the clock when the session holds it.
func (c *clockWriter) Sync(ctx context.Context) error {
	c.lock()
	defer c.unlock()
	w, err := c.held()
	if err != nil || w == nil {
		return err
	}
	return w.Sync(ctx)
}

// EndStretch ends the stretch of the writer of the clock when the session holds it.
func (c *clockWriter) EndStretch(ctx context.Context) error {
	c.lock()
	defer c.unlock()
	w, err := c.held()
	if err != nil || w == nil {
		return err
	}
	return w.EndStretch(ctx)
}

// lockEntries locks the entries of ws in rising order of clock, and returns the
// function that unlocks them.
func lockEntries(ws []*clockWriter) func() {
	entries := make([]*clockEntry, 0, len(ws))
	for _, c := range ws {
		if c.entry != nil {
			entries = append(entries, c.entry)
		}
	}
	slices.SortFunc(entries, func(a, b *clockEntry) int {
		return cmp.Compare(a.clock, b.clock)
	})
	for _, e := range entries {
		e.mu.Lock()
	}
	return func() {
		for _, e := range entries {
			e.mu.Unlock()
		}
	}
}

// followAll moves the writer of each shared clock of ws to its leader. The caller holds
// the entries' locks.
func followAll(ctx context.Context, ts *cesium.DB, ws []*clockWriter) {
	for _, c := range ws {
		if c.entry != nil {
			c.entry.follow(ctx, ts)
		}
	}
}
