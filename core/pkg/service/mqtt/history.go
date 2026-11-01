// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

const (
	// historyDelay is how long a task waits for more historical values before it
	// writes the ones it has, so that a burst lands in one domain. A live value waits
	// as well: the claim of the live writer leaves a gap once it writes that value.
	historyDelay = 500 * time.Millisecond
	// historyPoll is how often a task checks on a write of historical values.
	historyPoll = 100 * time.Millisecond
	// historyFlushCount is the count of waiting values that ends historyDelay early.
	historyFlushCount = 1 << 16
	// historyLimit is the count of waiting values that a task holds in memory.
	historyLimit = 1 << 20
	// gapLimit is the count of gaps with no waiting values that a task remembers for
	// one tag. An older outage takes no more values.
	gapLimit = 64
)

// historyRetry covers a live writer that has not yet reached the value that moves its
// claim out of a gap. One write shares it across its fills, which bounds how long a
// blocked write holds up a stop of the task.
var historyRetry = breaker.Config{
	BaseInterval: 100 * time.Millisecond,
	Scale:        2,
	MaxRetries:   4,
}

// historicalValue is one value that an edge node stored while offline.
type historicalValue struct {
	data  []byte
	stamp telem.TimeStamp
}

// gap is time that a tag has no data for, and the historical values that wait to fill
// it. A gap ends at the first live value after it, and is open until that value.
type gap struct {
	values []historicalValue
	telem.TimeRange
}

func (g gap) open() bool { return g.End == telem.TimeStampMax }

// tagGaps holds the gaps of one tag in time order. Only the last one can be open.
type tagGaps struct {
	gaps []gap
	tag  readTag
}

func (t *tagGaps) open() *gap {
	if n := len(t.gaps); n > 0 && t.gaps[n-1].open() {
		return &t.gaps[n-1]
	}
	return nil
}

// fill is the values of one tag for one gap. A writer spans the time between its first
// and last values, so that one writer cannot fill two gaps.
type fill struct {
	values []historicalValue
	tag    readTag
}

// history writes the historical values of a read task. Each fill goes through its own
// writer in persist mode, on a routine of its own, so that a slow or failed write
// never holds back live values. One fill failing leaves the others to go on. Only the
// reader of the task calls history, and it must call close.
type history struct {
	framer  *framer.Service
	tags    map[channel.Key]*tagGaps
	flushed chan error
	sCtx    signal.Context
	cancel  context.CancelFunc
	// flushAt is when the waiting values are next due to be written. Zero when none
	// are.
	flushAt time.Time
	subject control.Subject
	// count is the count of waiting values.
	count int
	// flushing is true from the start of a write until tick takes its result.
	flushing bool
}

// newHistory returns a history with an open gap for each tag that s stores. A tag on a
// virtual channel, or in a task that saves no data, has no time to fill.
func newHistory(s *readSource) *history {
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.ins))
	h := &history{
		framer:  s.framer,
		subject: s.subject,
		tags:    make(map[channel.Key]*tagGaps),
		flushed: make(chan error, 1),
		sCtx:    sCtx,
		cancel:  cancel,
	}
	if len(s.stored) == 0 {
		return h
	}
	for _, tag := range s.tags {
		if tag.index != 0 {
			h.tags[tag.index] = &tagGaps{tag: tag}
			h.end(tag.index, s.lastStamp[tag.index])
		}
	}
	return h
}

// end opens a gap after last, the last timestamp written to index. It returns false
// when index needs no end: its tag stores no data, or the tag already has an open gap.
func (h *history) end(index channel.Key, last telem.TimeStamp) bool {
	t, ok := h.tags[index]
	if !ok || t.open() != nil {
		return false
	}
	t.gaps = append(t.gaps, gap{Start: last + 1, End: telem.TimeStampMax})
	if len(t.gaps) > gapLimit {
		closed := t.gaps[:len(t.gaps)-1]
		i := slices.IndexFunc(closed, func(g gap) bool { return len(g.values) == 0 })
		if i != -1 {
			t.gaps = slices.Delete(t.gaps, i, i+1)
		}
	}
	return true
}

// live ends the open gap of index, if it has one, at stamp, the timestamp of a live
// value. It returns an error for waiting values at or after stamp.
func (h *history) live(now time.Time, index channel.Key, stamp telem.TimeStamp) error {
	t, ok := h.tags[index]
	if !ok {
		return nil
	}
	g := t.open()
	if g == nil {
		return nil
	}
	g.End = stamp
	if len(g.values) == 0 {
		return nil
	}
	h.flushAt = now.Add(historyDelay)
	count := len(g.values)
	g.values = slices.DeleteFunc(g.values, func(v historicalValue) bool {
		return v.stamp >= stamp
	})
	if rejected := count - len(g.values); rejected > 0 {
		h.count -= rejected
		return errors.Newf(
			"rejected historical values of %s: their timestamps are not before the "+
				"first live value after the outage",
			t.tag,
		)
	}
	return nil
}

// add takes a historical value of tag. It returns an error for a value that fits no
// gap, which would write over stored data. A tag that stores no data drops the value.
func (h *history) add(
	now time.Time,
	tag readTag,
	stamp telem.TimeStamp,
	data []byte,
) error {
	t, ok := h.tags[tag.index]
	if !ok {
		return nil
	}
	i := slices.IndexFunc(t.gaps, func(g gap) bool {
		return g.ContainsStamp(stamp)
	})
	if i == -1 {
		return errors.Newf(
			"rejected a historical value of %s: its timestamp is not inside an outage",
			tag,
		)
	}
	if h.count >= historyLimit {
		return errors.Newf(
			"rejected a historical value of %s: %d values already wait to be written",
			tag, h.count,
		)
	}
	t.gaps[i].values = append(
		t.gaps[i].values, historicalValue{data: data, stamp: stamp},
	)
	h.count++
	if h.count >= historyFlushCount {
		h.flushAt = now
	} else {
		h.flushAt = now.Add(historyDelay)
	}
	return nil
}

// deadline returns the next time that tick has work to do.
func (h *history) deadline(now time.Time) (deadline time.Time, ok bool) {
	if h.flushing {
		return now.Add(historyPoll), true
	}
	return h.flushAt, !h.flushAt.IsZero()
}

// tick takes the result of a finished write, and starts the next one when values are
// due. It returns the error of the finished write.
func (h *history) tick(now time.Time) (err error) {
	select {
	case err = <-h.flushed:
		h.flushing = false
	default:
	}
	if h.flushing || h.flushAt.IsZero() || now.Before(h.flushAt) {
		return err
	}
	h.flushAt = time.Time{}
	fills := h.take(false)
	if len(fills) == 0 {
		return err
	}
	h.flushing = true
	var flushErr error
	h.sCtx.Go(
		func(ctx context.Context) error {
			flushErr = h.write(ctx, fills)
			return nil
		},
		signal.RecoverWithErrOnPanic(),
		signal.Defer(func() { h.flushed <- flushErr }),
	)
	return err
}

// close waits for the write in progress, then writes every waiting value. The caller
// must have closed the live writer: its claim then blocks no gap, open ones included.
func (h *history) close() error {
	defer h.cancel()
	var err error
	if h.flushing {
		err = <-h.flushed
	}
	fills := h.take(true)
	h.sCtx.Go(
		func(ctx context.Context) error { return h.write(ctx, fills) },
		signal.RecoverWithErrOnPanic(),
	)
	return errors.Combine(err, h.sCtx.Wait())
}

// take removes the waiting values of every closed gap, and of open gaps too if all is
// true. A gap then starts after its last taken value, which keeps a value that an edge
// node sends twice from reaching what the task has written.
func (h *history) take(all bool) []fill {
	var fills []fill
	for _, t := range h.tags {
		for i := range t.gaps {
			g := &t.gaps[i]
			if len(g.values) == 0 || (g.open() && !all) {
				continue
			}
			fills = append(fills, fill{tag: t.tag, values: g.values})
			last := slices.MaxFunc(g.values, compareStamps)
			g.Start = last.stamp + 1
			h.count -= len(g.values)
			g.values = nil
		}
	}
	return fills
}

func compareStamps(a, b historicalValue) int { return cmp.Compare(a.stamp, b.stamp) }

// write writes fills, and returns the first error. The status of a task holds one
// warning, so the first failure stands for all of them.
func (h *history) write(ctx context.Context, fills []fill) error {
	brk, err := breaker.NewBreaker(ctx, historyRetry)
	if err != nil {
		return err
	}
	for _, f := range fills {
		err = errors.Combine(err, h.writeFill(ctx, &brk, f))
	}
	return err
}

func (h *history) writeFill(
	ctx context.Context,
	brk *breaker.Breaker,
	f fill,
) error {
	slices.SortStableFunc(f.values, compareStamps)
	values := slices.CompactFunc(f.values, func(a, b historicalValue) bool {
		return a.stamp == b.stamp
	})
	var (
		stamps = telem.Series{DataType: telem.TimestampT}
		data   = telem.Series{DataType: f.tag.dataType}
		keys   = channel.Keys{f.tag.index, f.tag.channel}
	)
	for _, v := range values {
		stamps.Data = telem.ByteOrder.AppendUint64(stamps.Data, uint64(v.stamp))
		data.Data = append(data.Data, v.data...)
	}
	fr := frame.NewMulti(keys, []telem.Series{stamps, data})
	for {
		err := h.persist(ctx, values[0].stamp, keys, fr)
		if !errors.Is(err, control.ErrUnauthorized) || !brk.Wait() {
			return errors.Wrapf(err, "failed to write historical values of %s", f.tag)
		}
	}
}

// persist writes fr, which starts at start, through a writer of its own.
func (h *history) persist(
	ctx context.Context,
	start telem.TimeStamp,
	keys channel.Keys,
	fr framer.Frame,
) (err error) {
	w, err := h.framer.OpenWriter(ctx, framer.WriterConfig{
		ControlSubject:    h.subject,
		Keys:              keys,
		Start:             start,
		Mode:              framer.WriterModePersistOnly,
		ErrOnUnauthorized: new(true),
		Sync:              new(true),
	})
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, w.Close()) }()
	authorized, err := w.Write(fr)
	if err == nil && !authorized {
		return control.ErrUnauthorized
	}
	return err
}
