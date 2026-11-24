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
	"path"
	"slices"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// logFrame is a frame of a log file and its offset.
type logFrame struct {
	off   int64
	frame log.Frame
}

// framesOf returns the valid frames of log 1 of database db in fs, in order, from the
// first frame to the first that fails at its offset and at the next block.
func framesOf(fs xfs.FS, db uint64) []logFrame {
	GinkgoHelper()
	b := MustSucceed(xfs.ReadFile(fs, log.Name(1)))
	var out []logFrame
	for off := int64(log.FramesStart); ; {
		f, err := log.DecodeFrame(b[off:], db, 1, uint64(off))
		if err != nil {
			next := (off + log.BlockSize - 1) &^ (log.BlockSize - 1)
			if next == off || next >= int64(len(b)) {
				return out
			}
			if f, err = log.DecodeFrame(b[next:], db, 1, uint64(next)); err != nil {
				return out
			}
			off = next
		}
		out = append(out, logFrame{off: off, frame: f})
		off += int64(f.Size())
	}
}

// rootFiles returns the bytes of each file under fs by its path.
func rootFiles(fs xfs.FS) map[string][]byte {
	GinkgoHelper()
	out := make(map[string][]byte)
	var walk func(dir string)
	walk = func(dir string) {
		for _, info := range MustSucceed(fs.List(dir)) {
			name := path.Join(dir, info.Name())
			if info.IsDir() {
				walk(name)
				continue
			}
			out[name] = MustSucceed(xfs.ReadFile(fs, name))
		}
	}
	walk("")
	return out
}

// recordsOf returns the records of the entries of f.
func recordsOf(f logFrame) []log.Record {
	GinkgoHelper()
	var (
		dec     log.Decoder
		out     []log.Record
		entries = f.frame.Entries
	)
	for range f.frame.Count {
		n := MustSucceed(dec.Reset(entries))
		for {
			rec, ok := MustSucceed2(dec.Next())
			if !ok {
				break
			}
			out = append(out, log.Clone(rec))
		}
		entries = entries[n:]
	}
	return out
}

// frameWith returns the index among frames of the first frame that holds a record that
// match matches.
func frameWith(frames []logFrame, match func(log.Record) bool) int {
	GinkgoHelper()
	for i, f := range frames {
		if slices.ContainsFunc(recordsOf(f), match) {
			return i
		}
	}
	Fail("no frame holds the record")
	return 0
}

// plainFrame returns the index among frames of the first frame from from on that holds
// entries and no Create, and that a frame follows.
func plainFrame(frames []logFrame, from int) int {
	GinkgoHelper()
	created := func(rec log.Record) bool {
		_, ok := rec.(*log.Create)
		return ok
	}
	for i := from; i < len(frames)-1; i++ {
		records := recordsOf(frames[i])
		if frames[i].frame.Count > 0 && !slices.ContainsFunc(records, created) {
			return i
		}
	}
	Fail("no frame holds entries and no Create")
	return 0
}

// segmentIDs returns the segments of the directory of key in fs, in rising order.
func segmentIDs(fs xfs.FS, key cesium.ChannelKey) []uint64 {
	GinkgoHelper()
	sizes := segmentSizes(fs, key)
	out := make([]uint64, 0, len(sizes))
	for name := range sizes {
		hex := strings.TrimSuffix(name, ".seg")
		out = append(out, MustSucceed(strconv.ParseUint(hex, 16, 64)))
	}
	slices.Sort(out)
	return out
}

// segmentSizes returns the size of each segment file of the directory of key in fs.
func segmentSizes(fs xfs.FS, key cesium.ChannelKey) map[string]int64 {
	GinkgoHelper()
	out := make(map[string]int64)
	for _, info := range MustSucceed(fs.List(strconv.Itoa(int(key)))) {
		if strings.HasSuffix(info.Name(), ".seg") {
			out[info.Name()] = info.Size()
		}
	}
	return out
}

var _ = Describe("Repair", func() {
	const (
		idx  cesium.ChannelKey = 1
		data cesium.ChannelKey = 2
	)
	var (
		fs xfs.FS
		id uint64
	)
	// repair repairs fs with the identity id, and counts the calls of confirm.
	repair := func(ctx context.Context, confirms *int) (cesium.Report, error) {
		return cesium.Repair(ctx, "", func(cesium.Report) bool {
			*confirms++
			return false
		}, cesium.WithFS(fs), cesium.WithIdentity(id))
	}
	// open opens fs on the pointer log with the identity id.
	open := func(ctx context.Context) (*cesium.DB, error) {
		return cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithIdentity(id),
		)
	}
	// write writes, commits, and syncs 4 frames of 5 samples to both keys from 1 second
	// after from.
	write := func(ctx context.Context, db *cesium.DB, from telem.TimeStamp) {
		GinkgoHelper()
		keys := []cesium.ChannelKey{idx, data}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    from + telem.SecondTS,
		}))
		for c := range telem.TimeStamp(4) {
			stamps := make([]telem.TimeStamp, 5)
			for i := range stamps {
				stamps[i] = from + (c*5+telem.TimeStamp(i)+1)*telem.SecondTS
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeriesV[int64](1, 2, 3, 4, 5),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Sync()).To(Succeed())
		}
		Expect(w.Close()).To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		fs, id = xfs.NewMem(), 0
		db := MustSucceed(open(ctx))
		id = db.Identity()
		stop := GoRun(ctx, db.Run)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      idx,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		write(ctx, db, 0)
		Expect(errors.Join(stop(), db.Close())).To(Succeed())
	})
	// spoil changes byte at of frame i of log 1, and returns that frame and where its
	// bad bytes start: the end of the frame before it, or the start of the frames.
	spoil := func(i, at int) (logFrame, uint64) {
		GinkgoHelper()
		frames := framesOf(fs, id)
		Expect(i).To(BeNumerically("<", len(frames)-1))
		f := frames[i]
		b := MustSucceed(xfs.ReadFile(fs, log.Name(1)))
		b[f.off+int64(at)] ^= 0xFF
		Expect(xfs.WriteFile(fs, log.Name(1), b)).To(Succeed())
		if i == 0 {
			return f, log.FramesStart
		}
		prev := frames[i-1]
		return f, uint64(prev.off) + uint64(prev.frame.Size())
	}
	// middle returns the index of the first frame of log 1 from the middle on that
	// holds entries.
	middle := func() int {
		GinkgoHelper()
		frames := framesOf(fs, id)
		for i := len(frames) / 2; i < len(frames)-1; i++ {
			if frames[i].frame.Count > 0 {
				return i
			}
		}
		Fail("no frame from the middle on holds entries")
		return 0
	}

	It("Should report no damage of a healthy root, call no confirm, and write nothing",
		func(ctx SpecContext) {
			before := rootFiles(fs)
			confirms := 0
			rep := MustSucceed(repair(ctx, &confirms))
			Expect(rep.Damage).To(BeEmpty())
			Expect(rep.Lost).To(BeEmpty())
			Expect(confirms).To(BeZero())
			Expect(rootFiles(fs)).To(Equal(before))
		})

	It("Should report a bad frame that valid frames follow, and each record that it "+
		"held", func(ctx SpecContext) {
		// A changed check leaves the entries whole.
		f, start := spoil(middle(), 4)
		Expect(open(ctx)).Error().To(SatisfyAll(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("holds a valid frame at")),
		))
		before := rootFiles(fs)
		confirms := 0
		rep := MustSucceed(repair(ctx, &confirms))
		last := f.frame.Sequence + uint64(f.frame.Count) - 1
		Expect(rep.Damage).To(Equal([]cesium.Damage{{
			Generation:    1,
			Offset:        start,
			FirstSequence: f.frame.Sequence,
			LastSequence:  last,
		}}))
		Expect(rep.Lost).ToNot(BeEmpty())
		for _, rec := range rep.Lost {
			Expect(rec.Sequence).To(BeNumerically(">=", f.frame.Sequence))
			Expect(rec.Sequence).To(BeNumerically("<=", last))
			Expect(rec.Kind).ToNot(BeEmpty())
			Expect(rec.Reason).To(ContainSubstring("from a damaged frame"))
		}
		Expect(rootFiles(fs)).To(Equal(before))
	})

	It("Should list by sequence alone each entry of a bad frame that does not decode",
		func(ctx SpecContext) {
			// A changed entry count bounds the entries by the frame's bytes.
			f, _ := spoil(middle(), log.FrameHeaderSize)
			rep := MustSucceed(repair(ctx, new(int)))
			Expect(rep.Damage).To(HaveLen(1))
			last := f.frame.Sequence + uint64(f.frame.Count) - 1
			Expect(rep.Lost).ToNot(BeEmpty())
			Expect(rep.Lost[len(rep.Lost)-1].Sequence).To(Equal(last))
			Expect(rep.Lost[0].Kind).To(BeEmpty())
		})

	const (
		newIdx  cesium.ChannelKey = 3
		newData cesium.ChannelKey = 4
	)
	// writeNew creates newIdx and newData, writes and syncs commits of n samples to
	// both, and returns the segment files of each.
	writeNew := func(ctx context.Context, n, commits int) [2]map[string]int64 {
		GinkgoHelper()
		db := MustSucceed(open(ctx))
		stop := GoRun(ctx, db.Run)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      newIdx,
				Name:     "new_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      newData,
				Name:     "new_data",
				Index:    newIdx,
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		keys := []cesium.ChannelKey{newIdx, newData}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    telem.SecondTS,
		}))
		for c := range commits {
			stamps := make([]telem.TimeStamp, n)
			values := make([]int64, n)
			for i := range stamps {
				stamps[i] = telem.TimeStamp(c*n+i+1) * telem.SecondTS
				values[i] = int64(c*n + i)
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeries(values),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Sync()).To(Succeed())
		}
		Expect(w.Close()).To(Succeed())
		Expect(errors.Join(stop(), db.Close())).To(Succeed())
		return [2]map[string]int64{
			segmentSizes(fs, newIdx),
			segmentSizes(fs, newData),
		}
	}
	Describe("Place", func() {
		// spoilCreates changes the check of the frame that holds the Create of the
		// segments of newIdx.
		spoilCreates := func() {
			GinkgoHelper()
			created := func(rec log.Record) bool {
				c, ok := rec.(*log.Create)
				return ok && c.Key == newIdx
			}
			spoil(frameWith(framesOf(fs, id), created), 4)
		}
		// unrecovered returns the files of key that files lists, with reason.
		unrecovered := func(
			key cesium.ChannelKey,
			files map[string]int64,
			reason string,
		) []cesium.File {
			GinkgoHelper()
			out := make([]cesium.File, 0, len(files))
			for name, size := range files {
				hex := strings.TrimSuffix(name, ".seg")
				seg := MustSucceed(strconv.ParseUint(hex, 16, 64))
				out = append(out, cesium.File{
					Key:     key,
					Segment: seg,
					Bytes:   telem.Size(size),
					Reason:  reason,
				})
			}
			return out
		}

		It("Should place each segment of over 64 KiB that a later Commit names",
			func(ctx SpecContext) {
				files := writeNew(ctx, 10_000, 2)
				Expect(files[0]).To(HaveLen(1))
				Expect(files[1]).To(HaveLen(1))
				spoilCreates()
				before := rootFiles(fs)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Damage).To(HaveLen(1))
				Expect(rep.Unrecovered).To(BeEmpty())
				Expect(rep.Buried).To(BeEmpty())
				Expect(rootFiles(fs)).To(Equal(before))
				written := telem.TimeRange{
					Start: telem.SecondTS,
					End:   10_000*telem.SecondTS + 1,
				}
				Expect(rep.Rebuilt).To(Equal([]cesium.Range{
					{
						Key:       newIdx,
						Segment:   segmentIDs(fs, newIdx)[0],
						Bytes:     80_000,
						Samples:   10_000,
						TimeRange: written,
					},
					{
						Key:       newData,
						Segment:   segmentIDs(fs, newData)[0],
						Bytes:     80_000,
						Samples:   10_000,
						TimeRange: written,
					},
				}))
			})

		It("Should list as unrecovered each segment under 64 KiB that a later Commit "+
			"names", func(ctx SpecContext) {
			files := writeNew(ctx, 5, 2)
			spoilCreates()
			rep := MustSucceed(repair(ctx, new(int)))
			Expect(rep.Unrecovered).To(Equal(append(
				unrecovered(newIdx, files[0], "it holds no full block"),
				unrecovered(newData, files[1], "it holds no full block")...,
			)))
			Expect(rep.Skipped).To(HaveLen(1))
			Expect(rep.Skipped[0]).To(MatchFields(IgnoreExtras, Fields{
				"Kind": Equal("Commit"),
				"TimeRange": Equal(telem.TimeRange{
					Start: 6 * telem.SecondTS,
					End:   10*telem.SecondTS + 1,
				}),
				"Reason": ContainSubstring("which no create made"),
			}))
		})

		It("Should list as unrecovered a segment file that no record names",
			func(ctx SpecContext) {
				Expect(xfs.WriteFile(fs, "2/00000000000000ff.seg", make([]byte, 70))).
					To(Succeed())
				spoil(middle(), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Unrecovered).To(Equal([]cesium.File{{
					Key:     data,
					Segment: 0xff,
					Bytes:   70,
					Reason:  "no record names it",
				}}))
			})

		// dropNew drops newData, and returns the frames of log 1 and the index of the
		// frame of the Drop.
		dropNew := func(ctx context.Context) ([]logFrame, int) {
			GinkgoHelper()
			db := MustSucceed(open(ctx))
			stop := GoRun(ctx, db.Run)
			Expect(db.Drop([]cesium.ChannelKey{newData})).To(Succeed())
			write(ctx, db, 100*telem.SecondTS)
			Expect(errors.Join(stop(), db.Close())).To(Succeed())
			frames := framesOf(fs, id)
			return frames, frameWith(frames, func(rec log.Record) bool {
				_, ok := rec.(*log.Drop)
				return ok
			})
		}
		// putBack writes files to the directory of newData.
		putBack := func(files map[string][]byte) {
			GinkgoHelper()
			dir := MustSucceed(fs.Sub(strconv.Itoa(int(newData))))
			for name, b := range files {
				Expect(xfs.WriteFile(dir, name, b)).To(Succeed())
			}
		}

		It("Should leave out the file of a segment that a Drop before the damage drops",
			func(ctx SpecContext) {
				files := writeNew(ctx, 5, 2)
				kept := make(map[string][]byte)
				for name := range files[1] {
					kept[name] = MustSucceed(xfs.ReadFile(fs, "4/"+name))
				}
				frames, dropped := dropNew(ctx)
				putBack(kept)
				spoil(plainFrame(frames, dropped+1), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Damage).To(HaveLen(1))
				Expect(rep.Unrecovered).To(BeEmpty())
			})

		It("Should leave out a segment file that no record names of a key that a Drop "+
			"after the damage drops", func(ctx SpecContext) {
			writeNew(ctx, 5, 2)
			frames, dropped := dropNew(ctx)
			putBack(map[string][]byte{"00000000000000ff.seg": make([]byte, 70)})
			i := plainFrame(frames, 1)
			Expect(i).To(BeNumerically("<", dropped))
			spoil(i, 4)
			rep := MustSucceed(repair(ctx, new(int)))
			Expect(rep.Damage).To(HaveLen(1))
			Expect(rep.Unrecovered).To(BeEmpty())
		})
	})

	Describe("Rebuild", func() {
		// commitFrame returns the index of the frame of log 1 that holds the first
		// Commit whose first stamp is first.
		commitFrame := func(first telem.TimeStamp) int {
			GinkgoHelper()
			return frameWith(framesOf(fs, id), func(rec log.Record) bool {
				c, ok := rec.(*log.Commit)
				return ok && telem.TimeStamp(c.First) == first
			})
		}
		// newFrame returns the index of the frame of log 1 that holds the Commit of the
		// first segment of newIdx whose first stamp is first.
		newFrame := func(first telem.TimeStamp) int {
			GinkgoHelper()
			seg := segmentIDs(fs, newIdx)[0]
			return frameWith(framesOf(fs, id), func(rec log.Record) bool {
				c, ok := rec.(*log.Commit)
				return ok && telem.TimeStamp(c.First) == first &&
					slices.ContainsFunc(c.Slots, func(s log.Slot) bool {
						return s.Segment == seg
					})
			})
		}
		// pair returns the ranges of the first segments of idx and data, of samples
		// samples of 8 bytes a second apart from first.
		pair := func(samples int64, first telem.TimeStamp) []cesium.Range {
			GinkgoHelper()
			tr := telem.TimeRange{
				Start: first,
				End:   first + telem.TimeStamp(samples-1)*telem.SecondTS + 1,
			}
			out := make([]cesium.Range, 0, 2)
			for _, key := range []cesium.ChannelKey{idx, data} {
				out = append(out, cesium.Range{
					Key:       key,
					Segment:   segmentIDs(fs, key)[0],
					Bytes:     telem.Size(samples * 8),
					Samples:   samples,
					TimeRange: tr,
				})
			}
			return out
		}
		tail := "the last block of a lost commit, whose checksum was in the bad frame"
		lateRange := telem.TimeRange{
			Start: 10_001 * telem.SecondTS,
			End:   16_384*telem.SecondTS + 1,
		}

		It("Should rebuild a lost commit up to the start of the later Commit",
			func(ctx SpecContext) {
				spoil(commitFrame(6*telem.SecondTS), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Damage).To(HaveLen(1))
				Expect(rep.Rebuilt).To(Equal(pair(5, 6*telem.SecondTS)))
				Expect(rep.Unplaced).To(BeEmpty())
				Expect(rep.Tails).To(BeEmpty())
				Expect(rep.Skipped).To(BeEmpty())
			})

		It("Should rebuild the gap of each of two bad frames of one log, in log order",
			func(ctx SpecContext) {
				writeNew(ctx, 5, 5)
				spoil(newFrame(6*telem.SecondTS), 4)
				spoil(newFrame(16*telem.SecondTS), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Damage).To(HaveLen(2))
				want := make([]cesium.Range, 0, 4)
				for _, key := range []cesium.ChannelKey{newIdx, newData} {
					for _, first := range []telem.TimeStamp{6, 16} {
						want = append(want, cesium.Range{
							Key:     key,
							Segment: segmentIDs(fs, key)[0],
							Bytes:   40,
							Samples: 5,
							TimeRange: telem.TimeRange{
								Start: first * telem.SecondTS,
								End:   (first+4)*telem.SecondTS + 1,
							},
						})
					}
				}
				Expect(rep.Rebuilt).To(Equal(want))
				Expect(rep.Unplaced).To(BeEmpty())
				Expect(rep.Skipped).To(BeEmpty())
			})

		It(
			"Should raise the limits above the values of the later entries when the bad "+
				"frame held a Reserve",
			func(ctx SpecContext) {
				frames := framesOf(fs, id)
				i := frameWith(frames, func(rec log.Record) bool {
					_, ok := rec.(*log.Reserve)
					return ok
				})
				spoil(i, 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Damage).To(HaveLen(1))
				Expect(rep.Lost).To(ContainElement(HaveField("Kind", "Reserve")))
				Expect(rep.Skipped).To(BeEmpty())
			},
		)

		It("Should start a rebuilt range at the committed length, so a delete of the "+
			"tail of its last pointer stays cut", func(ctx SpecContext) {
			db := MustSucceed(open(ctx))
			stop := GoRun(ctx, db.Run)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      newIdx,
					Name:     "new_time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      newData,
					Name:     "new_data",
					Index:    newIdx,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			keys := []cesium.ChannelKey{newIdx, newData}
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Start:    telem.SecondTS,
			}))
			// put writes n samples from stamp from seconds, and commits them when
			// committed is true.
			put := func(from, n int, committed bool) {
				stamps := make([]telem.TimeStamp, n)
				for i := range stamps {
					stamps[i] = telem.TimeStamp(from+i) * telem.SecondTS
				}
				MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeries(stamps),
					telem.NewSeries(make([]int64, n)),
				}), nil))
				if committed {
					MustSucceed2(w.Commit(ctx, nil))
					Expect(w.Sync()).To(Succeed())
				}
			}
			put(1, 5, true)
			put(6, 5, true)
			// Its full first block reaches the files, and no record names it.
			put(11, 10_000, false)
			Expect(w.Close()).To(Succeed())
			Expect(
				db.DeleteTimeRange(ctx, []cesium.ChannelKey{newData}, telem.TimeRange{
					Start: 9 * telem.SecondTS,
					End:   11 * telem.SecondTS,
				}),
			).To(Succeed())
			write(ctx, db, 100*telem.SecondTS)
			Expect(errors.Join(stop(), db.Close())).To(Succeed())
			spoil(commitFrame(106*telem.SecondTS), 4)
			rep := MustSucceed(repair(ctx, new(int)))
			ofNew := func(ranges []cesium.Range) []cesium.Range {
				return slices.DeleteFunc(ranges, func(r cesium.Range) bool {
					return r.Key != newIdx && r.Key != newData
				})
			}
			segs := [2]uint64{segmentIDs(fs, newIdx)[0], segmentIDs(fs, newData)[0]}
			Expect(ofNew(rep.Rebuilt)).To(Equal([]cesium.Range{{
				Key:     newIdx,
				Segment: segs[0],
				Bytes:   65_456,
				Samples: 8_182,
				TimeRange: telem.TimeRange{
					Start: 11 * telem.SecondTS,
					End:   8_192*telem.SecondTS + 1,
				},
				Open: true,
			}}))
			Expect(ofNew(rep.Unplaced)).To(Equal([]cesium.Range{{
				Key:     newData,
				Segment: segs[1],
				Bytes:   65_456,
				Samples: 8_182,
				Open:    true,
				Reason:  "no later record ends it, so its count is not known",
			}}))
		})

		It(
			"Should end a lost commit at the length of a later Seal",
			func(ctx SpecContext) {
				db := MustSucceed(open(ctx))
				stop := GoRun(ctx, db.Run)
				write(ctx, db, 100*telem.SecondTS)
				Expect(errors.Join(stop(), db.Close())).To(Succeed())
				spoil(commitFrame(16*telem.SecondTS), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Rebuilt).To(Equal(pair(5, 16*telem.SecondTS)))
				Expect(rep.Unplaced).To(BeEmpty())
				Expect(rep.Skipped).To(BeEmpty())
			},
		)

		It("Should list as a tail the unchecked last block of a lost last commit",
			func(ctx SpecContext) {
				spoil(commitFrame(16*telem.SecondTS), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Rebuilt).To(BeEmpty())
				Expect(rep.Unplaced).To(BeEmpty())
				tails := pair(5, 16*telem.SecondTS)
				for i := range tails {
					tails[i].Samples, tails[i].TimeRange = 0, telem.TimeRange{}
					tails[i].Open, tails[i].Reason = true, tail
				}
				Expect(rep.Tails).To(Equal(tails))
			})

		It("Should place the clock range of a lost last commit up to its last checked "+
			"block, and not its data", func(ctx SpecContext) {
			writeNew(ctx, 10_000, 2)
			spoil(commitFrame(10_001*telem.SecondTS), 4)
			rep := MustSucceed(repair(ctx, new(int)))
			segs := [2]uint64{segmentIDs(fs, newIdx)[0], segmentIDs(fs, newData)[0]}
			Expect(rep.Rebuilt).To(Equal([]cesium.Range{{
				Key:       newIdx,
				Segment:   segs[0],
				Bytes:     51_072,
				Samples:   6_384,
				TimeRange: lateRange,
				Open:      true,
			}}))
			Expect(rep.Unplaced).To(Equal([]cesium.Range{{
				Key:     newData,
				Segment: segs[1],
				Bytes:   51_072,
				Samples: 6_384,
				Open:    true,
				Reason:  "no later record ends it, so its count is not known",
			}}))
			Expect(rep.Tails).To(Equal([]cesium.Range{
				{
					Key:     newIdx,
					Segment: segs[0],
					Bytes:   28_928,
					Open:    true,
					Reason:  tail,
				},
				{
					Key:     newData,
					Segment: segs[1],
					Bytes:   28_928,
					Open:    true,
					Reason:  tail,
				},
			}))
		})

		DescribeTable(
			"Should place no range of a clock write whose clock stamps do not "+
				"rise inside the pointers around them",
			func(ctx SpecContext, at int) {
				writeNew(ctx, 10_000, 3)
				spoil(commitFrame(10_001*telem.SecondTS), 4)
				segs := [2]uint64{segmentIDs(fs, newIdx)[0], segmentIDs(fs, newData)[0]}
				name := fmt.Sprintf("%d/%016x.seg", newIdx, segs[0])
				b := MustSucceed(xfs.ReadFile(fs, name))
				// The stamp at stream offset at, in block 1, goes to 5 s, and the block
				// gets
				// the checksum of its new bytes.
				const slot = 65_536 + 4
				binary.LittleEndian.PutUint64(
					b[slot+at-65_536:],
					uint64(5*telem.SecondTS),
				)
				sum := place.Segment(id, segs[0], slot, 1,
					uint8(telem.DataTypeCodeTimestamp), newIdx).
					Sum(b[slot : slot+65_536])
				binary.LittleEndian.PutUint32(b[slot+65_536:], sum)
				Expect(xfs.WriteFile(fs, name, b)).To(Succeed())
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Rebuilt).To(BeEmpty())
				rise := "the stamps of its clock do not rise inside the pointers around them"
				Expect(rep.Unplaced).To(Equal([]cesium.Range{
					{
						Key:     newIdx,
						Segment: segs[0],
						Bytes:   80_000,
						Samples: 10_000,
						Reason:  rise,
					},
					{
						Key:     newData,
						Segment: segs[1],
						Bytes:   80_000,
						Samples: 10_000,
						Reason:  rise,
					},
				}))
			},
			Entry("a stamp that falls", 90_000),
			Entry("a first stamp before the pointer before it", 80_000),
		)

		It("Should move each segment to the end of its gap, where a later Seal ends it",
			func(ctx SpecContext) {
				writeNew(ctx, 10_000, 2)
				db := MustSucceed(open(ctx))
				stop := GoRun(ctx, db.Run)
				keys := []cesium.ChannelKey{newIdx, newData}
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    30_000 * telem.SecondTS,
				}))
				MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeries([]telem.TimeStamp{30_000 * telem.SecondTS}),
					telem.NewSeriesV[int64](1),
				}), nil))
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Close()).To(Succeed())
				Expect(errors.Join(stop(), db.Close())).To(Succeed())
				spoil(newFrame(10_001*telem.SecondTS), 4)
				seg := segmentIDs(fs, newData)[0]
				name := fmt.Sprintf("%d/%016x.seg", newData, seg)
				b := MustSucceed(xfs.ReadFile(fs, name))
				b[2*(65_536+4)+100] ^= 0xFF
				Expect(xfs.WriteFile(fs, name, b)).To(Succeed())
				rep := MustSucceed(repair(ctx, new(int)))
				// The data segment ends at the Seal's length with the checksum of its
				// changed bytes, so only its Seal breaks a rule.
				Expect(rep.Skipped).To(HaveLen(1))
				Expect(rep.Skipped[0]).To(MatchFields(IgnoreExtras, Fields{
					"Kind":   Equal("Seal"),
					"Reason": ContainSubstring("not its committed 160000"),
				}))
			})

		It("Should end a clock write at the shortest range of it in checked blocks",
			func(ctx SpecContext) {
				writeNew(ctx, 10_000, 3)
				spoil(commitFrame(10_001*telem.SecondTS), 4)
				segs := [2]uint64{segmentIDs(fs, newIdx)[0], segmentIDs(fs, newData)[0]}
				name := fmt.Sprintf("%d/%016x.seg", newData, segs[1])
				b := MustSucceed(xfs.ReadFile(fs, name))
				b[2*(65_536+4)+100] ^= 0xFF
				Expect(xfs.WriteFile(fs, name, b)).To(Succeed())
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Rebuilt).To(Equal([]cesium.Range{
					{
						Key:       newIdx,
						Segment:   segs[0],
						Bytes:     51_072,
						Samples:   6_384,
						TimeRange: lateRange,
					},
					{
						Key:       newData,
						Segment:   segs[1],
						Bytes:     51_072,
						Samples:   6_384,
						TimeRange: lateRange,
					},
				}))
				Expect(rep.Unplaced).To(Equal([]cesium.Range{
					{
						Key:     newIdx,
						Segment: segs[0],
						Bytes:   28_928,
						Samples: 3_616,
						Reason: "past the end of the samples that every range of its " +
							"clock write holds in checked blocks",
					},
					{
						Key:     newData,
						Segment: segs[1],
						Bytes:   28_928,
						Samples: 3_616,
						Reason:  "from a block that fails its check",
					},
				}))
			})
	})

	Describe("Removed", func() {
		It("Should take out of the state a sealed segment whose file is gone",
			func(ctx SpecContext) {
				db := MustSucceed(open(ctx))
				stop := GoRun(ctx, db.Run)
				write(ctx, db, 100*telem.SecondTS)
				Expect(errors.Join(stop(), db.Close())).To(Succeed())
				seg := segmentIDs(fs, idx)[0]
				Expect(fs.Remove(fmt.Sprintf("%d/%016x.seg", idx, seg))).To(Succeed())
				frames := framesOf(fs, id)
				spoil(plainFrame(frames, len(frames)-3), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Damage).To(HaveLen(1))
				Expect(
					rep.Removed,
				).To(Equal([]cesium.Segment{{Key: idx, Segment: seg}}))
			})

		It("Should take out of the state each segment of a key whose directory is gone",
			func(ctx SpecContext) {
				seg := segmentIDs(fs, data)[0]
				Expect(fs.Remove(strconv.Itoa(int(data)))).To(Succeed())
				spoil(plainFrame(framesOf(fs, id), 2), 4)
				rep := MustSucceed(repair(ctx, new(int)))
				Expect(rep.Damage).To(HaveLen(1))
				Expect(
					rep.Removed,
				).To(Equal([]cesium.Segment{{Key: data, Segment: seg}}))
			})
	})

	It(
		"Should list a sealed segment of the head with no pointer, which an open refuses",
		func(ctx SpecContext) {
			fs, id = xfs.NewMem(), 0
			db := MustSucceed(open(ctx))
			id = db.Identity()
			Expect(db.Close()).To(Succeed())
			Expect(fs.Remove(checkpoint.Name(1))).To(Succeed())
			MustSucceed(checkpoint.Create(fs, id, 1, &state.Snapshot{
				Limits: state.Limits{Segments: 10, Alignments: 10},
				Channels: []*state.Channel{{Key: data, Segments: []state.Segment{{
					ID:       3,
					Clock:    idx,
					Version:  1,
					DataType: uint8(telem.DataTypeCodeInt64),
					Sealed:   true,
				}}}},
			}, 0))
			Expect(open(ctx)).Error().
				To(MatchError(ContainSubstring("sealed with no pointer")))
			before := rootFiles(fs)
			rep := MustSucceed(repair(ctx, new(int)))
			Expect(rep.Damage).To(BeEmpty())
			Expect(rep.Buried).To(Equal([]cesium.Segment{{Key: data, Segment: 3}}))
			Expect(rootFiles(fs)).To(Equal(before))
		},
	)

	It("Should refuse a root of the old layout", func(ctx SpecContext) {
		fs = xfs.NewMem()
		Expect(xfs.WriteFile(MustSucceed(fs.Sub("1")), "meta.json", []byte("{}"))).
			To(Succeed())
		Expect(repair(ctx, new(int))).Error().To(MatchError(errors.ErrUnsupported))
	})

	It("Should refuse another identity than the root's, and undo nothing",
		func(ctx SpecContext) {
			before := rootFiles(fs)
			id++
			Expect(repair(ctx, new(int))).Error().
				To(MatchError(ContainSubstring("which the caller holds")))
			Expect(rootFiles(fs)).To(Equal(before))
		})
})
