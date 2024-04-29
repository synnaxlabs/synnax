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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer Behavior", func() {
	for fsName, fs := range fileSystems {
		fs := fs()
		Context("FS: "+fsName, Ordered, func() {
			var db *cesium.DB
			BeforeAll(func() { db = openDBOnFS(fs) })
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(fs.Remove(rootPath)).To(Succeed())
			})
			Describe("Happy Path", func() {
				Context("Indexed", func() {
					Specify("Basic Write", func() {
						var (
							basic1      = GenerateCesiumChannelKey()
							basic1Index = GenerateCesiumChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic1Index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic1, Index: basic1Index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic1, basic1Index},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basic1Index, basic1},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
								telem.NewSeriesV[int64](1, 2, 3, 4),
							}),
						)
						Expect(ok).To(BeTrue())
						end, ok := w.Commit()
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						Expect(end).To(Equal(13*telem.SecondTS + 1))

						By("Reading the data back")
						frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1))
						Expect(frame.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
						tsFrame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1Index))
						Expect(tsFrame.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
					})
				})
				Context("Multiple Indexes", func() {
					Specify("Basic Writer", func() {
						var (
							basic1    = GenerateCesiumChannelKey()
							basicIdx1 = GenerateCesiumChannelKey()
							basic2    = GenerateCesiumChannelKey()
							basicIdx2 = GenerateCesiumChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basicIdx1, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic1, Index: basicIdx1, DataType: telem.Int64T},
							cesium.Channel{Key: basicIdx2, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic2, Index: basicIdx2, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic1, basicIdx1, basic2, basicIdx2},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the first index")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basicIdx1, basic1},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
								telem.NewSeriesV[int64](1, 2, 3, 4),
							},
						))
						Expect(ok).To(BeTrue())

						By("Writing more data to the second index")
						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basicIdx2, basic2},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14),
								telem.NewSeriesV[int64](1, 2, 3, 4, 5),
							},
						))
						Expect(ok).To(BeTrue())
						end, ok := w.Commit()
						Expect(ok).To(BeTrue())
						Expect(end).To(Equal(14*telem.SecondTS + 1))
						Expect(w.Close()).To(Succeed())
					})

				})

				Describe("Auto-commit", func() {
					Describe("Indexed channels", func() {
						var (
							index1 = GenerateCesiumChannelKey()
							basic1 = GenerateCesiumChannelKey()
							index2 = GenerateCesiumChannelKey()
							basic2 = GenerateCesiumChannelKey()
							basic3 = GenerateCesiumChannelKey()
						)
						BeforeEach(func() {
							By("Creating channels")
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{Key: index1, IsIndex: true, DataType: telem.TimeStampT},
								cesium.Channel{Key: basic1, Index: index1, DataType: telem.Int64T},
								cesium.Channel{Key: index2, IsIndex: true, DataType: telem.TimeStampT},
								cesium.Channel{Key: basic2, Index: index2, DataType: telem.Int64T},
								cesium.Channel{Key: basic3, Index: index2, DataType: telem.Float64T},
							)).To(Succeed())
						})
						It("Should automatically commit the writer for all channels with an end timestamp when it is not specified", func() {
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels: []cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								Start:    10 * telem.SecondTS,
								Mode:     cesium.NewWriterMode(cesium.EnableStream(), cesium.EnablePersist(), cesium.EnableAutoCommit()),
							}))

							By("Writing telemetry")
							ok := w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(10, 12, 13, 14),
									telem.NewSeriesV[int64](100, 102, 103, 104),
									telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
									telem.NewSeriesV[float64](100.00, 101.03, 102.06, 103.00, 104.00, 105.80),
								},
							))
							Expect(ok).To(BeTrue())

							By("Reading the telemetry")
							f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index1, basic1, index2, basic2, basic3))
							Expect(f.Get(index1)).To(HaveLen(1))
							Expect(f.Get(index1)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(14*telem.SecondTS + 1)))
							Expect(f.Get(index1)[0].Len()).To(Equal(int64(4)))
							Expect(f.Get(index1)[0].Data).To(Equal(telem.NewSecondsTSV(10, 12, 13, 14).Data))

							Expect(f.Get(basic1)).To(HaveLen(1))
							Expect(f.Get(basic1)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(14*telem.SecondTS + 1)))
							Expect(f.Get(basic1)[0].Len()).To(Equal(int64(4)))
							Expect(f.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 102, 103, 104).Data))

							Expect(f.Get(index2)).To(HaveLen(1))
							Expect(f.Get(index2)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(f.Get(index2)[0].Len()).To(Equal(int64(6)))
							Expect(f.Get(index2)[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 12, 13, 14, 15).Data))

							Expect(f.Get(basic2)).To(HaveLen(1))
							Expect(f.Get(basic2)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(f.Get(basic2)[0].Len()).To(Equal(int64(6)))
							Expect(f.Get(basic2)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105).Data))

							Expect(f.Get(basic3)).To(HaveLen(1))
							Expect(f.Get(basic3)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(f.Get(basic3)[0].Len()).To(Equal(int64(6)))
							Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[float64](100.00, 101.03, 102.06, 103.00, 104.00, 105.80).Data))

							By("Closing resources")
							Expect(w.Close()).To(Succeed())
						})
					})
				})
			})
			Describe("Stream Only Mode", func() {
				It("Should not persist data", func() {
					var (
						basic1      = GenerateCesiumChannelKey()
						basic1Index = GenerateCesiumChannelKey()
					)
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic1Index, IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: basic1, Index: basic1Index, DataType: telem.Int64T},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic1, basic1Index},
						Start:    10 * telem.SecondTS,
						Mode:     cesium.NewWriterMode(cesium.EnableStream()),
					}))

					By("Writing data to the channel")
					ok := w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{basic1Index, basic1},
						[]telem.Series{
							telem.NewSecondsTSV(10, 11, 12, 13),
							telem.NewSeriesV[int64](1, 2, 3, 4),
						}),
					)
					Expect(ok).To(BeTrue())
					end, ok := w.Commit()
					Expect(ok).To(BeTrue())
					Expect(w.Close()).To(Succeed())
					Expect(end).To(Equal(13*telem.SecondTS + 1))

					By("Reading the data back")
					frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1))
					Expect(frame.Series).To(HaveLen(0))
					tsFrame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1Index))
					Expect(tsFrame.Series).To(HaveLen(0))
				})
			})
			Describe("Open Errors", func() {
				Specify("Channel that does not exist", func() {
					_, err := db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{55000},
							Start:    10 * telem.SecondTS,
						})
					Expect(err).To(MatchError(core.ChannelNotFound))
				})
			})
			Describe("Frame Errors", Ordered, func() {
				var (
					frameErr1 = GenerateCesiumChannelKey()
					frameErr2 = GenerateCesiumChannelKey()
				)
				BeforeAll(func() {
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: frameErr1, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						cesium.Channel{Key: frameErr2, DataType: telem.Int64T, Rate: 1 * telem.Hz},
					))
				})
				Specify("Uneven Frame", func() {
					w := MustSucceed(
						db.OpenWriter(ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{frameErr1, frameErr2},
								Start:    10 * telem.SecondTS,
							}),
					)
					ok := w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{frameErr1, frameErr2},
						[]telem.Series{
							telem.NewSeriesV[int64](1, 2, 3, 4),
							telem.NewSeriesV[int64](1, 2, 3),
						}),
					)
					Expect(ok).To(BeTrue())
					_, ok = w.Commit()
					Expect(ok).To(BeFalse())
					err := w.Close()
					Expect(err).To(MatchError(validate.Error))
					Expect(err.Error()).To(ContainSubstring("same length"))
				})
				Specify("Frame Without All Channels", func() {
					w := MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{frameErr1, frameErr2},
							Start:    10 * telem.SecondTS,
						}))
					ok := w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{frameErr1},
						[]telem.Series{
							telem.NewSeriesV[int64](1, 2, 3, 4),
						},
					))
					Expect(ok).To(BeTrue())
					_, ok = w.Commit()
					Expect(ok).To(BeFalse())
					err := w.Close()
					Expect(err).To(MatchError(validate.Error))
					Expect(err.Error()).To(ContainSubstring("one and only one"))
				})
				Specify("Frame with Duplicate Channels", func() {
					w := MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{frameErr1, frameErr2},
							Start:    10 * telem.SecondTS,
						}))
					ok := w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{frameErr1, frameErr1},
						[]telem.Series{
							telem.NewSeriesV[int64](1, 2, 3, 4),
							telem.NewSeriesV[int64](1, 2, 3, 4),
						},
					))
					Expect(ok).To(BeTrue())
					_, ok = w.Commit()
					Expect(ok).To(BeFalse())
					err := w.Close()
					Expect(err).To(MatchError(validate.Error))
					Expect(err.Error()).To(ContainSubstring("duplicate channel"))
				})
			})
			Describe("Index Errors", func() {
				Context("Discontinuous Index", func() {
					var (
						disc1      = GenerateCesiumChannelKey()
						disc1Index = GenerateCesiumChannelKey()
						disc2      = GenerateCesiumChannelKey()
						disc2Index = GenerateCesiumChannelKey()
					)
					Specify("Last sample is not the index", func() {
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: disc1Index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: disc1, Index: disc1Index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(
							ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{disc1Index},
								Start:    10 * telem.SecondTS,
							}))

						By("Writing data to the index correctly")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{disc1Index},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
							}),
						)
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						By("Writing data to channel where the last sample is not the index")
						w = MustSucceed(db.OpenWriter(
							ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{disc1},
								Start:    10 * telem.SecondTS,
							}))
						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{disc1},
							[]telem.Series{
								telem.NewSeriesV[int64](1, 2, 3, 4, 5),
							},
						))
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeFalse())
						err := w.Close()
						Expect(err).To(MatchError(index.ErrDiscontinuous))
					})
					Specify("Index not defined at all", func() {
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: disc2Index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: disc2, Index: disc2Index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(
							ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{disc2},
								Start:    10 * telem.SecondTS,
							}))
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{disc2},
							[]telem.Series{
								telem.NewSeriesV[int64](1, 2, 3, 4, 5),
							},
						))
						Expect(ok).To(BeTrue())
						_, ok = w.Commit()
						Expect(ok).To(BeFalse())
						Expect(w.Close()).To(MatchError(index.ErrDiscontinuous))
					})
				})
			})
			Describe("Data t Errors", func() {
				Specify("Invalid Data t for series", func() {
					var dtErr = GenerateCesiumChannelKey()
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      dtErr,
							DataType: telem.Int64T,
							Rate:     1,
						})).To(Succeed())
					w := MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{dtErr},
							Start:    10 * telem.SecondTS,
						}))
					w = MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{dtErr},
							Start:    15 * telem.SecondTS,
						}))
					ok := w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{dtErr},
						[]telem.Series{
							telem.NewSeriesV[float64](1, 2, 3, 4, 5),
						},
					))
					Expect(ok).To(BeTrue())
					_, ok = w.Commit()
					Expect(ok).To(BeFalse())
					err := w.Close()
					Expect(err).To(MatchError(validate.Error))
					Expect(err.Error()).To(ContainSubstring("expected int64, got float64"))
				})
			})

			Describe("Virtual Channels", func() {
				It("Should write to virtual channel", func() {
					var virtual1 = GenerateCesiumChannelKey()
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: virtual1, DataType: telem.Int64T, Virtual: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{virtual1},
						Start:    10 * telem.SecondTS,
					}))

					Expect(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{virtual1},
						[]telem.Series{telem.NewSeriesV[int64](1, 2, 3)},
					))).To(BeTrue())

					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Close", func() {
				It("Should not allow operations on a closed iterator", func() {
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: 100, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						i = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{100}, Start: 10 * telem.SecondTS}))
						e = core.EntityClosed("cesium.writer")
					)
					Expect(i.Close()).To(Succeed())
					Expect(i.Close()).To(Succeed())
					Expect(i.Write(cesium.Frame{Series: []telem.Series{{Data: []byte{1, 2, 3}}}})).To(BeFalse())
					_, ok := i.Commit()
					Expect(ok).To(BeFalse())
					Expect(i.Error()).To(MatchError(e))
				})
			})

			Describe("Close", func() {
				It("Should close properly with a control setup", func() {
					Expect(db.ConfigureControlUpdateChannel(ctx, 199)).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: 200, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: 201, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: 202, DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())

					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{200, 201, 202}, Start: 10 * telem.SecondTS}))
					Expect(w.Close()).To(Succeed())
				})
				It("Should not allow operations on a closed iterator", func() {
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: 300, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{100}, Start: 10 * telem.SecondTS}))
						e = core.EntityClosed("cesium.writer")
					)
					Expect(w.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
					Expect(w.Write(cesium.Frame{Series: []telem.Series{{Data: []byte{1, 2, 3}}}})).To(BeFalse())
					_, ok := w.Commit()
					Expect(ok).To(BeFalse())
					Expect(w.Error()).To(MatchError(e))
				})
			})
		})
	}
})
