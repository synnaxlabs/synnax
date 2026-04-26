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
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Variable-length channel", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				fs      xfs.FS
				indexDB *unary.DB
				dataDB  *unary.DB
			)
			BeforeAll(func(ctx SpecContext) {
				fs = openFS()
				indexDB = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("index")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "index",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				}))
				dataDB = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("data")),
					MetaCodec: json.Codec,
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
						MetaCodec: json.Codec,
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
						MetaCodec: json.Codec,
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
				It("Should serve reads from cache after commit without scanning length prefixes", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("flush-on-commit"))
					idx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "flush-on-commit-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					defer func() { Expect(idx.Close()).To(Succeed()) }()
					rec := xfs.NewRecorder(MustSucceed(subFS.Sub("data")))
					data := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        rec,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "flush-on-commit-data",
							DataType: telem.StringT,
							Index:    idx.Channel().Key,
						},
					}))
					defer func() { Expect(data.Close()).To(Succeed()) }()
					data.SetIndex(idx.Index())

					const sampleCount = 100
					indexSeries := make([]telem.TimeStamp, sampleCount)
					values := make([]string, sampleCount)
					for i := range sampleCount {
						indexSeries[i] = telem.TimeStamp(700+i) * telem.SecondTS
						values[i] = "sample"
					}
					Expect(unary.Write(ctx, idx, indexSeries[0],
						telem.NewSeriesV(indexSeries...),
					)).To(Succeed())
					Expect(unary.Write(ctx, data, indexSeries[0],
						telem.NewSeriesV(values...),
					)).To(Succeed())

					rec.Reset()
					readEnd := indexSeries[sampleCount-1] + telem.SecondTS
					frame := MustSucceed(data.Read(ctx, indexSeries[0].Range(readEnd)))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(HaveLen(sampleCount))

					// A cache miss does one 4-byte ReadAt per sample to walk length
					// prefixes, so the cache-hit path must do zero of them.
					var lengthPrefixReads int
					for _, e := range rec.Events() {
						if e.Op == xfs.OpReadAt && e.Length == 4 {
							lengthPrefixReads++
						}
					}
					Expect(lengthPrefixReads).To(BeZero())
				})

				It("Should rebuild a cold cache without per-sample length-prefix reads", func(ctx SpecContext) {
					// Write data through one DB instance and close it, then reopen
					// against the same FS so the second DB starts with an empty
					// cache. Reads must rebuild the cache by scanning the file,
					// which is the path the buffered scan optimizes.
					subFS := MustSucceed(fs.Sub("cold-rebuild"))
					seedIdx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "cold-rebuild-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					seedData := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("data")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "cold-rebuild-data",
							DataType: telem.StringT,
							Index:    seedIdx.Channel().Key,
						},
					}))
					seedData.SetIndex(seedIdx.Index())

					const sampleCount = 200
					indexSeries := make([]telem.TimeStamp, sampleCount)
					values := make([]string, sampleCount)
					for i := range sampleCount {
						indexSeries[i] = telem.TimeStamp(900+i) * telem.SecondTS
						values[i] = "rebuild-sample"
					}
					Expect(unary.Write(ctx, seedIdx, indexSeries[0],
						telem.NewSeriesV(indexSeries...),
					)).To(Succeed())
					Expect(unary.Write(ctx, seedData, indexSeries[0],
						telem.NewSeriesV(values...),
					)).To(Succeed())
					Expect(seedData.Close()).To(Succeed())
					Expect(seedIdx.Close()).To(Succeed())

					// Reopen against the same FS. Cache is empty.
					reopenIdx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      seedIdx.Channel().Key,
							Name:     "cold-rebuild-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					defer func() { Expect(reopenIdx.Close()).To(Succeed()) }()
					rec := xfs.NewRecorder(MustSucceed(subFS.Sub("data")))
					reopenData := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        rec,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      seedData.Channel().Key,
							Name:     "cold-rebuild-data",
							DataType: telem.StringT,
							Index:    reopenIdx.Channel().Key,
						},
					}))
					defer func() { Expect(reopenData.Close()).To(Succeed()) }()
					reopenData.SetIndex(reopenIdx.Index())

					rec.Reset()
					readEnd := indexSeries[sampleCount-1] + telem.SecondTS
					frame := MustSucceed(reopenData.Read(ctx, indexSeries[0].Range(readEnd)))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).To(HaveLen(sampleCount))

					// The rebuild walks length prefixes via a buffered scan, so
					// the recorder should see no 4-byte ReadAts even though the
					// cache started empty.
					var lengthPrefixReads int
					for _, e := range rec.Events() {
						if e.Op == xfs.OpReadAt && e.Length == 4 {
							lengthPrefixReads++
						}
					}
					Expect(lengthPrefixReads).To(BeZero())
				})

				It("Should populate the cache for both domains across a writer file rollover", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("rollover-flush"))
					idx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "rollover-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						FileSize: 40 * telem.Byte,
					}))
					defer func() { Expect(idx.Close()).To(Succeed()) }()
					rec := xfs.NewRecorder(MustSucceed(subFS.Sub("data")))
					data := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        rec,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "rollover-data",
							DataType: telem.StringT,
							Index:    idx.Channel().Key,
						},
						FileSize: 40 * telem.Byte,
					}))
					defer func() { Expect(data.Close()).To(Succeed()) }()
					data.SetIndex(idx.Index())

					// Seed the index with enough timestamps to back two data domains.
					Expect(unary.Write(ctx, idx, 800*telem.SecondTS,
						telem.NewSeriesSecondsTSV(800, 801, 802, 803, 804, 805),
					)).To(Succeed())

					// Open one data writer session that crosses a rollover.
					iw, _ := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:   800 * telem.SecondTS,
						Subject: xcontrol.Subject{Key: "rollover-flush"},
					}))
					// First batch: enough length-prefixed bytes to push the data file
					// past the 40-byte rollover threshold on the next commit.
					MustSucceed(iw.Write(telem.NewSeriesV(
						"sampleA", "sampleB", "sampleC", "sampleD", "sampleE",
					)))
					MustSucceed(iw.Commit(ctx))
					// Second batch lands in the second domain.
					MustSucceed(iw.Write(telem.NewSeriesV("sampleF")))
					MustSucceed(iw.Commit(ctx))
					MustSucceed(iw.Close())

					rec.Reset()
					frame := MustSucceed(data.Read(ctx,
						(800 * telem.SecondTS).Range(806*telem.SecondTS),
					))
					var got []string
					for i := 0; i < frame.Count(); i++ {
						got = append(got, telem.UnmarshalSeries[string](frame.SeriesAt(i))...)
					}
					Expect(got).To(Equal([]string{
						"sampleA", "sampleB", "sampleC", "sampleD", "sampleE", "sampleF",
					}))

					// Both domains' caches should have been published on commit /
					// rollover, so the read should not perform any length-prefix
					// scans on either data file.
					var lengthPrefixReads int
					for _, e := range rec.Events() {
						if e.Op == xfs.OpReadAt && e.Length == 4 {
							lengthPrefixReads++
						}
					}
					Expect(lengthPrefixReads).To(BeZero())
				})

				It("Should serve reads from cache after a second commit on the same domain without rebuilding", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("multi-commit-flush"))
					idx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "multi-commit-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					defer func() { Expect(idx.Close()).To(Succeed()) }()
					rec := xfs.NewRecorder(MustSucceed(subFS.Sub("data")))
					data := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        rec,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "multi-commit-data",
							DataType: telem.StringT,
							Index:    idx.Channel().Key,
						},
					}))
					defer func() { Expect(data.Close()).To(Succeed()) }()
					data.SetIndex(idx.Index())

					Expect(unary.Write(ctx, idx, 1000*telem.SecondTS,
						telem.NewSeriesSecondsTSV(1000, 1001, 1002, 1003, 1004),
					)).To(Succeed())

					iw, _ := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:   1000 * telem.SecondTS,
						Subject: xcontrol.Subject{Key: "multi-commit-flush"},
					}))
					MustSucceed(iw.Write(telem.NewSeriesV("a", "b", "c")))
					MustSucceed(iw.Commit(ctx))
					// Second commit on the same domain: publish should replace
					// the cache entry rather than waiting for the next read to
					// invalidate it via the size gate.
					MustSucceed(iw.Write(telem.NewSeriesV("d", "e")))
					MustSucceed(iw.Commit(ctx))
					MustSucceed(iw.Close())

					rec.Reset()
					frame := MustSucceed(data.Read(ctx,
						(1000 * telem.SecondTS).Range(1005*telem.SecondTS),
					))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).
						To(Equal([]string{"a", "b", "c", "d", "e"}))

					// The post-commit-2 cache entry should serve the read
					// directly. A rebuild scan would emit one 4-byte ReadAt
					// per sample.
					var lengthPrefixReads int
					for _, e := range rec.Events() {
						if e.Op == xfs.OpReadAt && e.Length == 4 {
							lengthPrefixReads++
						}
					}
					Expect(lengthPrefixReads).To(BeZero())
				})

				It("Should rebuild the cached offset table after the domain grows", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("cache-refresh"))
					idx2 := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
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
						MetaCodec: json.Codec,
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

			Describe("Control Handoff", func() {
				It("Should preserve byte offsets, alignment, and cached offsets when a higher-authority writer takes control", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("handoff-basic"))
					idx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "handoff-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					defer func() { Expect(idx.Close()).To(Succeed()) }()
					rec := xfs.NewRecorder(MustSucceed(subFS.Sub("data")))
					data := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        rec,
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "handoff-data",
							DataType: telem.StringT,
							Index:    idx.Channel().Key,
						},
					}))
					defer func() { Expect(data.Close()).To(Succeed()) }()
					data.SetIndex(idx.Index())

					Expect(unary.Write(ctx, idx, 700*telem.SecondTS,
						telem.NewSeriesSecondsTSV(700, 701, 702, 703, 704),
					)).To(Succeed())

					// Lower-authority writer takes initial control and commits
					// three samples, populating the cache for this domain.
					w1, t1 := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:     700 * telem.SecondTS,
						Authority: xcontrol.AuthorityAbsolute - 1,
						Subject:   xcontrol.Subject{Key: "w1"},
					}))
					Expect(t1.Occurred()).To(BeTrue())
					a1 := MustSucceed(w1.Write(telem.NewSeriesV("a", "b", "c")))
					Expect(a1.SampleIndex()).To(Equal(uint32(0)))
					MustSucceed(w1.Commit(ctx))

					// Higher-authority writer joins the same region and takes
					// control. With shared tracker state, its first write must
					// see the prior cumulative sample count and its writes must
					// be addressed at the correct domain byte offsets.
					w2, t2 := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:     700 * telem.SecondTS,
						Authority: xcontrol.AuthorityAbsolute,
						Subject:   xcontrol.Subject{Key: "w2"},
					}))
					Expect(t2.Occurred()).To(BeTrue())
					a2 := MustSucceed(w2.Write(telem.NewSeriesV("d", "e")))
					Expect(a2.SampleIndex()).To(Equal(uint32(3)))
					MustSucceed(w2.Commit(ctx))
					MustSucceed(w2.Close())
					MustSucceed(w1.Close())

					rec.Reset()
					frame := MustSucceed(data.Read(ctx,
						(700 * telem.SecondTS).Range(705*telem.SecondTS),
					))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).
						To(Equal([]string{"a", "b", "c", "d", "e"}))

					// w2's commit must publish a complete table for this
					// domain (covering w1's samples too), so the read above
					// is served entirely from cache.
					var lengthPrefixReads int
					for _, e := range rec.Events() {
						if e.Op == xfs.OpReadAt && e.Length == 4 {
							lengthPrefixReads++
						}
					}
					Expect(lengthPrefixReads).To(BeZero())
				})

				It("Should pick up the prior writer's uncommitted bytes when a higher-authority writer takes control", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("handoff-uncommitted"))
					idx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "handoff-uncommitted-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					defer func() { Expect(idx.Close()).To(Succeed()) }()
					data := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("data")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "handoff-uncommitted-data",
							DataType: telem.StringT,
							Index:    idx.Channel().Key,
						},
					}))
					defer func() { Expect(data.Close()).To(Succeed()) }()
					data.SetIndex(idx.Index())

					Expect(unary.Write(ctx, idx, 800*telem.SecondTS,
						telem.NewSeriesSecondsTSV(800, 801, 802, 803),
					)).To(Succeed())

					w1, _ := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:     800 * telem.SecondTS,
						Authority: xcontrol.AuthorityAbsolute - 1,
						Subject:   xcontrol.Subject{Key: "w1"},
					}))
					MustSucceed(w1.Write(telem.NewSeriesV("a", "b")))

					// Handoff happens before w1 commits. w2's first write must
					// continue from w1's cumulative byte position so the on-
					// disk records remain addressable.
					w2, _ := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:     800 * telem.SecondTS,
						Authority: xcontrol.AuthorityAbsolute,
						Subject:   xcontrol.Subject{Key: "w2"},
					}))
					a := MustSucceed(w2.Write(telem.NewSeriesV("c", "d")))
					Expect(a.SampleIndex()).To(Equal(uint32(2)))
					MustSucceed(w2.Commit(ctx))
					MustSucceed(w2.Close())
					MustSucceed(w1.Close())

					frame := MustSucceed(data.Read(ctx,
						(800 * telem.SecondTS).Range(804*telem.SecondTS),
					))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).
						To(Equal([]string{"a", "b", "c", "d"}))
				})

				It("Should preserve tracker state when control returns to the original writer", func(ctx SpecContext) {
					subFS := MustSucceed(fs.Sub("handoff-roundtrip"))
					idx := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("idx")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "handoff-roundtrip-idx",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					defer func() { Expect(idx.Close()).To(Succeed()) }()
					data := MustSucceed(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(subFS.Sub("data")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "handoff-roundtrip-data",
							DataType: telem.StringT,
							Index:    idx.Channel().Key,
						},
					}))
					defer func() { Expect(data.Close()).To(Succeed()) }()
					data.SetIndex(idx.Index())

					Expect(unary.Write(ctx, idx, 900*telem.SecondTS,
						telem.NewSeriesSecondsTSV(900, 901, 902, 903, 904, 905),
					)).To(Succeed())

					w1, _ := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:     900 * telem.SecondTS,
						Authority: xcontrol.AuthorityAbsolute - 1,
						Subject:   xcontrol.Subject{Key: "w1"},
					}))
					MustSucceed(w1.Write(telem.NewSeriesV("a", "b")))
					MustSucceed(w1.Commit(ctx))

					w2, _ := MustSucceed2(data.OpenWriter(ctx, unary.WriterConfig{
						Start:     900 * telem.SecondTS,
						Authority: xcontrol.AuthorityAbsolute,
						Subject:   xcontrol.Subject{Key: "w2"},
					}))
					MustSucceed(w2.Write(telem.NewSeriesV("c", "d")))
					MustSucceed(w2.Commit(ctx))
					MustSucceed(w2.Close())

					// w1 regains control. Its next write must continue from
					// the post-handoff cumulative count, not reset back to its
					// own pre-handoff position.
					a := MustSucceed(w1.Write(telem.NewSeriesV("e", "f")))
					Expect(a.SampleIndex()).To(Equal(uint32(4)))
					MustSucceed(w1.Commit(ctx))
					MustSucceed(w1.Close())

					frame := MustSucceed(data.Read(ctx,
						(900 * telem.SecondTS).Range(906*telem.SecondTS),
					))
					Expect(telem.UnmarshalSeries[string](frame.SeriesAt(0))).
						To(Equal([]string{"a", "b", "c", "d", "e", "f"}))
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
						MetaCodec:   json.Codec,
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
