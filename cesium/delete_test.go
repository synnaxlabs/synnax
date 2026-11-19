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
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db *cesium.DB
				fs fs.FS
			)
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				fs = openFS()
				db = mustOpenDBOnFS(ctx, fs)
			})
			Describe("Delete Channel", func() {
				var (
					uChannelKey = GenerateChannelKey()
					vChannelKey = GenerateChannelKey()
					uChannel    = cesium.Channel{
						Key:      uChannelKey,
						Name:     "Heraclitus",
						IsIndex:  true,
						DataType: telem.TimestampT,
					}
					vChannel = cesium.Channel{
						Key:      vChannelKey,
						Name:     "Parmenides",
						Virtual:  true,
						IsIndex:  false,
						DataType: telem.Int64T,
					}
				)
				Describe("Error paths", func() {
					Specify(
						"Deleting a nonexistent channel should be idempotent",
						func() {
							Expect(db.Drop([]cesium.ChannelKey{999})).To(Succeed())
						},
					)
					Specify(
						"Deleting a time range with a canceled context",
						func(ctx SpecContext) {
							key := GenerateChannelKey()
							Expect(db.CreateChannel(ctx, cesium.Channel{
								Key:      key,
								Name:     "Anaximander",
								IsIndex:  true,
								DataType: telem.TimestampT,
							})).To(Succeed())
							Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
								[]cesium.ChannelKey{key},
								[]telem.Series{telem.NewSeriesSecondsTSV(10, 11, 12)},
							))).To(Succeed())
							canceled, cancel := context.WithCancel(ctx)
							cancel()
							Expect(db.DeleteTimeRange(
								canceled, []cesium.ChannelKey{key}, telem.TimeRangeMax,
							)).To(MatchError(context.Canceled))
							fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key))
							Expect(fr.Get(key).Len()).To(Equal(int64(3)))
						},
					)
					Specify("Deleting a channel with db closed", func(ctx SpecContext) {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(ctx, sub)
						Expect(
							subDB.CreateChannel(
								ctx,
								cesium.Channel{
									Key:      key,
									Name:     "IndexChannel",
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
							),
						).To(Succeed())
						Expect(subDB.Close()).To(Succeed())

						Expect(
							subDB.Drop([]cesium.ChannelKey{key}),
						).To(MatchError(cesium.ErrDBClosed))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
					Describe(
						"Dropping a channel that is being written to",
						func() {
							Specify("Virtual Channel", func(ctx SpecContext) {
								Expect(db.CreateChannel(ctx, vChannel)).To(Succeed())
								w := MustSucceed(
									db.OpenWriter(
										ctx,
										cesium.WriterConfig{
											Channels: []cesium.ChannelKey{vChannelKey},
											Start:    10 * telem.SecondTS,
										},
									),
								)
								Expect(
									db.Drop([]cesium.ChannelKey{vChannelKey}),
								).To(Succeed())
								expectDropped(ctx, db, vChannelKey)
								// A virtual channel holds no files for its writers.
								expectRemoved(fs, vChannelKey)
								Expect(w.Close()).To(Succeed())
							})

							Specify("Unary Channel", func(ctx SpecContext) {
								Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
								w := MustSucceed(
									db.OpenWriter(
										ctx,
										cesium.WriterConfig{
											Channels: []cesium.ChannelKey{uChannelKey},
											Start:    10 * telem.SecondTS,
										},
									),
								)
								Expect(
									db.Drop([]cesium.ChannelKey{uChannelKey}),
								).To(Succeed())
								expectDropped(ctx, db, uChannelKey)
								Expect(
									fs.Exists(channelKeyToPath(uChannelKey)),
								).To(BeTrue())
								Expect(w.Close()).To(Succeed())
								expectRemoved(fs, uChannelKey)
							})
							Specify(
								"Two writers on a unary channel",
								func(ctx SpecContext) {
									Expect(
										db.CreateChannel(ctx, uChannel),
									).To(Succeed())
									cfg := cesium.WriterConfig{
										Start:    10 * telem.SecondTS,
										Channels: []cesium.ChannelKey{uChannelKey},
									}
									w1 := MustSucceed(db.OpenWriter(ctx, cfg))
									w2 := MustSucceed(db.OpenWriter(ctx, cfg))
									Expect(
										db.Drop([]cesium.ChannelKey{uChannelKey}),
									).To(Succeed())
									expectDropped(ctx, db, uChannelKey)
									Expect(w1.Close()).To(Succeed())
									Expect(
										fs.Exists(channelKeyToPath(uChannelKey)),
									).To(BeTrue())
									Expect(w2.Close()).To(Succeed())
									expectRemoved(fs, uChannelKey)
								},
							)
						},
					)
					Describe(
						"Dropping a channel that is being read from",
						func() {
							Specify("Unary Channel", func(ctx SpecContext) {
								Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
								i := MustSucceed(
									db.OpenIterator(
										cesium.IteratorConfig{
											Bounds:   telem.TimeRangeMax,
											Channels: []cesium.ChannelKey{uChannelKey},
										},
									),
								)
								Expect(
									db.Drop([]cesium.ChannelKey{uChannelKey}),
								).To(Succeed())
								expectDropped(ctx, db, uChannelKey)
								Expect(
									fs.Exists(channelKeyToPath(uChannelKey)),
								).To(BeTrue())
								Expect(i.Close()).To(Succeed())
								expectRemoved(fs, uChannelKey)
							})

							Specify(
								"Unary Channel double reader",
								func(ctx SpecContext) {
									Expect(
										db.CreateChannel(ctx, uChannel),
									).To(Succeed())
									i := MustSucceed(
										db.OpenIterator(
											cesium.IteratorConfig{
												Bounds: telem.TimeRangeMax,
												Channels: []cesium.ChannelKey{
													uChannelKey,
												},
											},
										),
									)
									i2 := MustSucceed(
										db.OpenIterator(
											cesium.IteratorConfig{
												Bounds: telem.TimeRangeMax,
												Channels: []cesium.ChannelKey{
													uChannelKey,
												},
											},
										),
									)
									Expect(
										db.Drop([]cesium.ChannelKey{uChannelKey}),
									).To(Succeed())
									expectDropped(ctx, db, uChannelKey)
									Expect(i.Close()).To(Succeed())
									Expect(
										fs.Exists(channelKeyToPath(uChannelKey)),
									).To(BeTrue())
									Expect(i2.Close()).To(Succeed())
									expectRemoved(fs, uChannelKey)
								},
							)
						},
					)
					Describe(
						"Dropping a channel that is being streamed from",
						func() {
							Describe("StreamIterator", func() {
								Specify("Unary", func(ctx SpecContext) {
									Expect(
										db.CreateChannel(ctx, uChannel),
									).To(Succeed())
									it := MustSucceed(
										db.NewStreamIterator(
											cesium.IteratorConfig{
												Bounds: telem.TimeRangeMax,
												Channels: []cesium.ChannelKey{
													uChannelKey,
												},
											},
										),
									)

									Expect(
										db.Drop([]cesium.ChannelKey{uChannelKey}),
									).To(Succeed())
									expectDropped(ctx, db, uChannelKey)
									Expect(
										fs.Exists(channelKeyToPath(uChannelKey)),
									).To(BeTrue())

									sCtx, cancel := signal.Isolated()
									i, _ := confluence.Attach(it, 1)
									it.Flow(sCtx)

									i.Close()
									Expect(sCtx.Wait()).To(Succeed())
									cancel()

									expectRemoved(fs, uChannelKey)
								})
							})

							Specify(
								"Dropping an index channel that other channels rely on should keep it",
								func(ctx SpecContext) {
									var (
										dependent = GenerateChannelKey()
										dependee  = GenerateChannelKey()
									)
									Expect(db.CreateChannel(
										ctx,
										cesium.Channel{
											Key:      dependent,
											Name:     "dependent",
											IsIndex:  true,
											DataType: telem.TimestampT,
										},
										cesium.Channel{
											Key:      dependee,
											Name:     "dependee",
											Index:    dependent,
											DataType: telem.Int64T,
										},
									)).To(Succeed())
									written := telem.MultiFrame(
										[]cesium.ChannelKey{dependent, dependee},
										[]telem.Series{
											telem.NewSeriesSecondsTSV(10, 11, 12),
											telem.NewSeriesV[int64](1, 2, 3),
										},
									)
									Expect(
										db.Write(ctx, 10*telem.SecondTS, written),
									).To(Succeed())

									By("Dropping channel")
									Expect(
										db.Drop([]cesium.ChannelKey{dependent}),
									).To(Succeed())
									MustSucceed(db.RetrieveChannel(ctx, dependent))
									fr := MustSucceed(db.Read(
										ctx,
										telem.TimeRangeMax,
										dependent,
										dependee,
									))
									Expect(fr.Get(dependent).Series[0]).To(
										telem.MatchSeriesData(
											telem.NewSeriesSecondsTSV(10, 11, 12),
										),
									)
									Expect(
										fr.Get(dependee).Series[0],
									).To(telem.MatchSeriesDataV[int64](1, 2, 3))
									w := MustSucceed(
										db.OpenWriter(ctx, cesium.WriterConfig{
											Channels: []cesium.ChannelKey{dependent},
											Start:    20 * telem.SecondTS,
										}),
									)
									Expect(w.Close()).To(Succeed())

									By("Dropping channel that depend on it")
									Expect(
										db.Drop([]cesium.ChannelKey{dependee}),
									).To(Succeed())

									By("Dropping the index channel again")
									Expect(
										db.Drop([]cesium.ChannelKey{dependent}),
									).To(Succeed())
									Expect(
										db.RetrieveChannel(ctx, 12),
									).Error().
										To(MatchError(cesium.ErrChannelNotFound))
								},
							)
						},
					)
				})
				Describe(
					"Deleting Index Channel when other channels depend on it",
					func() {
						It(
							"Should not allow such deletion when another channel is indexed by it on the same time range",
							func(ctx SpecContext) {
								By(
									"Creating an indexed channel and a channel indexed by it",
								)
								var (
									indexChannelKey = GenerateChannelKey()
									dataChannelKey  = GenerateChannelKey()
								)
								Expect(db.CreateChannel(
									ctx,
									cesium.Channel{
										Name:     "One",
										Key:      indexChannelKey,
										IsIndex:  true,
										DataType: telem.TimestampT,
									},
									cesium.Channel{
										Name:     "Orange",
										Key:      dataChannelKey,
										Index:    indexChannelKey,
										DataType: telem.Int64T,
									},
								)).To(Succeed())
								w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels: []cesium.ChannelKey{
										dataChannelKey,
										indexChannelKey,
									},
									Start: 10 * telem.SecondTS,
								}))

								By("Writing data to the channel")
								MustSucceed(w.Write(ctx, telem.MultiFrame(
									[]cesium.ChannelKey{
										dataChannelKey,
										indexChannelKey,
									},
									[]telem.Series{
										telem.NewSeriesV[int64](100, 101, 102),
										telem.NewSeriesSecondsTSV(10, 11, 12),
									},
								),
								))
								MustSucceed(w.Commit(ctx))
								Expect(w.Close()).To(Succeed())

								// Before deletion:
								// 10 11 12 13 14 15 16 17 18 19
								//  0  1  2  3  4  5  6  7  8  9

								By("Deleting channel data")
								Expect(db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{indexChannelKey},
									telem.TimeRange{
										Start: 12 * telem.SecondTS,
										End:   17 * telem.SecondTS,
									},
								)).To(MatchError(ContainSubstring("depending")))
							},
						)
					},
				)
				Describe("Happy paths", func() {
					var key cesium.ChannelKey
					BeforeEach(func() { key = GenerateChannelKey() })

					It(
						"Should delete an index unary channel WITHOUT OPENING WRITERS ON IT",
						func(ctx SpecContext) {
							By("Creating a channel")
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "Braincell",
									Key:      key,
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
							)).To(Succeed())

							By("Deleting the channel")
							Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
						},
					)

					It("Should delete an index unary channel", func(ctx SpecContext) {
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Name:     "Wir",
								Key:      key,
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
						)).To(Succeed())
						Expect(
							db.WriteSeries(
								ctx,
								key,
								10*telem.SecondTS,
								telem.NewSeriesSecondsTSV(10, 11, 12, 13),
							),
						).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())

						By("Deleting the channel")
						Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())

						By("The deletion should be complete when Drop returns")
						expectRemoved(fs, key)
						Expect(
							db.RetrieveChannel(ctx, key),
						).Error().
							To(MatchError(cesium.ErrChannelNotFound))

						By("We can also create a channel of the same key")
						Expect(
							db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "Martin",
									Key:      key,
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
							),
						).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, key))
						Expect(ch.Key).To(Equal(key))
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
					})
					It("Should delete a unary channel", func(ctx SpecContext) {
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Name:     "Foy",
								Key:      key,
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
						)).To(Succeed())
						Expect(
							db.WriteSeries(
								ctx,
								key,
								10*telem.SecondTS,
								telem.NewSeriesSecondsTSV(10, 11, 12, 13),
							),
						).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
						Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
						expectRemoved(fs, key)
						Expect(
							db.RetrieveChannel(ctx, key),
						).Error().
							To(MatchError(cesium.ErrChannelNotFound))

						Expect(
							db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "Vance",
									Key:      key,
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
							),
						).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, key))
						Expect(ch.Key).To(Equal(key))
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
					})
					It("Should delete a virtual channel", func(ctx SpecContext) {
						Expect(
							db.CreateChannel(
								ctx,
								cesium.Channel{
									Key:      key,
									Name:     "VirtualChannel",
									Virtual:  true,
									DataType: telem.Int64T,
								},
							),
						).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
						Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
						expectRemoved(fs, key)
						Expect(
							db.RetrieveChannel(ctx, key),
						).Error().
							To(MatchError(cesium.ErrChannelNotFound))

						Expect(
							db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "Sky",
									Key:      key,
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
							),
						).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, key))
						Expect(ch.Key).To(Equal(key))
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
					})
					It(
						"Should remove a dropped channel's files when the last iterator closes",
						func(ctx SpecContext) {
							Expect(db.CreateChannel(ctx, cesium.Channel{
								Key:      key,
								Name:     "Xenophanes",
								IsIndex:  true,
								DataType: telem.TimestampT,
							})).To(Succeed())
							i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
								Bounds:   telem.TimeRangeMax,
								Channels: []cesium.ChannelKey{key},
							}))
							Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
							Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
							Expect(i.Close()).To(Succeed())
							expectRemoved(fs, key)
						},
					)
					It(
						"Should keep the index readable for an open iterator on its dropped data channel",
						func(ctx SpecContext) {
							data := GenerateChannelKey()
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{
									Key:      key,
									Name:     "Empedocles",
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
								cesium.Channel{
									Key:      data,
									Name:     "Anaxagoras",
									Index:    key,
									DataType: telem.Int64T,
								},
							)).To(Succeed())
							Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
								[]cesium.ChannelKey{key, data},
								[]telem.Series{
									telem.NewSeriesSecondsTSV(10, 11, 12),
									telem.NewSeriesV[int64](1, 2, 3),
								},
							))).To(Succeed())
							i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
								Bounds:   telem.TimeRangeMax,
								Channels: []cesium.ChannelKey{data},
							}))
							Expect(
								db.Drop([]cesium.ChannelKey{data, key}),
							).To(Succeed())
							expectDropped(ctx, db, key)
							Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
							Expect(i.SeekGE(11 * telem.SecondTS)).To(BeTrue())
							Expect(i.Next(1 * telem.Second)).To(BeTrue())
							Expect(
								i.Value().Get(data).Series[0],
							).To(telem.MatchSeriesDataV[int64](2))
							Expect(i.Close()).To(Succeed())
							expectRemoved(fs, data)
							expectRemoved(fs, key)
						},
					)
					It(
						"Should remove a leftover deleted directory before it renames",
						func(ctx SpecContext) {
							Expect(db.CreateChannel(ctx, cesium.Channel{
								Key:      key,
								Name:     "Pythagoras",
								IsIndex:  true,
								DataType: telem.TimestampT,
							})).To(Succeed())
							leftover := MustSucceed(
								fs.Sub(channelKeyToPath(key) + "-DELETE"),
							)
							Expect(
								MustSucceed(
									leftover.Open("meta.json", os.O_CREATE),
								).Close(),
							).To(Succeed())
							Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
							expectRemoved(fs, key)
						},
					)
					It(
						"Should remove a dropped channel's files when its last writer closes",
						func(ctx SpecContext) {
							Expect(db.CreateChannel(ctx, cesium.Channel{
								Key:      key,
								Name:     "Democritus",
								IsIndex:  true,
								DataType: telem.TimestampT,
							})).To(Succeed())
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels: []cesium.ChannelKey{key},
								Start:    10 * telem.SecondTS,
							}))
							MustSucceed(w.Write(ctx, telem.UnaryFrame(
								key,
								telem.NewSeriesSecondsTSV(10, 11, 12),
							)))
							MustSucceed(w.Commit(ctx))
							Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
							Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
							Expect(w.Close()).To(Succeed())
							expectRemoved(fs, key)
						},
					)
				})
			})

			Describe("Delete Channels", Ordered, func() {
				var (
					index1   = GenerateChannelKey()
					data1    = GenerateChannelKey()
					index2   = GenerateChannelKey()
					data2    = GenerateChannelKey()
					data3    = GenerateChannelKey()
					index3   = GenerateChannelKey()
					virtual1 = GenerateChannelKey()
					channels = []cesium.Channel{
						{
							Name:     "Certs",
							Key:      index1,
							IsIndex:  true,
							DataType: telem.TimestampT,
							Index:    index1,
						},
						{Name: "2", Key: data1, DataType: telem.Int64T, Index: index1},
						{
							Name:     "Years",
							Key:      index2,
							IsIndex:  true,
							DataType: telem.TimestampT,
							Index:    index2,
						},
						{
							Name:     "Ago",
							Key:      data2,
							DataType: telem.Int64T,
							Index:    index2,
						},
						{
							Name:     "329",
							Key:      data3,
							DataType: telem.Int64T,
							Index:    index2,
						},
						{
							Name:     "TLS",
							Key:      index3,
							IsIndex:  true,
							Index:    index3,
							DataType: telem.TimestampT,
						},
						virtualChannel(virtual1, "Virtual"),
					}
				)
				BeforeEach(func(ctx SpecContext) {
					Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
				})
				AfterEach(func() {
					Expect(
						db.Drop(
							lo.Map(
								channels,
								func(c cesium.Channel, _ int) channel.Key { return c.Key },
							),
						),
					).To(Succeed())
					for _, c := range channels {
						expectRemoved(fs, c.Key)
					}
				})
				Describe("Happy paths", func() {
					It("Should be idempotent", func() {
						Expect(
							db.Drop([]cesium.ChannelKey{index1, data1}),
						).To(Succeed())
						Expect(
							db.Drop([]cesium.ChannelKey{index1, data1}),
						).To(Succeed())
						Expect(
							db.Drop([]cesium.ChannelKey{index1, data1}),
						).To(Succeed())
					})
					DescribeTable(
						"Deleting permutations of channels",
						func(ctx SpecContext, chs ...channel.Key) {
							Expect(db.Drop(chs)).To(Succeed())
							for _, c := range chs {
								Expect(
									db.RetrieveChannel(ctx, c),
								).Error().
									To(MatchError(cesium.ErrChannelNotFound))
								expectRemoved(fs, c)
							}
						},
						Entry("1 index 1 data", index1, data1),
						Entry("1 data", data1),
						Entry("2 data", data1, data2),
						Entry("1 index, 2 data", index2, data2, data3),
						Entry("data and unrelated index", data1, data2, index3),
						Entry("1 virtual", virtual1),
						Entry("a repeated key", data1, data1),
						Entry(
							"all",
							data1,
							data2,
							data3,
							index1,
							index2,
							index3,
							virtual1,
						),
					)
				})
				Describe("Error paths", func() {
					Specify("Deleting a channel with db closed", func(ctx SpecContext) {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(ctx, sub)
						Expect(
							subDB.CreateChannel(
								ctx,
								cesium.Channel{
									Key:      key,
									Name:     "IndexChannel",
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
							),
						).To(Succeed())
						Expect(subDB.Close()).To(Succeed())

						Expect(
							subDB.Drop([]cesium.ChannelKey{key}),
						).To(MatchError(cesium.ErrDBClosed))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
				})
				Describe("Channels in use", func() {
					BeforeEach(func(ctx SpecContext) {
						Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
							[]cesium.ChannelKey{index2, data2, data3},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(10, 11, 12),
								telem.NewSeriesV[int64](1, 2, 3),
								telem.NewSeriesV[int64](4, 5, 6),
							},
						))).To(Succeed())
					})
					It(
						"Should drop every channel when one has an open writer",
						func(ctx SpecContext) {
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Start:    20 * telem.SecondTS,
								Channels: []cesium.ChannelKey{data3},
							}))
							Expect(
								db.Drop([]cesium.ChannelKey{data2, data3}),
							).To(Succeed())
							expectDropped(ctx, db, data2)
							expectRemoved(fs, data2)
							expectDropped(ctx, db, data3)
							Expect(fs.Exists(channelKeyToPath(data3))).To(BeTrue())
							Expect(w.Close()).To(Succeed())
							expectRemoved(fs, data3)
						},
					)
					It(
						"Should keep an index that has outside data",
						func(ctx SpecContext) {
							Expect(
								db.Drop([]cesium.ChannelKey{data2, index2}),
							).To(Succeed())
							expectDropped(ctx, db, data2)
							expectRemoved(fs, data2)
							MustSucceed(db.RetrieveChannel(ctx, index2))
							fr := MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, index2, data3),
							)
							Expect(fr.Get(index2).Series[0]).To(telem.MatchSeriesData(
								telem.NewSeriesSecondsTSV(10, 11, 12),
							))
							Expect(
								fr.Get(data3).Series[0],
							).To(telem.MatchSeriesDataV[int64](4, 5, 6))
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Start:    20 * telem.SecondTS,
								Channels: []cesium.ChannelKey{index2},
							}))
							Expect(w.Close()).To(Succeed())
						},
					)
					It(
						"Should drop every channel when a virtual writer is open",
						func(ctx SpecContext) {
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Start:    20 * telem.SecondTS,
								Channels: []cesium.ChannelKey{virtual1},
							}))
							Expect(
								db.Drop([]cesium.ChannelKey{data2, virtual1}),
							).To(Succeed())
							expectDropped(ctx, db, data2)
							expectRemoved(fs, data2)
							expectDropped(ctx, db, virtual1)
							// A virtual channel holds no files for its writers.
							expectRemoved(fs, virtual1)
							Expect(w.Close()).To(Succeed())
						},
					)
				})
			})

			Describe("Delete chunks", Ordered, func() {
				var (
					basic1      = GenerateChannelKey()
					basic1Index = GenerateChannelKey()
					basic2Index = GenerateChannelKey()
					basic3Index = GenerateChannelKey()
					basic3      = GenerateChannelKey()
				)
				Describe("Error paths", func() {
					It(
						"Should return an error for deleting a non-existent channel",
						func(ctx SpecContext) {
							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{9999},
									telem.TimeRangeMax,
								),
							).To(MatchError(channel.ErrNotFound))
						},
					)
				})

				DescribeTable(
					"Should delete a run of repeated timestamps as one",
					func(ctx SpecContext, tr telem.TimeRange, expected []int64) {
						idx, data := GenerateChannelKey(), GenerateChannelKey()
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      idx,
								Name:     "Zeno",
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
							cesium.Channel{
								Key:      data,
								Name:     "Parmenides",
								Index:    idx,
								DataType: telem.Int64T,
							},
						)).To(Succeed())
						Expect(db.Write(ctx, telem.SecondTS, telem.MultiFrame(
							[]cesium.ChannelKey{idx, data},
							[]telem.Series{
								telem.NewSeriesSecondsTSV(1, 2, 2, 2, 3),
								telem.NewSeriesV[int64](10, 20, 30, 40, 50),
							},
						))).To(Succeed())
						Expect(db.DeleteTimeRange(
							ctx,
							[]cesium.ChannelKey{data},
							tr,
						)).To(Succeed())
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
						got := make([]int64, 0, f.Len())
						for _, s := range f.RawSeries() {
							got = append(got, s.Unmarshal[int64]()...)
						}
						Expect(got).To(Equal(expected))
					},
					Entry(
						"starting at the run",
						(2*telem.SecondTS).Range(3*telem.SecondTS),
						[]int64{10, 50},
					),
					Entry(
						"ending at the run",
						(1*telem.SecondTS).Range(2*telem.SecondTS),
						[]int64{20, 30, 40, 50},
					),
				)

				Describe("Simple Index-based channel", func() {
					It("Should delete chunks of a channel", func(ctx SpecContext) {
						By("Creating an indexed channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      basic1Index,
								Name:     "Democritus",
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
							cesium.Channel{
								Key:      basic1,
								Name:     "Epicurus",
								Index:    basic1Index,
								DataType: telem.Int64T,
							},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic1, basic1Index},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						MustSucceed(w.Write(ctx, telem.MultiFrame(
							[]cesium.ChannelKey{basic1, basic1Index},
							[]telem.Series{
								telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
								telem.NewSeriesSecondsTSV(
									10,
									11,
									12,
									13,
									14,
									15,
									16,
									17,
									18,
									19,
								),
							},
						),
						))
						MustSucceed(w.Commit(ctx))
						Expect(w.Close()).To(Succeed())

						// Before deletion:
						// 10 11 12 13 14 15 16 17 18 19
						//  0  1  2  3  4  5  6  7  8  9

						By("Deleting channel data")
						Expect(
							db.DeleteTimeRange(
								ctx,
								[]cesium.ChannelKey{basic1},
								telem.TimeRange{
									Start: 12 * telem.SecondTS,
									End:   17 * telem.SecondTS,
								},
							),
						).To(Succeed())

						// After deletion:
						// 10 11 12 13 14 15 16 17 18 19
						//  0  1                 7  8  9

						frame := MustSucceed(
							db.Read(
								ctx,
								telem.TimeRange{
									Start: 10 * telem.SecondTS,
									End:   20 * telem.SecondTS,
								},
								basic1,
							),
						)
						Expect(frame.Count()).To(Equal(2))
						Expect(
							frame.SeriesAt(0).TimeRange.End,
						).To(Equal(12 * telem.SecondTS))

						series0Data := frame.SeriesAt(0).Unmarshal[int64]()
						Expect(series0Data).To(ContainElement(int64(0)))
						Expect(series0Data).To(ContainElement(int64(1)))
						Expect(series0Data).ToNot(ContainElement(int64(2)))

						Expect(
							frame.SeriesAt(1).TimeRange.Start,
						).To(Equal(17 * telem.SecondTS))
						series1Data := frame.SeriesAt(1).Unmarshal[int64]()

						Expect(series1Data).ToNot(ContainElement(int64(6)))
						Expect(series1Data).To(ContainElement(int64(7)))
						Expect(series1Data).To(ContainElement(int64(8)))
						Expect(series1Data).To(ContainElement(int64(9)))
					})
				})

				Describe("Deleting simple index channel", func() {
					It(
						"Should Delete chunks off the index channel",
						func(ctx SpecContext) {
							By("Creating an indexed channel")
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{
									Key:      basic2Index,
									Name:     "Zeno",
									IsIndex:  true,
									DataType: telem.TimestampT,
								},
							)).To(Succeed())
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels: []cesium.ChannelKey{basic2Index},
								Start:    10 * telem.SecondTS,
							}))

							By("Writing data to the channel")
							MustSucceed(w.Write(ctx, telem.MultiFrame(
								[]cesium.ChannelKey{basic2Index},
								[]telem.Series{
									telem.NewSeriesSecondsTSV(
										10,
										11,
										12,
										13,
										14,
										15,
										16,
										17,
										18,
										19,
									),
								},
							),
							))
							MustSucceed(w.Commit(ctx))
							Expect(w.Close()).To(Succeed())

							// Before deletion:
							// 10 11 12 13 14 15 16 17 18 19
							//  0  1  2  3  4  5  6  7  8  9

							By("Deleting channel data")
							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{basic2Index},
									telem.TimeRange{
										Start: 12 * telem.SecondTS,
										End:   17 * telem.SecondTS,
									},
								),
							).To(Succeed())

							// After deletion:
							// 10 11                17 18 19

							frame := MustSucceed(
								db.Read(
									ctx,
									telem.TimeRange{
										Start: 10 * telem.SecondTS,
										End:   20 * telem.SecondTS,
									},
									basic2Index,
								),
							)
							Expect(frame.Count()).To(Equal(2))

							series0Data := frame.SeriesAt(0).Unmarshal[telem.TimeStamp]()
							Expect(series0Data).To(ContainElement(10 * telem.SecondTS))
							Expect(series0Data).To(ContainElement(11 * telem.SecondTS))
							Expect(
								series0Data,
							).ToNot(ContainElement(12 * telem.SecondTS))

							Expect(
								frame.SeriesAt(1).TimeRange.Start,
							).To(Equal(17 * telem.SecondTS))
							series1Data := frame.SeriesAt(1).Unmarshal[telem.TimeStamp]()

							Expect(
								series1Data,
							).ToNot(ContainElement(16 * telem.SecondTS))
							Expect(series1Data).To(ContainElement(17 * telem.SecondTS))
							Expect(series1Data).To(ContainElement(18 * telem.SecondTS))
							Expect(series1Data).To(ContainElement(19 * telem.SecondTS))
						},
					)
				})

				Describe(
					"Deleting Index Channel when other channels depend on it",
					func() {
						It(
							"Should not allow such deletion when another channel is indexed by it on the same time range",
							func(ctx SpecContext) {
								By(
									"Creating an indexed channel and a channel indexed by it",
								)
								Expect(db.CreateChannel(
									ctx,
									cesium.Channel{
										Name:     "Finish",
										Key:      basic3Index,
										IsIndex:  true,
										DataType: telem.TimestampT,
									},
									cesium.Channel{
										Name:     "My",
										Key:      basic3,
										Index:    basic3Index,
										DataType: telem.Int64T,
									},
								)).To(Succeed())
								w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels: []cesium.ChannelKey{basic3, basic3Index},
									Start:    10 * telem.SecondTS,
								}))

								By("Writing data to the channel")
								MustSucceed(w.Write(ctx, telem.MultiFrame(
									[]cesium.ChannelKey{basic3, basic3Index},
									[]telem.Series{
										telem.NewSeriesV[int64](100, 101, 102),
										telem.NewSeriesSecondsTSV(10, 11, 12),
									},
								),
								))
								MustSucceed(w.Commit(ctx))
								Expect(w.Close()).To(Succeed())

								// Before deletion:
								// 10 11 12 13 14 15 16 17 18 19
								//  0  1  2  3  4  5  6  7  8  9

								By("Deleting channel data")
								Expect(db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{basic3Index},
									telem.TimeRange{
										Start: 12 * telem.SecondTS,
										End:   17 * telem.SecondTS,
									},
								)).To(MatchError(ContainSubstring("depending")))
							},
						)
					},
				)

				Describe("Delete chunks in multiple channels", func() {
					var (
						index1 = GenerateChannelKey()
						basic1 = GenerateChannelKey()
						basic2 = GenerateChannelKey()
					)
					Describe("Happy paths", func() {
						Specify("Indexed channels", func(ctx SpecContext) {
							By("Creating channels")
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{
									Name:     "Best",
									Key:      index1,
									DataType: telem.TimestampT,
									IsIndex:  true,
								},
								cesium.Channel{
									Name:     "Life",
									Key:      basic1,
									DataType: telem.Int64T,
									Index:    index1,
								},
								cesium.Channel{
									Name:     "Bohmer",
									Key:      basic2,
									DataType: telem.Int64T,
									Index:    index1,
								},
							)).To(Succeed())
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels: []cesium.ChannelKey{index1, basic1, basic2},
								Start:    10 * telem.SecondTS,
							}))

							By("Writing data to the channel")
							MustSucceed(w.Write(ctx, telem.MultiFrame(
								[]cesium.ChannelKey{index1, basic1, basic2},
								[]telem.Series{
									telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104),
								},
							),
							))
							MustSucceed(w.Commit(ctx))
							Expect(w.Close()).To(Succeed())

							By("Deleting channel data")
							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{index1, basic1, basic2},
									telem.TimeRange{
										Start: 12 * telem.SecondTS,
										End:   14 * telem.SecondTS,
									},
								),
							).To(Succeed())

							frame := MustSucceed(
								db.Read(
									ctx,
									telem.TimeRange{
										Start: 10 * telem.SecondTS,
										End:   19 * telem.SecondTS,
									},
									basic1,
									basic2,
								),
							)
							basic1S := frame.Get(basic1)
							Expect(basic1S.Series).To(HaveLen(2))
							Expect(
								basic1S.Series[0].TimeRange.End,
							).To(Equal(12 * telem.SecondTS))
							Expect(
								basic1S.Series[0].Data,
							).To(Equal(telem.NewSeriesV[int64](100, 101).Data))
							Expect(
								basic1S.Series[1].TimeRange.Start,
							).To(Equal(14 * telem.SecondTS))
							Expect(
								basic1S.Series[1].Data,
							).To(Equal(telem.NewSeriesV[int64](104).Data))

							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{index1, basic1, basic2},
									telem.TimeRange{
										Start: 11 * telem.SecondTS,
										End:   20 * telem.SecondTS,
									},
								),
							).To(Succeed())

							frame = MustSucceed(
								db.Read(
									ctx,
									telem.TimeRange{
										Start: 10 * telem.SecondTS,
										End:   19 * telem.SecondTS,
									},
									basic1,
									basic2,
								),
							)
							basic1S = frame.Get(basic1)
							Expect(basic1S.Series).To(HaveLen(1))
							Expect(
								basic1S.TimeRange().End,
							).To(Equal(11 * telem.SecondTS))
							Expect(
								basic1S.Data(),
							).To(Equal(telem.NewSeriesV[int64](100).Data))

							By("Asserting that writes are still successful")
							w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels: []cesium.ChannelKey{index1, basic1, basic2},
								Start:    11 * telem.SecondTS,
							}))
							MustSucceed(w.Write(ctx, telem.MultiFrame(
								[]cesium.ChannelKey{index1, basic1, basic2},
								[]telem.Series{
									telem.NewSeriesSecondsTSV(11, 12, 13, 14),
									telem.NewSeriesV[int64](101, 102, 103, 104),
									telem.NewSeriesV[int64](101, 102, 103, 104),
								},
							),
							))
							MustSucceed(w.Commit(ctx))
							Expect(w.Close()).To(Succeed())

							Expect(w.Close()).To(Succeed())
							frame = MustSucceed(
								db.Read(
									ctx,
									telem.TimeRange{
										Start: 10 * telem.SecondTS,
										End:   19 * telem.SecondTS,
									},
									basic1,
									basic2,
								),
							)
							basic1S = frame.Get(basic1)
							Expect(basic1S.Series).To(HaveLen(2))
							Expect(
								basic1S.Series[0].TimeRange.End,
							).To(Equal(11 * telem.SecondTS))
							Expect(
								basic1S.Series[0].Data,
							).To(Equal(telem.NewSeriesV[int64](100).Data))
							Expect(
								basic1S.Series[1].TimeRange.Start,
							).To(Equal(11 * telem.SecondTS))
							Expect(
								basic1S.Series[1].Data,
							).To(Equal(telem.NewSeriesV[int64](101, 102, 103, 104).Data))
						})
					})
				})
				Describe("Error paths", func() {
					var (
						data     cesium.ChannelKey
						index    cesium.ChannelKey
						channels []cesium.Channel
					)
					BeforeEach(func(ctx SpecContext) {
						data = GenerateChannelKey()
						index = GenerateChannelKey()
						channels = []cesium.Channel{
							{
								Name:     "Meredi",
								Key:      index,
								IsIndex:  true,
								DataType: telem.TimestampT,
							},
							{
								Name:     "Performed",
								Key:      data,
								Index:    index,
								DataType: telem.Int64T,
							},
						}

						Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
					})
					It(
						"Should return ChannelNotFound when a channel does not exist",
						func(ctx SpecContext) {
							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{math.MaxUint32 - 10},
									telem.TimeRangeMax,
								),
							).To(MatchError(cesium.ErrChannelNotFound))
						},
					)
					It(
						"Should not delete any data when one channel does not exist",
						func(ctx SpecContext) {
							Expect(db.Write(ctx, 0, telem.MultiFrame(
								[]channel.Key{data, index},
								[]telem.Series{
									telem.NewSeriesV[int64](0, 1, 2, 3),
									telem.NewSeriesSecondsTSV(0, 1, 2, 3),
								},
							))).To(Succeed())
							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{
										data,
										index,
										math.MaxUint32 - 10,
									},
									(1 * telem.SecondTS).Range(2*telem.SecondTS),
								),
							).To(MatchError(cesium.ErrChannelNotFound))
							f := MustSucceed(
								db.Read(ctx, telem.TimeRangeMax, data, index),
							)
							Expect(f.Get(data).Series).To(HaveLen(1))
						},
					)
					It(
						"Should not return an error when trying to delete time range from virtual channel",
						func(ctx SpecContext) {
							virtualKey := GenerateChannelKey()
							Expect(
								db.CreateChannel(
									ctx,
									cesium.Channel{
										Key:      virtualKey,
										Name:     "VirtualChannel",
										Virtual:  true,
										DataType: telem.Int64T,
									},
								),
							).To(Succeed())
							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{virtualKey},
									telem.TimeRangeMax,
								),
							).To(Succeed())
						},
					)
					It(
						"Should not allow deletion of any channel while there is a writer that could write over it",
						func(ctx SpecContext) {
							Expect(
								db.WriteSeries(
									ctx,
									index,
									10*telem.SecondTS,
									telem.NewSeriesSecondsTSV(10, 11, 12, 13),
								),
							).To(Succeed())
							Expect(
								db.WriteSeries(
									ctx,
									data,
									10*telem.SecondTS,
									telem.NewSeriesV[int64](10, 11, 12, 13),
								),
							).To(Succeed())
							w := MustSucceed(
								db.OpenWriter(
									ctx,
									cesium.WriterConfig{
										Channels: []cesium.ChannelKey{data},
										Start:    9 * telem.SecondTS,
									},
								),
							)

							Expect(db.DeleteTimeRange(
								ctx,
								[]cesium.ChannelKey{data},
								(8 * telem.SecondTS).Range(11*telem.SecondTS),
							)).To(MatchError(ContainSubstring("cannot access time range")))

							By("Closing the writer and asserting we can now delete")
							Expect(w.Close()).To(Succeed())
							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{data},
									(8 * telem.SecondTS).Range(11*telem.SecondTS),
								),
							).To(Succeed())
							f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))

							Expect(
								f.SeriesAt(0).TimeRange,
							).To(Equal((11 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
							Expect(
								f.SeriesAt(0).Data,
							).To(Equal(telem.NewSeriesV[int64](11, 12, 13).Data))
						},
					)
					It(
						"Should delete normally if there is an open iterator on the channel",
						func(ctx SpecContext) {
							Expect(
								db.WriteSeries(
									ctx,
									index,
									10*telem.SecondTS,
									telem.NewSeriesSecondsTSV(10, 11, 12, 13),
								),
							).To(Succeed())
							Expect(
								db.WriteSeries(
									ctx,
									data,
									10*telem.SecondTS,
									telem.NewSeriesV[int64](10, 11, 12, 13),
								),
							).To(Succeed())
							i := MustSucceed(
								db.OpenIterator(
									cesium.IteratorConfig{
										Bounds:   telem.TimeRangeMax,
										Channels: []cesium.ChannelKey{data},
									},
								),
							)

							Expect(i.SeekFirst()).To(BeTrue())
							Expect(i.Next(10 * telem.Second)).To(BeTrue())
							Expect(
								i.Value().SeriesAt(0).Data,
							).To(Equal(telem.NewSeriesV[int64](10, 11, 12, 13).Data))

							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{data},
									(8 * telem.SecondTS).Range(11*telem.SecondTS),
								),
							).To(Succeed())
							Expect(i.SeekFirst()).To(BeTrue())
							Expect(i.Next(10 * telem.Second)).To(BeTrue())
							Expect(
								i.Value().SeriesAt(0).Data,
							).To(Equal(telem.NewSeriesV[int64](11, 12, 13).Data))

							Expect(i.Close()).To(Succeed())
						},
					)
					It(
						"Should not allow deletion of an index channel when there is a data channel depending on an index channel and has data for it",
						func(ctx SpecContext) {
							Expect(
								db.WriteSeries(
									ctx,
									index,
									10*telem.SecondTS,
									telem.NewSeriesSecondsTSV(10, 11, 12, 13),
								),
							).To(Succeed())
							Expect(
								db.WriteSeries(
									ctx,
									data,
									10*telem.SecondTS,
									telem.NewSeriesV[int64](10, 11, 12, 13),
								),
							).To(Succeed())

							Expect(db.DeleteTimeRange(
								ctx,
								[]cesium.ChannelKey{index},
								(8 * telem.SecondTS).Range(11*telem.SecondTS),
							)).To(MatchError(ContainSubstring(fmt.Sprintf("channel %v: cannot delete the index with channel %v depending on it on the time range %v", channels[0], channels[1], (8 * telem.SecondTS).Range(11*telem.SecondTS)))))
						},
					)
					It(
						"Should not allow deletion of an index channel when there is a data channel with a writer open before the time range",
						func(ctx SpecContext) {
							Expect(
								db.WriteSeries(
									ctx,
									index,
									10*telem.SecondTS,
									telem.NewSeriesSecondsTSV(10, 11, 12, 13),
								),
							).To(Succeed())
							w := MustSucceed(
								db.OpenWriter(
									ctx,
									cesium.WriterConfig{
										Channels: []cesium.ChannelKey{data},
										Start:    9 * telem.SecondTS,
									},
								),
							)

							Expect(db.DeleteTimeRange(
								ctx,
								[]cesium.ChannelKey{index},
								(8 * telem.SecondTS).Range(10*telem.SecondTS),
							)).To(MatchError(ContainSubstring(fmt.Sprintf("channel %v: cannot delete the index with channel %v depending on it on the time range %v", channels[0], channels[1], (8 * telem.SecondTS).Range(10*telem.SecondTS)))))

							Expect(
								db.DeleteTimeRange(
									ctx,
									[]cesium.ChannelKey{index},
									(8 * telem.SecondTS).Range(9*telem.SecondTS),
								),
							).To(Succeed())
							Expect(w.Close()).To(Succeed())
						},
					)
					It(
						"Should not allow deletion when there are multiple channels and one of them fails",
						func(ctx SpecContext) {
							data2 := GenerateChannelKey()
							data3 := GenerateChannelKey()
							newChannels := []cesium.Channel{
								{
									Name:     "JONAH",
									Key:      data2,
									Index:    index,
									DataType: telem.Int64T,
								},
								{
									Name:     "The Whale",
									Key:      data3,
									Index:    index,
									DataType: telem.Int64T,
								},
							}

							Expect(db.CreateChannel(ctx, newChannels...)).To(Succeed())

							Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
								[]cesium.ChannelKey{index, data, data2, data3},
								[]telem.Series{
									telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104),
								},
							),
							)).To(Succeed())

							Expect(db.DeleteTimeRange(
								ctx,
								[]cesium.ChannelKey{data, data2, index},
								(11 * telem.SecondTS).Range(14*telem.SecondTS),
							)).To(MatchError(ContainSubstring(fmt.Sprintf("channel %v: cannot delete the index with channel %v depending on it", channels[0], newChannels[1]))))
						},
					)
				})
			})
		})
	}
})

var _ = Describe("Removal", func() {
	var (
		fsys *FaultyFS
		db   *cesium.DB
		key  cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		fsys = WrapFaultyFS(FileSystems["memFS"]())
		// Each spec closes db itself, so db does not run.
		db = DeferClose(openDBOnFS(ctx, fsys))
		key = GenerateChannelKey()
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key:      key,
			Name:     "Anaximander",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
	})

	holdRemoval := func() (removing chan struct{}, release func()) {
		removing, held := make(chan struct{}), make(chan struct{})
		release = sync.OnceFunc(func() { close(held) })
		DeferCleanup(release)
		fsys.SetOptions(WithBeforeRemove(sync.OnceFunc(func() {
			close(removing)
			<-held
		})))
		return removing, release
	}

	It("Should remove the files before Drop returns", func() {
		removing, release := holdRemoval()
		dropped := make(chan error, 1)
		go func() { dropped <- db.Drop([]cesium.ChannelKey{key}) }()
		Eventually(removing).Should(BeClosed())
		Consistently(dropped).ShouldNot(Receive())
		release()
		Eventually(dropped).Should(Receive(BeNil()))
		expectRemoved(fsys, key)
	})

	It("Should leave the data after a failed removal and remove it on Open", func(
		ctx SpecContext,
	) {
		deleted := channelKeyToPath(key) + "-DELETE"
		fsys.SetOptions(WithBeforeRemove(func() {
			if MustSucceed(fsys.Exists(deleted)) {
				fsys.SetOptions(WithFailRemove())
			}
		}))
		Expect(db.Drop([]cesium.ChannelKey{key})).To(MatchError(ErrFault))
		Expect(db.Close()).To(Succeed())
		Expect(deletedDirs(fsys, key)).To(HaveLen(1))
		fsys.SetOptions()
		db = mustOpenDBOnFS(ctx, fsys)
		Expect(deletedDirs(fsys, key)).To(BeEmpty())
		Expect(db.RetrieveChannel(ctx, key)).Error().
			To(MatchError(cesium.ErrChannelNotFound))
	})

	It("Should remove a v0.61 deleted directory on Open", func(ctx SpecContext) {
		name := channelKeyToPath(key) + "-DELETE-123"
		leftover := MustSucceed(fsys.Sub(name))
		Expect(
			MustSucceed(leftover.Open("meta.json", os.O_CREATE)).Close(),
		).To(Succeed())
		Expect(db.Close()).To(Succeed())
		db = mustOpenDBOnFS(ctx, fsys)
		Expect(fsys.Exists(name)).To(BeFalse())
		MustSucceed(db.RetrieveChannel(ctx, key))
	})

	It(
		"Should drop later channels after a failed rename and keep the failed directory",
		func(ctx SpecContext) {
			later := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      later,
				Name:     "Thales",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			fsys.SetOptions(WithFailRename(channelKeyToPath(key)))
			Expect(
				db.Drop([]cesium.ChannelKey{key, later}),
			).To(MatchError(ErrFault))
			for _, k := range []cesium.ChannelKey{key, later} {
				Expect(db.RetrieveChannel(ctx, k)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
			}
			Expect(fsys.Exists(channelKeyToPath(later))).To(BeFalse())
			Expect(fsys.Exists(channelKeyToPath(key))).To(BeTrue())
			Expect(deletedDirs(fsys, key)).To(BeEmpty())
		},
	)

	It(
		"Should keep the files of an index when the removal of its data channel fails",
		func(ctx SpecContext) {
			data := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      data,
				Name:     "Thales",
				Index:    key,
				DataType: telem.Int64T,
			})).To(Succeed())
			Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
				[]cesium.ChannelKey{key, data},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesV[int64](1, 2, 3),
				},
			))).To(Succeed())
			fsys.SetOptions(WithFailRename(channelKeyToPath(data)))
			Expect(
				db.Drop([]cesium.ChannelKey{key, data}),
			).To(MatchError(ErrFault))
			for _, k := range []cesium.ChannelKey{key, data} {
				Expect(db.RetrieveChannel(ctx, k)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
				Expect(fsys.Exists(channelKeyToPath(k))).To(BeTrue())
				Expect(deletedDirs(fsys, k)).To(BeEmpty())
			}
		},
	)

	It("Should delete a channel whose files fail to close", func(ctx SpecContext) {
		fsys.SetOptions(WithFailClose("index.domain"))
		Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
		fsys.SetOptions()
		Expect(db.RetrieveChannel(ctx, key)).Error().
			To(MatchError(cesium.ErrChannelNotFound))
		Expect(fsys.Exists(channelKeyToPath(key))).To(BeFalse())
		Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
	})

	It("Should not delete a channel once the DB is closed", func() {
		Expect(db.Close()).To(Succeed())
		Expect(db.Drop([]cesium.ChannelKey{key})).To(MatchError(cesium.ErrDBClosed))
	})
})

// deletedDirs returns the deleted directories of key that still wait for removal.
func deletedDirs(fsys fs.FS, key cesium.ChannelKey) ([]string, error) {
	prefix := channelKeyToPath(key) + "-DELETE"
	entries, err := fsys.List("")
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, f := range entries {
		if strings.HasPrefix(f.Name(), prefix) {
			dirs = append(dirs, f.Name())
		}
	}
	return dirs, nil
}

// expectRemoved asserts that the directory of key and its deleted directories are
// gone.
func expectRemoved(fsys fs.FS, key cesium.ChannelKey) {
	GinkgoHelper()
	Expect(fsys.Exists(channelKeyToPath(key))).To(BeFalse())
	Expect(deletedDirs(fsys, key)).To(BeEmpty())
}

// expectDropped asserts that a new writer or iterator on key fails because the
// channel is not found.
func expectDropped(ctx context.Context, db *cesium.DB, key cesium.ChannelKey) {
	GinkgoHelper()
	Expect(db.OpenWriter(ctx, cesium.WriterConfig{
		Channels: []cesium.ChannelKey{key},
		Start:    telem.SecondTS,
	})).Error().To(MatchError(cesium.ErrChannelNotFound))
	Expect(db.OpenIterator(cesium.IteratorConfig{
		Bounds:   telem.TimeRangeMax,
		Channels: []cesium.ChannelKey{key},
	})).Error().To(MatchError(cesium.ErrChannelNotFound))
}
