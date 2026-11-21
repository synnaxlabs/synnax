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

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
)

// clockEntry is the Cesium writer of one live clock, which the session that holds the
// clock's control writes through. Each transfer of the clock moves it as one event,
// under mu: a Close of the old holder's writer, which makes its commits durable, then
// an
// open for the new holder from the old writer's last commit end. The uncommitted
// samples of the old holder drop.
//
// Lock order: mu, then a Cesium claim, then a gate region. The skip predicate of a
// session never takes mu, and no path takes mu while it holds a claim or a region lock.
type clockEntry struct {
	mu    sync.Mutex
	clock cesium.ChannelKey
	ctl   *controlled
	// members holds the sessions that write the clock, by control subject key.
	members map[string]*clockMember
	holder  *clockMember
	w       *cesium.Writer
	// end is the end of the last commit of a writer of the entry.
	end telem.TimeStamp
	// handing is true when leave closed the holder's writer in the event that follow
	// ends.
	handing bool
}

// clockMember is one session that writes the clock of an entry.
type clockMember struct {
	subject string
	// keys are the keys of the session on the clock, and start the start of its writer.
	keys  []cesium.ChannelKey
	start telem.TimeStamp
	// end is the end of the last commit of the session, or start, which a commit
	// without the clock returns, as a writer with nothing to commit does.
	end telem.TimeStamp
	// pending is true when the session wrote samples since its last commit, and dropped
	// is true when a handoff dropped them. The next commit then reports a refusal.
	pending, dropped bool
	// err is the error of the writer that the entry opened or closed for the member,
	// which its next call returns.
	err error
}

// follow moves the writer of e to the session that leads the clock's control, when it
// is not the holder. The caller holds e.mu.
func (e *clockEntry) follow(ctx context.Context, ts *cesium.DB) {
	var leader *clockMember
	if st := e.ctl.c.LeadingState(); st != nil {
		leader = e.members[st.Subject.Key]
	}
	handoff := e.holder != nil || e.handing
	e.handing = false
	if leader == e.holder {
		return
	}
	if e.w != nil {
		if err := e.w.Close(); err != nil && e.holder != nil {
			e.holder.err = err
		}
		e.w = nil
	}
	if e.holder != nil && e.holder.pending {
		e.holder.pending, e.holder.dropped = false, true
	}
	e.holder = leader
	if leader == nil || leader.err != nil {
		return
	}
	// A handoff starts the new writer at the old one's last commit end, and a first
	// open at the session's own start, which refuses stored data there.
	start := leader.start
	if handoff {
		start = max(e.end, start)
	}
	e.w, leader.err = ts.OpenWriter(ctx, cesium.WriterConfig{
		Channels: leader.keys,
		Clock:    e.clock,
		Start:    start,
	})
}

// leave removes m from e, and closes the writer of e when m holds it. The caller holds
// e.mu.
func (e *clockEntry) leave(m *clockMember) error {
	delete(e.members, m.subject)
	if e.holder != m {
		return nil
	}
	e.holder = nil
	if e.w == nil {
		return nil
	}
	err := e.w.Close()
	e.w, e.handing = nil, true
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

func (c *clockWriter) lock() func() {
	if !c.shares {
		return func() {}
	}
	c.entry.mu.Lock()
	return c.entry.mu.Unlock
}

// Write writes fr through the writer of the clock, and leaves out the keys of the clock
// when the session does not hold it.
func (c *clockWriter) Write(
	ctx context.Context,
	fr cesium.Frame,
	skip func(cesium.ChannelKey) bool,
) (cesium.Frame, bool, error) {
	defer c.lock()()
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

// Commit commits the writer of the clock. A session that does not hold the clock
// commits nothing and returns the end of its last commit, or a refusal once after a
// handoff dropped its samples.
func (c *clockWriter) Commit(
	ctx context.Context,
	skip func(cesium.ChannelKey) bool,
) (telem.TimeStamp, bool, error) {
	defer c.lock()()
	w, err := c.held()
	if err != nil {
		return 0, false, err
	}
	if w == nil {
		if c.member.dropped {
			c.member.dropped = false
			return 0, true, nil
		}
		return c.member.end, false, nil
	}
	end, skipped, err := w.Commit(ctx, skip)
	if err == nil && !skipped && c.own == nil {
		c.member.end, c.member.pending, c.member.dropped = end, false, false
		c.entry.end = max(c.entry.end, end)
	}
	return end, skipped, err
}

// Sync syncs the writer of the clock when the session holds it.
func (c *clockWriter) Sync() error {
	defer c.lock()()
	w, err := c.held()
	if err != nil || w == nil {
		return err
	}
	return w.Sync()
}

// EndStretch ends the stretch of the writer of the clock when the session holds it.
func (c *clockWriter) EndStretch(ctx context.Context) error {
	defer c.lock()()
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
