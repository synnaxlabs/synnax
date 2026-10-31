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
	"slices"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/alignment"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// alignmentRun is a run of samples one second apart, written by one writer.
type alignmentRun struct {
	start telem.TimeStamp
	count int
}

func (r alignmentRun) stamps() []telem.TimeStamp {
	stamps := make([]telem.TimeStamp, r.count)
	for i := range stamps {
		stamps[i] = r.start + telem.TimeStamp(i)*telem.SecondTS
	}
	return stamps
}

// alignmentLayout describes how an index channel and the data channels that share it
// are written.
type alignmentLayout struct {
	name  string
	index []alignmentRun
	// data holds the runs of the data channels, written after the index. When it is
	// empty, the data channels are written in the same frames as the index.
	data []alignmentRun
}

// alignmentRead describes one way to read every channel of a layout.
type alignmentRead struct {
	bounds   telem.TimeRange
	backward bool
	span     telem.TimeSpan
	chunk    int64
	factor   uint32
}

var _ = Describe("Alignment", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			for _, layout := range []alignmentLayout{
				{
					name:  "Data written with its index",
					index: []alignmentRun{{10 * telem.SecondTS, 20}},
				},
				{
					name:  "Data that starts partway through an index domain",
					index: []alignmentRun{{10 * telem.SecondTS, 20}},
					data:  []alignmentRun{{12 * telem.SecondTS, 16}},
				},
				{
					name:  "Data that starts on a rolled over domain's first sample",
					index: []alignmentRun{{10 * telem.SecondTS, 20}},
					data:  []alignmentRun{{16 * telem.SecondTS, 8}},
				},
				{
					name:  "Data in two domains",
					index: []alignmentRun{{10 * telem.SecondTS, 20}},
					data: []alignmentRun{
						{10 * telem.SecondTS, 4},
						{17 * telem.SecondTS, 10},
					},
				},
				{
					name: "Index in two separate runs",
					index: []alignmentRun{
						{10 * telem.SecondTS, 9},
						{40 * telem.SecondTS, 9},
					},
				},
			} {
				Context(layout.name, Ordered, ContinueOnFailure, func() {
					var (
						db                             *cesium.DB
						idxKey, i64Key, i32Key, strKey cesium.ChannelKey
						dataKeys                       []cesium.ChannelKey
						// stampAt maps the alignment of each index sample to its
						// timestamp, and alignmentOf maps it back.
						stampAt     map[telem.Alignment]telem.TimeStamp
						alignmentOf map[telem.TimeStamp]telem.Alignment
						dataStamps  []telem.TimeStamp
						indexStamps []telem.TimeStamp
					)
					// series encodes the timestamp of each sample in its value.
					series := func(
						key cesium.ChannelKey,
						stamps []telem.TimeStamp,
					) telem.Series {
						GinkgoHelper()
						switch key {
						case idxKey:
							return telem.NewSeriesV(stamps...)
						case i64Key:
							values := make([]int64, len(stamps))
							for i, ts := range stamps {
								values[i] = int64(ts)
							}
							return telem.NewSeriesV(values...)
						case i32Key:
							values := make([]int32, len(stamps))
							for i, ts := range stamps {
								values[i] = int32(ts / telem.SecondTS)
							}
							return telem.NewSeriesV(values...)
						case strKey:
							values := make([]string, len(stamps))
							for i, ts := range stamps {
								values[i] = strconv.Itoa(int(ts / telem.SecondTS))
							}
							return telem.NewSeriesV(values...)
						}
						Fail("unknown channel key")
						return telem.Series{}
					}
					stampsOf := func(s telem.Series) []telem.TimeStamp {
						GinkgoHelper()
						switch s.DataType {
						case telem.TimestampT:
							return s.Unmarshal[telem.TimeStamp]()
						case telem.Int64T:
							stamps := make([]telem.TimeStamp, 0, s.Len())
							for _, v := range s.Unmarshal[int64]() {
								stamps = append(stamps, telem.TimeStamp(v))
							}
							return stamps
						case telem.Int32T:
							stamps := make([]telem.TimeStamp, 0, s.Len())
							for _, v := range s.Unmarshal[int32]() {
								stamps = append(
									stamps,
									telem.TimeStamp(v)*telem.SecondTS,
								)
							}
							return stamps
						}
						stamps := make([]telem.TimeStamp, 0, s.Len())
						for _, v := range s.Unmarshal[string]() {
							stamps = append(
								stamps,
								telem.TimeStamp(
									MustSucceed(strconv.Atoi(v)),
								)*telem.SecondTS,
							)
						}
						return stamps
					}
					// write commits run three samples at a time, so the file size cap
					// forces rollovers.
					write := func(
						ctx SpecContext,
						keys []cesium.ChannelKey,
						run alignmentRun,
					) {
						GinkgoHelper()
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: keys,
							Start:    run.start,
						}))
						for chunk := range slices.Chunk(run.stamps(), 3) {
							s := make([]telem.Series, len(keys))
							for i, k := range keys {
								s[i] = series(k, chunk)
							}
							Expect(MustSucceed(
								w.Write(telem.MultiFrame(keys, s)),
							)).To(BeTrue())
						}
						Expect(w.Close()).To(Succeed())
					}
					BeforeAll(func(ctx SpecContext) {
						ShouldNotLeakGoroutines()
						db = mustOpenDBOnFS(
							ctx,
							openFS(),
							cesium.WithFileSizeCap(40*telem.Byte),
						)
						idxKey, i64Key, i32Key, strKey = GenerateChannelKey(),
							GenerateChannelKey(), GenerateChannelKey(),
							GenerateChannelKey()
						dataKeys = []cesium.ChannelKey{i64Key, i32Key, strKey}
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idxKey,
								Name:     "Amundsen",
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
							cesium.Channel{
								Key:      i64Key,
								Name:     "Scott",
								Index:    idxKey,
								DataType: telem.Int64T,
							},
							cesium.Channel{
								Key:      i32Key,
								Name:     "Nansen",
								Index:    idxKey,
								DataType: telem.Int32T,
							},
							cesium.Channel{
								Key:      strKey,
								Name:     "Peary",
								Index:    idxKey,
								DataType: telem.StringT,
							},
						)).To(Succeed())
						dataStamps, indexStamps = nil, nil
						for _, run := range layout.index {
							keys := []cesium.ChannelKey{idxKey}
							if len(layout.data) == 0 {
								keys = append(keys, dataKeys...)
								dataStamps = append(dataStamps, run.stamps()...)
							}
							write(ctx, keys, run)
							indexStamps = append(indexStamps, run.stamps()...)
						}
						for _, run := range layout.data {
							write(ctx, dataKeys, run)
							dataStamps = append(dataStamps, run.stamps()...)
						}
						fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, idxKey))
						Expect(fr.Count()).To(BeNumerically(">", len(layout.index)),
							"the file size cap never forced a rollover")
						stampAt = make(map[telem.Alignment]telem.TimeStamp)
						alignmentOf = make(map[telem.TimeStamp]telem.Alignment)
						for s := range fr.Series() {
							for i, ts := range s.Unmarshal[telem.TimeStamp]() {
								a := s.Alignment.AddSamples(uint32(i))
								stampAt[a], alignmentOf[ts] = ts, a
							}
						}
						Expect(stampAt).To(HaveLen(len(indexStamps)))
					})
					DescribeTable(
						"Reading",
						func(ctx SpecContext, r alignmentRead) {
							i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
								Bounds: r.bounds,
								Channels: append(
									[]cesium.ChannelKey{idxKey},
									dataKeys...,
								),
								AutoChunkSize: r.chunk,
								Reduction: telem.Reduction{
									Variant: telem.StrideReduction{Factor: r.factor},
								},
							}))
							seek, step := i.SeekFirst, i.Next
							if r.backward {
								seek, step = i.SeekLast, i.Prev
							}
							factor := max(r.factor, 1)
							read := make(map[cesium.ChannelKey][]telem.TimeStamp)
							collect := func() {
								GinkgoHelper()
								for key, s := range i.Value().Entries() {
									Expect(s.Multiple()).To(Equal(factor))
									for j, ts := range stampsOf(s) {
										a := s.Alignment.AddSamples(
											uint32(j) * s.Multiple(),
										)
										Expect(stampAt).To(
											HaveKeyWithValue(a, ts),
											"channel %d sample at %s has alignment %s",
											key, ts, a,
										)
										Expect(a.SampleIndex() % factor).To(BeZero())
										Expect(s.TimeRange.ContainsStamp(ts)).To(
											BeTrue(),
											"sample at %s is outside %s",
											ts, s.TimeRange,
										)
										read[key] = append(read[key], ts)
									}
								}
							}
							if seek() {
								if r.span == cesium.AutoSpan ||
									r.span == telem.TimeSpanMax {
									for step(r.span) {
										collect()
									}
								} else {
									// An empty span returns false, so the read steps
									// over all of the bounds.
									for range (r.bounds.Span() + r.span - 1) / r.span {
										if step(r.span) {
											collect()
										}
									}
								}
							}
							Expect(i.Close()).To(Succeed())
							expected := func(
								stamps []telem.TimeStamp,
							) []telem.TimeStamp {
								var kept []telem.TimeStamp
								for _, ts := range stamps {
									if r.bounds.ContainsStamp(ts) &&
										alignmentOf[ts].SampleIndex()%factor == 0 {
										kept = append(kept, ts)
									}
								}
								return kept
							}
							got := read[idxKey]
							slices.Sort(got)
							Expect(got).To(Equal(expected(indexStamps)))
							for _, key := range dataKeys {
								got = read[key]
								slices.Sort(got)
								Expect(got).To(
									Equal(expected(dataStamps)),
									"channel %d", key,
								)
							}
						},
						Entry(
							"Should align a read of every sample at once",
							alignmentRead{
								bounds: telem.TimeRangeMax,
								span:   telem.TimeSpanMax,
							},
						),
						Entry("Should align forward chunks of 1", alignmentRead{
							bounds: telem.TimeRangeMax,
							span:   cesium.AutoSpan,
							chunk:  1,
						}),
						Entry("Should align forward chunks of 4", alignmentRead{
							bounds: telem.TimeRangeMax,
							span:   cesium.AutoSpan,
							chunk:  4,
						}),
						Entry("Should align forward chunks of 7", alignmentRead{
							bounds: telem.TimeRangeMax,
							span:   cesium.AutoSpan,
							chunk:  7,
						}),
						Entry("Should align backward chunks of 1", alignmentRead{
							bounds:   telem.TimeRangeMax,
							backward: true,
							span:     cesium.AutoSpan,
							chunk:    1,
						}),
						Entry("Should align backward chunks of 5", alignmentRead{
							bounds:   telem.TimeRangeMax,
							backward: true,
							span:     cesium.AutoSpan,
							chunk:    5,
						}),
						Entry("Should align backward chunks of 100", alignmentRead{
							bounds:   telem.TimeRangeMax,
							backward: true,
							span:     cesium.AutoSpan,
							chunk:    100,
						}),
						Entry("Should align forward spans of 3 seconds", alignmentRead{
							bounds: (5 * telem.SecondTS).Range(55 * telem.SecondTS),
							span:   3 * telem.Second,
						}),
						Entry("Should align backward spans of 4 seconds", alignmentRead{
							bounds:   (5 * telem.SecondTS).Range(55 * telem.SecondTS),
							backward: true,
							span:     4 * telem.Second,
						}),
						Entry(
							"Should align forward reads within inexact bounds",
							alignmentRead{
								bounds: (12*telem.SecondTS + telem.SecondTS/2).
									Range(45*telem.SecondTS + telem.SecondTS/2),
								span:  cesium.AutoSpan,
								chunk: 4,
							},
						),
						Entry(
							"Should align backward reads within inexact bounds",
							alignmentRead{
								bounds: (12*telem.SecondTS + telem.SecondTS/2).
									Range(45*telem.SecondTS + telem.SecondTS/2),
								backward: true,
								span:     cesium.AutoSpan,
								chunk:    4,
							},
						),
						Entry(
							"Should keep the same samples when downsampling at once",
							alignmentRead{
								bounds: telem.TimeRangeMax,
								span:   telem.TimeSpanMax,
								factor: 2,
							},
						),
						Entry(
							"Should keep the same samples when downsampling forward",
							alignmentRead{
								bounds: telem.TimeRangeMax,
								span:   cesium.AutoSpan,
								chunk:  4,
								factor: 3,
							},
						),
						Entry(
							"Should keep the same samples when downsampling backward",
							alignmentRead{
								bounds:   telem.TimeRangeMax,
								backward: true,
								span:     cesium.AutoSpan,
								chunk:    5,
								factor:   2,
							},
						),
						Entry(
							"Should read past forward chunks that keep no samples",
							alignmentRead{
								bounds: telem.TimeRangeMax,
								span:   cesium.AutoSpan,
								chunk:  1,
								factor: 2,
							},
						),
						Entry(
							"Should read past backward chunks that keep no samples",
							alignmentRead{
								bounds:   telem.TimeRangeMax,
								backward: true,
								span:     cesium.AutoSpan,
								chunk:    2,
								factor:   3,
							},
						),
						Entry(
							"Should downsample the same samples within inexact bounds",
							alignmentRead{
								bounds: (12*telem.SecondTS + telem.SecondTS/2).
									Range(45*telem.SecondTS + telem.SecondTS/2),
								span:   3 * telem.Second,
								factor: 3,
							},
						),
					)
				})
			}
		})
	}
})

var _ = Describe("NewAlignmentCounter", func() {
	It("Should hand out leading domains after the zero leading alignment", func() {
		c := cesium.NewAlignmentCounter()
		Expect(c.Next()).To(Equal(alignment.Leading(1, 0)))
		Expect(c.Next()).To(Equal(alignment.Leading(2, 0)))
	})
})
