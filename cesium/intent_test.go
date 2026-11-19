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
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	xfstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete intents", func() {
	var (
		mem       *xfs.MemFS
		db        *cesium.DB
		idx, data cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		mem = xfs.NewMem()
		db = mustOpenDBOnFS(ctx, mem)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
			[]cesium.ChannelKey{idx, data},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12),
				telem.NewSeriesV[int64](1, 2, 3),
			},
		))).To(Succeed())
	})
	// crash returns a DB opened on a copy of the files, as a restart after a crash at
	// this point would see them.
	crash := func(ctx SpecContext) (*cesium.DB, *xfs.MemFS) {
		GinkgoHelper()
		cp := xfs.NewMem()
		Expect(xfs.CopyDir(mem, cp, "")).To(Succeed())
		return mustOpenDBOnFS(ctx, cp), cp
	}
	expectGone := func(ctx SpecContext, db *cesium.DB, fs xfs.FS, key cesium.ChannelKey) {
		GinkgoHelper()
		Expect(db.RetrieveChannel(ctx, key)).Error().
			To(MatchError(cesium.ErrChannelNotFound))
		Expect(fs.Exists(channelKeyToPath(key))).To(BeFalse())
	}
	intents := func(fs xfs.FS) int64 {
		GinkgoHelper()
		return MustSucceed(fs.Stat("delete-intents")).Size()
	}

	Describe("Open", func() {
		It("Should skip the directory of a key with an intent", func(ctx SpecContext) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			restarted, fs := crash(ctx)
			Expect(restarted.RetrieveChannel(ctx, data)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
			Expect(fs.Exists(channelKeyToPath(data))).To(BeTrue())
		})
	})

	Describe("Drop", func() {
		It("Should clear the intent after it removes the files", func(ctx SpecContext) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			Expect(intents(mem)).To(Equal(int64(8)))
			Expect(db.Drop([]cesium.ChannelKey{data})).To(Succeed())
			Expect(intents(mem)).To(BeZero())
			Expect(mem.Exists(channelKeyToPath(data))).To(BeFalse())
		})
	})

	Describe("Sweep", func() {
		It(
			"Should clear the intent of a key whose record survived and open it",
			func(ctx SpecContext) {
				Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
				restarted, fs := crash(ctx)
				Expect(restarted.Sweep(ctx, []cesium.ChannelKey{idx, data})).
					To(BeEmpty())
				Expect(intents(fs)).To(BeZero())
				fr := MustSucceed(restarted.Read(ctx, telem.TimeRangeMax, data))
				Expect(
					fr.Get(data).Series[0],
				).To(telem.MatchSeriesDataV[int64](1, 2, 3))
			},
		)

		It("Should drop a key with an intent and no record", func(ctx SpecContext) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			restarted, fs := crash(ctx)
			Expect(restarted.Sweep(ctx, []cesium.ChannelKey{idx})).To(BeEmpty())
			expectGone(ctx, restarted, fs, data)
			Expect(intents(fs)).To(BeZero())
		})

		It("Should drop a key with no record and no samples", func(ctx SpecContext) {
			empty := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: empty, Name: "empty", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(db.Sweep(ctx, []cesium.ChannelKey{idx, data})).To(BeEmpty())
			expectGone(ctx, db, mem, empty)
		})

		It(
			"Should keep a key with samples, no record, and no intent, and refuse it",
			func(ctx SpecContext) {
				kept := MustSucceed(db.Sweep(ctx, []cesium.ChannelKey{idx}))
				Expect(kept).To(HaveLen(1))
				Expect(kept[0].Key).To(Equal(data))
				Expect(kept[0].Size).To(BeNumerically(">", 0))
				Expect(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{data},
					Start:    20 * telem.SecondTS,
				})).Error().To(MatchError(cesium.ErrChannelNotFound))
				Expect(db.Read(ctx, telem.TimeRangeMax, data)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
				Expect(mem.Exists(channelKeyToPath(data))).To(BeTrue())
			},
		)

		It(
			"Should keep the index of a kept channel, and the intent of the index",
			func(ctx SpecContext) {
				Expect(db.Intend([]cesium.ChannelKey{idx})).To(Succeed())
				kept := MustSucceed(db.Sweep(ctx, nil))
				Expect(kept).To(ConsistOf(HaveField("Key", data)))
				Expect(mem.Exists(channelKeyToPath(idx))).To(BeTrue())
				Expect(intents(mem)).To(Equal(int64(8)))
			},
		)
	})

	Describe("Purge", func() {
		It("Should remove a kept key", func(ctx SpecContext) {
			Expect(db.Sweep(ctx, []cesium.ChannelKey{idx})).To(HaveLen(1))
			Expect(db.Purge([]cesium.ChannelKey{data})).To(Succeed())
			expectGone(ctx, db, mem, data)
		})

		It("Should refuse a key that Sweep did not keep", func(ctx SpecContext) {
			Expect(db.Purge([]cesium.ChannelKey{idx})).
				To(MatchError(query.ErrNotFound))
			MustSucceed(db.RetrieveChannel(ctx, idx))
		})
	})

	Describe("Crashes", func() {
		It(
			"Should sync the rewritten intents before it renames them over the old file",
			func(ctx SpecContext) {
				var ops []string
				logged := mustOpenDBOnFS(ctx, opLogFS{FS: xfs.NewMem(), ops: &ops})
				Expect(logged.CreateChannel(ctx, cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				})).To(Succeed())
				Expect(logged.Intend([]cesium.ChannelKey{idx})).To(Succeed())
				ops = nil
				Expect(logged.Drop([]cesium.ChannelKey{idx})).To(Succeed())
				Expect(ops).To(HaveExactElements(
					"sync delete-intents.tmp",
					"rename delete-intents.tmp delete-intents",
					"syncdir",
				))
			},
		)

		It("Should ignore a torn intent append", func(ctx SpecContext) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			f := MustSucceed(mem.Open("delete-intents", os.O_RDWR))
			MustSucceed(f.WriteAt([]byte{1, 2, 3, 4, 5}, 8))
			Expect(f.Close()).To(Succeed())
			restarted, fs := crash(ctx)
			Expect(intents(fs)).To(Equal(int64(8)))
			Expect(restarted.Sweep(ctx, []cesium.ChannelKey{idx})).To(BeEmpty())
			expectGone(ctx, restarted, fs, data)
		})

		It(
			"Should keep a channel when a crash comes after the intent and before the "+
				"commit",
			func(ctx SpecContext) {
				Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
				restarted, _ := crash(ctx)
				Expect(restarted.Sweep(ctx, []cesium.ChannelKey{idx, data})).
					To(BeEmpty())
				MustSucceed(restarted.Read(ctx, telem.TimeRangeMax, data))
			},
		)

		It(
			"Should drop a channel when a crash comes after the commit and before Drop",
			func(ctx SpecContext) {
				Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
				restarted, fs := crash(ctx)
				Expect(restarted.Sweep(ctx, []cesium.ChannelKey{idx})).To(BeEmpty())
				expectGone(ctx, restarted, fs, data)
			},
		)

		It("Should finish a Drop that a crash cut short", func(ctx SpecContext) {
			faulty := xfstestutil.WrapFaultyFS(xfs.NewMem())
			fdb := mustOpenDBOnFS(ctx, faulty)
			Expect(fdb.CreateChannel(ctx, cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(fdb.Write(ctx, 10*telem.SecondTS, telem.UnaryFrame(
				idx,
				telem.NewSeriesSecondsTSV(10, 11),
			))).To(Succeed())
			Expect(fdb.Intend([]cesium.ChannelKey{idx})).To(Succeed())
			faulty.SetOptions(xfstestutil.WithFailRemove(
				channelKeyToPath(idx) + "-DELETE/meta.json",
			))
			Expect(
				fdb.Drop([]cesium.ChannelKey{idx}),
			).To(MatchError(xfstestutil.ErrFault))
			faulty.SetOptions()
			cp := xfs.NewMem()
			Expect(xfs.CopyDir(faulty, cp, "")).To(Succeed())
			restarted := mustOpenDBOnFS(ctx, cp)
			Expect(restarted.Sweep(ctx, nil)).To(BeEmpty())
			expectGone(ctx, restarted, cp, idx)
			Expect(cp.Exists(channelKeyToPath(idx) + "-DELETE")).To(BeFalse())
			Expect(intents(cp)).To(BeZero())
		})

		It(
			"Should finish a Sweep whose rewrite a crash cut short",
			func(ctx SpecContext) {
				faulty := xfstestutil.WrapFaultyFS(xfs.NewMem())
				fdb := mustOpenDBOnFS(ctx, faulty)
				Expect(fdb.CreateChannel(ctx, cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				})).To(Succeed())
				Expect(fdb.Intend([]cesium.ChannelKey{idx})).To(Succeed())
				cp := xfs.NewMem()
				Expect(xfs.CopyDir(faulty, cp, "")).To(Succeed())
				restarted := mustOpenDBOnFS(ctx, xfstestutil.WrapFaultyFS(cp,
					xfstestutil.WithFailRename("delete-intents.tmp"),
				))
				Expect(
					restarted.Sweep(ctx, nil),
				).Error().
					To(MatchError(xfstestutil.ErrFault))
				again := xfs.NewMem()
				Expect(xfs.CopyDir(cp, again, "")).To(Succeed())
				last := mustOpenDBOnFS(ctx, again)
				Expect(last.Sweep(ctx, nil)).To(BeEmpty())
				expectGone(ctx, last, again, idx)
				Expect(intents(again)).To(BeZero())
			},
		)
	})
})

// opLogFS records the syncs, renames, and directory syncs of the intents file.
type opLogFS struct {
	xfs.FS
	ops *[]string
}

func (f opLogFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil || name != "" {
		return sub, err
	}
	return opLogFS{FS: sub, ops: f.ops}, nil
}

func (f opLogFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !strings.HasPrefix(name, "delete-intents") {
		return file, err
	}
	return opLogFile{File: file, name: name, ops: f.ops}, nil
}

func (f opLogFS) Rename(oldPath, newPath string) error {
	if strings.HasPrefix(oldPath, "delete-intents") {
		*f.ops = append(*f.ops, "rename "+oldPath+" "+newPath)
	}
	return f.FS.Rename(oldPath, newPath)
}

func (f opLogFS) SyncDir(name string) error {
	if name == "" {
		*f.ops = append(*f.ops, "syncdir")
	}
	return f.FS.SyncDir(name)
}

type opLogFile struct {
	xfs.File
	ops  *[]string
	name string
}

func (f opLogFile) Sync() error {
	*f.ops = append(*f.ops, "sync "+f.name)
	return f.File.Sync()
}
