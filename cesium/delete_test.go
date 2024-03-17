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
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"runtime"
)

var _ = Describe("Delete", Ordered, func() {
	var db *cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Delete a channel when there are no iterators/writers", func() {
		Specify("Deleting a nonexistent channel", func() {
			Expect(db.DeleteChannel(9)).To(MatchError(query.Error))
		})
		Specify("Deleting an index channel that other channels rely on", func() {
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 15, IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: 16, Index: 15, DataType: telem.Int64T},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{15, 16},
				Start:    10 * telem.SecondTS,
			}))

			By("Writing data to the channel")
			ok := w.Write(cesium.NewFrame(
				[]cesium.ChannelKey{16, 15},
				[]telem.Series{
					telem.NewSeriesV[int64](100, 101, 102),
					telem.NewSecondsTSV(10, 11, 12),
				}),
			)
			Expect(ok).To(BeTrue())
			_, ok = w.Commit()
			Expect(ok).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Expect(w.Close()).To(Succeed())

			By("Deleting channel")
			Expect(db.DeleteChannel(15)).ToNot(Succeed())
		})
		Specify("Simple unary channel", func() {
			By("Creating a channel")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 1, Rate: 10 * telem.Hz, DataType: telem.Float64T},
			)).To(Succeed())
			By("Creating a writer on it")
			w := MustSucceed(db.OpenWriter(
				ctx,
				cesium.WriterConfig{
					Channels: []cesium.ChannelKey{1},
					Start:    10 * telem.SecondTS,
				}))
			By("Trying to delete it")
			Expect(db.DeleteChannel(1).Error()).To(ContainSubstring("1 unclosed"))
			By("Closing the writer")
			Expect(w.Close()).To(Succeed())
			By("Trying to delete it again")
			Expect(db.DeleteChannel(1)).To(Succeed())
			By("Trying to retrieve the channel")
			_, err := db.RetrieveChannel(ctx, 1)
			Expect(err).To(MatchError(query.Error))
		})

		Specify("Simple virtual channel", func() {
			By("Creating a channel")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 1, DataType: telem.Float64T, Virtual: true},
			)).To(Succeed())
			By("Creating a writer on it")
			w := MustSucceed(db.OpenWriter(
				ctx,
				cesium.WriterConfig{
					Channels: []cesium.ChannelKey{1},
					Start:    10 * telem.SecondTS,
				}))
			By("Trying to delete it")
			Expect(db.DeleteChannel(1).Error()).To(ContainSubstring("1 unclosed"))
			By("Closing the writer")
			Expect(w.Close()).To(Succeed())
			By("Trying to delete it again")
			Expect(db.DeleteChannel(1)).To(Succeed())
			By("Trying to retrieve the channel")
			_, err := db.RetrieveChannel(ctx, 1)
			Expect(err).To(MatchError(query.Error))
		})

		Specify("Indexed unary channels", func() {
			By("Creating two channels")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: 2, IsIndex: true, DataType: telem.TimeStampT},
				cesium.Channel{Key: 3, Index: 2, DataType: telem.Int64T},
			)).To(Succeed())
			By("Creating writers on them")
			w := MustSucceed(db.OpenWriter(
				ctx,
				cesium.WriterConfig{
					Channels: []cesium.ChannelKey{2, 3},
					Start:    10 * telem.SecondTS,
				}))

			By("Trying to delete them")
			Expect(db.DeleteChannel(2).Error()).To(ContainSubstring("depending"))
			Expect(db.DeleteChannel(3).Error()).To(ContainSubstring("1 unclosed"))

			By("Closing the writer")
			Expect(w.Close()).To(Succeed())
			By("Trying to delete them again")
			Expect(db.DeleteChannel(3)).To(Succeed())
			Expect(db.DeleteChannel(2)).To(Succeed())

			By("Trying to retrieve the channels")
			_, err2 := db.RetrieveChannel(ctx, 2)
			_, err3 := db.RetrieveChannel(ctx, 3)
			Expect(err2).To(MatchError(query.Error))
			Expect(err3).To(MatchError(query.Error))
		})

		Specify("Controlled streamwriters", func() {
			var (
				controlKey cesium.ChannelKey = 7
				basic3     cesium.ChannelKey = 8
			)
			Expect(db.ConfigureControlUpdateChannel(ctx, controlKey)).To(Succeed())
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: basic3, DataType: telem.Int64T, Rate: 1 * telem.Hz},
			)).To(Succeed())
			streamer := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
				Channels: []cesium.ChannelKey{controlKey},
			}))
			i, o := confluence.Attach(streamer, 1)
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			streamer.Flow(sCtx, confluence.CloseInletsOnExit())
			// Do a best effort schedule for the streamer to boot up
			runtime.Gosched()
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels:       []cesium.ChannelKey{basic3},
				ControlSubject: control.Subject{Name: "Writer"},
				Start:          10 * telem.SecondTS,
			}))
			Expect(db.DeleteChannel(basic3).Error()).To(ContainSubstring("1 unclosed"))
			var r cesium.StreamerResponse
			Eventually(o.Outlet()).Should(Receive(&r))
			Expect(r.Frame.Keys).To(HaveLen(1))
			u := MustSucceed(cesium.DecodeControlUpdate(ctx, r.Frame.Series[0]))
			t, ok := lo.Find(u.Transfers, func(t controller.Transfer) bool {
				return t.To.Resource == basic3
			})
			Expect(ok).To(BeTrue())
			Expect(t.To.Subject.Name).To(Equal("Writer"))
			Expect(w.Close()).To(Succeed())
			Eventually(o.Outlet()).Should(Receive(&r))
			Expect(r.Frame.Keys).To(HaveLen(1))
			i.Close()
			Expect(sCtx.Wait()).To(Succeed())

			Expect(db.DeleteChannel(basic3)).To(Succeed())
			Expect(db.DeleteChannel(controlKey).Error()).To(ContainSubstring("cannot delete update digest channel"))
		})
	})
	Describe("Deleting Index Channel when other channels depend on it", func() {
		It("Should not allow such deletion when another channel is indexed by it on the sa me time range", func() {
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

			Expect(err.Error()).To(ContainSubstring("depending"))
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
		It("Should not allow such deletion when another channel is indexed by it on the sa me time range", func() {
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

			Expect(err.Error()).To(ContainSubstring("depending"))
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
