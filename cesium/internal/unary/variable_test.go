// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	xcontrol "github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Variable-length channel", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				fs      xfs.FS
				cleanUp func() error
				indexDB *unary.DB
				dataDB  *unary.DB
			)
			BeforeAll(func(ctx SpecContext) {
				fs, cleanUp = makeFS()
				indexDB = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("index")),
					MetaCodec: codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "index",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				}))
				dataDB = MustSucceed(unary.Open(ctx, unary.Config{
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

			Describe("Write + Read", func() {
				It("Should write and read string data", func(ctx SpecContext) {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS,
						telem.NewSeriesSecondsTSV(10, 11, 12),
					)).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 10*telem.SecondTS,
						telem.NewSeriesV("hello", "world", "foo"),
					)).To(Succeed())
					frame := MustSucceed(dataDB.Read(ctx, (10 * telem.SecondTS).Range(13*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(Equal([]string{"hello", "world", "foo"}))
				})
				It("Should write and read JSON data", func(ctx SpecContext) {
					jsonDB := MustSucceed(unary.Open(ctx, unary.Config{
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

					Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS,
						telem.NewSeriesSecondsTSV(20, 21),
					)).To(Succeed())
					jsonSeries := MustSucceed(telem.NewJSONSeriesV(
						map[string]any{"key": "value"},
						map[string]any{"num": 42},
					))
					Expect(unary.Write(ctx, jsonDB, 20*telem.SecondTS, jsonSeries)).To(Succeed())
					frame := MustSucceed(jsonDB.Read(ctx, (20 * telem.SecondTS).Range(22*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					series := frame.SeriesAt(0)
					Expect(series.Len()).To(Equal(int64(2)))
					Expect(string(series.At(0))).To(Equal(`{"key":"value"}`))
					Expect(string(series.At(1))).To(Equal(`{"num":42}`))
				})
				It("Should write and read bytes data", func(ctx SpecContext) {
					bytesDB := MustSucceed(unary.Open(ctx, unary.Config{
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

					Expect(unary.Write(ctx, indexDB, 30*telem.SecondTS,
						telem.NewSeriesSecondsTSV(30, 31, 32),
					)).To(Succeed())
					Expect(unary.Write(ctx, bytesDB, 30*telem.SecondTS,
						telem.NewSeriesV([]byte{1, 2, 3}, []byte{4, 5}, []byte{6}),
					)).To(Succeed())
					frame := MustSucceed(bytesDB.Read(ctx, (30 * telem.SecondTS).Range(33*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					samples := telem.UnmarshalSeries[[]byte](frame.SeriesAt(0))
					Expect(samples).To(Equal([][]byte{{1, 2, 3}, {4, 5}, {6}}))
				})
				It("Should write and read empty strings", func(ctx SpecContext) {
					Expect(unary.Write(ctx, indexDB, 40*telem.SecondTS,
						telem.NewSeriesSecondsTSV(40, 41, 42),
					)).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 40*telem.SecondTS,
						telem.NewSeriesV("", "nonempty", ""),
					)).To(Succeed())
					frame := MustSucceed(dataDB.Read(ctx, (40 * telem.SecondTS).Range(43*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(Equal([]string{"", "nonempty", ""}))
				})
				It("Should write and read strings containing newlines", func(ctx SpecContext) {
					Expect(unary.Write(ctx, indexDB, 50*telem.SecondTS,
						telem.NewSeriesSecondsTSV(50, 51),
					)).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 50*telem.SecondTS,
						telem.NewSeriesV("line1\nline2", "no newline"),
					)).To(Succeed())
					frame := MustSucceed(dataDB.Read(ctx, (50 * telem.SecondTS).Range(52*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(Equal([]string{"line1\nline2", "no newline"}))
				})
			})

			Describe("Offset cache", func() {
				It("Should rebuild the cached offset table after the domain grows", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("cache-refresh"))
					idx2 := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "idx2",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					defer func() { Expect(idx2.Close()).To(Succeed()) }()
					data2 := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("data")),
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "data2",
							DataType: telem.StringT,
							Index:    idx2.Channel().Key,
						},
					}))
					defer func() { Expect(data2.Close()).To(Succeed()) }()
					data2.SetIndex(idx2.Index())

					// Seed the index across both commits so both resolve against
					// the same domain.
					Expect(unary.Write(ctx, idx2, 600*telem.SecondTS,
						telem.NewSeriesSecondsTSV(600, 601, 602, 603, 604),
					)).To(Succeed())

					iw, _ := MustSucceed2(data2.OpenWriter(ctx, unary.WriterConfig{
						Start:   600 * telem.SecondTS,
						Subject: xcontrol.Subject{Key: "cache-refresh"},
					}))
					MustSucceed(iw.Write(telem.NewSeriesV("a", "b", "c")))
					MustSucceed(iw.Commit(ctx))

					// First read populates the cache for this domain with
					// sampleCount = 3, domainSize = bytes of "a","b","c".
					first := MustSucceed(data2.Read(ctx, (600 * telem.SecondTS).Range(603*telem.SecondTS)))
					Expect(telem.UnmarshalSeries[string](first.SeriesAt(0))).
						To(Equal([]string{"a", "b", "c"}))

					// Append two more samples against the same domain and commit.
					MustSucceed(iw.Write(telem.NewSeriesV("d", "e")))
					MustSucceed(iw.Commit(ctx))
					MustSucceed(iw.Close())

					// Sub-range read of the first four samples. The iterator view
					// ends inside the domain, so approximateEnd resolves endSample
					// from the index (= 4) and then calls byteOffset(4) on the
					// cached table. Without the size-based invalidation the
					// cache still reports sampleCount = 3, so 4 >= sampleCount
					// makes byteOffset return iter.Size() (end of the whole
					// domain) and the read spills into the fifth sample.
					sub := MustSucceed(data2.Read(ctx, (600 * telem.SecondTS).Range(604*telem.SecondTS)))
					Expect(telem.UnmarshalSeries[string](sub.SeriesAt(0))).
						To(Equal([]string{"a", "b", "c", "d"}))
				})
			})

			Describe("Writer", func() {
				It("Should track alignment correctly across writes", func(ctx SpecContext) {
					Expect(unary.Write(ctx, indexDB, 60*telem.SecondTS,
						telem.NewSeriesSecondsTSV(60, 61, 62, 63, 64),
					)).To(Succeed())
					w, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
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
					w, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
						Start:   70 * telem.SecondTS,
						Subject: xcontrol.Subject{Key: "wrong-type"},
					}))
					Expect(w.Write(telem.NewSeriesV[int64](1, 2, 3))).Error().To(MatchError(validate.ErrValidation))
					MustSucceed(w.Close())
				})
			})

			Describe("Iterator", func() {
				It("Should iterate forward through data", func(ctx SpecContext) {
					Expect(unary.Write(ctx, indexDB, 100*telem.SecondTS,
						telem.NewSeriesSecondsTSV(100, 101, 102, 103, 104),
					)).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 100*telem.SecondTS,
						telem.NewSeriesV("a", "b", "c", "d", "e"),
					)).To(Succeed())
					iter := MustSucceed(dataDB.OpenIterator(unary.IterRange(
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
					iter := MustSucceed(dataDB.OpenIterator(unary.IterRange(
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
					iter := MustSucceed(dataDB.OpenIterator(unary.IterRange(
						(101 * telem.SecondTS).Range(104 * telem.SecondTS),
					)))
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					Expect(iter.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
					f := iter.Value()
					Expect(f.Count()).To(Equal(1))
					Expect(telem.UnmarshalSeries[string](f.SeriesAt(0))).To(Equal([]string{"b", "c", "d"}))
					Expect(iter.Close()).To(Succeed())
				})
				It("Should iterate across multiple domains", func(ctx SpecContext) {
					Expect(unary.Write(ctx, indexDB, 200*telem.SecondTS,
						telem.NewSeriesSecondsTSV(200, 201, 202),
					)).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 200*telem.SecondTS,
						telem.NewSeriesV("x", "y", "z"),
					)).To(Succeed())

					Expect(unary.Write(ctx, indexDB, 203*telem.SecondTS,
						telem.NewSeriesSecondsTSV(203, 204),
					)).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 203*telem.SecondTS,
						telem.NewSeriesV("w", "v"),
					)).To(Succeed())

					iter := MustSucceed(dataDB.OpenIterator(unary.IterRange(
						(200 * telem.SecondTS).Range(205 * telem.SecondTS),
					)))
					Expect(iter.SeekFirst(ctx)).To(BeTrue())
					var all []string
					for iter.Next(ctx, telem.TimeSpanMax) {
						f := iter.Value()
						for i := 0; i < f.Count(); i++ {
							all = append(all, telem.UnmarshalSeries[string](f.SeriesAt(i))...)
						}
					}
					Expect(all).To(Equal([]string{"x", "y", "z", "w", "v"}))
					Expect(iter.Close()).To(Succeed())
				})
			})

			Describe("Delete", func() {
				It("Should delete data in a time range", func(ctx SpecContext) {
					Expect(unary.Write(ctx, indexDB, 300*telem.SecondTS,
						telem.NewSeriesSecondsTSV(300, 301, 302, 303, 304),
					)).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 300*telem.SecondTS,
						telem.NewSeriesV("del0", "del1", "del2", "del3", "del4"),
					)).To(Succeed())

					Expect(dataDB.Delete(ctx, (302 * telem.SecondTS).Range(304*telem.SecondTS))).To(Succeed())

					frame := MustSucceed(dataDB.Read(ctx, (300 * telem.SecondTS).Range(305*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(2))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(Equal([]string{"del0", "del1"}))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(1))).To(Equal([]string{"del4"}))
				})
			})

			Describe("GarbageCollect", func() {
				It("Should garbage collect after deletion", func(ctx SpecContext) {
					gcDB := MustSucceed(unary.Open(ctx, unary.Config{
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

					Expect(unary.Write(ctx, indexDB, 500*telem.SecondTS,
						telem.NewSeriesSecondsTSV(500, 501, 502, 503, 504),
					)).To(Succeed())
					Expect(unary.Write(ctx, gcDB, 500*telem.SecondTS,
						telem.NewSeriesV("gc0", "gc1", "gc2", "gc3", "gc4"),
					)).To(Succeed())

					Expect(gcDB.Delete(ctx, (502 * telem.SecondTS).Range(504*telem.SecondTS))).To(Succeed())
					Expect(gcDB.GarbageCollect(ctx)).To(Succeed())

					frame := MustSucceed(gcDB.Read(ctx, (500 * telem.SecondTS).Range(505*telem.SecondTS)))
					Expect(frame.Count()).To(Equal(2))
					Expect(gcDB.Close()).To(Succeed())
				})
			})
		})
	}
})
