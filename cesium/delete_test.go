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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/confluence"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"strconv"
)

var _ = Describe("Delete Channel", Ordered, func() {
	var (
		db       *cesium.DB
		uChannel cesium.ChannelKey = 10
		vChannel cesium.ChannelKey = 11
		udb                        = cesium.Channel{Key: uChannel, IsIndex: false, Rate: 1 * telem.Hz, DataType: telem.Int64T}
		vdb                        = cesium.Channel{Key: vChannel, Virtual: true, IsIndex: false, DataType: telem.Int64T}
	)
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Error paths", func() {
		Specify("Deleting a nonexistent channel should be idempotent", func() {
			Expect(db.DeleteChannel(9)).To(Succeed())
		})
		Describe("Deleting a channel that is being written to should error", func() {
			Specify("Virtual Channel", func() {
				Expect(db.CreateChannel(ctx, vdb)).To(Succeed())
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{vChannel}, Start: 10 * telem.SecondTS}))
				Expect(db.DeleteChannel(vChannel)).To(MatchError(ContainSubstring("currently 1 unclosed writers")))
				Expect(w.Close()).To(Succeed())
				Expect(db.DeleteChannel(vChannel)).To(Succeed())
			})

			Specify("Unary Channel", func() {
				Expect(db.CreateChannel(ctx, udb)).To(Succeed())
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{uChannel}, Start: 10 * telem.SecondTS}))
				Expect(db.DeleteChannel(uChannel)).To(MatchError(ContainSubstring("currently 1 unclosed writers/iterators")))
				Expect(w.Close()).To(Succeed())
				Expect(db.DeleteChannel(uChannel)).To(Succeed())
			})
		})
		Describe("Deleting a channel that is being read from should error", func() {
			Specify("Unary Channel", func() {
				Expect(db.CreateChannel(ctx, udb)).To(Succeed())
				i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannel}}))
				Expect(db.DeleteChannel(uChannel)).To(MatchError(ContainSubstring("currently 1 unclosed writers/iterators")))
				Expect(i.Close()).To(Succeed())
				Expect(db.DeleteChannel(uChannel)).To(Succeed())
			})

			Specify("Unary Channel double reader", func() {
				Expect(db.CreateChannel(ctx, udb)).To(Succeed())
				i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannel}}))
				i2 := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannel}}))
				Expect(db.DeleteChannel(uChannel)).To(MatchError(ContainSubstring("currently 2 unclosed writers/iterators")))
				Expect(i.Close()).To(Succeed())
				Expect(i2.Close()).To(Succeed())
				Expect(db.DeleteChannel(uChannel)).To(Succeed())
			})
		})
		Describe("Deleting a channel that is being streamed from should error", func() {
			Specify("Virtual Channel", func() {
				By("Creating a channel")
				Expect(db.CreateChannel(ctx, vdb)).To(Succeed())
				By("Opening a streamer on the channel")
				s := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{Channels: []cesium.ChannelKey{vChannel}}))
				sCtx, cancel := signal.WithCancel(ctx)

				By("Start streaming")
				i, _ := confluence.Attach(s, 1)
				s.Flow(sCtx, confluence.CloseInletsOnExit())

				By("Expecting delete channel to fail because there is an open streamer")
				err := db.DeleteChannel(vChannel)
				Expect(err).ToNot(HaveOccurred())

				By("All other operations should still happen without error")
				cancel()
				i.Close()
			})

			Specify("Unary Channel", func() {
				By("Creating a channel")
				Expect(db.CreateChannel(ctx, udb)).To(Succeed())
				By("Opening a streamer on the channel")
				s := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{Channels: []cesium.ChannelKey{uChannel}}))
				sCtx, cancel := signal.WithCancel(ctx)

				By("Start streaming")
				i, _ := confluence.Attach(s, 1)
				s.Flow(sCtx, confluence.CloseInletsOnExit())

				By("Expecting delete channel to fail because there is an open streamer")
				err := db.DeleteChannel(uChannel)
				Expect(err).ToNot(HaveOccurred())

				By("All other operations should still happen without error")
				cancel()
				i.Close()
			})

			Describe("StreamIterator", func() {
				Specify("Unary", func() {
					Expect(db.CreateChannel(ctx, udb)).To(Succeed())
					it := MustSucceed(db.NewStreamIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{uChannel}}))

					err := db.DeleteChannel(uChannel)
					Expect(err).To(MatchError(ContainSubstring("1 unclosed writers/iterators")))

					sCtx, cancel := signal.Isolated()
					i, _ := confluence.Attach(it, 1)
					it.Flow(sCtx)

					i.Close()
					Expect(sCtx.Wait()).To(Succeed())
					cancel()

					Expect(db.DeleteChannel(uChannel)).To(Succeed())
				})
			})

			Describe("StreamWriter", func() {
				Specify("Virtual", func() {
					Expect(db.CreateChannel(ctx, vdb)).To(Succeed())
					it := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []cesium.ChannelKey{vChannel}}))

					err := db.DeleteChannel(vChannel)
					Expect(err).To(MatchError(ContainSubstring("1 unclosed writers")))

					sCtx, cancel := signal.Isolated()
					i, _ := confluence.Attach(it, 1)
					it.Flow(sCtx)

					i.Close()
					Expect(sCtx.Wait()).To(Succeed())
					cancel()

					Expect(db.DeleteChannel(vChannel)).To(Succeed())
				})

				Specify("Unary", func() {
					Expect(db.CreateChannel(ctx, udb)).To(Succeed())
					it1 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []cesium.ChannelKey{uChannel}}))
					it2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []cesium.ChannelKey{uChannel}}))

					err := db.DeleteChannel(uChannel)
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

					Expect(db.DeleteChannel(uChannel)).To(Succeed())
				})
			})
			Specify("Deleting an index channel that other channels rely on should error", func() {
				Expect(db.CreateChannel(
					ctx,
					cesium.Channel{Key: 15, IsIndex: true, DataType: telem.TimeStampT},
					cesium.Channel{Key: 16, Index: 15, DataType: telem.Int64T},
					cesium.Channel{Key: 17, Index: 15, DataType: telem.Int16T},
				)).To(Succeed())

				By("Deleting channel")
				err := db.DeleteChannel(15)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("could not delete index channel with other channels depending on it")))

				By("Deleting channels that depend on it")
				Expect(db.DeleteChannel(16)).To(Succeed())
				Expect(db.DeleteChannel(15)).ToNot(Succeed())
				Expect(db.DeleteChannel(17)).To(Succeed())

				By("Deleting the index channel again")
				Expect(db.DeleteChannel(15)).To(Succeed())
				_, err = db.RetrieveChannel(ctx, 15)
				Expect(err).To(MatchError(core.ChannelNotFound))
			})
			Specify("Deleting control digest channel should error", func() {
				var controlKey cesium.ChannelKey = 7
				Expect(db.ConfigureControlUpdateChannel(ctx, controlKey)).To(Succeed())
				Expect(db.DeleteChannel(controlKey)).To(MatchError(ContainSubstring("1 unclosed writers")))
			})
		})
	})
	Describe("Deleting Index Channel when other channels depend on it", func() {
		It("Should not allow such deletion when another channel is indexed by it on the same time range", func() {
			By("Creating an indexed channel and a channel indexed by it")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 21, IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: 22, Index: 21, DataType: telem.Int64T},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{22, 21},
				Start:    10 * telem.SecondTS,
			}))

			By("Writing data to the channel")
			ok := w.Write(cesium.NewFrame(
				[]cesium.ChannelKey{22, 21},
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
			err := db.DeleteTimeRange(ctx, 21, telem.TimeRange{
				Start: 12 * telem.SecondTS,
				End:   17 * telem.SecondTS,
			})

			Expect(err).To(MatchError(ContainSubstring("depending")))
		})
	})
	Describe("Happy paths", func() {
		var (
			fsDB *cesium.DB
			fs                   = MustSucceed(xfs.Default.Sub("./testdata"))
			key  core.ChannelKey = 101
		)

		BeforeAll(func() {
			fsDB = MustSucceed(cesium.Open("", cesium.WithFS(fs)))
		})
		AfterAll(func() {
			Expect(xfs.Default.Remove("./testdata")).To(Succeed())
			Expect(fsDB.Close()).To(Succeed())
		})
		It("Should delete an index unary channel", func() {
			Expect(fsDB.CreateChannel(
				ctx,
				cesium.Channel{Key: key, IsIndex: true, DataType: telem.TimeStampT},
			)).To(Succeed())
			Expect(fsDB.WriteArray(ctx, key, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13))).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(key)))).To(BeTrue())
			Expect(fsDB.DeleteChannel(key)).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(key)))).To(BeFalse())
			_, err := fsDB.RetrieveChannel(ctx, key)
			Expect(err).To(MatchError(core.ChannelNotFound))
		})
		It("Should delete a unary channel", func() {
			Expect(fsDB.CreateChannel(
				ctx,
				cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T},
			)).To(Succeed())
			Expect(fsDB.WriteArray(ctx, key, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13))).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(key)))).To(BeTrue())
			Expect(fsDB.DeleteChannel(key)).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(key)))).To(BeFalse())
			_, err := fsDB.RetrieveChannel(ctx, key)
			Expect(err).To(MatchError(core.ChannelNotFound))
		})
		It("Should delete a virtual channel", func() {
			Expect(fsDB.CreateChannel(ctx, cesium.Channel{Key: key, Virtual: true, DataType: telem.Int64T})).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(key)))).To(BeTrue())
			Expect(fsDB.DeleteChannel(key)).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(key)))).To(BeFalse())
			_, err := fsDB.RetrieveChannel(ctx, key)
			Expect(err).To(MatchError(core.ChannelNotFound))
		})
	})
})

var _ = Describe("Delete Channels", Ordered, func() {
	var (
		db       *cesium.DB
		fs                         = MustSucceed(xfs.Default.Sub("./testdata2"))
		index1   cesium.ChannelKey = 1
		data1    cesium.ChannelKey = 2
		index2   cesium.ChannelKey = 3
		data2    cesium.ChannelKey = 4
		data3    cesium.ChannelKey = 5
		rate     cesium.ChannelKey = 6
		index3   cesium.ChannelKey = 7
		channels                   = []cesium.Channel{{Key: index1, IsIndex: true, DataType: telem.TimeStampT, Index: index1},
			{Key: data1, DataType: telem.Int64T, Index: index1},
			{Key: index2, IsIndex: true, DataType: telem.TimeStampT, Index: index2},
			{Key: data2, DataType: telem.Int64T, Index: index2},
			{Key: data3, DataType: telem.Int64T, Index: index2},
			{Key: rate, DataType: telem.Int64T, Rate: 2 * telem.Hz},
			{Key: index3, IsIndex: true, Index: index3, DataType: telem.TimeStampT},
		}
	)
	BeforeAll(func() {
		db = MustSucceed(cesium.Open("", cesium.WithFS(fs)))
	})

	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	BeforeEach(func() { Expect(db.CreateChannel(ctx, channels...)).To(Succeed()) })
	AfterEach(func() {
		Expect(db.DeleteChannels(lo.Map(channels, func(c cesium.Channel, _ int) core.ChannelKey { return c.Key })...)).To(Succeed())
		Expect(xfs.Default.Remove("./testdata2"))
	})
	Describe("Happy paths", func() {
		It("Should be idempotent", func() {
			Expect(db.DeleteChannels(index1, data1)).To(Succeed())
			Expect(db.DeleteChannels(index1, data1)).To(Succeed())
			Expect(db.DeleteChannels(index1, data1)).To(Succeed())
		})
		DescribeTable("Deleting permutations of channels", func(chs ...core.ChannelKey) {
			Expect(db.DeleteChannels(chs...)).To(Succeed())
			for _, c := range chs {
				_, err := db.RetrieveChannel(ctx, c)
				Expect(err).To(MatchError(core.ChannelNotFound))
				Expect(fs.Exists(strconv.Itoa(int(c)))).To(BeFalse())
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
	Describe("Interrupted deletion", func() {
		It("Should delete all channels before encountering an error", func() {
			w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []core.ChannelKey{data2, data3}}))
			w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Start: 10 * telem.SecondTS, Channels: []core.ChannelKey{data2}}))
			Expect(db.DeleteChannels(rate, data1, data2)).To(MatchError(ContainSubstring("2 unclosed writer")))
			Expect(w1.Close()).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(rate)))).To(BeFalse())
			Expect(fs.Exists(strconv.Itoa(int(data1)))).To(BeFalse())
			Expect(fs.Exists(strconv.Itoa(int(data2)))).To(BeTrue())
			_, err := db.RetrieveChannel(ctx, rate)
			Expect(err).To(MatchError(core.ChannelNotFound))
			_, err = db.RetrieveChannel(ctx, data1)
			Expect(err).To(MatchError(core.ChannelNotFound))
			_, err = db.RetrieveChannel(ctx, data2)
			Expect(err).To(BeNil())
			Expect(db.DeleteChannels(rate, data1, data3)).To(Succeed())
			Expect(w2.Close()).To(Succeed())
		})
		It("Should delete all channels, but not index channels, before encountering an error", func() {
			i1 := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{rate, data3}}))
			i2 := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{data3}}))
			Expect(db.DeleteChannels(index1, index2, data1, data2, data3)).To(MatchError(ContainSubstring("2 unclosed writer")))
			Expect(i1.Close()).To(Succeed())
			Expect(i2.Close()).To(Succeed())
			Expect(fs.Exists(strconv.Itoa(int(data2)))).To(BeFalse())
			_, err := db.RetrieveChannel(ctx, data2)
			Expect(err).To(MatchError(core.ChannelNotFound))
			Expect(fs.Exists(strconv.Itoa(int(index1)))).To(BeTrue())
			Expect(fs.Exists(strconv.Itoa(int(index2)))).To(BeTrue())
			_, err = db.RetrieveChannel(ctx, index1)
			Expect(err).To(BeNil())
			_, err = db.RetrieveChannel(ctx, index2)
			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("Delete chunks", Ordered, func() {
	var db *cesium.DB
	var (
		basic1      cesium.ChannelKey = 1
		basic2      cesium.ChannelKey = 2
		basic2index cesium.ChannelKey = 3
		basic3index cesium.ChannelKey = 5
		basic4index cesium.ChannelKey = 4
		basic4      cesium.ChannelKey = 6
		basic5      cesium.ChannelKey = 7
		basic6      cesium.ChannelKey = 8
	)
	BeforeAll(func() {
		db = openMemDB()
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Error paths", func() {
		It("Should be idempotent for deletion of non-existent channel", func() {
			Expect(db.DeleteTimeRange(ctx, 1, telem.TimeRangeMax)).To(MatchError(core.ChannelNotFound))
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
			Expect(db.DeleteTimeRange(ctx, basic1, telem.TimeRange{
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
			Expect(db.DeleteTimeRange(ctx, basic2, telem.TimeRange{
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
			Expect(db.DeleteTimeRange(ctx, basic3index, telem.TimeRange{
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
			err := db.DeleteTimeRange(ctx, basic4index, telem.TimeRange{
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
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{basic5},
					Start:    telem.TimeStamp(10*i) * telem.SecondTS,
				}))
				ok := w.Write(cesium.NewFrame(
					[]cesium.ChannelKey{basic5},
					[]telem.Series{
						telem.NewSeriesV[int64](data...),
					}),
				)
				Expect(ok).To(BeTrue())
				_, ok = w.Commit()
				Expect(ok).To(BeTrue())
				Expect(w.Close()).To(Succeed())
			}

			// should have been written to 10 - 99
			By("Deleting channel data")
			Expect(db.DeleteTimeRange(ctx, basic5, telem.TimeRange{
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
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{basic6},
					Start:    telem.TimeStamp(10*i) * telem.SecondTS,
				}))
				ok := w.Write(cesium.NewFrame(
					[]cesium.ChannelKey{basic6},
					[]telem.Series{
						telem.NewSeriesV[int64](data...),
					}),
				)
				Expect(ok).To(BeTrue())
				_, ok = w.Commit()
				Expect(ok).To(BeTrue())
				Expect(w.Close()).To(Succeed())
			}

			// should have been written to 10 - 99
			By("Deleting channel data")
			Expect(db.DeleteTimeRange(ctx, basic6, telem.TimeRange{
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
	})
})
