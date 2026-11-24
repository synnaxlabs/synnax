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
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// switchLog appends Intends of 1,000 keys each to db on fs until the log switches to
// generation, which asks for the file of the log after it. It first waits for the file
// of generation, since the live log goes on past three quarters until that file is
// ready. The checkpointer writes the head of the switch after it returns.
func switchLog(db *cesium.DB, fs xfs.FS, generation uint64) error {
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(time.Millisecond) {
		info, err := fs.Stat(log.Name(generation))
		if err == nil && info.Size() == log.FileSize {
			break
		}
		if time.Now().After(deadline) {
			return errors.Newf("the file of log %d is not ready: %v", generation, err)
		}
	}
	keys := make([]cesium.ChannelKey, 1_000)
	for i := range keys {
		keys[i] = cesium.ChannelKey(1_000_000 + i)
	}
	for range log.FileSize / 16_000 {
		if switched, err := fs.Exists(
			log.Name(generation + 1),
		); err != nil ||
			switched {
			return err
		}
		if err := db.Intend(keys); err != nil {
			return err
		}
	}
	return errors.Newf("the log did not switch to %d", generation)
}

// waitHead waits for the head of generation to read whole from fs.
func waitHead(fs xfs.FS, generation uint64) {
	GinkgoHelper()
	Eventually(func() error {
		_, _, err := checkpoint.Read(fs, generation)
		return err
	}).WithTimeout(30 * time.Second).Should(Succeed())
}

// gateFS makes the first open of a part file after arm wait until the spec closes
// release.
type gateFS struct {
	xfs.FS
	mu               sync.Mutex
	armed            bool
	waiting, release chan struct{}
}

func (f *gateFS) Sub(name string) (xfs.FS, error) {
	if name != "" {
		return f.FS.Sub(name)
	}
	return f, nil
}

func (f *gateFS) Open(name string, flag int) (xfs.File, error) {
	f.mu.Lock()
	held := f.armed && strings.HasSuffix(name, ".part")
	waiting, release := f.waiting, f.release
	f.armed = f.armed && !held
	f.mu.Unlock()
	if held {
		waiting <- struct{}{}
		<-release
	}
	return f.FS.Open(name, flag)
}

func (f *gateFS) arm() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed, f.waiting, f.release = true, make(chan struct{}, 1), make(chan struct{})
}

// partedDB opens a database of the pointer log layout on fs whose switch writes every
// sealed segment to a part.
func partedDB(ctx context.Context, fs xfs.FS) *cesium.DB {
	GinkgoHelper()
	return MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithPointerLog(),
		cesium.WithPartSize(1),
	))
}

// writeParted writes writers stretches of samples of data, timed by idx, to db, then
// switches log 1, so the samples move to parts. It returns the values of data.
func writeParted(
	ctx context.Context,
	db *cesium.DB,
	fs xfs.FS,
	idx, data cesium.ChannelKey,
	writers, samples int,
) []int64 {
	GinkgoHelper()
	values := make([]int64, 0, writers*samples)
	for w := range writers {
		stamps := make([]telem.TimeStamp, samples)
		vals := make([]int64, samples)
		for i := range samples {
			stamps[i] = telem.TimeStamp(w*samples+i+1) * telem.SecondTS
			vals[i] = int64(w*samples + i)
		}
		values = append(values, vals...)
		writer := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Clock:    idx,
			Start:    stamps[0],
		}))
		MustSucceed2(writer.Write(ctx, telem.MultiFrame(
			[]cesium.ChannelKey{idx, data},
			[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(vals)},
		), nil))
		MustSucceed2(writer.Commit(ctx, nil))
		Expect(writer.Close()).To(Succeed())
	}
	Expect(switchLog(db, fs, 2)).To(Succeed())
	waitHead(fs, 2)
	Expect(partFile(fs)).ToNot(BeEmpty())
	return values
}

// partFile returns the name of a part file in the root of fs, or "".
func partFile(fs xfs.FS) string {
	GinkgoHelper()
	for _, info := range MustSucceed(fs.List("")) {
		if strings.HasSuffix(info.Name(), ".part") {
			return info.Name()
		}
	}
	return ""
}

// flipByte changes one byte at off of the file name in fs.
func flipByte(fs xfs.FS, name string, off int64) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_RDWR))
	b := make([]byte, 1)
	MustSucceed(f.ReadAt(b, off))
	b[0] ^= 0xFF
	MustSucceed(f.WriteAt(b, off))
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("Reads through parts", func() {
	It("Should read the samples of segments that a part holds through the part "+
		"cache", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db := partedDB(ctx, fs)
		stop := StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
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
		values := writeParted(ctx, db, fs, idx, data, 3, 10)
		stop()
		Expect(db.Close()).To(Succeed())

		db = partedDB(ctx, fs)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		// A writer's open waits for the state owner, which applies the tags of the
		// parts that the open loaded first.
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Clock:    idx,
			Start:    telem.TimeStamp(1_000_000) * telem.SecondTS,
		}))
		Expect(w.Close()).To(Succeed())

		part := partFile(fs)
		flipByte(fs, part, checkpoint.BodyStart+40)
		Expect(db.Read(ctx, telem.TimeRangeMax, data)).Error().
			To(MatchError(place.ErrCorrupt))
		flipByte(fs, part, checkpoint.BodyStart+40)
		frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
		got := make([]int64, 0, len(values))
		for _, s := range frame.Get(data).Series {
			got = append(got, s.Unmarshal[int64]()...)
		}
		Expect(got).To(Equal(values))
	})

	It("Should read what a delete over parted segments leaves, before and after the "+
		"switch that folds its kills, and after a reopen", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db := partedDB(ctx, fs)
		stop := StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
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
		values := writeParted(ctx, db, fs, idx, data, 3, 10)
		stop()
		Expect(db.Close()).To(Succeed())
		db = partedDB(ctx, fs)
		stop = StartRun(db.Run)
		// Samples 4 to 23 sit at seconds 5 to 24, across all three parted segments.
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, telem.TimeRange{
			Start: 5 * telem.SecondTS,
			End:   25 * telem.SecondTS,
		})).To(Succeed())
		want := append(slices.Clone(values[:4]), values[24:]...)
		read := func() []int64 {
			GinkgoHelper()
			frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
			got := make([]int64, 0, len(want))
			for _, s := range frame.Get(data).Series {
				got = append(got, s.Unmarshal[int64]()...)
			}
			return got
		}
		Expect(read()).To(Equal(want))
		Expect(switchLog(db, fs, 3)).To(Succeed())
		waitHead(fs, 3)
		Eventually(read).Should(Equal(want))
		stop()
		Expect(db.Close()).To(Succeed())
		db = partedDB(ctx, fs)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		Expect(read()).To(Equal(want))
	})

	It("Should read a part that two switches leave out while a load that pins it "+
		"reads it", func(ctx SpecContext) {
		gate := &gateFS{FS: xfs.NewMem()}
		db := partedDB(ctx, gate)
		stop := StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
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
		writeParted(ctx, db, gate, idx, data, 2, 15)
		part := partFile(gate)
		stop()
		Expect(db.Close()).To(Succeed())

		db = partedDB(ctx, gate)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{idx},
		}))
		gate.arm()
		sought := make(chan bool, 1)
		go func() { sought <- iter.SeekFirst() }()
		Eventually(gate.waiting).WithTimeout(10 * time.Second).Should(Receive())
		// The kills of the data records leave the part half dead, so switch 3 writes it
		// again, and switch 4 leaves the last head that names it.
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, telem.TimeRangeMax)).
			To(Succeed())
		for generation := uint64(3); generation <= 4; generation++ {
			Expect(switchLog(db, gate, generation)).To(Succeed())
			waitHead(gate, generation)
		}
		Expect(gate.Exists(part)).To(BeTrue())
		close(gate.release)
		Expect(<-sought).To(BeTrue())
		samples := 0
		for iter.Next(cesium.AutoSpan) {
			for _, s := range iter.Value().Get(idx).Series {
				samples += int(s.Len())
			}
		}
		Expect(iter.Error()).ToNot(HaveOccurred())
		Expect(samples).To(Equal(30))
		Expect(iter.Close()).To(Succeed())
		// With the pin gone, the next switch removes the part.
		Expect(switchLog(db, gate, 5)).To(Succeed())
		waitHead(gate, 5)
		Eventually(func() (bool, error) { return gate.Exists(part) }).Should(BeFalse())
	})
})

var _ = Describe("Claims over parts", func() {
	It("Should refuse a writer that starts in parted data, and a commit that reaches "+
		"it", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db := partedDB(ctx, fs)
		stop := StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
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
		// The samples sit at seconds 1 to 30 in three segments.
		writeParted(ctx, db, fs, idx, data, 3, 10)
		stop()
		Expect(db.Close()).To(Succeed())
		db = partedDB(ctx, fs)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		DeferCleanup(StartRun(db.Run))
		Expect(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{idx, data},
			Start:    15 * telem.SecondTS,
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("inside its stored data from 11000000000")),
		))
		Expect(db.Write(ctx, telem.SecondTS/2, telem.MultiFrame(
			[]cesium.ChannelKey{idx, data},
			[]telem.Series{
				telem.NewSeries(
					[]telem.TimeStamp{telem.SecondTS / 2, 2 * telem.SecondTS},
				),
				telem.NewSeries([]int64{1, 2}),
			},
		))).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("overlaps its stored data from 1000000000")),
		))
	})
})
