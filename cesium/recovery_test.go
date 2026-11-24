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
	"strconv"
	"strings"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	xerrors "github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Recovery", func() {
	var fs xfs.FS
	BeforeEach(func() { fs = xfs.NewMem() })
	open := func(ctx context.Context) (*cesium.DB, error) {
		return cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		)
	}
	// cycle opens the database, runs it, and closes it.
	cycle := func(ctx context.Context) {
		GinkgoHelper()
		db := MustSucceed(open(ctx))
		stop := StartRun(db.Run)
		stop()
		Expect(db.Close()).To(Succeed())
	}
	// frameOffsets returns the offsets of the frames of log 1, which hold only marks.
	frameOffsets := func() []int64 {
		GinkgoHelper()
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDONLY))
		defer func() { Expect(f.Close()).To(Succeed()) }()
		var offsets []int64
		b := make([]byte, 4)
		for off := int64(log.FramesStart); off < 1<<20; off += log.BlockSize {
			MustSucceed(f.ReadAt(b, off))
			if binary.LittleEndian.Uint32(b) == 0x464C5043 {
				offsets = append(offsets, off)
			}
		}
		return offsets
	}
	// frameHeader returns the bytes of the frame header at off of log 1.
	frameHeader := func(off int64) []byte {
		GinkgoHelper()
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDONLY))
		defer func() { Expect(f.Close()).To(Succeed()) }()
		b := make([]byte, log.FrameHeaderSize)
		MustSucceed(f.ReadAt(b, off))
		return b
	}
	flip := func(off int64) {
		GinkgoHelper()
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		b := make([]byte, 1)
		MustSucceed(f.ReadAt(b, off))
		b[0] ^= 0xFF
		MustSucceed(f.WriteAt(b, off))
		Expect(f.Close()).To(Succeed())
	}

	It("Should open a new database in an empty root, and open it again", func(
		ctx SpecContext,
	) {
		cycle(ctx)
		Expect(fs.Exists("0000000000000001.head")).To(BeTrue())
		Expect(fs.Exists(log.Name(1))).To(BeTrue())
		cycle(ctx)
		Expect(frameOffsets()).To(HaveLen(2))
	})

	It("Should close a database that never ran", func(ctx SpecContext) {
		db := MustSucceed(open(ctx))
		Expect(db.Close()).To(Succeed())
		cycle(ctx)
	})

	It("Should refuse a root that holds the old layout", func(ctx SpecContext) {
		old := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fs)))
		Expect(old.CreateChannel(ctx, cesium.Channel{
			Key:      1,
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
		Expect(old.Close()).To(Succeed())
		Expect(open(ctx)).Error().To(And(
			MatchError(xerrors.ErrUnsupported),
			MatchError(ContainSubstring("cannot open a root of the old layout")),
		))
	})

	It("Should read a log of synced frames at most twice at open", func(
		ctx SpecContext,
	) {
		for range 8 {
			cycle(ctx)
		}
		Expect(frameOffsets()).To(HaveLen(8))
		info := MustSucceed(fs.Stat(log.Name(1)))
		var read atomic.Int64
		fs = logReadFS{FS: fs, read: &read}
		cycle(ctx)
		Expect(read.Load()).To(BeNumerically("<=", 2*info.Size()+log.BlockSize))
	})

	DescribeTable(
		"Should read a frame of entries again when its first read fails its check",
		func(ctx SpecContext, at int64) {
			db := MustSucceed(open(ctx))
			stop := StartRun(db.Run)
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      1,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{1},
				Start:    telem.SecondTS,
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(
				[]cesium.ChannelKey{1},
				[]telem.Series{telem.NewSeriesSecondsTSV(1, 2)},
			), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			stop()
			Expect(db.Close()).To(Succeed())
			var entries int64
			for _, off := range frameOffsets() {
				if size, ok := log.FrameSize(frameHeader(off)); ok &&
					size > log.FrameHeaderSize {
					entries = off
					break
				}
			}
			Expect(entries).ToNot(BeZero())
			fs = damageFS{FS: fs, at: entries + at, done: &atomic.Bool{}}
			db = MustSucceed(open(ctx))
			DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
			DeferCleanup(StartRun(db.Run))
			fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, 1))
			Expect(fr.Get(1).Series[0]).To(
				MatchSeriesData(telem.NewSeriesSecondsTSV(1, 2)),
			)
		},
		Entry("in the header", int64(1)),
		Entry("in the entries", int64(log.FrameHeaderSize)),
	)

	It("Should make each new log file the size of the log file size option", func(
		ctx SpecContext,
	) {
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithLogFileSize(telem.Size(log.MinFileSize)),
		))
		stop := StartRun(db.Run)
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key:      1,
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{1},
			Start:    telem.SecondTS,
		}))
		// Each commit syncs a frame of one block, so 1,000 commits pass three quarters
		// of the log and switch it.
		for i := range 1_000 {
			stamp := telem.TimeStamp(i+1) * telem.SecondTS
			MustSucceed2(w.Write(ctx, telem.UnaryFrame(
				cesium.ChannelKey(1),
				telem.NewSeries([]telem.TimeStamp{stamp}),
			), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
		Eventually(func() (bool, error) { return fs.Exists(log.Name(2)) }).
			Should(BeTrue())
		stop()
		Expect(db.Close()).To(Succeed())
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithLogFileSize(telem.Size(log.MinFileSize)),
		))
		StartRun(db.Run)()
		Expect(db.Close()).To(Succeed())
		var logs int
		for _, info := range MustSucceed(fs.List("")) {
			if strings.HasSuffix(info.Name(), ".log") {
				logs++
				Expect(info.Size()).To(Equal(int64(log.MinFileSize)), info.Name())
			}
		}
		Expect(logs).To(BeNumerically(">=", 2))
	})

	DescribeTable("Should refuse a log file size that is not whole blocks of 4 MiB "+
		"or more", func(ctx SpecContext, size telem.Size) {
		Expect(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithLogFileSize(size),
		)).Error().To(MatchError(validate.ErrValidation))
		Expect(fs.List("")).To(BeEmpty())
	},
		Entry("under 4 MiB", telem.Size(log.MinFileSize-log.BlockSize)),
		Entry("not whole blocks", telem.Size(log.MinFileSize+1)),
	)

	It("Should cut a torn tail after the last frame", func(ctx SpecContext) {
		cycle(ctx)
		last := frameOffsets()[0]
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		MustSucceed(f.WriteAt([]byte("torn bytes of a frame"), last+log.BlockSize))
		Expect(f.Close()).To(Succeed())
		cycle(ctx)
	})

	It("Should refuse a log with a valid frame after a bad one", func(ctx SpecContext) {
		cycle(ctx)
		cycle(ctx)
		offsets := frameOffsets()
		Expect(offsets).To(HaveLen(2))
		flip(offsets[0] + 8)
		Expect(open(ctx)).Error().To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("holds a valid frame at")),
		))
	})

	It(
		"Should refuse a log with a valid frame after a bad one that only a second read "+
			"finds",
		func(ctx SpecContext) {
			cycle(ctx)
			cycle(ctx)
			offsets := frameOffsets()
			Expect(offsets).To(HaveLen(2))
			flip(offsets[0] + 8)
			fs = damageFS{FS: fs, at: offsets[1] + 8, done: &atomic.Bool{}}
			Expect(open(ctx)).Error().To(And(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring("holds a valid frame at")),
			))
		},
	)

	It("Should refuse a log whose valid frame after its last one only a second read "+
		"finds, across the end of a block of that read", func(ctx SpecContext) {
		// block is the size of the blocks of the second read.
		const block = 4 << 20
		cycle(ctx)
		cycle(ctx)
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		b := make([]byte, 1<<20)
		MustSucceed(f.ReadAt(b, 0))
		end := int64(0)
		for off := int64(log.FramesStart); off+log.FrameHeaderSize <= int64(len(b)); off += 8 {
			if n, ok := log.FrameSize(b[off:]); ok {
				end = off + int64(n)
			}
		}
		Expect(end).ToNot(BeZero())
		// The frame starts in the last 64 bytes of the first block from the end of the
		// last frame, and ends past it.
		at := (end + block - 64) &^ 7
		frame := log.Frame{Count: 1, Sequence: 1, Entries: make([]byte, 256)}
		MustSucceed(f.WriteAt(
			frame.Append(nil, binary.LittleEndian.Uint64(b[16:]), 1, uint64(at)),
			at,
		))
		Expect(f.Close()).To(Succeed())
		fs = tailFS{FS: fs, from: at, done: &atomic.Bool{}}
		Expect(open(ctx)).Error().To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("holds a valid frame at "+strconv.FormatInt(
				at,
				10,
			))),
		))
	})

	It("Should start a new database over a head 1 that is not complete and a log 1 "+
		"with no frame", func(ctx SpecContext) {
		cycle(ctx)
		f := MustSucceed(fs.Open("0000000000000001.head", os.O_RDWR))
		MustSucceed(f.WriteAt(make([]byte, 512), 0))
		Expect(f.Close()).To(Succeed())
		g := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		MustSucceed(g.WriteAt(make([]byte, 1<<20), 0))
		Expect(g.Close()).To(Succeed())
		cycle(ctx)
	})

	It("Should refuse a head 1 that is not complete when log 1 holds a frame", func(
		ctx SpecContext,
	) {
		cycle(ctx)
		f := MustSucceed(fs.Open("0000000000000001.head", os.O_RDWR))
		MustSucceed(f.WriteAt(make([]byte, 512), 0))
		Expect(f.Close()).To(Succeed())
		Expect(open(ctx)).Error().To(MatchError(place.ErrCorrupt))
	})
})

// logReadFS counts the bytes read from log 1.
type logReadFS struct {
	xfs.FS
	read *atomic.Int64
}

func (f logReadFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil || name != "" {
		return sub, err
	}
	return logReadFS{FS: sub, read: f.read}, nil
}

func (f logReadFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != log.Name(1) {
		return file, err
	}
	return logReadFile{File: file, read: f.read}, nil
}

type logReadFile struct {
	xfs.File
	read *atomic.Int64
}

func (f logReadFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	f.read.Add(int64(n))
	return n, err
}

// tailFS zeroes the bytes of log 1 from from on in the first read that covers from.
type tailFS struct {
	xfs.FS
	from int64
	done *atomic.Bool
}

func (f tailFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil || name != "" {
		return sub, err
	}
	return tailFS{FS: sub, from: f.from, done: f.done}, nil
}

func (f tailFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != log.Name(1) {
		return file, err
	}
	return tailFile{File: file, fs: f}, nil
}

type tailFile struct {
	xfs.File
	fs tailFS
}

func (f tailFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	covers := f.fs.from >= off && f.fs.from < off+int64(n)
	if covers && f.fs.done.CompareAndSwap(false, true) {
		clear(p[f.fs.from-off : n])
	}
	return n, err
}

// damageFS damages the byte at of log 1 in the first read that covers it.
type damageFS struct {
	xfs.FS
	at   int64
	done *atomic.Bool
}

func (f damageFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil || name != "" {
		return sub, err
	}
	return damageFS{FS: sub, at: f.at, done: f.done}, nil
}

func (f damageFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != log.Name(1) {
		return file, err
	}
	return damageFile{File: file, fs: f}, nil
}

type damageFile struct {
	xfs.File
	fs damageFS
}

func (f damageFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	covers := f.fs.at >= off && f.fs.at < off+int64(n)
	if covers && f.fs.done.CompareAndSwap(false, true) {
		p[f.fs.at-off] ^= 0xFF
	}
	return n, err
}
