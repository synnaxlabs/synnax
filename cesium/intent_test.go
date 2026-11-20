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
	"hash/crc32"
	"io"
	"maps"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	xfstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/set"
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
		Expect(write(ctx, db, 10*telem.SecondTS, telem.MultiFrame(
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
	// crashWith returns a DB opened on a copy of the files that damage changes.
	crashWith := func(ctx SpecContext, damage func(*xfs.MemFS)) (*cesium.DB, *xfs.MemFS) {
		GinkgoHelper()
		cp := xfs.NewMem()
		Expect(xfs.CopyDir(mem, cp, "")).To(Succeed())
		damage(cp)
		return mustOpenDBOnFS(ctx, cp), cp
	}
	expectGone := func(ctx SpecContext, db *cesium.DB, fs xfs.FS, key cesium.ChannelKey) {
		GinkgoHelper()
		Expect(db.RetrieveChannel(ctx, key)).Error().
			To(MatchError(cesium.ErrChannelNotFound))
		Expect(fs.Exists(channelKeyToPath(key))).To(BeFalse())
	}
	// intents returns the number of keys with an intent in the intents file of fs. It
	// skips a batch that fails its checksum, as Open does.
	intents := func(fs xfs.FS) int {
		GinkgoHelper()
		data := MustSucceed(xfs.ReadFile(fs, "delete-intents"))
		live := set.New[cesium.ChannelKey]()
		for off := 0; off+8 <= len(data); {
			count := int(binary.LittleEndian.Uint32(data[off:]))
			end := off + 8 + 8*count
			if count == 0 || end > len(data) {
				break
			}
			var at [8]byte
			binary.LittleEndian.PutUint64(at[:], uint64(off))
			table := crc32.MakeTable(crc32.Castagnoli)
			crc := crc32.Update(crc32.Update(0, table, at[:]), table, data[off+8:end])
			if crc != binary.LittleEndian.Uint32(data[off+4:]) {
				off = end
				continue
			}
			for r := off + 8; r < end; r += 8 {
				key := binary.LittleEndian.Uint32(data[r:])
				if binary.LittleEndian.Uint32(data[r+4:])&1 != 0 {
					live.Remove(key)
				} else {
					live.Add(key)
				}
			}
			off = end
		}
		return len(live)
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
		It(
			"Should clear the intents of a batch with syncs that do not grow with the batch",
			func(ctx SpecContext) {
				var ops []string
				logged := mustOpenDBOnFS(ctx, opLogFS{FS: xfs.NewMem(), ops: &ops})
				keys := make([]cesium.ChannelKey, 100)
				for i := range keys {
					keys[i] = GenerateChannelKey()
					Expect(logged.CreateChannel(ctx, cesium.Channel{
						Key:      keys[i],
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					})).To(Succeed())
				}
				Expect(logged.Intend(keys)).To(Succeed())
				ops = nil
				Expect(logged.Drop(keys)).To(Succeed())
				Expect(ops).To(HaveExactElements("syncdir", "sync delete-intents"))
			},
		)

		It("Should clear the intent after it removes the files", func(ctx SpecContext) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			Expect(intents(mem)).To(Equal(1))
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
				Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx, data})).
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
			Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx})).To(BeEmpty())
			expectGone(ctx, restarted, fs, data)
			Expect(intents(fs)).To(BeZero())
		})

		It("Should drop a key with no record and no samples", func(ctx SpecContext) {
			empty := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: empty, Name: "empty", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(sweep(ctx, db, []cesium.ChannelKey{idx, data})).To(BeEmpty())
			expectGone(ctx, db, mem, empty)
		})

		It(
			"Should keep a key with samples, no record, and no intent, and refuse it",
			func(ctx SpecContext) {
				size := db.Metrics().DiskSize
				kept := sweep(ctx, db, []cesium.ChannelKey{idx})
				Expect(db.Metrics().DiskSize).To(Equal(size))
				Expect(kept).To(HaveLen(1))
				Expect(kept[0].Key).To(Equal(data))
				Expect(kept[0].Size).To(BeNumerically(">", 0))
				Expect(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{data},
					Clock:    idx,
					Start:    20 * telem.SecondTS,
				})).Error().To(MatchError(cesium.ErrChannelNotFound))
				Expect(db.Read(ctx, telem.TimeRangeMax, data)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
				Expect(db.CreateChannel(ctx, cesium.Channel{
					Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
				})).To(MatchError(ContainSubstring("it already exists")))
				Expect(mem.Exists(channelKeyToPath(data))).To(BeTrue())
			},
		)

		It("Should decide once per open, and drop nothing on a later Sweep", func(
			ctx SpecContext,
		) {
			empty := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: empty, Name: "empty", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(sweep(ctx, db, []cesium.ChannelKey{idx, empty})).
				To(ConsistOf(HaveField("Key", data)))
			// A first Sweep would drop empty, which has no record and no samples here.
			Expect(sweep(ctx, db, []cesium.ChannelKey{idx})).
				To(ConsistOf(HaveField("Key", data)))
			Expect(mem.Exists(channelKeyToPath(empty))).To(BeTrue())
			Expect(mem.Exists(channelKeyToPath(idx))).To(BeTrue())
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			Expect(db.Drop([]cesium.ChannelKey{data})).To(Succeed())
			Expect(sweep(ctx, db, []cesium.ChannelKey{idx})).To(BeEmpty())
		})

		It("Should return the highest key that the DB holds, with an intent included",
			func(ctx SpecContext) {
				keptData, intendedIdx := GenerateChannelKey(), GenerateChannelKey()
				Expect(db.CreateChannel(
					ctx,
					cesium.Channel{
						Key:      intendedIdx,
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					cesium.Channel{
						Key:      keptData,
						Name:     "data",
						Index:    intendedIdx,
						DataType: telem.Int64T,
					},
				)).To(Succeed())
				Expect(write(ctx, db, 10*telem.SecondTS, telem.MultiFrame(
					[]cesium.ChannelKey{intendedIdx, keptData},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(10, 11),
						telem.NewSeriesV[int64](1, 2),
					},
				))).To(Succeed())
				Expect(db.Intend([]cesium.ChannelKey{intendedIdx})).To(Succeed())
				kept, highest := MustSucceed2(db.Sweep(
					ctx,
					[]cesium.ChannelKey{idx, data},
				))
				Expect(kept).To(ConsistOf(SatisfyAll(
					HaveField("Key", keptData),
					HaveField("Index", intendedIdx),
				)))
				Expect(highest).To(Equal(intendedIdx))
			},
		)

		It("Should make the intent of a drop durable before Sweep removes a key with "+
			"no samples", func(ctx SpecContext) {
			var ops []string
			logged := mustOpenDBOnFS(ctx, opLogFS{FS: xfs.NewMem(), ops: &ops})
			Expect(logged.CreateChannel(ctx, cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			ops = nil
			Expect(sweep(ctx, logged, nil)).To(BeEmpty())
			// The intent, the removal, the clear, then the empty file.
			Expect(ops).To(HaveExactElements(
				"sync delete-intents",
				"syncdir",
				"sync delete-intents",
				"sync delete-intents",
			))
		})

		It("Should open a kept channel after a delete drops its index", func(
			ctx SpecContext,
		) {
			Expect(sweep(ctx, db, nil)).To(ConsistOf(
				HaveField("Key", idx),
				HaveField("Key", data),
			))
			Expect(db.Intend([]cesium.ChannelKey{idx})).To(Succeed())
			Expect(db.Drop([]cesium.ChannelKey{idx})).To(Succeed())
			restarted, fs := crash(ctx)
			Expect(sweep(ctx, restarted, nil)).To(ConsistOf(HaveField("Key", data)))
			Expect(fs.Exists(channelKeyToPath(idx))).To(BeTrue())
		})

		It(
			"Should keep the index of a kept channel, and the intent of the index",
			func(ctx SpecContext) {
				Expect(db.Intend([]cesium.ChannelKey{idx})).To(Succeed())
				kept := sweep(ctx, db, nil)
				Expect(kept).To(ConsistOf(HaveField("Key", data)))
				Expect(mem.Exists(channelKeyToPath(idx))).To(BeTrue())
				Expect(intents(mem)).To(Equal(1))
			},
		)
	})

	Describe("Drop of a kept key", func() {
		It("Should remove a key that Sweep keeps", func(ctx SpecContext) {
			Expect(sweep(ctx, db, []cesium.ChannelKey{idx})).To(HaveLen(1))
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			Expect(db.Drop([]cesium.ChannelKey{data})).To(Succeed())
			expectGone(ctx, db, mem, data)
			Expect(intents(mem)).To(BeZero())
		})
	})

	Describe("Crashes", func() {
		It(
			"Should make a removal durable before it appends the clear of its intent",
			func(ctx SpecContext) {
				var ops []string
				logged := mustOpenDBOnFS(ctx, opLogFS{FS: xfs.NewMem(), ops: &ops})
				Expect(logged.CreateChannel(ctx, cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				})).To(Succeed())
				Expect(logged.Intend([]cesium.ChannelKey{idx})).To(Succeed())
				ops = nil
				Expect(logged.Drop([]cesium.ChannelKey{idx})).To(Succeed())
				Expect(ops).To(HaveExactElements("syncdir", "sync delete-intents"))
			},
		)

		Describe("Lost unsynced data", func() {
			var (
				lossy *lossyFS
				ldb   *cesium.DB
				other cesium.ChannelKey
			)
			BeforeEach(func(ctx SpecContext) {
				lossy = newLossyFS(xfs.NewMem())
				ldb = mustOpenDBOnFS(ctx, lossy)
				other = GenerateChannelKey()
				Expect(ldb.CreateChannel(
					ctx,
					cesium.Channel{
						Key:      idx,
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					cesium.Channel{
						Key:      other,
						Name:     "other",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
				)).To(Succeed())
				for _, k := range []cesium.ChannelKey{idx, other} {
					Expect(writeSeries(
						ctx, ldb,
						k,
						10*telem.SecondTS,
						telem.NewSeriesSecondsTSV(10, 11),
					)).To(Succeed())
				}
			})

			It("Should keep an appended intent", func(ctx SpecContext) {
				Expect(ldb.Intend([]cesium.ChannelKey{other})).To(Succeed())
				fs := lossy.crash()
				restarted := mustOpenDBOnFS(ctx, fs)
				Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx})).To(BeEmpty())
				expectGone(ctx, restarted, fs, other)
			})

			It("Should keep the intents that a clear leaves", func(ctx SpecContext) {
				Expect(ldb.Intend([]cesium.ChannelKey{idx, other})).To(Succeed())
				Expect(ldb.Drop([]cesium.ChannelKey{idx})).To(Succeed())
				fs := lossy.crash()
				restarted := mustOpenDBOnFS(ctx, fs)
				Expect(sweep(ctx, restarted, nil)).To(BeEmpty())
				expectGone(ctx, restarted, fs, other)
			})
		})

		It("Should read the intents after a corrupt record", func(ctx SpecContext) {
			other := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: other, Name: "other", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(writeSeries(
				ctx, db,
				other,
				10*telem.SecondTS,
				telem.NewSeriesSecondsTSV(10, 11, 12),
			)).To(Succeed())
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			Expect(db.Intend([]cesium.ChannelKey{other})).To(Succeed())
			restarted, fs := crashWith(ctx, func(cp *xfs.MemFS) {
				f := MustSucceed(cp.Open("delete-intents", os.O_RDWR))
				MustSucceed(f.WriteAt([]byte{0xFF}, 4))
				Expect(f.Close()).To(Succeed())
			})
			Expect(intents(fs)).To(Equal(1))
			Expect(restarted.CorruptIntents()).To(Equal(1))
			kept := sweep(ctx, restarted, []cesium.ChannelKey{idx})
			Expect(kept).To(ConsistOf(HaveField("Key", data)))
			expectGone(ctx, restarted, fs, other)
		})

		It("Should drop a whole batch when a crash keeps only part of it", func(
			ctx SpecContext,
		) {
			Expect(db.Intend([]cesium.ChannelKey{idx, data})).To(Succeed())
			restarted, fs := crashWith(ctx, func(cp *xfs.MemFS) {
				f := MustSucceed(cp.Open("delete-intents", os.O_RDWR))
				MustSucceed(f.WriteAt(make([]byte, 8), 16))
				Expect(f.Close()).To(Succeed())
			})
			Expect(intents(fs)).To(BeZero())
			Expect(restarted.CorruptIntents()).To(BeZero())
			Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx, data})).To(BeEmpty())
			Expect(fs.Exists(channelKeyToPath(data))).To(BeTrue())
		})

		It("Should keep an intent that one read of the file damages", func(
			ctx SpecContext,
		) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			cp := xfs.NewMem()
			Expect(xfs.CopyDir(mem, cp, "")).To(Succeed())
			restarted := mustOpenDBOnFS(ctx, &badReadFS{FS: cp, name: "delete-intents"})
			Expect(restarted.CorruptIntents()).To(BeZero())
			Expect(MustSucceed(cp.Stat("delete-intents")).Size()).To(Equal(int64(16)))
			Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx})).To(BeEmpty())
			expectGone(ctx, restarted, cp, data)
		})

		It("Should empty the intents file when Sweep leaves no intent", func(
			ctx SpecContext,
		) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			restarted, fs := crash(ctx)
			Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx})).To(BeEmpty())
			Expect(MustSucceed(fs.Stat("delete-intents")).Size()).To(BeZero())
			Expect(restarted.Intend([]cesium.ChannelKey{idx})).To(Succeed())
			again, cp := crash(ctx)
			Expect(again.CorruptIntents()).To(BeZero())
			Expect(intents(cp)).To(Equal(1))
		})

		It("Should fail every Intend after an append fails", func(ctx SpecContext) {
			faulty := xfstestutil.WrapFaultyFS(xfs.NewMem())
			fdb := mustOpenDBOnFS(ctx, faulty)
			faulty.SetOptions(xfstestutil.WithFailWriteAt("delete-intents"))
			Expect(fdb.Intend([]cesium.ChannelKey{idx})).
				To(MatchError(xfstestutil.ErrFault))
			faulty.SetOptions()
			Expect(fdb.Intend([]cesium.ChannelKey{idx})).
				To(MatchError(ContainSubstring("until a restart")))
		})

		It("Should ignore a torn intent append", func(ctx SpecContext) {
			Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
			f := MustSucceed(mem.Open("delete-intents", os.O_RDWR))
			MustSucceed(f.WriteAt([]byte{1, 2, 3, 4, 5}, 16))
			Expect(f.Close()).To(Succeed())
			restarted, fs := crash(ctx)
			Expect(intents(fs)).To(Equal(1))
			Expect(MustSucceed(fs.Stat("delete-intents")).Size()).To(Equal(int64(16)))
			Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx})).To(BeEmpty())
			expectGone(ctx, restarted, fs, data)
		})

		It(
			"Should keep a channel when a crash comes after the intent and before the "+
				"commit",
			func(ctx SpecContext) {
				Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
				restarted, _ := crash(ctx)
				Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx, data})).
					To(BeEmpty())
				MustSucceed(restarted.Read(ctx, telem.TimeRangeMax, data))
			},
		)

		It(
			"Should drop a channel when a crash comes after the commit and before Drop",
			func(ctx SpecContext) {
				Expect(db.Intend([]cesium.ChannelKey{data})).To(Succeed())
				restarted, fs := crash(ctx)
				Expect(sweep(ctx, restarted, []cesium.ChannelKey{idx})).To(BeEmpty())
				expectGone(ctx, restarted, fs, data)
			},
		)

		It("Should finish a Drop that a crash cut short", func(ctx SpecContext) {
			faulty := xfstestutil.WrapFaultyFS(xfs.NewMem())
			fdb := mustOpenDBOnFS(ctx, faulty)
			Expect(fdb.CreateChannel(ctx, cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(write(ctx, fdb, 10*telem.SecondTS, telem.UnaryFrame(
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
			Expect(sweep(ctx, restarted, nil)).To(BeEmpty())
			expectGone(ctx, restarted, cp, idx)
			Expect(cp.Exists(channelKeyToPath(idx) + "-DELETE")).To(BeFalse())
			Expect(intents(cp)).To(BeZero())
		})

		It(
			"Should finish a Sweep whose clear a crash cut short",
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
					xfstestutil.WithFailWriteAt("delete-intents"),
				))
				Expect(
					restarted.Sweep(ctx, nil),
				).Error().
					To(MatchError(xfstestutil.ErrFault))
				Expect(restarted.Sweep(ctx, nil)).Error().
					To(MatchError(xfstestutil.ErrFault))
				again := xfs.NewMem()
				Expect(xfs.CopyDir(cp, again, "")).To(Succeed())
				last := mustOpenDBOnFS(ctx, again)
				Expect(sweep(ctx, last, nil)).To(BeEmpty())
				expectGone(ctx, last, again, idx)
				Expect(intents(again)).To(BeZero())
			},
		)
	})
})

// sweep runs Sweep on db with the record keys keys, and returns the channels it keeps.
func sweep(
	ctx context.Context,
	db *cesium.DB,
	keys []cesium.ChannelKey,
) []cesium.KeptChannel {
	GinkgoHelper()
	kept, _ := MustSucceed2(db.Sweep(ctx, keys))
	return kept
}

// badReadFS damages the last byte of the first read of the file name.
type badReadFS struct {
	xfs.FS
	name string
	done atomic.Bool
}

func (f *badReadFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil || name != "" {
		return sub, err
	}
	return &badReadFS{FS: sub, name: f.name}, nil
}

func (f *badReadFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != f.name {
		return file, err
	}
	return badReadFile{File: file, fs: f}, nil
}

type badReadFile struct {
	xfs.File
	fs *badReadFS
}

func (f badReadFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	if n > 0 && f.fs.done.CompareAndSwap(false, true) {
		p[n-1] ^= 0xFF
	}
	return n, err
}

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

// lossyFS models a crash at the root of a file system for the intents files: an entry
// survives only once SyncDir syncs the root, and a file keeps only the contents of its
// last Sync.
type lossyFS struct {
	xfs.FS
	st *lossyState
}

type lossyState struct {
	inner   xfs.FS
	live    map[string]int
	durable map[string]int
	synced  map[int][]byte
	next    int
	mu      sync.Mutex
}

func newLossyFS(inner xfs.FS) *lossyFS {
	return &lossyFS{FS: inner, st: &lossyState{
		inner:   inner,
		live:    map[string]int{},
		durable: map[string]int{},
		synced:  map[int][]byte{},
	}}
}

// crash returns a copy of the files as a restart after a crash now would see them.
func (f *lossyFS) crash() xfs.FS {
	GinkgoHelper()
	f.st.mu.Lock()
	defer f.st.mu.Unlock()
	cp := xfs.NewMem()
	Expect(xfs.CopyDir(f.st.inner, cp, "")).To(Succeed())
	for _, name := range []string{"delete-intents", "delete-intents.tmp"} {
		Expect(cp.Remove(name)).To(Succeed())
	}
	for name, inode := range f.st.durable {
		Expect(xfs.WriteFile(cp, name, f.st.synced[inode])).To(Succeed())
	}
	return cp
}

func modeled(name string) bool { return strings.HasPrefix(name, "delete-intents") }

func (f *lossyFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil || name != "" {
		return sub, err
	}
	return &lossyFS{FS: sub, st: f.st}, nil
}

func (f *lossyFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !modeled(name) {
		return file, err
	}
	f.st.mu.Lock()
	defer f.st.mu.Unlock()
	inode, ok := f.st.live[name]
	if !ok {
		f.st.next++
		inode = f.st.next
		f.st.live[name] = inode
	}
	return lossyFile{File: file, st: f.st, inode: inode}, nil
}

func (f *lossyFS) Rename(oldPath, newPath string) error {
	if err := f.FS.Rename(oldPath, newPath); err != nil || !modeled(oldPath) {
		return err
	}
	f.st.mu.Lock()
	defer f.st.mu.Unlock()
	f.st.live[newPath] = f.st.live[oldPath]
	delete(f.st.live, oldPath)
	return nil
}

func (f *lossyFS) Remove(name string) error {
	if err := f.FS.Remove(name); err != nil || !modeled(name) {
		return err
	}
	f.st.mu.Lock()
	defer f.st.mu.Unlock()
	delete(f.st.live, name)
	return nil
}

func (f *lossyFS) SyncDir(name string) error {
	if err := f.FS.SyncDir(name); err != nil || name != "" {
		return err
	}
	f.st.mu.Lock()
	defer f.st.mu.Unlock()
	f.st.durable = maps.Clone(f.st.live)
	return nil
}

type lossyFile struct {
	xfs.File
	st    *lossyState
	inode int
}

func (f lossyFile) Sync() error {
	if err := f.File.Sync(); err != nil {
		return err
	}
	data, err := io.ReadAll(io.NewSectionReader(f.File, 0, 1<<62))
	if err != nil {
		return err
	}
	f.st.mu.Lock()
	defer f.st.mu.Unlock()
	f.st.synced[f.inode] = data
	return nil
}
