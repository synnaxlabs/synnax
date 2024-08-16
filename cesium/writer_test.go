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
	"encoding/binary"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"os"
	"strconv"
	"time"
)

var _ = Describe("Writer Behavior", func() {
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
			Describe("Happy Path", func() {
				Context("Indexed", func() {
					Specify("Basic Write", func() {
						var (
							basic1      = GenerateChannelKey()
							basic1Index = GenerateChannelKey()
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
						Expect(w.Error()).ToNot(HaveOccurred())
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
							basic1    = GenerateChannelKey()
							basicIdx1 = GenerateChannelKey()
							basic2    = GenerateChannelKey()
							basicIdx2 = GenerateChannelKey()
						)
						By("Creating the channels")
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
				Context("Rate channels", func() {
					It("Should write to many rate channels at once", func() {
						var (
							rate1 = GenerateChannelKey()
							rate2 = GenerateChannelKey()
							rate3 = GenerateChannelKey()
						)
						By("Creating the channels")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: rate1, Name: "Rate 1", Rate: 1 * telem.Hz, DataType: telem.Int64T},
							cesium.Channel{Key: rate2, Rate: 2 * telem.Hz, DataType: telem.Int64T},
							cesium.Channel{Key: rate3, Rate: 2 * telem.Hz, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{rate1, rate2, rate3},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{rate1, rate2, rate3},
							[]telem.Series{
								telem.NewSeriesV[int64](100),
								telem.NewSeriesV[int64](100, 105),
								telem.NewSeriesV[int64](100, 105),
							},
						))
						Expect(ok).To(BeTrue())
						t, ok := w.Commit()
						Expect(w.Error()).To(BeNil())
						Expect(ok).To(BeTrue())
						Expect(t).To(Equal(10500*telem.MillisecondTS + 1))

						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{rate1, rate2, rate3},
							[]telem.Series{
								telem.NewSeriesV[int64](110, 120, 130, 140),
								telem.NewSeriesV[int64](110, 115, 120, 125, 130, 135, 140, 145),
								telem.NewSeriesV[int64](110, 115, 120, 125, 130, 135, 140, 145),
							},
						))
						Expect(ok).To(BeTrue())
						t, ok = w.Commit()
						Expect(t).To(Equal(14*telem.SecondTS + 500*telem.MillisecondTS + 1))
						Expect(ok).To(BeTrue())

						Expect(w.Close()).To(Succeed())

						By("Checking that the data is correct")
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, rate1, rate2, rate3))
						Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](100, 110, 120, 130, 140).Data))
						Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](100, 105, 110, 115, 120, 125, 130, 135, 140, 145).Data))
						Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](100, 105, 110, 115, 120, 125, 130, 135, 140, 145).Data))
					})
				})
				Context("Rate, Index, and Data", func() {
					It("Should write properly", func() {
						var (
							rate1  = GenerateChannelKey()
							rate2  = GenerateChannelKey()
							index1 = GenerateChannelKey()
							data1  = GenerateChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: rate1, Rate: 2 * telem.Hz, DataType: telem.Int64T},
							cesium.Channel{Key: rate2, Rate: 2 * telem.Hz, DataType: telem.Int64T},
							cesium.Channel{Key: index1, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Key: data1, DataType: telem.Int64T, Index: index1},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{index1, data1, rate1, rate2},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index1, data1, rate1, rate2},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 13),
								telem.NewSeriesV[int64](10, 11, 13),
								telem.NewSeriesV[int64](100, 105, 110, 115, 120, 125, 130, 135),
								telem.NewSeriesV[int64](100, 105, 110, 115, 120, 125, 130, 135),
							},
						))
						Expect(ok).To(BeTrue())
						end, ok := w.Commit()
						Expect(end).To(Equal(13*telem.SecondTS + 500*telem.MillisecondTS + 1))
						Expect(ok).To(BeTrue())

						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index1, data1, rate1, rate2},
							[]telem.Series{
								telem.NewSecondsTSV(17),
								telem.NewSeriesV[int64](17),
								telem.NewSeriesV[int64](140, 145),
								telem.NewSeriesV[int64](140, 145),
							},
						))
						Expect(ok).To(BeTrue())
						end, ok = w.Commit()
						Expect(end).To(Equal(17*telem.SecondTS + 1))
						Expect(ok).To(BeTrue())

						Expect(w.Close()).To(Succeed())

						By("Checking that the data is correct")
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index1, data1, rate1, rate2))
						Expect(f.Series[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 13, 17).Data))
						Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](10, 11, 13, 17).Data))
						Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](100, 105, 110, 115, 120, 125, 130, 135, 140, 145).Data))
						Expect(f.Series[3].Data).To(Equal(telem.NewSeriesV[int64](100, 105, 110, 115, 120, 125, 130, 135, 140, 145).Data))
					})
					It("Should not write an empty frame", func() {

						var (
							rate1 = GenerateChannelKey()
							rate2 = GenerateChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: rate1, Rate: 2 * telem.Hz, DataType: telem.Int64T},
							cesium.Channel{Key: rate2, Rate: 2 * telem.Hz, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{rate1, rate2},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{rate1, rate2},
							[]telem.Series{
								{DataType: "int64"},
								{DataType: "int64"},
							},
						))
						Expect(ok).To(BeTrue())
						end, ok := w.Commit()
						Expect(ok).To(BeTrue())
						Expect(end).To(Equal(10 * telem.SecondTS))

						Expect(w.Close()).To(Succeed())
					})
				})
				Describe("Auto-commit", func() {
					Describe("Indexed channels", func() {
						var (
							index1 cesium.ChannelKey
							basic1 cesium.ChannelKey
							index2 cesium.ChannelKey
							basic2 cesium.ChannelKey
							basic3 cesium.ChannelKey
						)
						BeforeEach(func() {
							index1 = GenerateChannelKey()
							basic1 = GenerateChannelKey()
							index2 = GenerateChannelKey()
							basic2 = GenerateChannelKey()
							basic3 = GenerateChannelKey()

							By("Creating channels")
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{Key: index1, IsIndex: true, DataType: telem.TimeStampT},
								cesium.Channel{Key: basic1, Index: index1, DataType: telem.Int64T},
								cesium.Channel{Key: index2, IsIndex: true, DataType: telem.TimeStampT},
								cesium.Channel{Key: basic2, Index: index2, DataType: telem.Int64T},
								cesium.Channel{Key: basic3, Index: index2, DataType: telem.Uint32T},
							)).To(Succeed())
						})
						It("Should automatically commit the writer for all channels", func() {
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels:         []cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								Start:            10 * telem.SecondTS,
								Mode:             cesium.WriterPersistStream,
								EnableAutoCommit: config.True(),
							}))

							By("Writing telemetry")
							ok := w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(10, 12, 13, 14),
									telem.NewSeriesV[int64](100, 102, 103, 104),
									telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
									telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105),
								},
							))
							Expect(ok).To(BeTrue())
							Expect(w.Error()).ToNot(HaveOccurred())

							By("Reading the telemetry to assert they are committed")
							Eventually(func(g Gomega) {
								f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index1, basic1, index2, basic2, basic3))
								g.Expect(f.Get(index1)).To(HaveLen(1))
								g.Expect(f.Get(index1)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(14*telem.SecondTS + 1)))
								g.Expect(f.Get(index1)[0].Len()).To(Equal(int64(4)))
								g.Expect(f.Get(index1)[0].Data).To(Equal(telem.NewSecondsTSV(10, 12, 13, 14).Data))

								g.Expect(f.Get(basic1)).To(HaveLen(1))
								g.Expect(f.Get(basic1)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(14*telem.SecondTS + 1)))
								g.Expect(f.Get(basic1)[0].Len()).To(Equal(int64(4)))
								g.Expect(f.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 102, 103, 104).Data))

								g.Expect(f.Get(index2)).To(HaveLen(1))
								g.Expect(f.Get(index2)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
								g.Expect(f.Get(index2)[0].Len()).To(Equal(int64(6)))
								g.Expect(f.Get(index2)[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 12, 13, 14, 15).Data))

								g.Expect(f.Get(basic2)).To(HaveLen(1))
								g.Expect(f.Get(basic2)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
								g.Expect(f.Get(basic2)[0].Len()).To(Equal(int64(6)))
								g.Expect(f.Get(basic2)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105).Data))

								g.Expect(f.Get(basic3)).To(HaveLen(1))
								g.Expect(f.Get(basic3)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
								g.Expect(f.Get(basic3)[0].Len()).To(Equal(int64(6)))
								g.Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105).Data))
							}).Should(Succeed())

							By("Writing more telemetry")
							ok = w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(20, 22, 23, 24),
									telem.NewSeriesV[int64](200, 202, 203, 204),
									telem.NewSecondsTSV(20, 21, 22, 23, 24, 25),
									telem.NewSeriesV[int64](200, 201, 202, 203, 204, 205),
									telem.NewSeriesV[uint32](200, 201, 202, 203, 204, 205),
								},
							))

							Expect(ok).To(BeTrue())
							Expect(w.Error()).ToNot(HaveOccurred())

							By("Reading the telemetry to assert they are committed")
							Eventually(func(g Gomega) {
								f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index1, basic1, index2, basic2, basic3))
								g.Expect(f.Get(index1)[0].Data).To(Equal(telem.NewSecondsTSV(10, 12, 13, 14, 20, 22, 23, 24).Data))
								g.Expect(f.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 102, 103, 104, 200, 202, 203, 204).Data))
								g.Expect(f.Get(index2)[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 20, 21, 22, 23, 24, 25).Data))
								g.Expect(f.Get(basic2)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 200, 201, 202, 203, 204, 205).Data))
								g.Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105, 200, 201, 202, 203, 204, 205).Data))
							}).Should(Succeed())

							By("Closing the writer")
							Expect(w.Close()).To(Succeed())
						})

						It("Should block subsequent writes if a previous write encounters a commit error", func() {
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels:         []cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								Start:            10 * telem.SecondTS,
								Mode:             cesium.WriterPersistStream,
								EnableAutoCommit: config.True(),
							}))

							By("Writing telemetry")
							ok := w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(10, 12, 13, 14),
									telem.NewSeriesV[int64](100, 102, 103, 104),
									telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
									telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105),
								},
							))
							Expect(ok).To(BeTrue())
							Expect(w.Error()).ToNot(HaveOccurred())
							Expect(w.Close()).To(Succeed())

							By("Writing telemetry that would collide with previous domains")
							w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels:         []cesium.ChannelKey{index1, basic1},
								Start:            9 * telem.SecondTS,
								Mode:             cesium.WriterPersistStream,
								EnableAutoCommit: config.True(),
							}))
							ok = w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1},
								[]telem.Series{
									telem.NewSecondsTSV(9, 10, 11),
									telem.NewSeriesV[int64](99, 100, 101),
								},
							))

							Expect(ok).To(BeTrue())

							By("Checking that more writes to the writer would fail")
							Eventually(func() bool {
								return w.Write(cesium.Frame{})
							}).Should(BeFalse())

							By("Checking that the first commit did not succeed")
							f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index1, basic1, index2, basic2, basic3))

							Expect(f.Get(index1)[0].Data).To(Equal(telem.NewSecondsTSV(10, 12, 13, 14).Data))
							Expect(f.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 102, 103, 104).Data))
							Expect(f.Get(index2)[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 12, 13, 14, 15).Data))
							Expect(f.Get(basic2)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105).Data))
							Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105).Data))

							By("Resolving the error to be commit error")
							err := w.Error()
							Expect(err).To(MatchError(validate.Error))
							Expect(err).To(MatchError(ContainSubstring("overlaps with existing data occupying time range %v", (10 * telem.SecondTS).Range(14*telem.SecondTS+1))))

							By("Closing the writer")
							Expect(w.Close()).To(Succeed())
						})
						It("Should work with the write method", func() {
							start := 10 * telem.SecondTS

							for i := 0; i < 100; i++ {
								stamps := make([]telem.TimeStamp, 100)
								data := make([]int64, 100)
								for j := telem.TimeStamp(0); j < 100; j++ {
									stamps[j] = start + j*10*telem.MicrosecondTS
									data[j] = 1
								}
								w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels:         []cesium.ChannelKey{index1},
									Start:            start,
									Mode:             cesium.WriterPersistOnly,
									EnableAutoCommit: config.True(),
								}))
								Expect(w.Write(cesium.NewFrame([]cesium.ChannelKey{index1},
									[]telem.Series{
										telem.NewSeries(stamps),
									},
								))).To(BeTrue())
								Expect(w.Close()).To(Succeed())

								w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels:         []cesium.ChannelKey{basic1},
									Start:            start,
									Mode:             cesium.WriterPersistOnly,
									EnableAutoCommit: config.True(),
								}))
								Expect(w.Write(cesium.NewFrame([]cesium.ChannelKey{basic1},
									[]telem.Series{
										telem.NewSeries(data),
									},
								))).To(BeTrue())
								Expect(w.Close()).To(Succeed())
								start += 2 * telem.MillisecondTS
							}

							f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1))
							Expect(f.Series).To(HaveLen(100))
						})

						Describe("Auto-Persist", func() {
							It("Should auto persist on every commit when set to always auto persist", func() {
								By("Opening a writer")
								w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels:                 []cesium.ChannelKey{index1, basic1},
									Start:                    10 * telem.SecondTS,
									Mode:                     cesium.WriterPersistStream,
									AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
									EnableAutoCommit:         config.True(),
								}))

								By("Writing telemetry")
								ok := w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(10, 12),
										telem.NewSeriesV[int64](100, 102),
									},
								))
								Expect(ok).To(BeTrue())
								Expect(w.Error()).ToNot(HaveOccurred())

								By("Asserting that the telemetry has been persisted")
								f := MustSucceed(fs.Open(channelKeyToPath(index1)+"/index.domain", os.O_RDONLY))
								buf := make([]byte, 26)
								_, err := f.ReadAt(buf, 0)
								Expect(err).ToNot(HaveOccurred())
								Expect(f.Close()).To(Succeed())
								Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
								Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(12*telem.SecondTS + 1)))
								Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(16)))

								f = MustSucceed(fs.Open(channelKeyToPath(basic1)+"/index.domain", os.O_RDONLY))
								buf = make([]byte, 26)
								_, err = f.ReadAt(buf, 0)
								Expect(err).ToNot(HaveOccurred())
								Expect(f.Close()).To(Succeed())
								Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
								Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(12*telem.SecondTS + 1)))
								Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(16)))

								By("Closing the writer")
								Expect(w.Close()).To(Succeed())
							})

							It("Should auto persist every second when the interval is not set", func() {
								By("Opening a writer")
								w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels:         []cesium.ChannelKey{index1, basic1},
									Start:            10 * telem.SecondTS,
									Mode:             cesium.WriterPersistStream,
									EnableAutoCommit: config.True(),
								}))

								By("Writing telemetry")
								ok := w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(10, 11, 13),
										telem.NewSeriesV[int64](100, 101, 103),
									},
								))
								Expect(ok).To(BeTrue())
								Expect(w.Error()).ToNot(HaveOccurred())

								By("Checking that this telemetry is not persisted")
								s := MustSucceed(fs.Stat(channelKeyToPath(index1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))
								s = MustSucceed(fs.Stat(channelKeyToPath(basic1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))

								By("Sleeping to wait for the threshold to be met")
								time.Sleep(time.Duration(1000 * telem.Millisecond))

								By("Writing more telemetry")
								ok = w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(20, 22, 23, 24),
										telem.NewSeriesV[int64](200, 202, 203, 204),
									},
								))

								Expect(ok).To(BeTrue())
								Expect(w.Error()).ToNot(HaveOccurred())

								By("Asserting that the telemetry has been persisted")
								Eventually(func(g Gomega) {
									f := MustSucceed(fs.Open(channelKeyToPath(index1)+"/index.domain", os.O_RDONLY))
									buf := make([]byte, 26)
									_, err := f.Read(buf)
									g.Expect(err).ToNot(HaveOccurred())
									g.Expect(f.Close()).To(Succeed())
									g.Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
									g.Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(24*telem.SecondTS + 1)))
									g.Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(56)))

									f = MustSucceed(fs.Open(channelKeyToPath(basic1)+"/index.domain", os.O_RDONLY))
									buf = make([]byte, 26)
									_, err = f.Read(buf)
									g.Expect(err).ToNot(HaveOccurred())
									g.Expect(f.Close()).To(Succeed())
									g.Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
									g.Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(24*telem.SecondTS + 1)))
									g.Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(56)))
								}).Should(Succeed())

								By("Writing more telemetry")
								ok = w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(30, 31, 33),
										telem.NewSeriesV[int64](300, 301, 303),
									},
								))
								Expect(ok).To(BeTrue())

								By("Assert that the stranded telemetry be persisted on close")
								Expect(w.Close()).To(Succeed())

								f := MustSucceed(fs.Open(channelKeyToPath(index1)+"/index.domain", os.O_RDONLY))
								buf := make([]byte, 26)
								_, err := f.Read(buf)
								Expect(err).ToNot(HaveOccurred())
								Expect(f.Close()).To(Succeed())
								Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(33*telem.SecondTS + 1)))
								Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(80)))

								f = MustSucceed(fs.Open(channelKeyToPath(basic1)+"/index.domain", os.O_RDONLY))
								buf = make([]byte, 26)
								_, err = f.Read(buf)
								Expect(err).ToNot(HaveOccurred())
								Expect(f.Close()).To(Succeed())
								Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(33*telem.SecondTS + 1)))
								Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(80)))
							})

							It("Should auto persist once the time interval is reached", func() {
								By("Opening a writer")
								w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels:                 []cesium.ChannelKey{index1, basic1},
									Start:                    10 * telem.SecondTS,
									Mode:                     cesium.WriterPersistStream,
									EnableAutoCommit:         config.True(),
									AutoIndexPersistInterval: 200 * telem.Millisecond,
								}))

								By("Writing telemetry")
								ok := w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(10, 11, 13),
										telem.NewSeriesV[int64](100, 101, 103),
									},
								))
								Expect(ok).To(BeTrue())
								Expect(w.Error()).ToNot(HaveOccurred())

								By("Checking that this telemetry is not persisted")
								s := MustSucceed(fs.Stat(channelKeyToPath(index1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))
								s = MustSucceed(fs.Stat(channelKeyToPath(basic1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))

								By("Sleeping to wait for the threshold to be met")
								time.Sleep(time.Duration(200 * telem.Millisecond))

								By("Writing more telemetry")
								ok = w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(20, 22, 23, 24),
										telem.NewSeriesV[int64](200, 202, 203, 204),
									},
								))

								Expect(ok).To(BeTrue())
								Expect(w.Error()).ToNot(HaveOccurred())

								By("Asserting that the telemetry has been persisted")
								Eventually(func(g Gomega) {
									f := MustSucceed(fs.Open(channelKeyToPath(index1)+"/index.domain", os.O_RDONLY))
									buf := make([]byte, 26)
									_, err := f.ReadAt(buf, 0)
									g.Expect(err).ToNot(HaveOccurred())
									g.Expect(f.Close()).To(Succeed())
									g.Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
									g.Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(24*telem.SecondTS + 1)))
									g.Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(56)))

									f = MustSucceed(fs.Open(channelKeyToPath(basic1)+"/index.domain", os.O_RDONLY))
									buf = make([]byte, 26)
									_, err = f.ReadAt(buf, 0)
									g.Expect(err).ToNot(HaveOccurred())
									g.Expect(f.Close()).To(Succeed())
									g.Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
									g.Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(24*telem.SecondTS + 1)))
									g.Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(56)))
								}).Should(Succeed())

								By("Writing more telemetry")
								ok = w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(30, 31, 33),
										telem.NewSeriesV[int64](300, 301, 303),
									},
								))
								Expect(ok).To(BeTrue())

								ok = w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(40, 41),
										telem.NewSeriesV[int64](400, 401),
									},
								))
								Expect(ok).To(BeTrue())

								time.Sleep(time.Duration(200 * telem.Millisecond))

								ok = w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(43),
										telem.NewSeriesV[int64](403),
									},
								))
								Expect(ok).To(BeTrue())
								Expect(w.Error()).ToNot(HaveOccurred())

								By("Asserting that the telemetry has been persisted")
								Eventually(func(g Gomega) {
									f := MustSucceed(fs.Open(channelKeyToPath(index1)+"/index.domain", os.O_RDONLY))
									buf := make([]byte, 26)
									_, err := f.Read(buf)
									g.Expect(err).ToNot(HaveOccurred())
									g.Expect(f.Close()).To(Succeed())
									g.Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
									g.Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(43*telem.SecondTS + 1)))
									g.Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(13 * 8)))

									f = MustSucceed(fs.Open(channelKeyToPath(basic1)+"/index.domain", os.O_RDONLY))
									buf = make([]byte, 26)
									_, err = f.ReadAt(buf, 0)
									g.Expect(err).ToNot(HaveOccurred())
									g.Expect(f.Close()).To(Succeed())
									g.Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
									g.Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(43*telem.SecondTS + 1)))
									g.Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(13 * 8)))
								}).Should(Succeed())

								Expect(w.Close()).To(Succeed())
							})
						})
					})
				})
				Describe("Auto file cutoff", func() {
					var (
						db2                *cesium.DB
						index, basic, rate cesium.ChannelKey
					)

					BeforeEach(func() {
						index = GenerateChannelKey()
						basic = GenerateChannelKey()
						rate = GenerateChannelKey()
					})

					AfterEach(func() {
						Expect(db2.Close()).To(Succeed())
						Expect(fs.Remove("size-capped-db")).To(Succeed())
					})

					Specify("With AutoCommit", func() {
						db2 = MustSucceed(cesium.Open("size-capped-db",
							cesium.WithFS(fs),
							cesium.WithFileSize(40*telem.ByteSize),
							cesium.WithInstrumentation(PanicLogger()),
						))

						Expect(db2.CreateChannel(
							ctx,
							cesium.Channel{Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic, Index: index, DataType: telem.Int64T},
							cesium.Channel{Key: rate, Rate: 1 * telem.Hz, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:         []cesium.ChannelKey{index, basic, rate},
							Start:            10 * telem.SecondTS,
							EnableAutoCommit: config.True(),
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
								telem.NewSeriesV[int64](0, 1),
							},
						))
						Expect(ok).To(BeTrue())

						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26),
								telem.NewSeriesV[int64](200, 201, 202, 203, 204, 205, 206),
								telem.NewSeriesV[int64](2, 3, 4, 5),
							},
						))
						Expect(ok).To(BeTrue())

						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(30, 31, 33),
								telem.NewSeriesV[int64](300, 301, 303),
								telem.NewSeriesV[int64](6, 7, 8),
							},
						))
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						By("Asserting that the first two channels now have three files while the third one has two", func() {
							subFS := MustSucceed(fs.Sub("size-capped-db"))
							l := MustSucceed(subFS.List(strconv.Itoa(int(index))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(3))
							Expect(l[0].Size()).To(Equal(int64(6 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(7 * telem.Int64T.Density())))
							Expect(l[2].Size()).To(Equal(int64(3 * telem.Int64T.Density())))
							l = MustSucceed(subFS.List(strconv.Itoa(int(basic))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(3))
							Expect(l[0].Size()).To(Equal(int64(6 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(7 * telem.Int64T.Density())))
							Expect(l[2].Size()).To(Equal(int64(3 * telem.Int64T.Density())))
							l = MustSucceed(subFS.List(strconv.Itoa(int(rate))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(2))
							Expect(l[0].Size()).To(Equal(int64(6 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(3 * telem.Int64T.Density())))
						})

						By("Asserting that the data is correct", func() {
							f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax, index, basic, rate))
							indexF := f.Get(index)
							Expect(indexF).To(HaveLen(3))
							Expect(indexF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(indexF[1].TimeRange).To(Equal((15*telem.SecondTS + 1).Range(26*telem.SecondTS + 1)))
							Expect(indexF[2].TimeRange).To(Equal((26*telem.SecondTS + 1).Range(33*telem.SecondTS + 1)))

							basicF := f.Get(basic)
							Expect(basicF).To(HaveLen(3))
							Expect(basicF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(basicF[1].TimeRange).To(Equal((15*telem.SecondTS + 1).Range(26*telem.SecondTS + 1)))
							Expect(basicF[2].TimeRange).To(Equal((26*telem.SecondTS + 1).Range(33*telem.SecondTS + 1)))

							rateF := f.Get(rate)
							Expect(rateF).To(HaveLen(2))
							Expect(rateF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(rateF[1].TimeRange).To(Equal((15*telem.SecondTS + 1).Range(18*telem.SecondTS + 1)))
						})
					})

					Specify("With AutoCommit: should not commit a tiny domain", func() {
						db2 = MustSucceed(cesium.Open("size-capped-db",
							cesium.WithFS(fs),
							cesium.WithFileSize(80*telem.ByteSize),
							cesium.WithInstrumentation(PanicLogger()),
						))

						Expect(db2.CreateChannel(
							ctx,
							cesium.Channel{Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic, Index: index, DataType: telem.Int64T},
							cesium.Channel{Key: rate, Rate: 1 * telem.Hz, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:                 []cesium.ChannelKey{index, basic, rate},
							Start:                    10 * telem.SecondTS,
							EnableAutoCommit:         config.True(),
							AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 106),
								telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5),
							},
						))
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						w = MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:                 []cesium.ChannelKey{index, basic, rate},
							Start:                    17 * telem.SecondTS,
							EnableAutoCommit:         config.True(),
							AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
						}))

						// This should still go to file 1.
						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(17),
								telem.NewSeriesV[int64](107),
								telem.NewSeriesV[int64](6),
							},
						))
						Expect(ok).To(BeTrue())

						// This should still go to file 1.
						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(18, 19),
								telem.NewSeriesV[int64](108, 109),
								telem.NewSeriesV[int64](7, 8),
							},
						))

						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(20, 21, 22, 23, 24),
								telem.NewSeriesV[int64](200, 201, 202, 203, 204),
								telem.NewSeriesV[int64](9, 10),
							},
						))
						Expect(ok).To(BeTrue())
						Expect(w.Close()).To(Succeed())

						subFS := MustSucceed(fs.Sub("size-capped-db"))
						By("Asserting that the first two channels have 2 files, while the last channel has an oversize file", func() {
							l := MustSucceed(subFS.List(strconv.Itoa(int(index))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(2))
							Expect(l[0].Size()).To(Equal(int64(10 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(5 * telem.Int64T.Density())))
							l = MustSucceed(subFS.List(strconv.Itoa(int(basic))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(2))
							Expect(l[0].Size()).To(Equal(int64(10 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(5 * telem.Int64T.Density())))
							l = MustSucceed(subFS.List(strconv.Itoa(int(rate))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(2))
							Expect(l[0].Size()).To(Equal(int64(11 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(0 * telem.Int64T.Density())))
						})

						By("Closing an reopening the db")
						Expect(db2.Close()).To(Succeed())

						db2 = MustSucceed(cesium.Open("size-capped-db",
							cesium.WithFS(fs),
							cesium.WithFileSize(64*telem.ByteSize),
							cesium.WithInstrumentation(PanicLogger()),
						))

						By("Asserting that upon writing to the channels, the writes go to appropriate files", func() {
							w = MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
								Channels:                 []cesium.ChannelKey{index, basic, rate},
								Start:                    30 * telem.SecondTS,
								EnableAutoCommit:         config.True(),
								AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
							}))

							Expect(w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index, basic, rate},
								[]telem.Series{
									telem.NewSecondsTSV(30, 31),
									telem.NewSeriesV[int64](300, 301),
									telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5),
								},
							))).To(BeTrue())
							Expect(w.Close()).To(Succeed())

							Expect(MustSucceed(subFS.Stat(strconv.Itoa(int(basic)) + "/2.domain")).Size()).To(Equal(int64(7 * telem.Int64T.Density())))
							Expect(MustSucceed(subFS.Stat(strconv.Itoa(int(index)) + "/2.domain")).Size()).To(Equal(int64(7 * telem.TimeStampT.Density())))
							Expect(MustSucceed(subFS.Stat(strconv.Itoa(int(rate)) + "/2.domain")).Size()).To(Equal(int64(6 * telem.Int64T.Density())))
						})

						By("Asserting that the data is correct", func() {
							f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax, index, basic, rate))
							indexF := f.Get(index)
							Expect(indexF).To(HaveLen(4))
							Expect(indexF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(16*telem.SecondTS + 1)))
							Expect(indexF[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16).Data))
							Expect(indexF[1].TimeRange).To(Equal((17 * telem.SecondTS).Range(19*telem.SecondTS + 1)))
							Expect(indexF[1].Data).To(Equal(telem.NewSecondsTSV(17, 18, 19).Data))
							Expect(indexF[2].TimeRange).To(Equal((19*telem.SecondTS + 1).Range(24*telem.SecondTS + 1)))
							Expect(indexF[2].Data).To(Equal(telem.NewSecondsTSV(20, 21, 22, 23, 24).Data))
							Expect(indexF[3].TimeRange).To(Equal((30 * telem.SecondTS).Range(31*telem.SecondTS + 1)))
							Expect(indexF[3].Data).To(Equal(telem.NewSecondsTSV(30, 31).Data))

							basicF := f.Get(basic)
							Expect(basicF).To(HaveLen(4))
							Expect(basicF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(16*telem.SecondTS + 1)))
							Expect(basicF[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 106).Data))
							Expect(basicF[1].TimeRange).To(Equal((17 * telem.SecondTS).Range(19*telem.SecondTS + 1)))
							Expect(basicF[1].Data).To(Equal(telem.NewSeriesV[int64](107, 108, 109).Data))
							Expect(basicF[2].TimeRange).To(Equal((19*telem.SecondTS + 1).Range(24*telem.SecondTS + 1)))
							Expect(basicF[2].Data).To(Equal(telem.NewSeriesV[int64](200, 201, 202, 203, 204).Data))
							Expect(basicF[3].TimeRange).To(Equal((30 * telem.SecondTS).Range(31*telem.SecondTS + 1)))
							Expect(basicF[3].Data).To(Equal(telem.NewSeriesV[int64](300, 301).Data))

							rateF := f.Get(rate)
							Expect(rateF).To(HaveLen(3))
							Expect(rateF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(rateF[0].Data).To(Equal(telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5).Data))
							Expect(rateF[1].TimeRange).To(Equal((17 * telem.SecondTS).Range(21*telem.SecondTS + 1)))
							Expect(rateF[1].Data).To(Equal(telem.NewSeriesV[int64](6, 7, 8, 9, 10).Data))
							Expect(rateF[2].TimeRange).To(Equal((30 * telem.SecondTS).Range(35*telem.SecondTS + 1)))
							Expect(rateF[2].Data).To(Equal(telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5).Data))
						})
					})

					Specify("Without AutoCommit", func() {
						db2 = MustSucceed(cesium.Open("size-capped-db",
							cesium.WithFS(fs),
							cesium.WithFileSize(40*telem.ByteSize),
							cesium.WithInstrumentation(PanicLogger()),
						))

						Expect(db2.CreateChannel(
							ctx,
							cesium.Channel{Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic, Index: index, DataType: telem.Int64T},
							cesium.Channel{Key: rate, Rate: 1 * telem.Hz, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{index, basic, rate},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						ok := w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
								telem.NewSeriesV[int64](0, 1),
							},
						))
						Expect(ok).To(BeTrue())

						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26),
								telem.NewSeriesV[int64](200, 201, 202, 203, 204, 205, 206),
								telem.NewSeriesV[int64](2, 3, 4, 5),
							},
						))
						Expect(ok).To(BeTrue())

						_, ok = w.Commit()
						Expect(ok).To(BeTrue())

						ok = w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic, rate},
							[]telem.Series{
								telem.NewSecondsTSV(30, 31, 33),
								telem.NewSeriesV[int64](300, 301, 303),
								telem.NewSeriesV[int64](6, 7, 8),
							},
						))
						Expect(ok).To(BeTrue())
						w.Commit()
						Expect(w.Close()).To(Succeed())

						By("Asserting that all channels have two files", func() {
							subFS := MustSucceed(fs.Sub("size-capped-db"))
							l := MustSucceed(subFS.List(strconv.Itoa(int(index))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(2))
							Expect(l[0].Size()).To(Equal(int64(13 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(3 * telem.Int64T.Density())))
							l = MustSucceed(subFS.List(strconv.Itoa(int(basic))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(2))
							Expect(l[0].Size()).To(Equal(int64(13 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(3 * telem.Int64T.Density())))
							l = MustSucceed(subFS.List(strconv.Itoa(int(rate))))
							l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
								return item.Name() != "index.domain" && item.Name() != "counter.domain" && item.Name() != "meta.json" && item.Name() != "tombstone.domain"
							})
							Expect(l).To(HaveLen(2))
							Expect(l[0].Size()).To(Equal(int64(6 * telem.Int64T.Density())))
							Expect(l[1].Size()).To(Equal(int64(3 * telem.Int64T.Density())))
						})

						By("Asserting that the data is correct", func() {
							f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax, index, basic, rate))
							indexF := f.Get(index)
							Expect(indexF).To(HaveLen(2))
							Expect(indexF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(26*telem.SecondTS + 1)))
							Expect(indexF[1].TimeRange).To(Equal((26*telem.SecondTS + 1).Range(33*telem.SecondTS + 1)))

							Expect(f.Get(basic)).To(HaveLen(2))
							Expect(f.Get(basic)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(26*telem.SecondTS + 1)))
							Expect(f.Get(basic)[1].TimeRange).To(Equal((26*telem.SecondTS + 1).Range(33*telem.SecondTS + 1)))

							Expect(f.Get(rate)).To(HaveLen(2))
							Expect(f.Get(rate)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
							Expect(f.Get(rate)[1].TimeRange).To(Equal((15*telem.SecondTS + 1).Range(18*telem.SecondTS + 1)))
						})
					})
					It("Should not break when auto committing to not all channels", func() {
						db2 = MustSucceed(cesium.Open("size-capped-db",
							cesium.WithFS(fs),
							cesium.WithFileSize(40*telem.ByteSize),
							cesium.WithInstrumentation(PanicLogger()),
						))

						var (
							index2 = GenerateChannelKey()
							basic2 = GenerateChannelKey()
						)

						Expect(db2.CreateChannel(
							ctx,
							cesium.Channel{Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic, Index: index, DataType: telem.Int64T},
							cesium.Channel{Key: index2, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic2, Index: index2, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:                 []core.ChannelKey{index, basic, index2, basic2},
							Start:                    10 * telem.SecondTS,
							EnableAutoCommit:         config.True(),
							AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
						}))

						Expect(w.Write(cesium.NewFrame(
							[]core.ChannelKey{index, basic, index2, basic2},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15),
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15),
							},
						))).To(BeTrue())

						By("Asserting that only writing to two channels will not fail")
						Expect(w.Write(cesium.NewFrame(
							[]core.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(16, 17, 18),
								telem.NewSeriesV[int64](16, 17, 18),
							},
						))).To(BeTrue())
						Expect(w.Close()).To(Succeed())
					})
				})
				Describe("SetKV Authority", func() {
					It("Should set the authority of writers", func() {
						var (
							key  = GenerateChannelKey()
							key2 = GenerateChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.TimeStampT},
							cesium.Channel{Key: key2, Virtual: true, DataType: telem.StringT},
						)).To(Succeed())
						w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{key, key2},
							Start:          10 * telem.SecondTS,
							Authorities:    []control.Authority{control.Authority(100), control.Authority(110)},
							SendAuthErrors: config.True(),
						}))

						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:       []cesium.ChannelKey{key, key2},
							Start:          10 * telem.SecondTS,
							Authorities:    []control.Authority{control.Authority(110), control.Authority(100)},
							SendAuthErrors: config.True(),
						}))

						w1.Write(cesium.NewFrame(
							[]cesium.ChannelKey{key},
							[]telem.Series{
								telem.NewSeriesV[int64](1, 2, 3, 4),
							}),
						)
						Expect(w1.Error()).To(HaveOccurredAs(control.Unauthorized))

						Expect(w1.SetAuthority(cesium.WriterConfig{Channels: []cesium.ChannelKey{key, key2}, Authorities: []control.Authority{control.Absolute, control.Authority(0)}})).To(BeTrue())
						Expect(w1.Error()).ToNot(HaveOccurred())

						w2.Write(cesium.NewFrame(
							[]cesium.ChannelKey{key},
							[]telem.Series{
								telem.NewSeriesV[int64](1, 3, 4),
							},
						))
						Expect(w2.Error()).To(HaveOccurredAs(control.Unauthorized))

						w1.Write(cesium.NewFrame(
							[]cesium.ChannelKey{key2},
							[]telem.Series{
								{DataType: telem.StringT, Data: []byte("hehe")},
							},
						))

						Expect(w1.Error()).To(HaveOccurredAs(control.Unauthorized))

						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
					})
				})
			})
			Describe("Stream Only Mode", func() {
				It("Should not persist data", func() {
					var (
						basic1      = GenerateChannelKey()
						basic1Index = GenerateChannelKey()
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
						Mode:     cesium.WriterStreamOnly,
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
					Expect(end).To(Equal(13*telem.SecondTS + 1))

					By("Reading the data back")
					frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1))
					Expect(frame.Series).To(HaveLen(0))
					tsFrame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1Index))
					Expect(tsFrame.Series).To(HaveLen(0))
					Expect(w.Close()).To(Succeed())
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
					Expect(err).To(MatchError(core.ErrChannelNotFound))
				})
				Specify("Encounters channel that does not exist after already successfully creating some writers", func() {
					key1 := GenerateChannelKey()
					key2 := GenerateChannelKey()

					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: key1, Name: "nonexistent 1", DataType: telem.Int64T, Rate: 1 * telem.Hz},
						cesium.Channel{Key: key2, Name: "nonexistent 2", DataType: telem.Int64T, Rate: 1 * telem.Hz},
					)).To(Succeed())

					_, err := db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{key1, 88888}, Start: 10 * telem.SecondTS})
					Expect(err).To(MatchError(core.ErrChannelNotFound))
				})
			})
			Describe("Frame Errors", Ordered, func() {
				var (
					frameErr1 = GenerateChannelKey()
					frameErr2 = GenerateChannelKey()
				)
				BeforeAll(func() {
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: frameErr1, Name: "uneven 1", DataType: telem.Int64T, Rate: 1 * telem.Hz},
						cesium.Channel{Key: frameErr2, Name: "uneven 2", DataType: telem.Int64T, Rate: 1 * telem.Hz},
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
					Expect(w.Close()).To(Succeed())
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
					Expect(err).To(MatchError(ContainSubstring("exactly one")))
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
						disc1      = GenerateChannelKey()
						disc1Index = GenerateChannelKey()
						disc2      = GenerateChannelKey()
						disc2Index = GenerateChannelKey()
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
						Expect(w.Close()).To(MatchError(ContainSubstring("cannot find stamp start")))
					})
				})
			})
			Describe("Data Type Errors", func() {
				Specify("Invalid Data Type for series", func() {
					var (
						dtErr      = GenerateChannelKey()
						dtErrIndex = GenerateChannelKey()
					)
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      dtErr,
							DataType: telem.Int64T,
							Rate:     1,
						},
						cesium.Channel{
							Key:      dtErrIndex,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						})).To(Succeed())
					w := MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{dtErr, dtErrIndex},
							Start:    10 * telem.SecondTS,
						}))
					ok := w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{dtErr},
						[]telem.Series{
							telem.NewSeriesV[uint16](1, 2, 3, 4, 5),
						},
					))
					Expect(ok).To(BeTrue())
					_, ok = w.Commit()
					Expect(ok).To(BeFalse())
					err := w.Error()
					Expect(err).To(MatchError(validate.Error))
					Expect(err.Error()).To(ContainSubstring("expected int64, got uint16"))

					Expect(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{dtErrIndex},
						[]telem.Series{telem.NewSeriesV[int64](10, 11, 12, 13)},
					))).To(BeTrue())
					err = w.Error()
					Expect(err).ToNot(HaveOccurred())
					Expect(w.Close()).To(Succeed())
				})
			})
			Describe("ErrOnUnauthorized", func() {
				var (
					key        cesium.ChannelKey
					controlKey = GenerateChannelKey()
					w1         *cesium.Writer
					w2         *cesium.Writer
				)
				BeforeEach(func() {
					key = GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(db.ConfigureControlUpdateChannel(ctx, controlKey)).To(Succeed())
					w1 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{key}, Start: 1 * telem.SecondTS}))
				})
				AfterEach(func() {
					Expect(w1.Close()).To(Succeed())
				})
				Context("False", func() {
					It("Should not return an error if writer is not authorized to write", func() {
						w2 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{key}, Start: 1 * telem.SecondTS}))
						Consistently(func() bool {
							return w2.Write(cesium.NewFrame([]cesium.ChannelKey{key}, []telem.Series{telem.NewSeriesV[int64](1, 2, 3, 4)}))
						}).Should(BeTrue())
						Expect(w2.Error()).ToNot(HaveOccurred())
						Expect(w2.Close()).To(Succeed())
					})
				})
				Context("True", func() {
					It("Should return an error if writer is not authorized to write", func() {
						w2, err := db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{key}, Start: 1 * telem.SecondTS, ErrOnUnauthorized: config.True()})
						Expect(err).To(MatchError(control.Unauthorized))
						Expect(w2).To(BeNil())
					})
				})
			})
			Describe("Virtual Channels", func() {
				It("Should write to virtual channel", func() {
					var virtual1 = GenerateChannelKey()
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
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Name: "Close 1", DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{key}, Start: 10 * telem.SecondTS}))
						e = core.EntityClosed("cesium.writer")
					)
					Expect(w.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
					Expect(w.Write(cesium.Frame{Series: []telem.Series{{Data: []byte{1, 2, 3}}}})).To(BeFalse())
					_, ok := w.Commit()
					Expect(ok).To(BeFalse())
					err := w.Error()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(e.Error()))
				})
			})
			Describe("Close", func() {
				It("Should close properly with a control setup", func() {
					k1, k2, k3, k4 := GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
					Expect(db.ConfigureControlUpdateChannel(ctx, k1)).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k2, Name: "Close 2", DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k3, Name: "Close 3", DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k4, Name: "Close 4", DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())

					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{k2, k3, k4}, Start: 10 * telem.SecondTS}))
					Expect(w.Close()).To(Succeed())
				})
				It("Should not allow operations on a closed writer", func() {
					k := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k, Name: "Close 5", DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{k}, Start: 10 * telem.SecondTS}))
						e = core.EntityClosed("cesium.writer")
					)
					Expect(w.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
					Expect(w.Write(cesium.Frame{Series: []telem.Series{{Data: []byte{1, 2, 3}}}})).To(BeFalse())
					_, ok := w.Commit()
					Expect(ok).To(BeFalse())
					Expect(w.Error()).To(HaveOccurredAs(e))
				})

				It("Should not allow opening an iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.OpenWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow opening a stream iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.NewStreamWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow writing from a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					err := subDB.Write(ctx, 0, cesium.NewFrame([]cesium.ChannelKey{key}, []telem.Series{telem.NewSeriesV[int64](1, 2, 3)}))
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))
					err = subDB.WriteArray(ctx, key, 0, telem.NewSeriesV[int64](1, 2, 3))
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})
			})
		})
	}
})
