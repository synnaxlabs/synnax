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
	"math/rand/v2"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

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
	var before []int64
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
		after := frames(d.fs, d.id)
		Expect(len(after)).To(BeNumerically(">", len(before)))
		if c == 1 {
			d.bad = after[len(after)-1]
		}
		before = after
	}
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
	chs := []cesium.Channel{
		{Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT},
	}
	for _, k := range data {
		chs = append(
			chs,
			cesium.Channel{Key: k, Name: "data", Index: idx, DataType: telem.Int64T},
		)
	}
	Expect(r.db.CreateChannel(ctx, chs...)).To(Succeed())
}

// synced commits and syncs w, and returns the offset of the frame of log 1 that the
// sync adds.
func (r repairRoot) synced(ctx context.Context, w *cesium.Writer) int64 {
	GinkgoHelper()
	before := len(frames(r.fs, r.db.Identity()))
	MustSucceed2(w.Commit(ctx, nil))
	Expect(w.Sync()).To(Succeed())
	after := frames(r.fs, r.db.Identity())
	Expect(len(after)).To(BeNumerically(">", before))
	return after[len(after)-1]
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
	series := []telem.Series{telem.NewSeries(stamps)}
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

// onlySegment returns the one segment of key in fs that before does not hold.
func onlySegment(fs xfs.FS, key cesium.ChannelKey, before ...uint64) uint64 {
	GinkgoHelper()
	added := slices.DeleteFunc(segmentsOf(fs, key), func(s uint64) bool {
		return slices.Contains(before, s)
	})
	Expect(added).To(HaveLen(1))
	return added[0]
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
		Expect(rep.Repaired).To(BeFalse())
		Expect(bytesOf(d.fs)).To(Equal(before))
	})

	It("Should keep out the cut tail of an open segment whose takeover Seal was in "+
		"the bad frame", func(ctx SpecContext) {
		r := newRepairRoot(ctx)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		r.create(ctx, idx, data)
		w := MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(0)},
		))
		put(ctx, w, keys, 0, commits, nil)
		r.synced(ctx, w)
		Expect(w.Close()).To(Succeed())
		// The delete cuts the last pointer of each open segment, which keeps its
		// length.
		Expect(r.db.DeleteTimeRange(ctx, keys, telem.TimeRange{
			Start: repairStamp(commits / 2),
			End:   repairStamp(commits),
		})).To(Succeed())
		cut := [2]uint64{onlySegment(r.fs, idx), onlySegment(r.fs, data)}
		// The next writer seals each segment that it takes over in the entry of its
		// first commit.
		w = MustSucceed(r.db.OpenWriter(
			ctx,
			cesium.WriterConfig{Channels: keys, Start: repairStamp(commits)},
		))
		put(ctx, w, keys, commits, commits, nil)
		bad := r.synced(ctx, w)
		put(ctx, w, keys, 2*commits, commits, nil)
		r.synced(ctx, w)
		Expect(w.Close()).To(Succeed())
		id := r.spoil(bad)
		rep := repairOf(ctx, r.fs, id)
		Expect(rep.Damage).To(HaveLen(1))
		Expect(rep.Rebuilt).To(ConsistOf(
			rebuiltOf(idx, onlySegment(r.fs, idx, cut[0]), commits, commits),
			rebuiltOf(data, onlySegment(r.fs, data, cut[1]), commits, commits),
		))
		Expect(rep.Unplaced).To(BeEmpty())
		Expect(rep.Tails).To(BeEmpty())
		Expect(rep.Unrecovered).To(BeEmpty())
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
		r.synced(ctx, w)
		put(ctx, w, keys, 10, 10, nil)
		put(ctx, w, keys, 20, 10, func(k cesium.ChannelKey) bool { return k == d2 })
		bad := r.synced(ctx, w)
		put(ctx, w, keys, 30, 10, nil)
		r.synced(ctx, w)
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
		r.synced(ctx, w)
		Expect(segmentsOf(r.fs, data)).To(BeEmpty())
		put(ctx, w, keys, commits, commits, nil)
		bad := r.synced(ctx, w)
		put(ctx, w, keys, 2*commits, commits, nil)
		r.synced(ctx, w)
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
