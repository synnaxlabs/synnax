// Copyright 2025 Synnax Labs, Inc.
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
	"os"
	"runtime"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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
				ShouldNotLeakRoutinesJustBeforeEach()

				Context("Indexed", func() {
					Specify("Basic Write", func() {
						var (
							basic1      = GenerateChannelKey()
							basic1Index = GenerateChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Key: basic1Index, Name: "Shakespeare", IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic1, Name: "Marlowe", Index: basic1Index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic1, basic1Index},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basic1Index, basic1},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
								telem.NewSeriesV[int64](1, 2, 3, 4),
							}),
						))
						end := MustSucceed(w.Commit())
						Expect(w.Close()).To(Succeed())

						Expect(end).To(Equal(13*telem.SecondTS + 1))

						By("Reading the data back")
						frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1))
						Expect(frame.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
						tsFrame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic1Index))
						Expect(tsFrame.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
					})

					Context("Disjoint Domain Alignment", func() {
						It("Should keep streaming alignment values consistent even when the index has more domains than the data channel", func() {
							var (
								basic1      = GenerateChannelKey()
								basic1Index = GenerateChannelKey()
							)

							By("Creating an index channel")
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{Key: basic1Index, Name: "Orwell", IsIndex: true, DataType: telem.TimeStampT},
							)).To(Succeed())

							By("Writing to the Index Channel")
							w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels: []cesium.ChannelKey{basic1Index},
								Start:    10 * telem.SecondTS,
							}))
							MustSucceed(w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{basic1Index},
								[]telem.Series{
									telem.NewSecondsTSV(10, 11, 12, 13),
								},
							)))
							end := MustSucceed(w.Commit())
							Expect(w.Close()).To(Succeed())
							Expect(end).To(Equal(13*telem.SecondTS + 1))
							// Sleep for 20 ms and schedule to allow the current
							// frame to be processed by the relay, ensuring we don't
							// read the first written value out of the streamer.
							runtime.Gosched()
							time.Sleep(20 * time.Millisecond)

							By("Creating a data channel")
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{Key: basic1, Name: "Huxley", Index: basic1Index, DataType: telem.Int64T},
							)).To(Succeed())
							w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels: []cesium.ChannelKey{basic1, basic1Index},
								Start:    14 * telem.SecondTS,
							}))
							s := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
								Channels: []cesium.ChannelKey{basic1, basic1Index},
							}))
							i, o := confluence.Attach(s, 1)
							sCtx, cancel := signal.WithCancel(ctx)
							defer cancel()
							s.Flow(sCtx)
							MustSucceed(w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{basic1, basic1Index},
								[]telem.Series{
									telem.NewSeriesV[int64](1, 2),
									telem.NewSecondsTSV(14, 15),
								},
							)))
							f := <-o.Outlet()
							Expect(f.Frame.Series).To(HaveLen(2))
							basic1Alignment := f.Frame.Series[0].Alignment
							basicIndex1Alignment := f.Frame.Series[1].Alignment
							Expect(basicIndex1Alignment).To(Equal(basic1Alignment))
							i.Close()
							Expect(sCtx.Wait()).To(Succeed())
							Expect(w.Close()).To(Succeed())
						})

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
							cesium.Channel{Key: basicIdx1, Name: "Hemingway", IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic1, Name: "Fitzgerald", Index: basicIdx1, DataType: telem.Int64T},
							cesium.Channel{Key: basicIdx2, Name: "Steinbeck", IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: basic2, Name: "Faulkner", Index: basicIdx2, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{basic1, basicIdx1, basic2, basicIdx2},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the first index")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basicIdx1, basic1},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
								telem.NewSeriesV[int64](1, 2, 3, 4),
							},
						)))

						By("Writing more data to the second index")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{basicIdx2, basic2},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14),
								telem.NewSeriesV[int64](1, 2, 3, 4, 5),
							},
						)))
						end := MustSucceed(w.Commit())
						Expect(end).To(Equal(14*telem.SecondTS + 1))
						Expect(w.Close()).To(Succeed())
					})
				})

				Context("Index and Data", func() {
					It("Should write properly", func() {
						var (
							index1 = GenerateChannelKey()
							data1  = GenerateChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "Bruce", Key: index1, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Steen", Key: data1, DataType: telem.Int64T, Index: index1},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{index1, data1},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index1, data1},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 13),
								telem.NewSeriesV[int64](10, 11, 13),
							},
						)))
						end := MustSucceed(w.Commit())
						Expect(end).To(Equal(13*telem.SecondTS + +1*telem.NanosecondTS))

						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index1, data1},
							[]telem.Series{
								telem.NewSecondsTSV(17),
								telem.NewSeriesV[int64](17),
							},
						)))
						end = MustSucceed(w.Commit())
						Expect(end).To(Equal(17*telem.SecondTS + 1))
						Expect(w.Close()).To(Succeed())

						By("Checking that the data is correct")
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index1, data1))
						Expect(f.Series[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 13, 17).Data))
						Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](10, 11, 13, 17).Data))
					})
					It("Should not write an empty frame", func() {

						var (
							idx  = GenerateChannelKey()
							data = GenerateChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "Bird", Key: idx, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "Live Studio Session", Key: data, Index: idx, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{idx, data},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{idx, data},
							[]telem.Series{
								{DataType: "int64"},
								{DataType: "int64"},
							},
						)))
						end := MustSucceed(w.Commit())
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
								cesium.Channel{Name: "Innerbloom", Key: index1, IsIndex: true, DataType: telem.TimeStampT},
								cesium.Channel{Name: "Lane", Key: basic1, Index: index1, DataType: telem.Int64T},
								cesium.Channel{Name: "Eight", Key: index2, IsIndex: true, DataType: telem.TimeStampT},
								cesium.Channel{Name: "Remix", Key: basic2, Index: index2, DataType: telem.Int64T},
								cesium.Channel{Name: "Yup", Key: basic3, Index: index2, DataType: telem.Uint32T},
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
							MustSucceed(w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(10, 12, 13, 14),
									telem.NewSeriesV[int64](100, 102, 103, 104),
									telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
									telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105),
								},
							)))

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
							MustSucceed(w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(20, 22, 23, 24),
									telem.NewSeriesV[int64](200, 202, 203, 204),
									telem.NewSecondsTSV(20, 21, 22, 23, 24, 25),
									telem.NewSeriesV[int64](200, 201, 202, 203, 204, 205),
									telem.NewSeriesV[uint32](200, 201, 202, 203, 204, 205),
								},
							)))

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
							MustSucceed(w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(10, 12, 13, 14),
									telem.NewSeriesV[int64](100, 102, 103, 104),
									telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
									telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
									telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105),
								},
							)))
							Expect(w.Close()).To(Succeed())

							By("Writing telemetry that would collide with previous domains")
							w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels:         []cesium.ChannelKey{index1, basic1},
								Start:            9 * telem.SecondTS,
								Mode:             cesium.WriterPersistStream,
								EnableAutoCommit: config.True(),
							}))
							MustSucceed(w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1},
								[]telem.Series{
									telem.NewSecondsTSV(9, 10, 11),
									telem.NewSeriesV[int64](99, 100, 101),
								},
							)))

							By("Checking that more writes to the writer would fail")
							Eventually(func() error {
								_, err := w.Write(cesium.Frame{})
								return err
							}).Should(HaveOccurredAs(validate.Error))

							By("Checking that the first commit did not succeed")
							f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index1, basic1, index2, basic2, basic3))

							Expect(f.Get(index1)[0].Data).To(Equal(telem.NewSecondsTSV(10, 12, 13, 14).Data))
							Expect(f.Get(basic1)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 102, 103, 104).Data))
							Expect(f.Get(index2)[0].Data).To(Equal(telem.NewSecondsTSV(10, 11, 12, 13, 14, 15).Data))
							Expect(f.Get(basic2)[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105).Data))
							Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[uint32](100, 101, 102, 103, 104, 105).Data))

							By("Closing the writer")
							err := w.Close()
							resultMatcher := ContainSubstring("overlaps with existing data occupying time range %v", (10 * telem.SecondTS).Range(14*telem.SecondTS+1))
							Expect(err).To(MatchError(validate.Error))
							Expect(err).To(MatchError(resultMatcher))
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
								MustSucceed(w.Write(cesium.NewFrame([]cesium.ChannelKey{index1},
									[]telem.Series{
										telem.NewSeries(stamps),
									},
								)))
								Expect(w.Close()).To(Succeed())

								w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels:         []cesium.ChannelKey{basic1},
									Start:            start,
									Mode:             cesium.WriterPersistOnly,
									EnableAutoCommit: config.True(),
								}))
								MustSucceed(w.Write(cesium.NewFrame([]cesium.ChannelKey{basic1},
									[]telem.Series{
										telem.NewSeries(data),
									},
								)))
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
								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(10, 12),
										telem.NewSeriesV[int64](100, 102),
									},
								)))

								By("Closing the writer")
								Expect(w.Close()).To(Succeed())

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
								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(10, 11, 13),
										telem.NewSeriesV[int64](100, 101, 103),
									},
								)))

								By("Checking that this telemetry is not persisted")
								s := MustSucceed(fs.Stat(channelKeyToPath(index1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))
								s = MustSucceed(fs.Stat(channelKeyToPath(basic1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))

								By("Sleeping to wait for the threshold to be met")
								time.Sleep(time.Duration(1000 * telem.Millisecond))

								By("Writing more telemetry")
								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(20, 22, 23, 24),
										telem.NewSeriesV[int64](200, 202, 203, 204),
									},
								)))

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
								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(30, 31, 33),
										telem.NewSeriesV[int64](300, 301, 303),
									},
								)))

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
								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(10, 11, 13),
										telem.NewSeriesV[int64](100, 101, 103),
									},
								)))

								By("Checking that this telemetry is not persisted")
								s := MustSucceed(fs.Stat(channelKeyToPath(index1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))
								s = MustSucceed(fs.Stat(channelKeyToPath(basic1) + "/index.domain"))
								Expect(s.Size()).To(Equal(int64(0)))

								By("Sleeping to wait for the threshold to be met")
								time.Sleep(time.Duration(200 * telem.Millisecond))

								By("Writing more telemetry")
								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(20, 22, 23, 24),
										telem.NewSeriesV[int64](200, 202, 203, 204),
									},
								)))

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
								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(30, 31, 33),
										telem.NewSeriesV[int64](300, 301, 303),
									},
								)))

								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(40, 41),
										telem.NewSeriesV[int64](400, 401),
									},
								)))

								time.Sleep(time.Duration(200 * telem.Millisecond))

								MustSucceed(w.Write(cesium.NewFrame(
									[]cesium.ChannelKey{index1, basic1},
									[]telem.Series{
										telem.NewSecondsTSV(43),
										telem.NewSeriesV[int64](403),
									},
								)))

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
						db2          *cesium.DB
						index, basic cesium.ChannelKey
					)

					BeforeEach(func() {
						index = GenerateChannelKey()
						basic = GenerateChannelKey()
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
							cesium.Channel{Name: "Massane", Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "Waiting", Key: basic, Index: index, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:         []cesium.ChannelKey{index, basic},
							Start:            10 * telem.SecondTS,
							EnableAutoCommit: config.True(),
						}))

						By("Writing data to the channel")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
							},
						)))

						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26),
								telem.NewSeriesV[int64](200, 201, 202, 203, 204, 205, 206),
							},
						)))

						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(30, 31, 33),
								telem.NewSeriesV[int64](300, 301, 303),
							},
						)))
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
						})

						By("Asserting that the data is correct", func() {
							f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax, index, basic))
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
							cesium.Channel{Name: "A", Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "Collection", Key: basic, Index: index, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:                 []cesium.ChannelKey{index, basic},
							Start:                    10 * telem.SecondTS,
							EnableAutoCommit:         config.True(),
							AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
						}))

						By("Writing data to the channel")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 106),
							},
						)))
						Expect(w.Close()).To(Succeed())

						w = MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:                 []cesium.ChannelKey{index, basic},
							Start:                    17 * telem.SecondTS,
							EnableAutoCommit:         config.True(),
							AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
						}))

						// This should still go to file 1.
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(17),
								telem.NewSeriesV[int64](107),
							},
						)))

						// This should still go to file 1.
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(18, 19),
								telem.NewSeriesV[int64](108, 109),
							},
						)))

						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(20, 21, 22, 23, 24),
								telem.NewSeriesV[int64](200, 201, 202, 203, 204),
							},
						)))
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
						})

						By("Closing an reopening the db")
						Expect(db2.Close()).To(Succeed())

						db2 = MustSucceed(cesium.Open("size-capped-db",
							cesium.WithFS(fs),
							cesium.WithFileSize(64*telem.ByteSize),
							cesium.WithInstrumentation(PanicLogger()),
						))

						By("Asserting that upon writing to the channels, the writes go to appropriate files")
						w = MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:                 []cesium.ChannelKey{index, basic},
							Start:                    30 * telem.SecondTS,
							EnableAutoCommit:         config.True(),
							AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
						}))

						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(30, 31),
								telem.NewSeriesV[int64](300, 301),
							},
						)))
						Expect(w.Close()).To(Succeed())

						Expect(MustSucceed(subFS.Stat(strconv.Itoa(int(basic)) + "/2.domain")).Size()).To(Equal(int64(7 * telem.Int64T.Density())))
						Expect(MustSucceed(subFS.Stat(strconv.Itoa(int(index)) + "/2.domain")).Size()).To(Equal(int64(7 * telem.TimeStampT.Density())))

						By("Asserting that the data is correct")
						f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax, index, basic))
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
					})

					Specify("Without AutoCommit", func() {
						db2 = MustSucceed(cesium.Open("size-capped-db",
							cesium.WithFS(fs),
							cesium.WithFileSize(40*telem.ByteSize),
							cesium.WithInstrumentation(PanicLogger()),
						))

						Expect(db2.CreateChannel(
							ctx,
							cesium.Channel{Name: "An Odd", Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "Collection", Key: basic, Index: index, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels: []cesium.ChannelKey{index, basic},
							Start:    10 * telem.SecondTS,
						}))

						By("Writing data to the channel")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105),
							},
						)))

						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26),
								telem.NewSeriesV[int64](200, 201, 202, 203, 204, 205, 206),
							},
						)))

						MustSucceed(w.Commit())

						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(30, 31, 33),
								telem.NewSeriesV[int64](300, 301, 303),
							},
						)))
						MustSucceed(w.Commit())
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
						})

						By("Asserting that the data is correct", func() {
							f := MustSucceed(db2.Read(ctx, telem.TimeRangeMax, index, basic))
							indexF := f.Get(index)
							Expect(indexF).To(HaveLen(2))
							Expect(indexF[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(26*telem.SecondTS + 1)))
							Expect(indexF[1].TimeRange).To(Equal((26*telem.SecondTS + 1).Range(33*telem.SecondTS + 1)))

							Expect(f.Get(basic)).To(HaveLen(2))
							Expect(f.Get(basic)[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(26*telem.SecondTS + 1)))
							Expect(f.Get(basic)[1].TimeRange).To(Equal((26*telem.SecondTS + 1).Range(33*telem.SecondTS + 1)))
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
							cesium.Channel{Name: "O", Key: index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "P", Key: basic, Index: index, DataType: telem.Int64T},
							cesium.Channel{Name: "C", Key: index2, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "U", Key: basic2, Index: index2, DataType: telem.Int64T},
						)).To(Succeed())

						w := MustSucceed(db2.OpenWriter(ctx, cesium.WriterConfig{
							Channels:                 []core.ChannelKey{index, basic, index2, basic2},
							Start:                    10 * telem.SecondTS,
							EnableAutoCommit:         config.True(),
							AutoIndexPersistInterval: cesium.AlwaysIndexPersistOnAutoCommit,
						}))

						MustSucceed(w.Write(cesium.NewFrame(
							[]core.ChannelKey{index, basic, index2, basic2},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15),
								telem.NewSecondsTSV(10, 11, 12, 13, 14, 15),
								telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15),
							},
						)))

						By("Asserting that only writing to two channels will not fail")
						MustSucceed(w.Write(cesium.NewFrame(
							[]core.ChannelKey{index, basic},
							[]telem.Series{
								telem.NewSecondsTSV(16, 17, 18),
								telem.NewSeriesV[int64](16, 17, 18),
							},
						)))
						Expect(w.Close()).To(Succeed())
					})
				})

				Describe("Write Authority", func() {
					It("Should set the authority of writers", func() {
						var (
							key  = GenerateChannelKey()
							key2 = GenerateChannelKey()
						)
						By("Creating a channel")
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{
								Key:      key,
								Name:     "John",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							cesium.Channel{
								Key:      key2,
								Name:     "Krakauer",
								Virtual:  true,
								DataType: telem.StringT,
							},
						)).To(Succeed())
						w1 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{key, key2},
							Start:       10 * telem.SecondTS,
							Authorities: []control.Authority{control.Authority(100), control.Authority(110)},
							Sync:        config.True(),
						}))

						w2 := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{key, key2},
							Start:       10 * telem.SecondTS,
							Authorities: []control.Authority{control.Authority(110), control.Authority(100)},
							Sync:        config.True(),
						}))

						authorized := MustSucceed(w1.Write(cesium.NewFrame(
							[]cesium.ChannelKey{key},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
							}),
						))
						Expect(authorized).To(BeFalse())

						Expect(w1.SetAuthority(cesium.WriterConfig{
							Channels:    []cesium.ChannelKey{key, key2},
							Authorities: []control.Authority{control.Absolute, control.Authority(0)},
						})).To(Succeed())

						authorized = MustSucceed(w2.Write(cesium.NewFrame(
							[]cesium.ChannelKey{key},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
							},
						)))
						Expect(authorized).To(BeFalse())

						authorized = MustSucceed(w1.Write(cesium.NewFrame(
							[]cesium.ChannelKey{key},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
							},
						)))
						Expect(authorized).To(BeTrue())

						authorized = MustSucceed(w1.Write(cesium.NewFrame(
							[]cesium.ChannelKey{key2},
							[]telem.Series{
								{DataType: telem.StringT, Data: []byte("hehe")},
							},
						)))
						Expect(authorized).To(BeFalse())
						Expect(w1.Close()).To(Succeed())
						Expect(w2.Close()).To(Succeed())
					})
				})
			})

			Describe("Stream Only Mode", func() {
				ShouldNotLeakRoutinesJustBeforeEach()
				It("Should not persist data", func() {
					var (
						basic1      = GenerateChannelKey()
						basic1Index = GenerateChannelKey()
					)
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Name: "U", Key: basic1Index, IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Name: "A", Key: basic1, Index: basic1Index, DataType: telem.Int64T},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic1, basic1Index},
						Start:    10 * telem.SecondTS,
						Mode:     cesium.WriterStreamOnly,
					}))

					By("Writing data to the channel")
					MustSucceed(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{basic1Index, basic1},
						[]telem.Series{
							telem.NewSecondsTSV(10, 11, 12, 13),
							telem.NewSeriesV[int64](1, 2, 3, 4),
						}),
					))
					end := MustSucceed(w.Commit())
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
				ShouldNotLeakRoutinesJustBeforeEach()
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
					idx := GenerateChannelKey()
					data := GenerateChannelKey()

					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "nonexistent 1", DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Key: data, Name: "nonexistent 2", DataType: telem.Int64T, Index: idx},
					)).To(Succeed())

					_, err := db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{idx, 88888}, Start: 10 * telem.SecondTS})
					Expect(err).To(MatchError(core.ErrChannelNotFound))
				})
			})

			Describe("Frame Errors", Ordered, func() {
				ShouldNotLeakRoutinesJustBeforeEach()
				var (
					idx  = GenerateChannelKey()
					data = GenerateChannelKey()
				)
				BeforeAll(func() {
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: idx, Name: "uneven 1", DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Key: data, Name: "uneven 2", DataType: telem.Float32T, Index: idx},
					))
				})
				Specify("Uneven Frame", func() {
					w := MustSucceed(
						db.OpenWriter(ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{idx, data},
								Start:    10 * telem.SecondTS,
							}),
					)
					MustSucceed(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{idx, data},
						[]telem.Series{
							telem.NewSecondsTSV(10, 11, 12, 13),
							telem.NewSeriesV[float32](10, 11, 12),
						}),
					))
					_, err := w.Commit()
					Expect(err).To(HaveOccurredAs(validate.Error))
					err = w.Close()
					Expect(err).To(MatchError(validate.Error))
					Expect(err).To(MatchError(ContainSubstring("same length")))
				})
				Specify("Frame Without All Channels", func() {
					w := MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{idx, data},
							Start:    10 * telem.SecondTS,
						}))
					MustSucceed(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{idx},
						[]telem.Series{
							telem.NewSecondsTSV(10, 11, 12, 13),
						},
					)))
					_, err := w.Commit()
					Expect(err).To(HaveOccurred())
					err = w.Close()
					Expect(err).To(MatchError(validate.Error))
					Expect(err).To(MatchError(ContainSubstring("exactly one")))
				})
				Specify("Frame with Duplicate Channels", func() {
					w := MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{idx, data},
							Start:    10 * telem.SecondTS,
						}))
					MustSucceed(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{idx, idx},
						[]telem.Series{
							telem.NewSecondsTSV(10, 11, 12, 13),
							telem.NewSecondsTSV(14, 15, 16, 17),
						},
					)))
					_, err := w.Commit()
					Expect(err).To(HaveOccurred())
					err = w.Close()
					Expect(err).To(MatchError(validate.Error))
					Expect(err.Error()).To(ContainSubstring("duplicate channel"))
				})
			})

			Describe("Index Errors", func() {
				ShouldNotLeakRoutinesJustBeforeEach()
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
							cesium.Channel{Name: "B", Key: disc1Index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "D", Key: disc1, Index: disc1Index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(
							ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{disc1Index},
								Start:    10 * telem.SecondTS,
							}))

						By("Writing data to the index correctly")
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{disc1Index},
							[]telem.Series{
								telem.NewSecondsTSV(10, 11, 12, 13),
							}),
						))
						MustSucceed(w.Commit())
						Expect(w.Close()).To(Succeed())

						By("Writing data to channel where the last sample is not the index")
						w = MustSucceed(db.OpenWriter(
							ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{disc1},
								Start:    10 * telem.SecondTS,
							}))
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{disc1},
							[]telem.Series{
								telem.NewSeriesV[int64](1, 2, 3, 4, 5),
							},
						)))
						_, err := w.Commit()
						Expect(err).To(HaveOccurred())
						err = w.Close()
						Expect(err).To(MatchError(index.ErrDiscontinuous))
					})
					Specify("Index not defined at all", func() {
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "Gregory", Key: disc2Index, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Name: "Alan", Key: disc2, Index: disc2Index, DataType: telem.Int64T},
						)).To(Succeed())
						w := MustSucceed(db.OpenWriter(
							ctx,
							cesium.WriterConfig{
								Channels: []cesium.ChannelKey{disc2},
								Start:    10 * telem.SecondTS,
							}))
						MustSucceed(w.Write(cesium.NewFrame(
							[]cesium.ChannelKey{disc2},
							[]telem.Series{
								telem.NewSeriesV[int64](1, 2, 3, 4, 5),
							},
						)))
						_, err := w.Commit()
						Expect(err).To(HaveOccurred())
						Expect(w.Close()).To(MatchError(ContainSubstring("cannot find stamp start")))
					})
				})
			})

			Describe("Data Type Errors", func() {
				ShouldNotLeakRoutinesJustBeforeEach()
				Specify("Invalid Data Type for series", func() {
					var dtErr = GenerateChannelKey()
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{
							Name:     "Isakov",
							Key:      dtErr,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						})).To(Succeed())
					w := MustSucceed(db.OpenWriter(
						ctx,
						cesium.WriterConfig{
							Channels: []cesium.ChannelKey{dtErr},
							Start:    10 * telem.SecondTS,
						}))
					MustSucceed(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{dtErr},
						[]telem.Series{
							telem.NewSeriesV[uint16](1, 2, 3, 4, 5),
						},
					)))

					MustSucceed(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{dtErr},
						[]telem.Series{telem.NewSeriesV[int64](10, 11, 12, 13)},
					)))
					_, err := w.Commit()
					Expect(err).To(MatchError(validate.Error))
					Expect(w.Close()).To(MatchError(validate.Error))
				})
			})

			Describe("ErrOnUnauthorized", func() {
				ShouldNotLeakRoutinesJustBeforeEach()
				var (
					key        cesium.ChannelKey
					controlKey = GenerateChannelKey()
					w1         *cesium.Writer
					w2         *cesium.Writer
				)
				BeforeAll(func() {
					Expect(db.ConfigureControlUpdateChannel(ctx, controlKey, "sy_cesium_control")).To(Succeed())
				})
				BeforeEach(func() {
					key = GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Name: "We", Key: key, DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					w1 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{key}, Start: 1 * telem.SecondTS}))
				})
				AfterEach(func() {
					Expect(w1.Close()).To(Succeed())
				})
				Context("False", func() {
					It("Should not return an error if writer is not authorized to write", func() {
						w2 = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{key}, Start: 1 * telem.SecondTS}))
						Consistently(func() error {
							_, err := w2.Write(cesium.NewFrame([]cesium.ChannelKey{key}, []telem.Series{telem.NewSeriesV[int64](1, 2, 3, 4)}))
							return err
						}).Should(Succeed())
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
				ShouldNotLeakRoutinesJustBeforeEach()
				It("Should write to virtual channel", func() {
					var virtual1 = GenerateChannelKey()
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Name: "Knew", Key: virtual1, DataType: telem.Int64T, Virtual: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{virtual1},
						Start:    10 * telem.SecondTS,
					}))

					MustSucceed(w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{virtual1},
						[]telem.Series{telem.NewSeriesV[int64](1, 2, 3)},
					)))

					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Close", func() {
				ShouldNotLeakRoutinesJustBeforeEach()
				It("Should not allow operations on a closed writer", func() {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Name: "Close 1", DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					var (
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{key}, Start: 10 * telem.SecondTS}))
						e = core.EntityClosed("cesium.writer")
					)
					Expect(w.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
					_, err := w.Write(cesium.Frame{Series: []telem.Series{{Data: []byte{1, 2, 3}}}})
					Expect(err).To(HaveOccurred())
					_, err = w.Commit()
					Expect(err).To(HaveOccurredAs(e))
				})
			})

			Describe("Close", func() {

				It("Should close properly with a control setup", func() {
					k2, k3, k4 := GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k2, Name: "Close 2", DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k3, Name: "Close 3", DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k4, Name: "Close 4", DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{k2, k3, k4}, Start: 10 * telem.SecondTS}))
					Expect(w.Close()).To(Succeed())
				})

				It("Should not allow operations on a closed writer", func() {
					k := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k, Name: "Close 5", DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					var (
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{k}, Start: 10 * telem.SecondTS}))
						e = core.EntityClosed("cesium.writer")
					)
					Expect(w.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
					_, err := w.Write(cesium.Frame{Series: []telem.Series{{Data: []byte{1, 2, 3}}}})
					Expect(err).To(HaveOccurred())
					_, err = w.Commit()
					Expect(err).To(HaveOccurredAs(e))
				})

				It("Should not allow opening an iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Name: "It", Key: key, DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.OpenWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow opening a stream iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Name: "Our", Key: key, DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.NewStreamWriter(ctx, cesium.WriterConfig{Start: 0, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))
					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow writing from a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Name: "Was", Key: key, DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())
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
