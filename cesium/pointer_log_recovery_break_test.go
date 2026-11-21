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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// brkLogEnd walks the valid frames of the log of generation in b as recovery does, and
// returns the database, the end of the last valid frame, the sequence after it, and its
// crc field.
func brkLogEnd(
	b []byte,
	generation uint64,
) (db uint64, end int64, seq uint64, prev uint32) {
	GinkgoHelper()
	db = binary.LittleEndian.Uint64(b[16:])
	h := MustSucceed(log.DecodeHeader(b, db, generation, 0))
	seq, prev, end = h.Sequence, binary.LittleEndian.Uint32(b[60:64]), log.FramesStart
	for {
		found := false
		for _, off := range []int64{end, (end + log.BlockSize - 1) &^ (log.BlockSize - 1)} {
			f, err := log.DecodeFrame(b[off:], db, generation, uint64(off))
			if err == nil && f.Sequence == seq && f.PrevCRC == prev {
				end, seq, prev, found = off+int64(f.Size()), f.Sequence+uint64(f.Count),
					f.CRC, true
				break
			}
		}
		if !found {
			return db, end, seq, prev
		}
	}
}

func brkReadFile(fs xfs.FS, name string) []byte {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	b := make([]byte, MustSucceed(f.Stat()).Size())
	MustSucceed(f.ReadAt(b, 0))
	return b
}

// brkHookFS calls after once each WriteAt to a file whose name ends with suffix
// returns.
type brkHookFS struct {
	xfs.FS
	suffix string
	after  func()
}

func (f brkHookFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || len(name) < len(f.suffix) ||
		name[len(name)-len(f.suffix):] != f.suffix {
		return file, err
	}
	return brkHookFile{File: file, after: f.after}, nil
}

func (f brkHookFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return brkHookFS{FS: sub, suffix: f.suffix, after: f.after}, nil
}

type brkHookFile struct {
	xfs.File
	after func()
}

func (f brkHookFile) WriteAt(p []byte, off int64) (int, error) {
	n, err := f.File.WriteAt(p, off)
	f.after()
	return n, err
}

var _ = Describe("Pointer log recovery breaker", func() {
	open := func(ctx context.Context, fs xfs.FS) (*cesium.DB, error) {
		return cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		)
	}
	cycle := func(ctx context.Context, fs xfs.FS, work func(db *cesium.DB)) {
		GinkgoHelper()
		db := MustSucceed(open(ctx, fs))
		stop := StartRun(db.Run)
		if work != nil {
			work(db)
		}
		stop()
		Expect(db.Close()).To(Succeed())
	}

	It("Should put the Restart in the live log when the next log cannot be made", func(
		ctx SpecContext,
	) {
		disk := MustSucceed(fault.New(fault.Config{}))
		cycle(ctx, disk, nil)
		// The next log is missing, as when a full disk kept the checkpointer from
		// making it, and log 1 holds marks past three quarters, the last one not clean.
		Expect(disk.Remove(log.Name(2))).To(Succeed())
		Expect(disk.SyncDir("")).To(Succeed())
		f := MustSucceed(disk.Open(log.Name(1), os.O_RDWR))
		db, end, seq, prev := brkLogEnd(brkReadFile(disk, log.Name(1)), 1)
		for end <= log.FileSize/4*3+log.BlockSize {
			start := (end + log.BlockSize - 1) &^ (log.BlockSize - 1)
			mark := log.Frame{Sequence: seq, PrevCRC: prev}
			b := mark.Append(nil, db, 1, uint64(start))
			MustSucceed(f.WriteAt(b, start))
			end, prev = start+int64(len(b)), mark.CRC
		}
		Expect(f.Sync()).To(Succeed())
		Expect(f.Close()).To(Succeed())
		var used int64
		for _, info := range MustSucceed(disk.List("")) {
			used += info.Size()
		}
		full := MustSucceed(disk.Crash(fault.CrashConfig{
			Image: fault.Config{Capacity: used + 64<<10},
		}))
		cycle(ctx, full, nil)
	})

	It("Should open after a crash that left head 1 created and not yet sized", func(
		ctx SpecContext,
	) {
		fs := xfs.NewMem()
		cycle(ctx, fs, nil)
		// A crash in the first open, after the head file was created and before its
		// Truncate was durable: log 1 holds its prepared zeros, and head 1 no byte.
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		MustSucceed(f.WriteAt(make([]byte, 1<<20), 0))
		Expect(f.Close()).To(Succeed())
		Expect(fs.Remove(log.Name(2))).To(Succeed())
		h := MustSucceed(fs.Open("0000000000000001.head", os.O_RDWR))
		Expect(h.Truncate(0)).To(Succeed())
		Expect(h.Close()).To(Succeed())
		cycle(ctx, fs, nil)
	})

	DescribeTable("Should open with its channel list through one-time read faults",
		func(ctx SpecContext, runKey uint64) {
			disk := MustSucceed(fault.New(fault.Config{}))
			chs := []cesium.Channel{
				{Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT},
				{Key: 2, Name: "data", Index: 1, DataType: telem.Int64T},
			}
			cycle(ctx, disk, func(db *cesium.DB) {
				Expect(db.CreateChannel(ctx, chs...)).To(Succeed())
			})
			// Each first read of a block gets one flipped bit, and a second read of
			// the same bytes is good, as fault 5a reads.
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				Image: fault.Config{RunKey: runKey, ReadCorruptionRate: 1},
			}))
			cycle(ctx, image, func(db *cesium.DB) {
				got := MustSucceed(db.RetrieveChannels(ctx, 1, 2))
				for i, ch := range got {
					Expect(ch.Key).To(Equal(chs[i].Key))
					Expect(ch.Name).To(Equal(chs[i].Name))
					Expect(ch.DataType).To(Equal(chs[i].DataType))
					if !chs[i].IsIndex {
						Expect(ch.Index).To(Equal(chs[i].Index))
					}
				}
			})
		},
		Entry("run key 0", uint64(0)),
		Entry("run key 1", uint64(1)),
		Entry("run key 2", uint64(2)),
	)

	It("Should open after a crash in the middle of a channel create", func(
		ctx SpecContext,
	) {
		disk := MustSucceed(fault.New(fault.Config{}))
		var image *fault.FS
		hooked := brkHookFS{FS: disk, suffix: "channel", after: func() {
			if image == nil {
				// The create's names reach the disk, and its bytes do not.
				image = MustSucceed(disk.Crash(fault.CrashConfig{EntrySurvivalRate: 1}))
			}
		}}
		cycle(ctx, hooked, func(db *cesium.DB) {
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
		})
		Expect(image).ToNot(BeNil())
		cycle(ctx, image, nil)
	})

	It("Should open after a crash that tore the first header of log 1 before its sync",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			cycle(ctx, fs, nil)
			// Head 1 is complete and covers no entry. The header write of log 1 is the
			// first write to its file, so a crash before its sync can leave a prefix of
			// the copy at 0 and no frame.
			b := brkReadFile(fs, log.Name(1))
			torn := make([]byte, log.FramesStart)
			copy(torn, b[:32])
			f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
			MustSucceed(f.WriteAt(make([]byte, 1<<20), 0))
			MustSucceed(f.WriteAt(torn, 0))
			Expect(f.Close()).To(Succeed())
			Expect(fs.Remove(log.Name(2))).To(Succeed())
			cycle(ctx, fs, nil)
		})

	It("Should not drop acked entries from a log that is shorter than its header says",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			idx, data := GenerateChannelKey(), GenerateChannelKey()
			keys := []cesium.ChannelKey{idx, data}
			const syncs = 1200
			cycle(ctx, fs, func(db *cesium.DB) {
				Expect(db.CreateChannel(
					ctx,
					cesium.Channel{
						Key:      idx,
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					cesium.Channel{
						Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
					},
				)).To(Succeed())
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Clock:    idx,
					Start:    1 * telem.SecondTS,
				}))
				for i := range syncs {
					s := telem.TimeStamp(i + 1)
					MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
						telem.NewSeriesSecondsTSV(s),
						telem.NewSeriesV[int64](int64(s)),
					}), nil))
					MustSucceed2(w.Commit(ctx, nil))
					Expect(w.Sync()).To(Succeed())
				}
				Expect(w.Close()).To(Succeed())
			})
			_, end, _, _ := brkLogEnd(brkReadFile(fs, log.Name(1)), 1)
			Expect(end).To(BeNumerically(">", 4<<20))
			f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
			Expect(f.Truncate(4 << 20)).To(Succeed())
			Expect(f.Close()).To(Succeed())
			db, err := open(ctx, fs)
			if err != nil {
				Expect(err).To(MatchError(place.ErrCorrupt))
				return
			}
			stop := StartRun(db.Run)
			defer func() {
				stop()
				Expect(db.Close()).To(Succeed())
			}()
			fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
			Expect(fr.Get(data).Len()).To(Equal(int64(syncs)))
		})
})
