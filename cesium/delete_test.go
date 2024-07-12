// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"math"
	"strconv"
)

var _ = Describe("Delete", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db      *cesium.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
				db = openDBOnFS(fs)
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})
			Describe("Delete Channel", func() {
				var (
					uChannelKey = GenerateChannelKey()
					vChannelKey = GenerateChannelKey()
					uChannel    = cesium.Channel{Key: uChannelKey, IsIndex: false, Rate: 1 * telem.Hz, DataType: telem.Int64T}
					vChannel    = cesium.Channel{Key: vChannelKey, Virtual: true, IsIndex: false, DataType: telem.Int64T}
				)
				Describe("Error paths", func() {
					Specify("Deleting a nonexistent channel should be idempotent", func() {
						Expect(db.DeleteChannel(999)).To(Succeed())
					})
					Specify("Deleting a channel with db closed", func() {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(sub)
						Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.BytesT})).To(Succeed())
						Expect(subDB.Close()).To(Succeed())

						err := subDB.DeleteChannel(key)
						Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
					Describe("Deleting a channel that is being written to should error", func() {
						Specify("Virtual Channel", func() {
							Expect(db.CreateChannel(ctx, vChannel)).To(Succeed())
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{vChannelKey}, Start: 10 * telem.SecondTS}))
							Expect(db.DeleteChannel(vChannelKey)).To(MatchError(ContainSubstring("1 unclosed writers")))
							Expect(w.Close()).To(Succeed())
							Expect(db.DeleteChannel(vChannelKey)).To(Succeed())
						})

						Specify("Unary Channel", func() {
							Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{uChannelKey}, Start: 10 * telem.SecondTS}))
							Expect(db.DeleteChannel(uChannelKey)).To(MatchError(ContainSubstring("1 unclosed writers/iterators")))
							Expect(w.Close()).To(Succeed())
							Expect(db.DeleteChannel(uChannelKey)).To(Succeed())
						})
					})
					Describe("Deleting a channel that is being read from should error", func() {
						Specify("Unary Channel", func() {
							Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
							i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannelKey}}))
							Expect(db.DeleteChannel(uChannelKey)).To(MatchError(ContainSubstring("1 unclosed writers/iterators")))
							Expect(i.Close()).To(Succeed())
							Expect(db.DeleteChannel(uChannelKey)).To(Succeed())
						})

						Specify("Unary Channel double reader", func() {
							Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
							i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannelKey}}))
							i2 := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannelKey}}))
							Expect(db.DeleteChannel(uChannelKey)).To(MatchError(ContainSubstring("2 unclosed writers/iterators")))
							Expect(i.Close()).To(Succeed())
							Expect(i2.Close()).To(Succeed())
							Expect(db.DeleteChannel(uChannelKey)).To(Succeed())
						})
					})
					Describe("Deleting a channel that is being streamed from should error", func() {
						Specify("Virtual Channel", func() {
							By("Creating a channel")
							Expect(db.CreateChannel(ctx, vChannel)).To(Succeed())
							By("Opening a streamer on the channel")
							s := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{Channels: []cesium.ChannelKey{vChannelKey}}))
							sCtx, cancel := signal.WithCancel(ctx)

							By("Start streaming")
							i, _ := confluence.Attach(s, 1)
							s.Flow(sCtx, confluence.CloseInletsOnExit())

							By("Expecting delete channel to fail because there is an open streamer")
							err := db.DeleteChannel(vChannelKey)
							Expect(err).ToNot(HaveOccurred())

							By("All other operations should still happen without error")
							cancel()
							i.Close()
						})

						Specify("Unary Channel", func() {
							By("Creating a channel")
							Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
							By("Opening a streamer on the channel")
							s := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{Channels: []cesium.ChannelKey{uChannelKey}}))
							sCtx, cancel := signal.WithCancel(ctx)

							By("Start streaming")
							i, _ := confluence.Attach(s, 1)
							s.Flow(sCtx, confluence.CloseInletsOnExit())

							By("Expecting delete channel to fail because there is an open streamer")
							err := db.DeleteChannel(uChannelKey)
							Expect(err).ToNot(HaveOccurred())

							By("All other operations should still happen without error")
							cancel()
							i.Close()
						})

						Describe("StreamIterator", func() {
							Specify("Unary", func() {
								Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
								it := MustSucceed(db.NewStreamIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannelKey}}))

								err := db.DeleteChannel(uChannelKey)
								Expect(err).To(MatchError(ContainSubstring("1 unclosed writers/iterators")))

								sCtx, cancel := signal.Isolated()
								i, _ := confluence.Attach(it, 1)
								it.Flow(sCtx)

								i.Close()
								Expect(sCtx.Wait()).To(Succeed())
								cancel()

								Expect(db.DeleteChannel(uChannelKey)).To(Succeed())
							})
						})

						Describe("StreamWriter", func() {
							Specify("Virtual", func() {
								Expect(db.CreateChannel(ctx, vChannel)).To(Succeed())
								it := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []cesium.ChannelKey{vChannelKey}}))

								err := db.DeleteChannel(vChannelKey)
								Expect(err).To(MatchError(ContainSubstring("1 unclosed writers")))

								sCtx, cancel := signal.Isolated()
								i, _ := confluence.Attach(it, 1)
								it.Flow(sCtx)

								i.Close()
								Expect(sCtx.Wait()).To(Succeed())
								cancel()

								Expect(db.DeleteChannel(vChannelKey)).To(Succeed())
							})

							Specify("Unary", func() {
								Expect(db.CreateChannel(ctx, uChannel)).To(Succeed())
								it1 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []cesium.ChannelKey{uChannelKey}}))
								it2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []cesium.ChannelKey{uChannelKey}}))

								err := db.DeleteChannel(uChannelKey)
								Expect(err).To(MatchError(ContainSubstring("2 unclosed writers/iterators")))

								sCtx, cancel := signal.Isolated()
								i1, _ := confluence.Attach(it1, 1)
								i2, _ := confluence.Attach(it2, 1)
								it1.Flow(sCtx)
								it2.Flow(sCtx)

								i1.Close()
								i2.Close()
								Expect(sCtx.Wait()).To(Succeed())
								cancel()

								Expect(db.DeleteChannel(uChannelKey)).To(Succeed())
							})
						})
						Specify("Deleting an index channel that other channels rely on should error", func() {
							var (
								dependent = GenerateChannelKey()
								dependee  = GenerateChannelKey()
							)
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{Key: dependent, Name: "dependent", IsIndex: true, DataType: telem.TimeStampT},
								cesium.Channel{Key: dependee, Name: "dependee", Index: dependent, DataType: telem.Int64T},
							)).To(Succeed())

							By("Deleting channel")
							err := db.DeleteChannel(dependent)
							Expect(err).To(HaveOccurred())
							Expect(err).To(MatchError(ContainSubstring("cannot delete channel [dependent]<%d> because it indexes data in channel [dependee]<%d>", dependent, dependee)))

							By("Deleting channel that depend on it")
							Expect(db.DeleteChannel(dependee)).To(Succeed())

							By("Deleting the index channel again")
							Expect(db.DeleteChannel(dependent)).To(Succeed())
							_, err = db.RetrieveChannel(ctx, 12)
							Expect(err).To(MatchError(cesium.ErrChannelNotFound))
						})
						Specify("Deleting control digest channel should error", func() {
							controlKey := GenerateChannelKey()
							Expect(db.ConfigureControlUpdateChannel(ctx, controlKey)).To(Succeed())
							Expect(db.DeleteChannel(controlKey)).To(MatchError(ContainSubstring("1 unclosed writers")))
						})
					})
				})
				Describe("Deleting Index Channel when other channels depend on it", func() {
					It("Should not allow such deletion when another channel is indexed by it on the same time range", func() {
						By("Creating an indexed channel and a channel indexed by it")
						var (
							indexChannelKey = GenerateChannelKey()
							dataChannelKey  = GenerateChannelKey()
						)
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: indexChannelKey, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: dataChannelKey, Index: indexChannelKey, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{dataChannelKey, indexChannelKey},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{dataChannelKey, indexChannelKey},
							[]telem.Series{
								telem.NewSeriesV[int64](100, 101, 102),
								telem.NewSecondsTSV(10, 11, 12),
							}),
						)
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						// Before deletion:
						// 10 11 12 13 14 15 16 17 18 19
						//  0  1  2  3  4  5  6  7  8  9

						By("Deleting channel data")
						err := db.DeleteTimeRange(ctx, []cesium.ChannelKey{indexChannelKey}, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   17 * telem.SecondTS,
						})

						Expect(err).To(MatchError(ContainSubstring("depending")))
					})
				})
				Describe("Happy paths", func() {
					var key cesium.ChannelKey
					BeforeEach(func() { key = GenerateChannelKey() })

					It("Should delete an index unary channel", func() {
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: key, IsIndex: true, DataType: telem.TimeStampT},
						)).To(Succeed())
						Expect(db.WriteArray(ctx, key, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13))).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())

						By("Deleting the channel")
						Expect(db.DeleteChannel(key)).To(Succeed())

						By("Eventually, the deletion should be completed")
						Expect(fs.Exists(channelKeyToPath(key))).To(BeFalse())

						By("Eventually, the deletion should be completed")
						Eventually(MustSucceed(fs.Exists(channelKeyToPath(key)))).Should(BeFalse())
						for _, f := range MustSucceed(fs.List("")) {
							Eventually(f.Name()).ShouldNot(HavePrefix(channelKeyToPath(key) + "-DELETE-"))
						}
						_, err := db.RetrieveChannel(ctx, key)
						Expect(err).To(MatchError(cesium.ErrChannelNotFound))

						By("We can also create a channel of the same key")
						Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, IsIndex: true, DataType: telem.TimeStampT})).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, key))
						Expect(ch.Key).To(Equal(key))
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
					})
					It("Should delete a unary channel", func() {
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T},
						)).To(Succeed())
						Expect(db.WriteArray(ctx, key, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13))).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
						Expect(db.DeleteChannel(key)).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeFalse())
						Eventually(MustSucceed(fs.Exists(channelKeyToPath(key)))).Should(BeFalse())
						for _, f := range MustSucceed(fs.List("")) {
							Eventually(f.Name()).ShouldNot(HavePrefix(channelKeyToPath(key) + "-DELETE-"))
						}
						_, err := db.RetrieveChannel(ctx, key)
						Expect(err).To(MatchError(cesium.ErrChannelNotFound))

						Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, key))
						Expect(ch.Key).To(Equal(key))
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
					})
					It("Should delete a virtual channel", func() {
						Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Virtual: true, DataType: telem.Int64T})).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
						Expect(db.DeleteChannel(key)).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(key))).To(BeFalse())
						Eventually(MustSucceed(fs.Exists(channelKeyToPath(key)))).Should(BeFalse())
						for _, f := range MustSucceed(fs.List("")) {
							Eventually(f.Name()).ShouldNot(HavePrefix(channelKeyToPath(key) + "-DELETE-"))
						}
						_, err := db.RetrieveChannel(ctx, key)
						Expect(err).To(MatchError(cesium.ErrChannelNotFound))

						Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, key))
						Expect(ch.Key).To(Equal(key))
						Expect(fs.Exists(channelKeyToPath(key))).To(BeTrue())
					})
				})
			})

			Describe("Delete Channels", Ordered, func() {
				var (
					index1   = GenerateChannelKey()
					data1    = GenerateChannelKey()
					index2   = GenerateChannelKey()
					data2    = GenerateChannelKey()
					data3    = GenerateChannelKey()
					rate     = GenerateChannelKey()
					index3   = GenerateChannelKey()
					channels = []cesium.Channel{{Key: index1, IsIndex: true, DataType: telem.TimeStampT, Index: index1},
						{Key: data1, DataType: telem.Int64T, Index: index1},
						{Key: index2, IsIndex: true, DataType: telem.TimeStampT, Index: index2},
						{Key: data2, DataType: telem.Int64T, Index: index2},
						{Key: data3, DataType: telem.Int64T, Index: index2},
						{Key: rate, DataType: telem.Int64T, Rate: 2 * telem.Hz},
						{Key: index3, IsIndex: true, Index: index3, DataType: telem.TimeStampT},
					}
				)
				BeforeEach(func() {
					Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
				})
				AfterEach(func() {
					Expect(db.DeleteChannels(lo.Map(channels, func(c cesium.Channel, _ int) core.ChannelKey { return c.Key }))).To(Succeed())
					for _, c := range channels {
						Expect(fs.Exists(channelKeyToPath(c.Key))).To(BeFalse())
						for _, f := range MustSucceed(fs.List("")) {
							Eventually(f.Name()).ShouldNot(HavePrefix(strconv.Itoa(int(c.Key)) + "-DELETE-"))
						}
					}
				})
				Describe("Happy paths", func() {
					It("Should be idempotent", func() {
						Expect(db.DeleteChannels([]cesium.ChannelKey{index1, data1})).To(Succeed())
						Expect(db.DeleteChannels([]cesium.ChannelKey{index1, data1})).To(Succeed())
						Expect(db.DeleteChannels([]cesium.ChannelKey{index1, data1})).To(Succeed())
					})
					DescribeTable("Deleting permutations of channels", func(chs ...core.ChannelKey) {
						Expect(db.DeleteChannels(chs)).To(Succeed())
						for _, c := range chs {
							_, err := db.RetrieveChannel(ctx, c)
							Expect(err).To(MatchError(cesium.ErrChannelNotFound))
							Eventually(MustSucceed(fs.Exists(channelKeyToPath(c)))).Should(BeFalse())
							for _, f := range MustSucceed(fs.List("")) {
								Eventually(f.Name()).ShouldNot(HavePrefix(strconv.Itoa(int(c)) + "-DELETE-"))
							}
						}
					},
						Entry("1 index 1 data", index1, data1),
						Entry("1 data", data1),
						Entry("2 data", data1, data2),
						Entry("1 index, 2 data", index2, data2, data3),
						Entry("rate", rate),
						Entry("data and rate", data1, data2, data3, rate),
						Entry("rate and index", rate, index3),
						Entry("data and unrelated index", data1, data2, index3),
						Entry("all", data1, data2, data3, index1, index2, index3, rate),
					)
				})
				Describe("Error paths", func() {
					Specify("Deleting a channel with db closed", func() {
						sub := MustSucceed(fs.Sub("closed-fs"))
						key := cesium.ChannelKey(1)
						subDB := openDBOnFS(sub)
						Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.BytesT})).To(Succeed())
						Expect(subDB.Close()).To(Succeed())

						err := subDB.DeleteChannels([]cesium.ChannelKey{key})
						Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

						Expect(fs.Remove("closed-fs")).To(Succeed())
					})
				})
				Describe("Interrupted deletion", func() {
					It("Should delete all channels before encountering an error", func() {
						w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []core.ChannelKey{data2, data3}}))
						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []core.ChannelKey{data2}}))
						Expect(db.DeleteChannels([]cesium.ChannelKey{rate, data1, data2})).To(MatchError(ContainSubstring("2 unclosed writer")))
						Expect(w1.Close()).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(rate))).To(BeFalse())
						Expect(fs.Exists(channelKeyToPath(data1))).To(BeFalse())
						Expect(fs.Exists(channelKeyToPath(data2))).To(BeTrue())
						_, err := db.RetrieveChannel(ctx, rate)
						Expect(err).To(MatchError(cesium.ErrChannelNotFound))
						_, err = db.RetrieveChannel(ctx, data1)
						Expect(err).To(MatchError(cesium.ErrChannelNotFound))
						_, err = db.RetrieveChannel(ctx, data2)
						Expect(err).To(BeNil())
						Expect(db.DeleteChannels([]cesium.ChannelKey{rate, data1, data3})).To(Succeed())
						Expect(w2.Close()).To(Succeed())

						Expect(db.CreateChannel(ctx, cesium.Channel{Key: rate, Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
						ch := MustSucceed(db.RetrieveChannel(ctx, rate))
						Expect(ch.Key).To(Equal(rate))
						Expect(fs.Exists(channelKeyToPath(rate))).To(BeTrue())

						Expect(db.CreateChannel(ctx, cesium.Channel{Key: data1, Index: index2, DataType: telem.Int64T})).To(Succeed())
						ch = MustSucceed(db.RetrieveChannel(ctx, data1))
						Expect(ch.Key).To(Equal(data1))
						Expect(fs.Exists(channelKeyToPath(data1))).To(BeTrue())
					})
					It("Should delete all channels, but not index channels, before encountering an error", func() {
						i1 := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{rate, data3}}))
						i2 := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{data3}}))
						Expect(db.DeleteChannels([]cesium.ChannelKey{index1, index2, data1, data2, data3})).To(MatchError(ContainSubstring("2 unclosed writer")))
						Expect(i1.Close()).To(Succeed())
						Expect(i2.Close()).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(data2))).To(BeFalse())
						_, err := db.RetrieveChannel(ctx, data2)
						Expect(err).To(MatchError(cesium.ErrChannelNotFound))
						Expect(fs.Exists(channelKeyToPath(index1))).To(BeTrue())
						Expect(fs.Exists(channelKeyToPath(index2))).To(BeTrue())
						_, err = db.RetrieveChannel(ctx, index1)
						Expect(err).To(BeNil())
						_, err = db.RetrieveChannel(ctx, index2)
						Expect(err).To(BeNil())
					})
					It("Should error when there is an error in deleting index channels", func() {
						i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{index1}}))
						Expect(db.DeleteChannels([]cesium.ChannelKey{data1, index1})).To(MatchError(ContainSubstring("1 unclosed")))
						Expect(i.Close()).To(Succeed())
						Expect(fs.Exists(channelKeyToPath(data1))).To(BeFalse())
						_, err := db.RetrieveChannels(ctx, data1)
						Expect(err).To(HaveOccurredAs(cesium.ErrChannelNotFound))
					})
				})
			})

			Describe("Delete chunks", Ordered, func() {
				var (
					basic1      = GenerateChannelKey()
					basic2      = GenerateChannelKey()
					basic2index = GenerateChannelKey()
					basic3index = GenerateChannelKey()
					basic4index = GenerateChannelKey()
					basic4      = GenerateChannelKey()
					basic5      = GenerateChannelKey()
					basic6      = GenerateChannelKey()
					basic7      = GenerateChannelKey()
				)
				Describe("Error paths", func() {
					It("Should return an error for deleting a non-existent channel", func() {
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{9999}, telem.TimeRangeMax)).To(MatchError(core.ErrChannelNotFound))
					})
				})
				Describe("Simple Rate-based channel", func() {
					It("Should delete chunks of a channel", func() {
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic1, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic1},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basic1},
							[]telem.Series{
								telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18),
							}),
						)
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						// Data before deletion: 10, 11, 12, 13, 14, 15, 16, 17, 18

						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic1}, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   15 * telem.SecondTS,
						})).To(Succeed())

						// Data after deletion: 10, 11, __, __, __, 15, 16, 17, 18
						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, basic1)
						Expect(err).To(BeNil())
						Expect(frame.Series).To(HaveLen(2))

						Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ContainElement(10))
						Expect(series0Data).To(ContainElement(11))
						Expect(series0Data).ToNot(ContainElement(12))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(15 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).ToNot(ContainElement(14))
						Expect(series1Data).To(ContainElement(15))
						Expect(series1Data).To(ContainElement(16))
						Expect(series1Data).To(ContainElement(17))
						Expect(series1Data).To(ContainElement(18))
					})
				})

				Describe("Simple Index-based channel", func() {
					It("Should delete chunks of a channel", func() {
						By("Creating an indexed channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic2index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic2, Index: basic2index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic2, basic2index},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basic2, basic2index},
							[]telem.Series{
								telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19),
							}),
						)
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						// Before deletion:
						// 10 11 12 13 14 15 16 17 18 19
						//  0  1  2  3  4  5  6  7  8  9

						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic2}, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   17 * telem.SecondTS,
						})).To(Succeed())

						// After deletion:
						// 10 11 12 13 14 15 16 17 18 19
						//  0  1                 7  8  9

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS}, basic2)
						Expect(err).To(BeNil())
						Expect(frame.Series).To(HaveLen(2))
						Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))

						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ContainElement(0))
						Expect(series0Data).To(ContainElement(1))
						Expect(series0Data).ToNot(ContainElement(2))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)

						Expect(series1Data).ToNot(ContainElement(6))
						Expect(series1Data).To(ContainElement(7))
						Expect(series1Data).To(ContainElement(8))
						Expect(series1Data).To(ContainElement(9))
					})
				})

				Describe("Deleting simple index channel", func() {
					It("Should Delete chunks off the index channel", func() {
						By("Creating an indexed channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic3index, IsIndex: true, DataType: telem.TimeStampT},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic3index},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basic3index},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19),
							}),
						)
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						// Before deletion:
						// 10 11 12 13 14 15 16 17 18 19
						//  0  1  2  3  4  5  6  7  8  9

						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic3index}, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   17 * telem.SecondTS,
						})).To(Succeed())

						// After deletion:
						// 10 11                17 18 19

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 20 * telem.SecondTS}, basic3index)
						Expect(err).To(BeNil())
						Expect(frame.Series).To(HaveLen(2))

						series0Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[0].Data, telem.TimeStampT)
						Expect(series0Data).To(ContainElement(10 * telem.SecondTS))
						Expect(series0Data).To(ContainElement(11 * telem.SecondTS))
						Expect(series0Data).ToNot(ContainElement(12 * telem.SecondTS))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[telem.TimeStamp](frame.Series[1].Data, telem.TimeStampT)

						Expect(series1Data).ToNot(ContainElement(16 * telem.SecondTS))
						Expect(series1Data).To(ContainElement(17 * telem.SecondTS))
						Expect(series1Data).To(ContainElement(18 * telem.SecondTS))
						Expect(series1Data).To(ContainElement(19 * telem.SecondTS))
					})
				})

				Describe("Deleting Index Channel when other channels depend on it", func() {
					It("Should not allow such deletion when another channel is indexed by it on the same time range", func() {
						By("Creating an indexed channel and a channel indexed by it")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic4index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic4, Index: basic4index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic4, basic4index},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basic4, basic4index},
							[]telem.Series{
								telem.NewSeriesV[int64](100, 101, 102),
								telem.NewSecondsTSV(10, 11, 12),
							}),
						)
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						// Before deletion:
						// 10 11 12 13 14 15 16 17 18 19
						//  0  1  2  3  4  5  6  7  8  9

						By("Deleting channel data")
						err := db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic4index}, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   17 * telem.SecondTS,
						})

						Expect(err).To(MatchError(ContainSubstring("depending")))
					})
				})
				Describe("Deleting Time-based channel across multiple pointers", func() {
					It("Should complete such deletions with the appropriate pointers and tombstones", func() {
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic5, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						)).To(Succeed())

						By("Writing data to the channel")
						for i := 1; i <= 9; i++ {
							var data []int64
							for j := 0; j <= 9; j++ {
								data = append(data, int64(i*10+j))
							}
							Expect(db.WriteArray(ctx, basic5, telem.TimeStamp(10*i)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
						}

						// should have been written to 10 - 99
						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic5}, telem.TimeRange{
							Start: 33 * telem.SecondTS,
							End:   75 * telem.SecondTS,
						})).To(Succeed())

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS}, basic5)
						Expect(err).To(BeNil())
						Expect(frame.Series).To(HaveLen(6))

						Expect(frame.Series[2].TimeRange.End).To(Equal(33 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
						Expect(series0Data).To(ContainElement(31))
						Expect(series0Data).To(ContainElement(32))
						Expect(series0Data).ToNot(ContainElement(33))

						Expect(frame.Series[3].TimeRange.Start).To(Equal(75 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[3].Data, telem.Int64T)
						Expect(series1Data).ToNot(ContainElement(74))
						Expect(series1Data).To(ContainElement(75))

						Expect(frame.Series[5].TimeRange.End).To(BeNumerically("<", 100*telem.SecondTS))
					})

					It("Should work for deleting whole pointers", func() {
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic6, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						)).To(Succeed())

						By("Writing data to the channel")
						for i := 1; i <= 9; i++ {
							var data []int64
							for j := 0; j <= 9; j++ {
								data = append(data, int64(i*10+j))
							}
							Expect(db.WriteArray(ctx, basic6, telem.TimeStamp(10*i)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
						}

						// should have been written to 10 - 99
						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic6}, telem.TimeRange{
							Start: 20 * telem.SecondTS,
							End:   50 * telem.SecondTS,
						})).To(Succeed())

						frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS}, basic6)
						Expect(err).To(BeNil())
						Expect(frame.Series).To(HaveLen(6))

						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).ToNot(ContainElement(20))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(50 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).ToNot(ContainElement(49))
						Expect(series1Data).To(ContainElement(50))

						Expect(frame.Series[5].TimeRange.End).To(BeNumerically("<", 100*telem.SecondTS))
					})

					It("Should delete everything", func() {
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic7, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						)).To(Succeed())

						By("Writing data to the channel")
						for i := 1; i <= 9; i++ {
							var data []int64
							for j := 0; j <= 9; j++ {
								data = append(data, int64(i*10+j))
							}
							Expect(db.WriteArray(ctx, basic7, telem.TimeStamp(10*i)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
						}

						// should have been written to 10 - 99
						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic7}, telem.TimeRangeMax)).To(Succeed())

						frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic7))
						Expect(frame.Series).To(HaveLen(0))
					})
				})
			})
			Describe("Delete chunks in multiple channels", func() {
				var (
					rate1  = GenerateChannelKey()
					rate2  = GenerateChannelKey()
					index1 = GenerateChannelKey()
					basic1 = GenerateChannelKey()
					basic2 = GenerateChannelKey()
				)
				Describe("Happy paths", func() {
					Specify("Multiple rate channels", func() {
						By("Creating channels")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: rate1, DataType: telem.Int64T, Rate: 1 * telem.Hz},
							cesium.Channel{Key: rate2, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:         []cesium.ChannelKey{rate1},
							Start:            10 * telem.SecondTS,
							EnableAutoCommit: config.True(),
						}))
						w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:         []cesium.ChannelKey{rate2},
							Start:            11 * telem.SecondTS,
							EnableAutoCommit: config.True(),
						}))

						By("Writing data to the channel")
						Expect(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{rate1},
							[]telem.Series{
								telem.NewSeriesV[int64](10, 11, 12, 13, 14),
							}),
						)).To(BeTrue())
						Expect(w1.Write(cesium.NewFrame(
							[]cesium.ChannelKey{rate2},
							[]telem.Series{
								telem.NewSeriesV[int64](11, 12, 13, 14, 15),
							},
						))).To(BeTrue())
						Expect(w.Close()).To(Succeed())
						Expect(w1.Close()).To(Succeed())

						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{rate1, rate2}, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   13 * telem.SecondTS,
						})).To(Succeed())

						frame := MustSucceed(db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, rate1, rate2))
						Expect(frame.Get(rate1)).To(HaveLen(2))
						Expect(frame.Get(rate1)[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						Expect(frame.Get(rate1)[0].Data).To(Equal(telem.NewSeriesV[int64](10, 11).Data))
						Expect(frame.Get(rate1)[1].TimeRange.Start).To(Equal(13 * telem.SecondTS))
						Expect(frame.Get(rate1)[1].Data).To(Equal(telem.NewSeriesV[int64](13, 14).Data))

						Expect(frame.Get(rate2)).To(HaveLen(2))
						Expect(frame.Get(rate2)[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						Expect(frame.Get(rate2)[0].Data).To(Equal(telem.NewSeriesV[int64](11).Data))
						Expect(frame.Get(rate2)[1].TimeRange.Start).To(Equal(13 * telem.SecondTS))
						Expect(frame.Get(rate2)[1].Data).To(Equal(telem.NewSeriesV[int64](13, 14, 15).Data))

						By("Deleting data again")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{rate1, rate2}, telem.TimeRange{
							Start: 11 * telem.SecondTS,
							End:   20 * telem.SecondTS,
						})).To(Succeed())

						frame = MustSucceed(db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, rate1, rate2))
						Expect(frame.Get(rate1)).To(HaveLen(1))
						Expect(frame.Get(rate1)[0].TimeRange.End).To(Equal(11 * telem.SecondTS))
						Expect(frame.Get(rate1)[0].Data).To(Equal(telem.NewSeriesV[int64](10).Data))

						Expect(frame.Get(rate2)).To(HaveLen(0))
					})
					Specify("Indexed channels", func() {
						By("Creating channels")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: index1, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Key: basic1, DataType: telem.Int64T, Index: index1},
							cesium.Channel{Key: basic2, DataType: telem.Int64T, Index: index1},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:         []cesium.ChannelKey{index1, basic1, basic2},
							Start:            10 * telem.SecondTS,
							EnableAutoCommit: config.True(),
						}))

						By("Writing data to the channel")
						Expect(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index1, basic1, basic2},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104),
							}),
						)).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						By("Deleting channel data")
						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{index1, basic1, basic2}, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   14 * telem.SecondTS,
						})).To(Succeed())

						frame := MustSucceed(db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, basic1, basic2))
						Expect(frame.Get(basic1)).To(HaveLen(2))
						Expect(frame.Get(basic1)[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						Expect(frame.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101).Data))
						Expect(frame.Get(basic1)[1].TimeRange.Start).To(Equal(14 * telem.SecondTS))
						Expect(frame.Get(basic1)[1].Data).To(Equal(telem.NewSeriesV[int64](104).Data))

						Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{index1, basic1, basic2}, telem.TimeRange{
							Start: 11 * telem.SecondTS,
							End:   20 * telem.SecondTS,
						})).To(Succeed())

						frame = MustSucceed(db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, basic1, basic2))
						Expect(frame.Get(basic1)).To(HaveLen(1))
						Expect(frame.Get(basic1)[0].TimeRange.End).To(Equal(11 * telem.SecondTS))
						Expect(frame.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100).Data))

						By("Asserting that writes are still successful")
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:         []cesium.ChannelKey{index1, basic1, basic2},
							Start:            11 * telem.SecondTS,
							EnableAutoCommit: config.True(),
						}))
						Expect(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index1, basic1, basic2},
							[]telem.Series{
								telem.NewSecondsTSV(11, 12, 13, 14),
								telem.NewSeriesV[int64](101, 102, 103, 104),
								telem.NewSeriesV[int64](101, 102, 103, 104),
							}),
						)).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						Expect(w.Close()).To(Succeed())
						frame = MustSucceed(db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, basic1, basic2))
						Expect(frame.Get(basic1)).To(HaveLen(2))
						Expect(frame.Get(basic1)[0].TimeRange.End).To(Equal(11 * telem.SecondTS))
						Expect(frame.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100).Data))
						Expect(frame.Get(basic1)[1].TimeRange.Start).To(Equal(11 * telem.SecondTS))
						Expect(frame.Get(basic1)[1].Data).To(Equal(telem.NewSeriesV[int64](101, 102, 103, 104).Data))
					})
				})
			})
			Describe("Error paths", func() {
				var (
					data     cesium.ChannelKey
					index    cesium.ChannelKey
					channels []cesium.Channel
				)
				BeforeEach(func() {
					data = GenerateChannelKey()
					index = GenerateChannelKey()
					channels = []cesium.Channel{
						{Key: index, IsIndex: true, DataType: telem.TimeStampT},
						{Key: data, Index: index, DataType: telem.Int64T},
					}

					Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
				})
				It("Should return ChannelNotFound when a channel does not exist", func() {
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{math.MaxUint32 - 10}, telem.TimeRangeMax)).To(MatchError(cesium.ErrChannelNotFound))
				})
				It("Should not delete any data when one channel does not exist", func() {
					Expect(db.Write(ctx, 0, cesium.NewFrame(
						[]core.ChannelKey{data, index},
						[]telem.Series{telem.NewSeriesV[int64](0, 1, 2, 3), telem.NewSecondsTSV(0, 1, 2, 3)},
					))).To(Succeed())
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data, index, math.MaxUint32 - 10}, (1 * telem.SecondTS).Range(2*telem.SecondTS))).To(MatchError(cesium.ErrChannelNotFound))
					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data, index))
					Expect(f.Get(data)).To(HaveLen(1))
				})
				It("Should return an error when trying to delete timerange from virtual channel", func() {
					virtualKey := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: virtualKey, Virtual: true, DataType: telem.Int64T})).To(Succeed())
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{virtualKey}, telem.TimeRangeMax)).To(MatchError(ContainSubstring("cannot delete time range from virtual channel")))
				})
				It("Should not allow deletion of any channel while there is a writer that could write over it", func() {
					Expect(db.WriteArray(ctx, index, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13))).To(Succeed())
					Expect(db.WriteArray(ctx, data, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13))).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{data}, Start: 9 * telem.SecondTS}))

					err := db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, (8 * telem.SecondTS).Range(11*telem.SecondTS))
					Expect(err).To(MatchError(ContainSubstring("overlaps with a controlled region")))

					By("Closing the writer and asserting we can now delete")
					Expect(w.Close()).To(Succeed())
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, (8 * telem.SecondTS).Range(11*telem.SecondTS))).To(Succeed())
					f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))

					Expect(f.Series[0].TimeRange).To(Equal((11 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
					Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](11, 12, 13).Data))
				})
				It("Should delete normally if there is an open iterator on the channel", func() {
					Expect(db.WriteArray(ctx, index, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13))).To(Succeed())
					Expect(db.WriteArray(ctx, data, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13))).To(Succeed())
					i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{data}}))

					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(10 * telem.Second)).To(BeTrue())
					Expect(i.Value().Series[0].Data).To(Equal(telem.NewSeriesV[int64](10, 11, 12, 13).Data))

					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, (8 * telem.SecondTS).Range(11*telem.SecondTS))).To(Succeed())
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(10 * telem.Second)).To(BeTrue())
					Expect(i.Value().Series[0].Data).To(Equal(telem.NewSeriesV[int64](11, 12, 13).Data))

					Expect(i.Close()).To(Succeed())
				})
				It("Should not allow deletion of an index channel when there is a data channel depending on an index channel and has data for it", func() {
					Expect(db.WriteArray(ctx, index, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13))).To(Succeed())
					Expect(db.WriteArray(ctx, data, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13))).To(Succeed())

					err := db.DeleteTimeRange(ctx, []cesium.ChannelKey{index}, (8 * telem.SecondTS).Range(11*telem.SecondTS))
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("cannot delete index channel %v with channel %v depending on it on the time range %v", channels[0], channels[1], (8 * telem.SecondTS).Range(11*telem.SecondTS)))))
				})
				It("Should not allow deletion of an index channel when there is a data channel with a writer open before the timerange", func() {
					Expect(db.WriteArray(ctx, index, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13))).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{data}, Start: 9 * telem.SecondTS}))

					err := db.DeleteTimeRange(ctx, []cesium.ChannelKey{index}, (8 * telem.SecondTS).Range(10*telem.SecondTS))
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("cannot delete index channel %v with channel %v depending on it on the time range %v", channels[0], channels[1], (8 * telem.SecondTS).Range(10*telem.SecondTS)))))

					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{index}, (8 * telem.SecondTS).Range(9*telem.SecondTS))).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
				It("Should not allow deletion when there are multiple channels and one of them fails", func() {
					data2 := GenerateChannelKey()
					data3 := GenerateChannelKey()
					newChannels := []cesium.Channel{
						{Key: data2, Index: index, DataType: telem.Int64T},
						{Key: data3, Index: index, DataType: telem.Int64T},
					}

					Expect(db.CreateChannel(ctx, newChannels...)).To(Succeed())

					Expect(db.Write(ctx, 10*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{index, data, data2, data3},
						[]telem.Series{
							telem.NewSecondsTSV(10, 11, 12, 13, 14),
							telem.NewSeriesV[int64](100, 101, 102, 103, 104),
							telem.NewSeriesV[int64](100, 101, 102, 103, 104),
							telem.NewSeriesV[int64](100, 101, 102, 103, 104),
						}),
					)).To(Succeed())

					err := db.DeleteTimeRange(ctx, []cesium.ChannelKey{data, data2, index}, (11 * telem.SecondTS).Range(14*telem.SecondTS))
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("cannot delete index channel %v with channel %v depending on it", channels[0], newChannels[1]))))
				})
			})
		})
	}
})
