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
	// parked holds the uncommitted samples of the session's other channels that a
	// handoff took out of the clock's writer.
	parked *cesium.Pending
	// err is the error of the writer that the entry opened or closed for the member,
	// which its next call returns.
	err error
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
			e.w, e.owner = nil, nil
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
	cfg := cesium.WriterConfig{Channels: leader.keys, Clock: e.clock, Start: start}
	if e.w != nil {
		e.handTo(ctx, ts, leader, cfg)
		return
	}
	e.w, leader.err = ts.OpenWriter(ctx, cfg)
	e.owner = leader
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
	owner, adopt := e.owner, leader.parked
	w, parked, err := ts.Handoff(ctx, e.w, cfg, adopt)
	e.w, e.owner, leader.parked = w, leader, nil
	if err != nil {
		owner.dropped, owner.pending = owner.pending, false
		leader.dropped = leader.dropped || adopt != nil
		e.ins.L.Warn(
			"dropped the uncommitted samples that the next holder of a clock could not take",
			zap.Uint32("clock", e.clock),
			zap.Error(err),
		)
		e.w, leader.err = ts.OpenWriter(ctx, cfg)
		return
	}
	if parked == nil {
		return
	}
	if e.member(owner) {
		owner.parked = parked
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
func (e *clockEntry) leave(m *clockMember) error {
	delete(e.members, m.subject)
	var err error
	if m.parked != nil {
		err, m.parked = m.parked.Close(), nil
	}
	if e.holder != m {
		return err
	}
	e.end = max(e.end, m.last)
	e.holder, e.handing = nil, true
	if e.w != nil {
		err = errors.Join(err, e.w.Sync())
	}
	return err
}

// clockWriter is the handle of a session on the Cesium writer of one clock: its own, or
// the writer of the clock's entry, which only the holder of the clock writes through. A
// session that holds a gate on the clock and keeps its own writer still follows the
// entry, since its transfers move the shared writer.
type clockWriter struct {
	clock  cesium.ChannelKey
	own    *cesium.Writer
	entry  *clockEntry
	member *clockMember
	// shares is true when the session writes through the entry's writer.
	shares bool
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

// lock locks the entry of a shared writer to read. A session changes only its own
// member under it, and only the holder writes through the entry's writer.
func (c *clockWriter) lock() {
	if c.shares {
		c.entry.mu.RLock()
	}
}

func (c *clockWriter) unlock() {
	if c.shares {
		c.entry.mu.RUnlock()
	}
}

// Write writes fr through the writer of the clock, and leaves out the keys of the clock
// when the session does not hold it.
func (c *clockWriter) Write(
	ctx context.Context,
	fr cesium.Frame,
	skip func(cesium.ChannelKey) bool,
) (cesium.Frame, bool, error) {
	c.lock()
	defer c.unlock()
	w, err := c.held()
	if err != nil {
		return fr, false, err
	}
	if w == nil {
		return refuse(fr, c.member.keys)
	}
	written, skipped, err := w.Write(ctx, fr, skip)
	if err == nil && c.own == nil && holds(written, c.member.keys) {
		c.member.pending = true
	}
	return written, skipped, err
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
	return end, skipped, err
}

// commitParked commits the samples that a handoff parked on the session, once the clock
// holds their stamps, and drops them when the session lacks one of their channels. It
// returns true when the session has samples that it cannot commit. The caller holds the
// entry's lock.
func (c *clockWriter) commitParked(
	ctx context.Context,
	skip func(cesium.ChannelKey) bool,
) (telem.TimeStamp, bool, error) {
	m := c.member
	if m.parked == nil {
		if m.pending {
			return 0, true, nil
		}
		return m.end, false, nil
	}
	for _, k := range m.keys {
		if k != c.clock && skip != nil && skip(k) {
			err := m.parked.Close()
			m.parked = nil
			return 0, true, err
		}
	}
	end, held, err := m.parked.Commit(ctx)
	if err != nil {
		err = errors.Join(err, m.parked.Close())
		m.parked = nil
		return 0, false, err
	}
	if !held {
		return 0, true, nil
	}
	m.parked, m.end, m.pending = nil, end, false
	return end, false, nil
}

// Sync syncs the writer of the clock when the session holds it.
func (c *clockWriter) Sync() error {
	c.lock()
	defer c.unlock()
	w, err := c.held()
	if err != nil || w == nil {
		return err
	}
	return w.Sync()
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
