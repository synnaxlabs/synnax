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
	"cmp"
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// headParts is the parts of one head.
type headParts struct {
	generation uint64
	parts      []checkpoint.Part
}

// partReader loads the part records that a step of a state needs, as the engine does:
// from the parts of the newest head and of the head before it, through a cache.
type partReader struct {
	cache *checkpoint.Cache
	// keyParts returns the parts of a key, and false when the state lacks the key.
	keyParts    func(key uint32) (*state.Parts, bool)
	mu          sync.Mutex
	live, older headParts
	// loaded counts the preloads that held a part record.
	loaded int
}

// swap makes the parts of the head of generation the newest.
func (pr *partReader) swap(generation uint64, parts []checkpoint.Part) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.older, pr.live = pr.live, headParts{generation: generation, parts: parts}
}

func (pr *partReader) heads() (live, older headParts) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.live, pr.older
}

// partsOf returns the parts that kp names.
func (pr *partReader) partsOf(kp *state.Parts) []checkpoint.Part {
	GinkgoHelper()
	head, older := pr.heads()
	if kp.Generation != head.generation {
		head = older
	}
	Expect(head.generation).To(Equal(kp.Generation))
	byID := func(p checkpoint.Part, id uint64) int { return cmp.Compare(p.ID, id) }
	lo, found := slices.BinarySearchFunc(head.parts, kp.Oldest, byID)
	Expect(found).To(BeTrue())
	hi, found := slices.BinarySearchFunc(head.parts, kp.Newest, byID)
	Expect(found).To(BeTrue())
	return head.parts[lo : hi+1]
}

// preload returns the live part records of key that reach first to last, and with
// timed the live part records of other keys that name key as their clock.
func (pr *partReader) preload(
	key uint32,
	first, last telem.TimeStamp,
	timed bool,
) state.Preload {
	GinkgoHelper()
	live, _ := pr.heads()
	pre := state.Preload{Key: key, Generation: live.generation}
	if kp, ok := pr.keyParts(key); ok && kp != nil {
		pre.Generation = kp.Generation
		for _, p := range pr.partsOf(kp) {
			if p.First > last || first > p.Last {
				continue
			}
			r := MustSucceed(pr.cache.Records(p, key))
			for _, i := range r.Reaching(first, last, nil) {
				seg := r.Segments[i]
				if p.Live(r.First+i) && !partKilled(kp.Kills, p, seg.ID) &&
					reach(seg.Pointers, first, last) {
					seg.Part = p.ID
					pre.Segments = append(pre.Segments, seg)
				}
			}
		}
	}
	if len(pre.Segments) > 0 {
		pr.loaded++
	}
	if !timed {
		return pre
	}
	for _, p := range live.parts {
		if p.First > last || first > p.Last {
			continue
		}
		for _, t := range MustSucceed(pr.cache.Timed(p, key)) {
			kp, ok := pr.keyParts(t.Key)
			if !ok || kp == nil || !p.Live(t.Index) ||
				partKilled(kp.Kills, p, t.Segment.ID) ||
				!reach(t.Segment.Pointers, first, last) {
				continue
			}
			seg := t.Segment
			seg.Part = p.ID
			pre.Users = append(pre.Users, state.User{Key: t.Key, Segment: seg})
		}
	}
	return pre
}

// preloads returns the preloads of keys over first to last, in rising order of key.
func (pr *partReader) preloads(
	keys []uint32,
	first, last telem.TimeStamp,
	timed bool,
) []state.Preload {
	GinkgoHelper()
	pre := make([]state.Preload, 0, len(keys))
	for _, k := range keys {
		pre = append(pre, pr.preload(k, first, last, timed))
	}
	return pre
}

// readable returns ch with each empty slice nil, or nil when ch holds no segment and no
// part: such a Channel serves the same reads as none.
func readable(ch *state.Channel) *state.Channel {
	GinkgoHelper()
	if ch = canonOf(ch); ch != nil && ch.Segments == nil && ch.Parts == nil {
		return nil
	}
	return ch
}

func partKilled(kills []state.Kill, p checkpoint.Part, id uint64) bool {
	return slices.ContainsFunc(kills, func(k state.Kill) bool {
		return k.Segment == id && k.Generation >= p.Generation
	})
}

func reach(ptrs []state.Pointer, first, last telem.TimeStamp) bool {
	return slices.ContainsFunc(ptrs, func(p state.Pointer) bool {
		return p.First <= last && first <= p.Last
	})
}

// addPreload merges p into the preload of its key in pre.
func addPreload(pre []state.Preload, p state.Preload) []state.Preload {
	if len(p.Segments) == 0 && len(p.Users) == 0 {
		return pre
	}
	i := slices.IndexFunc(pre, func(o state.Preload) bool { return o.Key == p.Key })
	if i < 0 {
		return append(pre, p)
	}
	for _, rec := range p.Segments {
		if !slices.ContainsFunc(pre[i].Segments, func(s state.Segment) bool {
			return s.ID == rec.ID
		}) {
			pre[i].Segments = append(pre[i].Segments, rec)
		}
	}
	for _, u := range p.Users {
		if !slices.ContainsFunc(pre[i].Users, func(o state.User) bool {
			return o.Segment.ID == u.Segment.ID
		}) {
			pre[i].Users = append(pre[i].Users, u)
		}
	}
	return pre
}

// replayEntry replays e into r with the preloads that the engine gives its records.
func replayEntry(r *state.State, pr *partReader, e durableEntry) error {
	GinkgoHelper()
	records := e.records()
	var pre []state.Preload
	created := map[uint64]uint32{}
	for _, rec := range records {
		switch rec := rec.(type) {
		case *log.Create:
			created[rec.Segment] = rec.Key
		case *log.Commit:
			first, last := telem.TimeStamp(rec.First), telem.TimeStamp(rec.Last)
			for _, sl := range rec.Slots {
				key, ok := r.KeyOf(sl.Segment)
				if !ok {
					key, ok = created[sl.Segment]
				}
				if ok {
					pre = addPreload(pre, pr.preload(key, first, last, false))
				}
			}
		case *log.Tombstone:
			start, end := telem.TimeStamp(rec.Start), telem.TimeStamp(rec.End)
			pre = addPreload(pre, pr.preload(rec.Key, start, end-1, true))
		case *log.Drop:
			pre = addPreload(
				pre,
				pr.preload(rec.Key, telem.TimeStampMin, telem.TimeStampMax, true),
			)
		case *log.Check:
			pre = addPreload(
				pre,
				pr.preload(rec.Key, telem.TimeStampMin, telem.TimeStampMax, false),
			)
		}
	}
	slices.SortFunc(
		pre,
		func(a, b state.Preload) int { return cmp.Compare(a.Key, b.Key) },
	)
	return r.Replay(e.seq, records, pre...)
}

// convertedHead is the first head of a converted root: on clock 5, sealed version 0
// segments 1 and 2 of key 5 and 3 and 4 of key 6, in old files 1 and 2, below every
// stamp that the workload writes.
func convertedHead() *state.Snapshot {
	old := func(id uint64, file uint16, number uint32, first int64) state.Segment {
		return state.Segment{
			ID: id, Clock: 5, File: file, DataType: 4, Sealed: true, Length: 24,
			Pointers: []state.Pointer{pointer(number, 0, first, first+20, 0, 24, 3)},
		}
	}
	return &state.Snapshot{
		Limits: state.Limits{Segments: 5, Alignments: 3},
		Channels: []*state.Channel{
			{Key: 5, Segments: []state.Segment{old(1, 1, 1, 10), old(2, 2, 2, 40)}},
			{Key: 6, Segments: []state.Segment{old(3, 1, 1, 10), old(4, 2, 2, 40)}},
		},
	}
}

var _ = Describe(
	"Check: the state replayed from its newest head and the live state",
	func() {
		It(
			"Should agree in every field after each event, across switches and their "+
				"parts",
			func() {
				const size = 4 << 20
				// reached counts what the workload reached over every run key, so the
				// check
				// fails when it stops reaching the window it exists for.
				var reached struct {
					tagged, inWindow, replaces, drops, loaded, checks, partChecks int
				}
				for runKey := range uint64(64) {
					rnd := rand.New(rand.NewPCG(runKey, 11))
					logDisk := MustSucceed(fault.New(fault.Config{}))
					Expect(log.Prepare(logDisk, 1, size)).To(Succeed())
					heads := xfs.NewMem()
					var (
						mu      sync.Mutex
						entries []durableEntry
						failed  error
						l       *log.Log
						st      *state.State
						goExe   = func(f func()) { go f() }
						// steps holds the steps of the checkpointer, which the workload
						// runs at a later event, so events land between a switch and
						// its Parted.
						steps = make(chan func(), 4)
					)
					fail := func(err error) {
						mu.Lock()
						defer mu.Unlock()
						failed = errors.Join(failed, err)
					}
					live := &partReader{
						cache: checkpoint.NewCache(heads, 7, 64<<20),
						keyParts: func(key uint32) (*state.Parts, bool) {
							ch := st.Channel(key)
							if ch == nil {
								return nil, false
							}
							return ch.Parts, true
						},
						live: headParts{generation: 1},
					}
					ckpt := MustSucceed(checkpoint.New(checkpoint.Config{
						FS:       heads,
						DB:       7,
						LogSize:  size,
						Prepared: func(uint64, error) {},
						Executor: func(f func()) { steps <- f },
						PartSize: 1,
						Cache:    live.cache,
						Parted: func(
							generation uint64,
							parts []checkpoint.Part,
							tags []state.Tag,
							ranges []state.Range,
						) {
							swap := func() { live.swap(generation, parts) }
							mu.Lock()
							reached.tagged += len(tags)
							mu.Unlock()
							fail(st.Parted(generation, tags, ranges, swap))
						},
					}))
					cfg := config(&appended{})
					cfg.Append = func(e *log.Entry) { l.Append(e) }
					cfg.OnHead = func(g uint64, s *state.Snapshot) { ckpt.Head(g, s) }
					st = MustSucceed(state.Open(cfg, convertedHead()))
					l = MustSucceed(log.Start(log.Config{
						FS:       logDisk,
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
					wg.Go(func() { fail(ckpt.Run(ctx)) })
					add := func(records ...log.Record) {
						GinkgoHelper()
						var e log.Entry
						for _, r := range records {
							e.Add(r)
						}
						MustSucceed(l.Append(&e).Wait())
					}
					// withPreloads runs op with the preloads of keys, and again while
					// it
					// returns ErrStale, as the engine does.
					withPreloads := func(
						keys []uint32,
						first, last telem.TimeStamp,
						timed bool,
						op func([]state.Preload) error,
					) {
						GinkgoHelper()
						for {
							err := op(live.preloads(keys, first, last, timed))
							if !errors.Is(err, state.ErrStale) {
								Expect(err).ToNot(HaveOccurred())
								return
							}
						}
					}
					var (
						points []livePoint
						// from holds the newest head of each point, and copies the
						// files of the checkpointer when each head lands, since later
						// switches clean
						// older heads.
						from       []uint64
						copies     = map[uint64]xfs.FS{}
						dead       []state.Dead
						generation = uint64(1)
						parting    bool
					)
					point := func(event string) {
						GinkgoHelper()
						dead = append(dead, MustSucceed(st.TakeDead())...)
						if parting {
							return
						}
						points = append(points, livePoint{
							event:   event,
							durable: MustSucceed(st.Snapshot()),
							visible: st.Channel(6),
							dead:    slices.Clone(dead),
						})
						from = append(from, generation)
					}
					keys := []state.WriterKey{
						{Key: 5, DataType: 4},
						{Key: 6, DataType: 4},
					}
					both := []uint32{5, 6}
					// write opens a writer on both keys from start, seals what a closed
					// writer left, and commits one to three runs of stamps, one for
					// each
					// sample, then may seal.
					write := func(start telem.TimeStamp) telem.TimeStamp {
						GinkgoHelper()
						var w *state.Writer
						withPreloads(
							both,
							start,
							start,
							false,
							func(pre []state.Preload) error {
								var err error
								w, err = st.OpenWriter(state.WriterConfig{
									Keys: keys, Clock: 5, Start: start, Bounded: true,
								}, pre...)
								return err
							},
						)
						var seals []log.Record
						for _, k := range both {
							if _, held := MustSucceed2(w.Held(k)); held {
								seal := MustSucceed(w.Seal(k))
								seals = append(seals, &seal)
							}
						}
						if len(seals) > 0 {
							add(seals...)
							point("a takeover Seal")
						}
						id5, id6 := MustSucceed(
							w.NewSegment(5),
						), MustSucceed(
							w.NewSegment(6),
						)
						add(create(id5, 5, 5), create(id6, 6, 5))
						point("a Create")
						n := MustSucceed(w.NewAlignment())
						var pos uint32
						for range 1 + rnd.IntN(3) {
							k := uint32(1 + rnd.IntN(4))
							first := start + telem.TimeStamp(pos)
							last := first + telem.TimeStamp(k) - 1
							cm := commit(
								k,
								n,
								pos,
								int64(first),
								int64(last),
								slot(id5, 8*(pos+k)),
								slot(id6, 8*(pos+k)),
							)
							withPreloads(
								both,
								first,
								last,
								false,
								func(pre []state.Preload) error {
									return w.Commit(cm, pre...)
								},
							)
							add(cm)
							point("a commit")
							pos += k
						}
						if rnd.IntN(2) == 0 {
							s5, s6 := MustSucceed(w.Seal(5)), MustSucceed(w.Seal(6))
							add(&s5, &s6)
							point("a Seal")
						}
						Expect(w.Close()).To(Succeed())
						return telem.TimeStamp(pos)
					}
					// deleteRange deletes the stamps of key 6 from lo to hi. The stamp
					// of each sample is the first stamp of its pointer plus its place
					// in it, so
					// the cuts follow from the edges.
					deleteRange := func(lo, hi telem.TimeStamp) {
						GinkgoHelper()
						var d *state.Delete
						withPreloads(
							[]uint32{6},
							lo,
							hi-1,
							true,
							func(pre []state.Preload) error {
								var err error
								d, err = st.BeginDelete(6, lo, hi, pre...)
								return err
							},
						)
						if !d.Empty {
							t := &log.Tombstone{
								Key:   6,
								Start: int64(lo),
								End:   int64(hi),
							}
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
					drop := func() {
						GinkgoHelper()
						withPreloads(
							[]uint32{6},
							telem.TimeStampMin,
							telem.TimeStampMax,
							true,
							func(pre []state.Preload) error {
								return st.BeginDrop(6, pre...)
							},
						)
						add(&log.Drop{Key: 6})
						reached.drops++
						point("a Drop")
					}
					// replace rewrites a candidate of key 6 into a new segment that
					// holds
					// only its live samples, as a compaction does.
					replace := func(start telem.TimeStamp) {
						GinkgoHelper()
						snap := MustSucceed(st.Snapshot())
						var old *state.Segment
						for _, ch := range snap.Channels {
							for i, seg := range ch.Segments {
								if ch.Key == 6 &&
									slices.Contains(snap.Candidates, seg.ID) {
									old = &ch.Segments[i]
								}
							}
						}
						if old == nil {
							return
						}
						var w *state.Writer
						withPreloads(
							[]uint32{6},
							start,
							start,
							false,
							func(pre []state.Preload) error {
								var err error
								w, err = st.OpenWriter(state.WriterConfig{
									Keys:  keys[1:],
									Clock: 5,
									Start: start,
								}, pre...)
								return err
							},
						)
						var seals []log.Record
						if _, held := MustSucceed2(w.Held(6)); held {
							seal := MustSucceed(w.Seal(6))
							seals = append(seals, &seal)
						}
						id := MustSucceed(w.NewSegment(6))
						add(append(seals, create(id, 6, 5))...)
						Expect(w.Close()).To(Succeed())
						r := &log.Replace{Old: old.ID, New: id, TailCRC: 7}
						for _, p := range old.Pointers {
							r.Pointers = append(r.Pointers, log.Pointer{
								Offset:    r.Length,
								Length:    p.Length,
								Samples:   p.Samples,
								Alignment: uint64(p.Alignment),
								First:     int64(p.First),
								Last:      int64(p.Last),
							})
							r.Length += p.Length
						}
						add(r)
						reached.replaces++
						point("a Replace")
					}
					// check checks a version 0 segment of key 5 or 6 that has no
					// checks, as the check pass does: once, with the preload of its
					// key over all time, since a target that left the state stays
					// stale.
					check := func() {
						GinkgoHelper()
						key := uint32(5 + rnd.IntN(2))
						pre := live.preloads(
							[]uint32{key},
							telem.TimeStampMin,
							telem.TimeStampMax,
							false,
						)
						var targets []state.Segment
						if ch := st.Channel(key); ch != nil {
							targets = append(targets, ch.Segments...)
						}
						for _, p := range pre {
							targets = append(targets, p.Segments...)
						}
						i := slices.IndexFunc(targets, func(s state.Segment) bool {
							return s.Version == 0 && !s.Checked
						})
						if i < 0 {
							return
						}
						t := targets[i]
						err := st.Check(&log.Check{Segment: t.ID, Key: key}, pre...)
						if errors.Is(err, state.ErrStale) {
							return
						}
						Expect(err).ToNot(HaveOccurred())
						reached.checks++
						if t.Part != 0 {
							reached.partChecks++
						}
						point("a Check")
					}
					// part runs the held step of the checkpointer, which writes the
					// head and
					// its parts, then hands the Parted to the state.
					part := func() {
						GinkgoHelper()
						(<-steps)()
						parting = false
						cp := xfs.NewMem()
						Expect(xfs.CopyDir(heads, cp, "")).To(Succeed())
						copies[generation] = cp
						point("a Parted")
					}
					point("the open")
					// written holds the start and the stamp count of each writer, so
					// most
					// deletes cut samples.
					var written []telem.TimeRange
					for i := range 32 {
						if parting {
							reached.inWindow++
						}
						switch k := rnd.IntN(18); {
						case k >= 16:
							check()
						case k < 3 && !parting:
							generation++
							Expect(st.Switch(generation)).To(Succeed())
							parting = true
						case k < 6 && parting:
							part()
						case k == 6:
							drop()
						case k < 10:
							replace(telem.TimeStamp(1000 * (i + 100)))
						case k < 13 && len(written) > 0:
							w := written[rnd.IntN(len(written))]
							lo := w.Start + telem.TimeStamp(rnd.Int64N(int64(w.Span())))
							deleteRange(
								lo,
								lo+1+telem.TimeStamp(rnd.Int64N(int64(w.Span()))),
							)
						default:
							start := telem.TimeStamp(1000 * (i + 1))
							written = append(written, start.Range(start+write(start)))
						}
					}
					if parting {
						part()
					}
					closed := make(chan error, 1)
					go func() { closed <- ckpt.Close() }()
					for done := false; !done; {
						select {
						case f := <-steps:
							f()
						case err := <-closed:
							Expect(err).ToNot(HaveOccurred())
							done = true
						}
					}
					Expect(l.Close()).To(Succeed())
					Expect(st.Close()).To(Succeed())
					cancel()
					wg.Wait()
					mu.Lock()
					Expect(failed).ToNot(HaveOccurred())
					for i, p := range points {
						g := from[i]
						rcfg := config(&appended{})
						rcfg.Generation = g
						start := convertedHead()
						rp := &partReader{live: headParts{generation: 1}}
						if g > 1 {
							hdr, snap := MustSucceed2(checkpoint.Read(copies[g], g))
							start = snap
							rp.cache = checkpoint.NewCache(copies[g], 7, 64<<20)
							rp.live = headParts{generation: g, parts: hdr.Parts}
						}
						r := MustSucceed(state.Open(rcfg, start))
						rp.keyParts = r.ReplayParts
						at := func(field string) string {
							return fmt.Sprintf(
								"run key %d: %s after %s, at entry %d, from head %d",
								runKey,
								field,
								p.event,
								p.durable.Sequence,
								g,
							)
						}
						for _, e := range entries {
							if e.seq > start.Sequence && e.seq <= p.durable.Sequence {
								Expect(
									replayEntry(r, rp, e),
								).To(Succeed(), at("replay"))
							}
						}
						reached.loaded += rp.loaded
						ran := make(chan error, 1)
						runCtx, stop := context.WithCancel(context.Background())
						go func() { ran <- r.Run(runCtx) }()
						durable := MustSucceed(r.Snapshot())
						visible := r.Channel(6)
						Expect(r.Close()).To(Succeed())
						stop()
						Expect(<-ran).To(Succeed())
						got, want := canonOf(durable), canonOf(p.durable)
						// The checkpointer owns the part limit: a head carries it, and
						// the live state keeps the value it opened with, of which a
						// switch takes
						// the larger.
						got.Limits.Parts, want.Limits.Parts = 0, 0
						Expect(got.Kills).To(Equal(want.Kills), at("kills"))
						Expect(
							got.Candidates,
						).To(Equal(want.Candidates), at("candidates"))
						Expect(got).To(Equal(want), at("durable"))
						Expect(
							readable(visible),
						).To(Equal(readable(p.visible)), at("visible"))
					}
					mu.Unlock()
				}
				GinkgoWriter.Printf("reached %+v\n", reached)
				Expect(reached.tagged).To(BeNumerically(">", 0))
				Expect(reached.inWindow).To(BeNumerically(">", 0))
				Expect(reached.replaces).To(BeNumerically(">", 0))
				Expect(reached.drops).To(BeNumerically(">", 0))
				Expect(reached.loaded).To(BeNumerically(">", 0))
				Expect(reached.checks).To(BeNumerically(">", 0))
				Expect(reached.partChecks).To(BeNumerically(">", 0))
			},
		)
	},
)
