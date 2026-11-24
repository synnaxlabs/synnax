// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// canon returns a copy of v in which each empty slice is nil, so two states that differ
// only in an empty slice compare equal. It copies, and so never changes what a state
// still holds.
func canon(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		out := reflect.New(v.Elem().Type())
		out.Elem().Set(canon(v.Elem()))
		return out
	case reflect.Slice:
		if v.Len() == 0 {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(canon(v.Index(i)))
		}
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		for i := range v.NumField() {
			if out.Field(i).CanSet() {
				out.Field(i).Set(canon(v.Field(i)))
			}
		}
		return out
	default:
		return v
	}
}

func canonOf[T any](v T) T {
	GinkgoHelper()
	out, ok := reflect.TypeAssert[T](canon(reflect.ValueOf(v)))
	Expect(ok).To(BeTrue())
	return out
}

// durableEntry is a durable entry of the log, encoded, so each replay decodes its own
// records. A decoder reuses its record of each type, so records clones each one.
type durableEntry struct {
	seq uint64
	b   []byte
}

func (e durableEntry) records() []log.Record {
	GinkgoHelper()
	var d log.Decoder
	MustSucceed(d.Reset(e.b))
	var records []log.Record
	for {
		r, ok := MustSucceed2(d.Next())
		if !ok {
			return records
		}
		records = append(records, log.Clone(r))
	}
}

// livePoint is what the live state held after one event.
type livePoint struct {
	event   string
	durable *state.Snapshot
	visible *state.Channel
	dead    []state.Dead
}

var _ = Describe("Check: the live and the replayed state", func() {
	It("Should agree in every field after each event", func(ctx SpecContext) {
		const size = 4 << 20
		keys := []state.WriterKey{{Key: 5, DataType: 4}}
		for runKey := range uint64(64) {
			rnd := rand.New(rand.NewPCG(runKey, 7))
			disk := MustSucceed(fault.New(fault.Config{}))
			Expect(log.Prepare(disk, 1, size)).To(Succeed())
			var (
				mu      sync.Mutex
				entries []durableEntry
				failed  error
				l       *log.Log
				goExe   = func(f func()) { go f() }
			)
			fail := func(err error) {
				mu.Lock()
				defer mu.Unlock()
				failed = errors.Join(failed, err)
			}
			cfg := config(&appended{})
			cfg.Append = func(e *log.Entry) { l.Append(e) }
			st := MustSucceed(state.Open(cfg, head(1, 1)))
			l = MustSucceed(log.Start(log.Config{
				FS:       disk,
				Executor: goExe,
				OnDurable: func(seq uint64, records []log.Record) {
					var e log.Entry
					for _, r := range records {
						e.Add(r)
					}
					mu.Lock()
					b := slices.Clone(e.Bytes())
					entries = append(entries, durableEntry{seq: seq, b: b})
					mu.Unlock()
					fail(st.Durable(seq, records))
				},
				OnRefused: func(records []log.Record, err error) {
					fail(st.Refused(records, err))
				},
				// No next file, so every entry stays in log 1.
				Prepare: func(next uint64) {
					go l.Prepared(next, errors.New("no next log"))
				},
				Switched:   func(uint64) {},
				DB:         7,
				Generation: 1,
				Sequence:   1,
			}))
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Go(func() { fail(st.Run(ctx)) })
			wg.Go(func() { fail(l.Run(ctx)) })
			add := func(r log.Record) {
				GinkgoHelper()
				var e log.Entry
				e.Add(r)
				MustSucceed(l.Append(&e).Wait())
			}
			var (
				points []livePoint
				dead   []state.Dead
			)
			point := func(event string) {
				GinkgoHelper()
				dead = append(dead, MustSucceed(st.TakeDead())...)
				points = append(points, livePoint{
					event:   event,
					durable: MustSucceed(st.Snapshot()),
					visible: st.Channel(5),
					dead:    slices.Clone(dead),
				})
			}
			// deleteRange deletes the stamps from lo to hi. The stamp of each sample is
			// the first stamp of its pointer plus its place in it, so the cuts follow
			// from the edges.
			deleteRange := func(lo, hi telem.TimeStamp) {
				GinkgoHelper()
				d := MustSucceed(st.BeginDelete(5, lo, hi))
				if !d.Empty {
					t := &log.Tombstone{Key: 5, Start: int64(lo), End: int64(hi)}
					if e := d.Left; e != nil {
						i := uint32(lo - e.Pointer.First)
						t.Left = &log.Cut{
							Segment:   e.Segment.ID,
							Offset:    e.Pointer.Offset + 8*i,
							Alignment: uint64(e.Pointer.Alignment) + uint64(i),
							Stamp:     int64(lo) - 1,
						}
					}
					if e := d.Right; e != nil {
						j := uint32(hi - e.Pointer.First)
						t.Right = &log.Cut{
							Segment:   e.Segment.ID,
							Offset:    e.Pointer.Offset + 8*j,
							Alignment: uint64(e.Pointer.Alignment) + uint64(j),
							Stamp:     int64(hi),
						}
					}
					Expect(d.Check(t)).To(Succeed())
					add(t)
				}
				Expect(d.Release()).To(Succeed())
				point("a delete")
			}
			// Each writer writes stamps from its start, one for each sample, in one to
			// three commits, then seals its segment or leaves it for the next writer to
			// seal. Deletes then take random ranges of what the writers wrote.
			point("the open")
			end := telem.TimeStamp(0)
			for i := range 8 {
				start := telem.TimeStamp(1000 * (i + 1))
				w := MustSucceed(st.OpenWriter(ctx, state.WriterConfig{
					Keys: keys, Clock: 5, Start: start, Bounded: true,
				}))
				if _, held := MustSucceed2(w.Held(5)); held {
					seal := MustSucceed(w.Seal(5))
					add(&seal)
					point("a takeover Seal")
				}
				id := MustSucceed(w.NewSegment(5))
				add(create(id, 5, 5))
				point("a Create")
				n := MustSucceed(w.NewAlignment())
				var pos uint32
				for range 1 + rnd.IntN(3) {
					k := uint32(1 + rnd.IntN(4))
					first := int64(start) + int64(pos)
					last := first + int64(k) - 1
					cm := commit(k, n, pos, first, last, slot(id, 8*(pos+k)))
					Expect(w.Commit(cm)).To(Succeed())
					add(cm)
					point("a commit")
					pos += k
				}
				end = start + telem.TimeStamp(pos)
				if rnd.IntN(2) == 0 {
					seal := MustSucceed(w.Seal(5))
					add(&seal)
					point("a Seal")
				}
				Expect(w.Close()).To(Succeed())
				point("a close")
				for range rnd.IntN(3) {
					lo := 1000 + telem.TimeStamp(rnd.Int64N(int64(end-1000)))
					deleteRange(lo, lo+1+telem.TimeStamp(rnd.IntN(12)))
				}
			}
			Expect(l.Close()).To(Succeed())
			Expect(st.Close()).To(Succeed())
			cancel()
			wg.Wait()
			mu.Lock()
			Expect(failed).ToNot(HaveOccurred())
			for _, p := range points {
				r := MustSucceed(state.Open(config(&appended{}), head(1, 1)))
				for _, e := range entries {
					if e.seq <= p.durable.Sequence {
						Expect(r.Replay(e.seq, e.records())).To(Succeed())
					}
				}
				ran := make(chan error, 1)
				runCtx, stop := context.WithCancel(context.Background())
				go func() { ran <- r.Run(runCtx) }()
				durable := MustSucceed(r.Snapshot())
				visible := r.Channel(5)
				replayedDead := MustSucceed(r.TakeDead())
				Expect(r.Close()).To(Succeed())
				stop()
				Expect(<-ran).To(Succeed())
				at := func(field string) string {
					return fmt.Sprintf(
						"run key %d: %s after %s, at entry %d",
						runKey,
						field,
						p.event,
						p.durable.Sequence,
					)
				}
				Expect(canonOf(durable).Kills).
					To(Equal(canonOf(p.durable).Kills), at("kills"))
				Expect(canonOf(durable)).To(Equal(canonOf(p.durable)), at("durable"))
				Expect(canonOf(visible)).To(Equal(canonOf(p.visible)), at("visible"))
				Expect(replayedDead).To(ConsistOf(p.dead), at("dead"))
			}
			mu.Unlock()
		}
	})
})
