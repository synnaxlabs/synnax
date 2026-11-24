// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// damaged is a root with a bad frame in the middle of log 1 and valid frames after it
// (fault 5b).
type damaged struct {
	fs        xfs.FS
	id        uint64
	idx, data cesium.ChannelKey
	// first is the alignment of the first sample, and bad the offset of the frame that
	// holds the second of the three commits.
	first telem.Alignment
	bad   int64
}

// commits is the number of samples in each of the three commits of a damaged root.
const commits = 10000

// frames returns the offsets of the frames of log 1 that hold entries.
func frames(fs xfs.FS, id uint64) []int64 {
	GinkgoHelper()
	f := MustSucceed(fs.Open(log.Name(1), os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	info := MustSucceed(f.Stat())
	b := make([]byte, info.Size())
	MustSucceed(f.ReadAt(b, 0))
	var out []int64
	end := info.Size() - log.FrameHeaderSize
	for off := int64(log.FramesStart); off <= end; off += 8 {
		if binary.LittleEndian.Uint32(b[off:]) != 0x464C5043 {
			continue
		}
		fr, err := log.DecodeFrame(b[off:], id, 1, uint64(off))
		if err == nil && fr.Count > 0 {
			out = append(out, off)
		}
	}
	return out
}

// commitAt returns the offset of the frame of log 1 that holds the Commit whose first
// stamp is first.
func commitAt(fs xfs.FS, id uint64, first telem.TimeStamp) int64 {
	GinkgoHelper()
	return frameOf(fs, id, func(rec log.Record) bool {
		c, ok := rec.(*log.Commit)
		return ok && c.First == int64(first)
	})
}

// frameOf returns the offset of the first frame of log 1 that holds a record that
// match matches.
func frameOf(fs xfs.FS, id uint64, match func(log.Record) bool) int64 {
	GinkgoHelper()
	b := MustSucceed(xfs.ReadFile(fs, log.Name(1)))
	for _, off := range frames(fs, id) {
		fr := MustSucceed(log.DecodeFrame(b[off:], id, 1, uint64(off)))
		var dec log.Decoder
		entries := fr.Entries
		for range fr.Count {
			n := MustSucceed(dec.Reset(entries))
			for {
				rec, ok := MustSucceed2(dec.Next())
				if !ok {
					break
				}
				if match(rec) {
					return off
				}
			}
			entries = entries[n:]
		}
	}
	Fail("no frame holds the record")
	return 0
}

// damage builds a root with three synced commits of commits samples each, then a
// delete of 10 samples inside the second commit, and flips a byte in the frame of the
// second commit.
func damage(ctx context.Context) damaged {
	GinkgoHelper()
	d := damaged{fs: xfs.NewMem()}
	db := MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(d.fs),
		cesium.WithPointerLog(),
		cesium.WithRandom(rand.NewPCG(1, 2)),
	))
	d.id = db.Identity()
	stop := StartRun(db.Run)
	d.idx, d.data = GenerateChannelKey(), GenerateChannelKey()
	keys := []cesium.ChannelKey{d.idx, d.data}
	Expect(db.CreateChannel(
		ctx,
		cesium.Channel{
			Key: d.idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		},
		cesium.Channel{Key: d.data, Name: "data", Index: d.idx, DataType: telem.Int64T},
	)).To(Succeed())
	w := MustSucceed(db.OpenWriter(
		ctx,
		cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
	))
	for c := range 3 {
		stamps := make([]telem.TimeStamp, commits)
		values := make([]int64, commits)
		for i := range stamps {
			p := c*commits + i
			stamps[i], values[i] = repairStamp(p), int64(p)
		}
		fr, _ := MustSucceed2(w.Write(ctx, telem.MultiFrame(
			keys,
			[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(values)},
		), nil))
		if c == 0 {
			d.first = fr.Get(d.idx).Series[0].Alignment
		}
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Sync()).To(Succeed())
	}
	d.bad = commitAt(d.fs, d.id, repairStamp(commits))
	Expect(w.Close()).To(Succeed())
	Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRange{
		Start: repairStamp(15000),
		End:   repairStamp(15010),
	})).To(Succeed())
	stop()
	Expect(db.Close()).To(Succeed())
	Expect(frames(d.fs, d.id)[len(frames(d.fs, d.id))-1]).To(BeNumerically(">", d.bad))
	flipLog(d.fs, d.bad+log.FrameHeaderSize+16)
	return d
}

// flipLog flips every bit of the byte at offset at of log 1.
func flipLog(fs xfs.FS, at int64) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
	b := make([]byte, 1)
	MustSucceed(f.ReadAt(b, at))
	b[0] ^= 0xFF
	MustSucceed(f.WriteAt(b, at))
	Expect(f.Close()).To(Succeed())
}

func repairStamp(p int) telem.TimeStamp {
	return telem.SecondTS + telem.TimeStamp(p)*telem.MillisecondTS
}

// repairRoot is a new root in memory on the pointer log, and the stop of the run of its
// database.
type repairRoot struct {
	fs   xfs.FS
	db   *cesium.DB
	stop func()
}

func newRepairRoot(ctx context.Context) repairRoot {
	GinkgoHelper()
	fs := xfs.NewMem()
	db := MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithRandom(rand.NewPCG(1, 2)),
	))
	return repairRoot{fs: fs, db: db, stop: StartRun(db.Run)}
}

// create creates the index idx and an int64 channel of it for each key of data.
func (r repairRoot) create(
	ctx context.Context,
	idx cesium.ChannelKey,
	data ...cesium.ChannelKey,
) {
	GinkgoHelper()
	chs := make([]cesium.Channel, 0, 1+len(data))
	chs = append(chs, cesium.Channel{
		Key:      idx,
		Name:     "time",
		IsIndex:  true,
		DataType: telem.TimestampT,
	})
	for _, k := range data {
		chs = append(
			chs,
			cesium.Channel{Key: k, Name: "data", Index: idx, DataType: telem.Int64T},
		)
	}
	Expect(r.db.CreateChannel(ctx, chs...)).To(Succeed())
}

// synced commits and syncs w, whose commit starts at position p, and returns the
// offset of the frame of log 1 that holds the commit.
func (r repairRoot) synced(ctx context.Context, w *cesium.Writer, p int) int64 {
	GinkgoHelper()
	MustSucceed2(w.Commit(ctx, nil))
	Expect(w.Sync()).To(Succeed())
	return commitAt(r.fs, r.db.Identity(), repairStamp(p))
}

// spoil closes the database of r, flips a byte in the entries of the frame of log 1 at
// bad, and returns the identity of the database.
func (r repairRoot) spoil(bad int64) uint64 {
	GinkgoHelper()
	id := r.db.Identity()
	r.stop()
	Expect(r.db.Close()).To(Succeed())
	all := frames(r.fs, id)
	Expect(all[len(all)-1]).To(BeNumerically(">", bad))
	flipLog(r.fs, bad+log.FrameHeaderSize+16)
	return id
}

// put writes one frame of n samples from position p to keys, whose first key is the
// clock, except the keys for which skip returns true.
func put(
	ctx context.Context,
	w *cesium.Writer,
	keys []cesium.ChannelKey,
	p, n int,
	skip func(cesium.ChannelKey) bool,
) {
	GinkgoHelper()
	stamps, values := make([]telem.TimeStamp, n), make([]int64, n)
	for i := range n {
		stamps[i], values[i] = repairStamp(p+i), int64(p+i)
	}
	series := make([]telem.Series, 0, len(keys))
	series = append(series, telem.NewSeries(stamps))
	for range keys[1:] {
		series = append(series, telem.NewSeries(values))
	}
	MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, series), skip))
}

// repairOf returns the report of a repair of fs with the identity id that confirms
// nothing.
func repairOf(ctx context.Context, fs xfs.FS, id uint64) cesium.Report {
	GinkgoHelper()
	return MustSucceed(cesium.Repair(
		ctx,
		"",
		func(cesium.Report) bool { return false },
		cesium.WithFS(fs),
		cesium.WithIdentity(id),
	))
}

// segmentsOf returns the segments of the files of the directory of key in fs, in
// rising order.
func segmentsOf(fs xfs.FS, key cesium.ChannelKey) []uint64 {
	GinkgoHelper()
	var out []uint64
	for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
		if hex, ok := strings.CutSuffix(info.Name(), ".seg"); ok {
			out = append(out, MustSucceed(strconv.ParseUint(hex, 16, 64)))
		}
	}
	slices.Sort(out)
	return out
}

// onlySegment returns the one segment of key in fs.
func onlySegment(fs xfs.FS, key cesium.ChannelKey) uint64 {
	GinkgoHelper()
	segs := segmentsOf(fs, key)
	Expect(segs).To(HaveLen(1))
	return segs[0]
}

// bytesOf returns the bytes of each file under fs by its path.
func bytesOf(fs xfs.FS) map[string][]byte {
	GinkgoHelper()
	out := make(map[string][]byte)
	var walk func(dir string)
	walk = func(dir string) {
		for _, info := range MustSucceed(fs.List(dir)) {
			name := path.Join(dir, info.Name())
			if info.IsDir() {
				walk(name)
			} else {
				out[name] = MustSucceed(xfs.ReadFile(fs, name))
			}
		}
	}
	walk("")
	return out
}

// rebuiltOf returns the range of segment of key with n samples of 8 bytes from position
// p.
func rebuiltOf(key cesium.ChannelKey, segment uint64, p, n int) cesium.Range {
	return cesium.Range{
		Key:     key,
		Segment: segment,
		Bytes:   telem.Size(8 * n),
		Samples: int64(n),
		TimeRange: telem.TimeRange{
			Start: repairStamp(p),
			End:   repairStamp(p+n-1) + 1,
		},
	}
}

// taken is a root whose second writer took over the cut segments of the first, with
// the identity of its database, the cut segments of its clock and its data, and the
// next segments of each.
type taken struct {
	fs        xfs.FS
	id        uint64
	idx, data cesium.ChannelKey
	cut, next [2]uint64
}

var _ = Describe("Check: repair of log damage", func() {
	It("Should build a root that the open refuses with a valid frame after the bad one",
		func(ctx SpecContext) {
			d := damage(ctx)
			Expect(cesium.Open(
				ctx,
				"",
				cesium.WithFS(d.fs),
				cesium.WithPointerLog(),
				cesium.WithIdentity(d.id),
			)).Error().To(SatisfyAll(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring("holds a valid frame at")),
			))
		})

	It("Should rebuild a lost middle commit over whole and partial blocks from its "+
		"stamps, and change no file", func(ctx SpecContext) {
		d := damage(ctx)
		before := bytesOf(d.fs)
		rep := repairOf(ctx, d.fs, d.id)
		Expect(rep.Damage).To(HaveLen(1))
		Expect(rep.Damage[0].Generation).To(Equal(uint64(1)))
		Expect(rep.Rebuilt).To(ConsistOf(
			rebuiltOf(d.idx, onlySegment(d.fs, d.idx), commits, commits),
			rebuiltOf(d.data, onlySegment(d.fs, d.data), commits, commits),
		))
		Expect(rep.Unplaced).To(BeEmpty())
		Expect(rep.Tails).To(BeEmpty())
		Expect(rep.Unrecovered).To(BeEmpty())
		Expect(rep.Buried).To(BeEmpty())
		// The later delete names the alignment of the lost commit, and the repair cuts
		// it again against the rebuilt range.
		Expect(rep.Skipped).To(BeEmpty())
		Expect(rep.Repaired).To(BeFalse())
		Expect(bytesOf(d.fs)).To(Equal(before))
	})

	// takeover writes written samples with a first writer, then deletes those from kept
	// on, which cuts the last pointer of each open segment and keeps its length. A
	// second writer seals each segment that it takes over in the entry of its first
	// commit, whose frame goes bad, and commits again after it. When rewritten is true,
	// takeover waits until a rewrite removes the file of each cut segment.
	takeover := func(ctx context.Context, written, kept int, rewritten bool) taken {
		GinkgoHelper()
		r := newRepairRoot(ctx)
		t := taken{fs: r.fs, idx: GenerateChannelKey(), data: GenerateChannelKey()}
		keys := []cesium.ChannelKey{t.idx, t.data}
		r.create(ctx, t.idx, t.data)
		w := MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
		))
		put(ctx, w, keys, 0, written, nil)
		r.synced(ctx, w, 0)
		Expect(w.Close()).To(Succeed())
		Expect(r.db.DeleteTimeRange(ctx, keys, telem.TimeRange{
			Start: repairStamp(kept),
			End:   repairStamp(written),
		})).To(Succeed())
		t.cut = [2]uint64{onlySegment(r.fs, t.idx), onlySegment(r.fs, t.data)}
		w = MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(written)},
		))
		put(ctx, w, keys, written, commits, nil)
		bad := r.synced(ctx, w, written)
		put(ctx, w, keys, written+commits, commits, nil)
		r.synced(ctx, w, written+commits)
		Expect(w.Close()).To(Succeed())
		if rewritten {
			Eventually(func() []uint64 {
				return append(segmentsOf(r.fs, t.idx), segmentsOf(r.fs, t.data)...)
			}).WithTimeout(time.Minute).Should(SatisfyAll(
				Not(ContainElement(t.cut[0])),
				Not(ContainElement(t.cut[1])),
			))
		}
		t.id = r.spoil(bad)
		// The second writer takes its segments before a rewrite can take one.
		for i, key := range keys {
			t.next[i] = slices.Min(slices.DeleteFunc(
				segmentsOf(r.fs, key),
				func(s uint64) bool { return s <= t.cut[i] },
			))
		}
		return t
	}

	It("Should keep out the cut tail of an open segment whose takeover Seal was in "+
		"the bad frame", func(ctx SpecContext) {
		// The cut leaves the dead share under the threshold of a rewrite, and the last
		// pointer ends inside the last full block.
		const written = 3 * commits
		t := takeover(ctx, written, written-5500, false)
		Expect(segmentsOf(t.fs, t.idx)).To(Equal([]uint64{t.cut[0], t.next[0]}))
		Expect(segmentsOf(t.fs, t.data)).To(Equal([]uint64{t.cut[1], t.next[1]}))
		rep := repairOf(ctx, t.fs, t.id)
		Expect(rep.Damage).To(HaveLen(1))
		Expect(rep.Rebuilt).To(ConsistOf(
			rebuiltOf(t.idx, t.next[0], written, commits),
			rebuiltOf(t.data, t.next[1], written, commits),
		))
		Expect(rep.Unplaced).To(BeEmpty())
		Expect(rep.Tails).To(BeEmpty())
		Expect(rep.Unrecovered).To(BeEmpty())
	})

	It("Should report a root whose later Replace removed the file of a segment whose "+
		"takeover Seal was in the bad frame", func(ctx SpecContext) {
		// The cut puts each sealed segment over the threshold of a rewrite, whose
		// Replace removes its file in a frame after the bad one.
		t := takeover(ctx, commits, commits/2, true)
		rep := repairOf(ctx, t.fs, t.id)
		Expect(rep.Damage).To(HaveLen(1))
		Expect(rep.Rebuilt).To(ConsistOf(
			rebuiltOf(t.idx, t.next[0], commits, commits),
			rebuiltOf(t.data, t.next[1], commits, commits),
		))
		Expect(rep.Unplaced).To(BeEmpty())
		Expect(rep.Tails).To(BeEmpty())
		Expect(rep.Removed).To(ConsistOf(
			cesium.Segment{Key: t.idx, Segment: t.cut[0]},
			cesium.Segment{Key: t.data, Segment: t.cut[1]},
		))
	})

	It("Should report a root whose later Replace removed the file of an open segment "+
		"with no bytes in the gap", func(ctx SpecContext) {
		r := newRepairRoot(ctx)
		cut := []cesium.ChannelKey{GenerateChannelKey(), GenerateChannelKey()}
		lost := []cesium.ChannelKey{GenerateChannelKey(), GenerateChannelKey()}
		r.create(ctx, cut[0], cut[1])
		r.create(ctx, lost[0], lost[1])
		open := func(keys []cesium.ChannelKey, p int) *cesium.Writer {
			GinkgoHelper()
			return MustSucceed(r.db.OpenWriter(
				ctx,
				cesium.WriterConfig{Channels: keys, Start: repairStamp(p)},
			))
		}
		w := open(cut, 0)
		put(ctx, w, cut, 0, commits, nil)
		r.synced(ctx, w, 0)
		Expect(w.Close()).To(Succeed())
		Expect(r.db.DeleteTimeRange(ctx, cut, telem.TimeRange{
			Start: repairStamp(commits / 2),
			End:   repairStamp(commits),
		})).To(Succeed())
		old := []uint64{onlySegment(r.fs, cut[0]), onlySegment(r.fs, cut[1])}
		const from = 10 * commits
		w = open(lost, from)
		put(ctx, w, lost, from, commits, nil)
		bad := r.synced(ctx, w, from)
		put(ctx, w, lost, from+commits, commits, nil)
		r.synced(ctx, w, from+commits)
		Expect(w.Close()).To(Succeed())
		// The takeover Seals of the cut segments, after the bad frame, start their
		// rewrite.
		w = open(cut, commits)
		put(ctx, w, cut, commits, commits, nil)
		r.synced(ctx, w, commits)
		Expect(w.Close()).To(Succeed())
		Eventually(func() []uint64 {
			return append(segmentsOf(r.fs, cut[0]), segmentsOf(r.fs, cut[1])...)
		}).WithTimeout(time.Minute).Should(SatisfyAll(
			Not(ContainElement(old[0])),
			Not(ContainElement(old[1])),
		))
		id := r.spoil(bad)
		rep := repairOf(ctx, r.fs, id)
		Expect(rep.Rebuilt).To(ConsistOf(
			rebuiltOf(lost[0], onlySegment(r.fs, lost[0]), from, commits),
			rebuiltOf(lost[1], onlySegment(r.fs, lost[1]), from, commits),
		))
		Expect(rep.Unplaced).To(BeEmpty())
		Expect(rep.Tails).To(BeEmpty())
		Expect(rep.Unrecovered).To(BeEmpty())
	})

	It("Should report a root whose lost Tombstone buried a sealed segment and removed "+
		"its file", func(ctx SpecContext) {
		r := newRepairRoot(ctx)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		r.create(ctx, idx, data)
		open := func(p int) *cesium.Writer {
			GinkgoHelper()
			return MustSucceed(r.db.OpenWriter(
				ctx,
				cesium.WriterConfig{Channels: keys, Start: repairStamp(p)},
			))
		}
		w := open(0)
		put(ctx, w, keys, 0, commits, nil)
		r.synced(ctx, w, 0)
		Expect(w.Close()).To(Succeed())
		old := onlySegment(r.fs, idx)
		// The takeover Seals make the first segments sealed before the delete empties
		// them.
		w = open(commits)
		put(ctx, w, keys, commits, commits, nil)
		r.synced(ctx, w, commits)
		Expect(r.db.DeleteTimeRange(ctx, keys, telem.TimeRange{
			Start: repairStamp(0),
			End:   repairStamp(commits),
		})).To(Succeed())
		Eventually(func() []uint64 {
			return segmentsOf(r.fs, idx)
		}).WithTimeout(time.Minute).ShouldNot(ContainElement(old))
		put(ctx, w, keys, 2*commits, commits, nil)
		r.synced(ctx, w, 2*commits)
		Expect(w.Close()).To(Succeed())
		bad := frameOf(r.fs, r.db.Identity(), func(rec log.Record) bool {
			t, ok := rec.(*log.Tombstone)
			return ok && t.Key == idx
		})
		id := r.spoil(bad)
		rep := repairOf(ctx, r.fs, id)
		Expect(rep.Damage).To(HaveLen(1))
		names := func(rg cesium.Range) bool { return rg.Segment == old }
		Expect(slices.ContainsFunc(rep.Rebuilt, names)).To(BeFalse())
		Expect(slices.ContainsFunc(rep.Unplaced, names)).To(BeFalse())
		Expect(slices.ContainsFunc(rep.Tails, names)).To(BeFalse())
		Expect(rep.Unrecovered).To(BeEmpty())
		Expect(rep.Removed).To(Equal([]cesium.Segment{{Key: idx, Segment: old}}))
	})

	It("Should leave unplaced a rebuilt range whose commit overlaps the stored data "+
		"of another writer", func(ctx SpecContext) {
		r := newRepairRoot(ctx)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		r.create(ctx, idx, data)
		open := func(p int) *cesium.Writer {
			GinkgoHelper()
			return MustSucceed(r.db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Start:    repairStamp(p),
				Mode:     cesium.WriterModePersistOnly,
			}))
		}
		// The two commits of a share the bad frame, and the commit of b between them
		// in time is durable before it. A rebuilt range of a runs over the data of b.
		a := open(0)
		put(ctx, a, keys, 0, commits, nil)
		MustSucceed2(a.Commit(ctx, nil))
		b := open(commits)
		put(ctx, b, keys, commits, commits, nil)
		r.synced(ctx, b, commits)
		Expect(b.Close()).To(Succeed())
		put(ctx, a, keys, 2*commits, commits, nil)
		bad := r.synced(ctx, a, 0)
		Expect(commitAt(r.fs, r.db.Identity(), repairStamp(2*commits))).To(Equal(bad))
		put(ctx, a, keys, 3*commits, commits, nil)
		r.synced(ctx, a, 3*commits)
		Expect(a.Close()).To(Succeed())
		id := r.spoil(bad)
		rep := repairOf(ctx, r.fs, id)
		Expect(rep.Rebuilt).To(BeEmpty())
		reason := fmt.Sprintf(
			"commit of key %d from %d to %d overlaps its pointer from %d to %d: "+
				"refused by the state",
			idx,
			repairStamp(0),
			repairStamp(3*commits-1),
			repairStamp(commits),
			repairStamp(2*commits-1),
		)
		unplaced := func(key cesium.ChannelKey) cesium.Range {
			return cesium.Range{
				Key:     key,
				Segment: segmentsOf(r.fs, key)[0],
				Bytes:   8 * 2 * commits,
				Samples: 2 * commits,
				Reason:  reason,
			}
		}
		Expect(rep.Unplaced).To(ConsistOf(unplaced(idx), unplaced(data)))
	})

	It("Should time the first segment of a key that the second bad frame made by the "+
		"stamps of that frame", func(ctx SpecContext) {
		r := newRepairRoot(ctx)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		r.create(ctx, idx, data)
		w := MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
		))
		// The skip keeps data out of the frames up to the second bad one, which makes
		// its first segment. Both bad frames hold as many samples of the clock.
		skip := func(k cesium.ChannelKey) bool { return k == data }
		var bad []int64
		for c := range 5 {
			s := skip
			if c > 2 {
				s = nil
			}
			put(ctx, w, keys, c*commits, commits, s)
			if off := r.synced(ctx, w, c*commits); c%2 == 1 {
				bad = append(bad, off)
			}
		}
		Expect(w.Close()).To(Succeed())
		id := r.spoil(bad[1])
		flipLog(r.fs, bad[0]+log.FrameHeaderSize+16)
		rep := repairOf(ctx, r.fs, id)
		Expect(rep.Damage).To(HaveLen(2))
		var listed int64
		for _, rg := range append(slices.Clone(rep.Rebuilt), rep.Unplaced...) {
			if rg.Key == data {
				listed += rg.Samples
			}
		}
		Expect(listed).To(Equal(int64(commits)), "%+v", rep)
		for _, rg := range rep.Rebuilt {
			if rg.Key == data {
				Expect(rg.TimeRange.Start).To(Equal(repairStamp(3*commits)), "%+v", rep)
			}
		}
	})

	It("Should list the lost commits of both bad frames of a log",
		func(ctx SpecContext) {
			r := newRepairRoot(ctx)
			idx, data := GenerateChannelKey(), GenerateChannelKey()
			keys := []cesium.ChannelKey{idx, data}
			r.create(ctx, idx, data)
			w := MustSucceed(r.db.OpenWriter(
				ctx,
				cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
			))
			var bad []int64
			for c := range 5 {
				put(ctx, w, keys, 10*c, 10, nil)
				if off := r.synced(ctx, w, 10*c); c%2 == 1 {
					bad = append(bad, off)
				}
			}
			Expect(w.Close()).To(Succeed())
			id := r.spoil(bad[1])
			flipLog(r.fs, bad[0]+log.FrameHeaderSize+16)
			rep := repairOf(ctx, r.fs, id)
			Expect(rep.Damage).To(HaveLen(2))
			for _, key := range keys {
				var listed int64
				for _, rg := range append(slices.Clone(rep.Rebuilt), rep.Unplaced...) {
					if rg.Key == key {
						listed += rg.Samples
					}
				}
				Expect(listed).To(Equal(int64(20)), "key %d: %+v", key, rep)
			}
		})

	It("Should fail a repair of a root that lost the file of an open segment that no "+
		"record retires", func(ctx SpecContext) {
		d := damage(ctx)
		name := fmt.Sprintf("%d/%016x.seg", d.idx, onlySegment(d.fs, d.idx))
		Expect(d.fs.Remove(name)).To(Succeed())
		Expect(cesium.Repair(
			ctx,
			"",
			func(cesium.Report) bool { return false },
			cesium.WithFS(d.fs),
			cesium.WithIdentity(d.id),
		)).Error().To(SatisfyAll(
			MatchError(os.ErrNotExist),
			MatchError(ContainSubstring(name)),
		))
	})

	It("Should leave unplaced a data range of a lost commit with fewer samples than "+
		"its clock, as a skip leaves it", func(ctx SpecContext) {
		r := newRepairRoot(ctx)
		idx, d1, d2 := GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, d1, d2}
		r.create(ctx, idx, d1, d2)
		w := MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
		))
		put(ctx, w, keys, 0, 10, nil)
		r.synced(ctx, w, 0)
		put(ctx, w, keys, 10, 10, nil)
		put(ctx, w, keys, 20, 10, func(k cesium.ChannelKey) bool { return k == d2 })
		bad := r.synced(ctx, w, 10)
		put(ctx, w, keys, 30, 10, nil)
		r.synced(ctx, w, 30)
		Expect(w.Close()).To(Succeed())
		id := r.spoil(bad)
		rep := repairOf(ctx, r.fs, id)
		Expect(rep.Rebuilt).To(ConsistOf(
			rebuiltOf(idx, onlySegment(r.fs, idx), 10, 20),
			rebuiltOf(d1, onlySegment(r.fs, d1), 10, 20),
		))
		Expect(rep.Unplaced).To(Equal([]cesium.Range{{
			Key:     d2,
			Segment: onlySegment(r.fs, d2),
			Bytes:   80,
			Samples: 10,
			Reason:  "it holds another number of samples than its clock",
		}}))
		Expect(rep.Tails).To(BeEmpty())
		Expect(rep.Unrecovered).To(BeEmpty())
	})

	It("Should place the first segment of a key that a lost commit made through the "+
		"clock of the later Commit that names it", func(ctx SpecContext) {
		r := newRepairRoot(ctx)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		r.create(ctx, idx, data)
		w := MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
		))
		// The skip makes the first segment of data in the bad frame, after the
		// segment of its clock.
		skip := func(k cesium.ChannelKey) bool { return k == data }
		put(ctx, w, keys, 0, commits, skip)
		r.synced(ctx, w, 0)
		Expect(segmentsOf(r.fs, data)).To(BeEmpty())
		put(ctx, w, keys, commits, commits, nil)
		bad := r.synced(ctx, w, commits)
		put(ctx, w, keys, 2*commits, commits, nil)
		r.synced(ctx, w, 2*commits)
		Expect(w.Close()).To(Succeed())
		id := r.spoil(bad)
		rep := repairOf(ctx, r.fs, id)
		Expect(rep.Unrecovered).To(BeEmpty())
		Expect(rep.Rebuilt).To(ConsistOf(
			rebuiltOf(idx, onlySegment(r.fs, idx), commits, commits),
			rebuiltOf(data, onlySegment(r.fs, data), commits, commits),
		))
		Expect(rep.Unplaced).To(BeEmpty())
		Expect(rep.Tails).To(BeEmpty())
	})
})
