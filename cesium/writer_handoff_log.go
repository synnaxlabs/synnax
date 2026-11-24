// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"cmp"
	"context"
	"slices"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// handoffLog moves the writer of w on the pointer log layout to the keys of cfg. The
// writer keeps its uncommitted runs, so the next commit of the new writer commits them,
// and its stretch goes on after their last stamp. The keys of w that cfg lacks leave in
// a Pending, the keys of adopt join with their samples, and the other keys of cfg open
// from the start of the first run. With no run and no adopt, it closes w and opens a
// writer of cfg that goes on in the stretch of w. A writer that stores no samples
// commits its runs first, and adopts no Pending.
func (db *DB) handoffLog(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
	adopt *Pending,
) (_ *Writer, pending *Pending, err error) {
	l := w.log
	if adopt != nil && adopt.log == nil {
		return nil, nil, errors.Wrap(
			validate.ErrValidation,
			"a handoff on the pointer log layout adopts no pending samples of the "+
				"old layout",
		)
	}
	if l.unstored {
		if adopt != nil {
			return nil, nil, errors.Wrap(
				validate.ErrValidation,
				"a writer that stores no samples adopts no pending samples",
			)
		}
		// The commit of a writer that stores nothing only ends its runs.
		if _, _, err = l.commit(ctx, nil); err != nil {
			return nil, nil, err
		}
	}
	if len(l.runs) == 0 && adopt == nil {
		next, err := db.reopenLog(ctx, w, cfg)
		return next, nil, err
	}
	if adopt != nil {
		for _, lk := range adopt.log.keys {
			if !slices.Contains(cfg.Channels, lk.ch.Key) {
				return nil, nil, errors.Wrap(
					validate.ErrValidation,
					"the writer lacks a key of the pending",
				)
			}
		}
	}
	defer func() {
		if err != nil && pending != nil {
			err = errors.Join(err, pending.Close())
			pending = nil
		}
	}()
	if split := l.keysOutside(cfg.Channels); len(split) > 0 {
		if pending, err = l.split(split); err != nil {
			return nil, nil, err
		}
	}
	if adopt != nil {
		if err = l.join(adopt.log); err != nil {
			return nil, nil, err
		}
	}
	if err = db.addKeys(l, cfg.Channels); err != nil {
		return nil, nil, err
	}
	w.closed = true
	return &Writer{WriterConfig: cfg, log: l}, pending, nil
}

// reopenLog closes w, which holds no uncommitted sample, and opens a writer of cfg that
// goes on in the stretch of w.
func (db *DB) reopenLog(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
) (*Writer, error) {
	run := *w.log
	if err := w.Close(); err != nil {
		return nil, err
	}
	next, err := db.OpenWriter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	next.log.number, next.log.next = run.number, run.next
	next.log.last, next.log.lastCommit = run.last, run.lastCommit
	return next, nil
}

// keysOutside returns the keys of w that keys lacks.
func (w *logWriter) keysOutside(keys []ChannelKey) []ChannelKey {
	var out []ChannelKey
	for _, lk := range w.keys {
		if !slices.Contains(keys, lk.ch.Key) {
			out = append(out, lk.ch.Key)
		}
	}
	return out
}

// split moves keys out of w into a writer of data against the stamps of the runs of w,
// and returns it as a Pending, or nil when keys hold no uncommitted sample.
func (w *logWriter) split(keys []ChannelKey) (*Pending, error) {
	st, err := w.st.Split(keys)
	if err != nil {
		return nil, err
	}
	p := &logWriter{
		e:          w.e,
		st:         st,
		history:    w.history,
		clock:      w.clock,
		clockCh:    w.clockCh,
		byKey:      make(map[ChannelKey]*logKey, len(keys)),
		unstored:   w.unstored,
		lastCommit: w.lastCommit,
	}
	var keep, take []int
	for i, lk := range w.keys {
		if slices.Contains(keys, lk.ch.Key) {
			take = append(take, i)
			p.keys = append(p.keys, lk)
			p.byKey[lk.ch.Key] = lk
			delete(w.byKey, lk.ch.Key)
		} else {
			keep = append(keep, i)
		}
	}
	w.keys = pick(w.keys, keep)
	for i := range w.runs {
		r := &w.runs[i]
		pr := run{
			alignment: r.alignment,
			samples:   r.samples,
			first:     r.first,
			last:      r.last,
			from:      r.first,
		}
		if r.ends != nil {
			pr.ends, r.ends = pick(r.ends, take), pick(r.ends, keep)
		}
		p.runs = append(p.runs, pr)
	}
	p.dropEmptyRuns()
	if len(p.runs) == 0 {
		return nil, p.close()
	}
	last := p.runs[len(p.runs)-1]
	p.stamps = &placer{
		e:       w.e,
		clock:   w.clock,
		start:   p.runs[0].first,
		started: true,
		cur: state.Pointer{
			First:     last.first,
			Last:      last.last,
			Alignment: last.alignment,
			Samples:   last.samples,
		},
		next: last.alignment.AddSamples(last.samples),
	}
	return &Pending{log: p}, nil
}

// dropEmptyRuns drops the runs in which no key of w has bytes.
func (w *logWriter) dropEmptyRuns() {
	from := w.ends(nil, false)
	pending := w.ends(nil, true)
	w.runs = slices.DeleteFunc(w.runs, func(r run) bool {
		to := r.ends
		if to == nil {
			to = pending
		}
		grows := false
		for i, e := range to {
			grows = grows ||
				(e.id != 0 && (e.id != from[i].id || e.length != from[i].length))
		}
		from = to
		return !grows
	})
	if n := len(w.runs); n > 0 {
		w.runs[n-1].ends = nil
	}
}

// join moves the keys of a, a writer of a Pending, into w, with the commits of a that
// it has not logged. The runs of a that w still carries take the samples of a, and the
// others, on stamps that the clock committed, come first as runs of data alone.
func (w *logWriter) join(a *logWriter) error {
	if err := w.st.Join(a.st); err != nil {
		return err
	}
	committed := w.ends(nil, false)
	pending := a.ends(nil, true)
	end := func(j int) []segmentEnd {
		if a.runs[j].ends != nil {
			return a.runs[j].ends
		}
		return pending
	}
	joined := make([][]segmentEnd, len(w.runs))
	var data []run
	for j, ar := range a.runs {
		i := slices.IndexFunc(
			w.runs,
			func(r run) bool { return r.alignment == ar.alignment },
		)
		if i >= 0 {
			joined[i] = end(j)
			continue
		}
		data = append(data, run{
			alignment: ar.alignment,
			samples:   ar.samples,
			first:     ar.first,
			last:      ar.last,
			from:      ar.first,
			ends:      slices.Concat(committed, end(j)),
		})
	}
	for i := range w.runs {
		if w.runs[i].ends == nil {
			continue
		}
		at := joined[i]
		if at == nil {
			at = pending
		}
		w.runs[i].ends = slices.Concat(w.runs[i].ends, at)
	}
	for _, lk := range a.keys {
		// No write of w has frame zero.
		lk.frame = 0
		w.keys = append(w.keys, lk)
		w.byKey[lk.ch.Key] = lk
	}
	// The next sync of w, which writes the clock, logs the commits of a with or after
	// the stamps that they need.
	w.entry.Take(&a.entry)
	w.sealed = append(w.sealed, a.sealed...)
	w.runs = append(data, w.runs...)
	a.keys, a.runs, a.sealed = nil, nil, nil
	return nil
}

// addKeys makes l write each key of keys that it does not, with claims from the start
// of its first run.
func (db *DB) addKeys(l *logWriter, keys []ChannelKey) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var (
		added []*logKey
		wks   []state.WriterKey
	)
	for _, key := range keys {
		if _, ok := l.byKey[key]; ok {
			continue
		}
		ch, ok := db.channelOf(key)
		if !ok {
			return channel.NewNotFoundError(key)
		}
		if ch.Clock() != l.clock {
			return channel.Wrap(key, errors.Wrapf(
				validate.ErrValidation,
				"channel is not timed by the writer clock <%d>",
				l.clock,
			))
		}
		lk := &logKey{ch: ch, code: telem.CodeOf(ch.DataType)}
		added = append(added, lk)
		wks = append(wks, state.WriterKey{Key: key, DataType: uint8(lk.code)})
	}
	if len(added) == 0 {
		return nil
	}
	slices.SortFunc(
		wks,
		func(a, b state.WriterKey) int { return cmp.Compare(a.Key, b.Key) },
	)
	keyed := make([]ChannelKey, len(wks))
	for i, k := range wks {
		keyed[i] = k.Key
	}
	start := l.runs[0].first
	if _, err := l.e.withPreloads(nil, keyed, start, start, false, func(
		pre []state.Preload,
	) error {
		return l.st.Add(wks, start, pre...)
	}); err != nil {
		return err
	}
	for _, lk := range added {
		l.keys = append(l.keys, lk)
		l.byKey[lk.ch.Key] = lk
	}
	for i := range l.runs {
		if l.runs[i].ends != nil {
			l.runs[i].ends = append(l.runs[i].ends, make([]segmentEnd, len(added))...)
		}
	}
	return nil
}

// commitLog commits the samples of p, a Pending of the pointer log layout, once the
// clock holds the exact stamps of each of their runs, and releases p once those stamps
// are durable. It returns false until then, and never waits.
func (p *Pending) commitLog(ctx context.Context) (telem.TimeStamp, bool, error) {
	l := p.log
	if len(l.runs) > 0 {
		for _, r := range l.runs {
			var held bool
			if _, err := l.e.withPreloads(
				nil,
				[]ChannelKey{l.clock},
				r.first,
				r.last,
				false,
				func(pre []state.Preload) (err error) {
					held, err = l.st.Stamped(
						r.alignment,
						r.samples,
						r.first,
						r.last,
						pre...,
					)
					return err
				},
			); err != nil || !held {
				return 0, false, err
			}
		}
		if _, _, err := l.commit(ctx, nil); err != nil {
			return 0, false, err
		}
	}
	if err := l.st.CheckStamps(l.need); errors.Is(err, state.ErrNotDurable) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, err
	}
	return l.lastCommit, true, p.Close()
}

func pick[T any](s []T, at []int) []T {
	out := make([]T, 0, len(at))
	for _, i := range at {
		out = append(out, s[i])
	}
	return out
}
