// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/fixed"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/variable"
	xcontrol "github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				fs      xfs.FS
				cleanUp func() error
				indexDB *fixed.DB
				dataDB  *variable.DB
			)
			BeforeAll(func(ctx SpecContext) {
				fs, cleanUp = makeFS()
				indexDB = MustSucceed(fixed.Open(ctx, fixed.Config{
					FS:        MustSucceed(fs.Sub("index")),
					MetaCodec: codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "index",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				}))
				dataDB = MustSucceed(variable.Open(ctx, variable.Config{
					FS:        MustSucceed(fs.Sub("data")),
					MetaCodec: codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "strings",
						DataType: telem.StringT,
						Index:    indexDB.Channel().Key,
					},
				}))
				dataDB.SetIndex(indexDB.Index())
			})
			AfterAll(func() {
				Expect(dataDB.Close()).To(Succeed())
				Expect(indexDB.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Open", func() {
				It("Should not open a virtual channel as a variable DB", func(ctx SpecContext) {
					Expect(variable.Open(ctx, variable.Config{
						FS:        MustSucceed(fs.Sub("virtual-reject")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "virtual",
							DataType: telem.StringT,
							Virtual:  true,
						},
					})).Error().To(HaveOccurred())
				})
				It("Should not open a fixed-density channel as a variable DB", func(ctx SpecContext) {
					_, err := variable.Open(ctx, variable.Config{
						FS:        MustSucceed(fs.Sub("fixed-reject")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "float",
							DataType: telem.Float64T,
							Index:    indexDB.Channel().Key,
						},
					})
					Expect(err).To(MatchError(variable.ErrNotVariable))
				})
			})

			Describe("Channel", func() {
				It("Should return the channel metadata", func() {
					ch := dataDB.Channel()
					Expect(ch.DataType).To(Equal(telem.StringT))
					Expect(ch.Name).To(Equal("strings"))
				})
			})

			Describe("Write + Read", func() {
				It("Should write and read string data", func(ctx SpecContext) {
					Expect(fixed.Write(ctx, indexDB, 10*telem.SecondTS,
						telem.NewSeriesSecondsTSV(10, 11, 12),
					)).To(Succeed())
					Expect(variable.Write(ctx, dataDB, 10*telem.SecondTS,
						telem.NewSeriesV("hello", "world", "foo"),
					)).To(Succeed())
					frame := MustSucceed(dataDB.Read(ctx, (10 * telem.SecondTS).Range(13*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(Equal([]string{"hello", "world", "foo"}))
				})
				It("Should write and read JSON data", func(ctx SpecContext) {
					jsonDB := MustSucceed(variable.Open(ctx, variable.Config{
						FS:        MustSucceed(fs.Sub("json-data")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "json",
							DataType: telem.JSONT,
							Index:    indexDB.Channel().Key,
						},
					}))
					defer func() { Expect(jsonDB.Close()).To(Succeed()) }()
					jsonDB.SetIndex(indexDB.Index())

					Expect(fixed.Write(ctx, indexDB, 20*telem.SecondTS,
						telem.NewSeriesSecondsTSV(20, 21),
					)).To(Succeed())
					jsonSeries := MustSucceed(telem.NewJSONSeriesV(
						map[string]any{"key": "value"},
						map[string]any{"num": 42},
					))
					Expect(variable.Write(ctx, jsonDB, 20*telem.SecondTS, jsonSeries)).To(Succeed())
					frame := MustSucceed(jsonDB.Read(ctx, (20 * telem.SecondTS).Range(22*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					Expect(frame.SeriesAt(0).Len()).To(Equal(int64(2)))
				})
				It("Should write and read bytes data", func(ctx SpecContext) {
					bytesDB := MustSucceed(variable.Open(ctx, variable.Config{
						FS:        MustSucceed(fs.Sub("bytes-data")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "bytes",
							DataType: telem.BytesT,
							Index:    indexDB.Channel().Key,
						},
					}))
					defer func() { Expect(bytesDB.Close()).To(Succeed()) }()
					bytesDB.SetIndex(indexDB.Index())

					Expect(fixed.Write(ctx, indexDB, 30*telem.SecondTS,
						telem.NewSeriesSecondsTSV(30, 31, 32),
					)).To(Succeed())
					Expect(variable.Write(ctx, bytesDB, 30*telem.SecondTS,
						telem.NewSeriesV([]byte{1, 2, 3}, []byte{4, 5}, []byte{6}),
					)).To(Succeed())
					frame := MustSucceed(bytesDB.Read(ctx, (30 * telem.SecondTS).Range(33*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					samples := telem.UnmarshalSeries[[]byte](frame.SeriesAt(0))
					Expect(samples).To(Equal([][]byte{{1, 2, 3}, {4, 5}, {6}}))
				})
				It("Should write and read empty strings", func(ctx SpecContext) {
					Expect(fixed.Write(ctx, indexDB, 40*telem.SecondTS,
						telem.NewSeriesSecondsTSV(40, 41, 42),
					)).To(Succeed())
					Expect(variable.Write(ctx, dataDB, 40*telem.SecondTS,
						telem.NewSeriesV("", "nonempty", ""),
					)).To(Succeed())
					frame := MustSucceed(dataDB.Read(ctx, (40 * telem.SecondTS).Range(43*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(Equal([]string{"", "nonempty", ""}))
				})
				It("Should write and read strings containing newlines", func(ctx SpecContext) {
					Expect(fixed.Write(ctx, indexDB, 50*telem.SecondTS,
						telem.NewSeriesSecondsTSV(50, 51),
					)).To(Succeed())
					Expect(variable.Write(ctx, dataDB, 50*telem.SecondTS,
						telem.NewSeriesV("line1\nline2", "no newline"),
					)).To(Succeed())
					frame := MustSucceed(dataDB.Read(ctx, (50 * telem.SecondTS).Range(52*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(Equal([]string{"line1\nline2", "no newline"}))
				})
			})

			Describe("HasDataFor", func() {
				It("Should return true when data exists in the range", func(ctx SpecContext) {
					has := MustSucceed(dataDB.HasDataFor(ctx, (10 * telem.SecondTS).Range(13*telem.SecondTS)))
					Expect(has).To(BeTrue())
				})
				It("Should return false when no data exists in the range", func(ctx SpecContext) {
					has := MustSucceed(dataDB.HasDataFor(ctx, (900 * telem.SecondTS).Range(901*telem.SecondTS)))
					Expect(has).To(BeFalse())
				})
			})

			Describe("Size", func() {
				It("Should return the total byte size of the database", func() {
					Expect(dataDB.Size()).To(BeNumerically(">", 0))
				})
			})

			Describe("Rename", func() {
				It("Should rename the channel in metadata", func(ctx SpecContext) {
					Expect(dataDB.RenameChannelInMeta(ctx, "renamed")).To(Succeed())
					Expect(dataDB.Channel().Name).To(Equal("renamed"))
					Expect(dataDB.RenameChannelInMeta(ctx, "strings")).To(Succeed())
				})
			})

			Describe("Writer", func() {
				It("Should track alignment correctly across writes", func(ctx SpecContext) {
					Expect(fixed.Write(ctx, indexDB, 60*telem.SecondTS,
						telem.NewSeriesSecondsTSV(60, 61, 62, 63, 64),
					)).To(Succeed())
					w, _ := MustSucceed2(dataDB.OpenWriter(ctx, variable.WriterConfig{
						Start:   60 * telem.SecondTS,
						Subject: xcontrol.Subject{Key: "alignment-test"},
					}))
					a1 := MustSucceed(w.Write(telem.NewSeriesV("a", "b")))
					Expect(a1.SampleIndex()).To(Equal(uint32(0)))
					a2 := MustSucceed(w.Write(telem.NewSeriesV("c", "d", "e")))
					Expect(a2.SampleIndex()).To(Equal(uint32(2)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())
				})
				It("Should reject writes with wrong data type", func(ctx SpecContext) {
					w, _ := MustSucceed2(dataDB.OpenWriter(ctx, variable.WriterConfig{
						Start:   70 * telem.SecondTS,
						Subject: xcontrol.Subject{Key: "wrong-type"},
					}))
					Expect(w.Write(telem.NewSeriesV[int64](1, 2, 3))).Error().To(HaveOccurred())
					MustSucceed(w.Close())
				})
				It("Should return error when writing to a closed writer", func(ctx SpecContext) {
					w, _ := MustSucceed2(dataDB.OpenWriter(ctx, variable.WriterConfig{
						Start:   80 * telem.SecondTS,
						Subject: xcontrol.Subject{Key: "closed-writer"},
					}))
					MustSucceed(w.Close())
					Expect(w.Write(telem.NewSeriesV("data"))).Error().To(MatchError(ContainSubstring("closed")))
				})
			})

			Describe("Iterator", func() {
				It("Should iterate forward through data", func(ctx SpecContext) {
					Expect(fixed.Write(ctx, indexDB, 100*telem.SecondTS,
						telem.NewSeriesSecondsTSV(100, 101, 102, 103, 104),
					)).To(Succeed())
					Expect(variable.Write(ctx, dataDB, 100*telem.SecondTS,
						telem.NewSeriesV("a", "b", "c", "d", "e"),
					)).To(Succeed())
					iter := MustSucceed(dataDB.OpenIterator(variable.IterRange(
						(100 * telem.SecondTS).Range(105 * telem.SecondTS),
					)))
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
					f := iter.Value()
					Expect(f.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](f.SeriesAt(0))).To(Equal([]string{"a", "b", "c", "d", "e"}))
					Expect(iter.Close()).To(Succeed())
				})
				It("Should iterate backward through data", func(ctx SpecContext) {
					iter := MustSucceed(dataDB.OpenIterator(variable.IterRange(
						(100 * telem.SecondTS).Range(105 * telem.SecondTS),
					)))
					Expect(iter.SeekLast(ctx)).To(BeTrue())
					Expect(iter.Prev(ctx, telem.TimeSpanMax)).To(BeTrue())
					f := iter.Value()
					Expect(f.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](f.SeriesAt(0))).To(Equal([]string{"a", "b", "c", "d", "e"}))
					Expect(iter.Close()).To(Succeed())
				})
				It("Should read a sub-range of a domain", func(ctx SpecContext) {
					iter := MustSucceed(dataDB.OpenIterator(variable.IterRange(
						(101 * telem.SecondTS).Range(104 * telem.SecondTS),
					)))
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
					f := iter.Value()
					Expect(f.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](f.SeriesAt(0))).To(Equal([]string{"b", "c", "d"}))
					Expect(iter.Close()).To(Succeed())
				})
				It("Should return false when seeking an empty range", func(ctx SpecContext) {
					iter := MustSucceed(dataDB.OpenIterator(variable.IterRange(
						(900 * telem.SecondTS).Range(901 * telem.SecondTS),
					)))
					Expect(iter.SeekFirst(ctx)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
				It("Should iterate across multiple domains", func(ctx SpecContext) {
					Expect(fixed.Write(ctx, indexDB, 200*telem.SecondTS,
						telem.NewSeriesSecondsTSV(200, 201, 202),
					)).To(Succeed())
					Expect(variable.Write(ctx, dataDB, 200*telem.SecondTS,
						telem.NewSeriesV("x", "y", "z"),
					)).To(Succeed())

					Expect(fixed.Write(ctx, indexDB, 203*telem.SecondTS,
						telem.NewSeriesSecondsTSV(203, 204),
					)).To(Succeed())
					Expect(variable.Write(ctx, dataDB, 203*telem.SecondTS,
						telem.NewSeriesV("w", "v"),
					)).To(Succeed())

					iter := MustSucceed(dataDB.OpenIterator(variable.IterRange(
						(200 * telem.SecondTS).Range(205 * telem.SecondTS),
					)))
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
					Expect(iter.Value().Count()).To(BeNumerically(">=", 1))
					Expect(iter.Close()).To(Succeed())
				})
			})

			Describe("Delete", func() {
				It("Should delete data in a time range", func(ctx SpecContext) {
					Expect(fixed.Write(ctx, indexDB, 300*telem.SecondTS,
						telem.NewSeriesSecondsTSV(300, 301, 302, 303, 304),
					)).To(Succeed())
					Expect(variable.Write(ctx, dataDB, 300*telem.SecondTS,
						telem.NewSeriesV("del0", "del1", "del2", "del3", "del4"),
					)).To(Succeed())

					Expect(dataDB.Delete(ctx, (302 * telem.SecondTS).Range(304*telem.SecondTS))).To(Succeed())

					frame := MustSucceed(dataDB.Read(ctx, (300 * telem.SecondTS).Range(305*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(2))
					first := telem.UnmarshalSeries[string](frame.SeriesAt(0))
					Expect(first).To(Equal([]string{"del0", "del1"}))
					second := telem.UnmarshalSeries[string](frame.SeriesAt(1))
					Expect(second).To(Equal([]string{"del4"}))
				})
				It("Should delete all data", func(ctx SpecContext) {
					deleteDB := MustSucceed(variable.Open(ctx, variable.Config{
						FS:        MustSucceed(fs.Sub("delete-all")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "delete-all",
							DataType: telem.StringT,
							Index:    indexDB.Channel().Key,
						},
					}))
					deleteDB.SetIndex(indexDB.Index())

					Expect(fixed.Write(ctx, indexDB, 400*telem.SecondTS,
						telem.NewSeriesSecondsTSV(400, 401, 402),
					)).To(Succeed())
					Expect(variable.Write(ctx, deleteDB, 400*telem.SecondTS,
						telem.NewSeriesV("a", "b", "c"),
					)).To(Succeed())

					Expect(deleteDB.Delete(ctx, telem.TimeRangeMax)).To(Succeed())
					frame := MustSucceed(deleteDB.Read(ctx, telem.TimeRangeMax))
					Expect(frame.Count()).To(Equal(0))
					Expect(deleteDB.Close()).To(Succeed())
				})
			})

			Describe("GarbageCollect", func() {
				It("Should garbage collect after deletion", func(ctx SpecContext) {
					gcDB := MustSucceed(variable.Open(ctx, variable.Config{
						FS:          MustSucceed(fs.Sub("gc")),
						MetaCodec:   codec,
						GCThreshold: 0.01,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "gc-test",
							DataType: telem.StringT,
							Index:    indexDB.Channel().Key,
						},
					}))
					gcDB.SetIndex(indexDB.Index())

					Expect(fixed.Write(ctx, indexDB, 500*telem.SecondTS,
						telem.NewSeriesSecondsTSV(500, 501, 502, 503, 504),
					)).To(Succeed())
					Expect(variable.Write(ctx, gcDB, 500*telem.SecondTS,
						telem.NewSeriesV("gc0", "gc1", "gc2", "gc3", "gc4"),
					)).To(Succeed())

					Expect(gcDB.Delete(ctx, (502 * telem.SecondTS).Range(504*telem.SecondTS))).To(Succeed())
					Expect(gcDB.GarbageCollect(ctx)).To(Succeed())

					frame := MustSucceed(gcDB.Read(ctx, (500 * telem.SecondTS).Range(505*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(2))
					Expect(gcDB.Close()).To(Succeed())
				})
			})

			Describe("Close", func() {
				It("Should be idempotent", func(ctx SpecContext) {
					closeDB := MustSucceed(variable.Open(ctx, variable.Config{
						FS:        MustSucceed(fs.Sub("close-test")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "close-test",
							DataType: telem.StringT,
							Index:    indexDB.Channel().Key,
						},
					}))
					Expect(closeDB.Close()).To(Succeed())
					Expect(closeDB.Close()).To(Succeed())
				})
			})
		})
	}
})
