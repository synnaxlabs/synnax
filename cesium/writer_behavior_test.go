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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"os"
	"time"
)

var _ = Describe("Writer Behavior", func() {
	for fsName, fsMaker := range fileSystems {
		fs := fsMaker()
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
							index1 cesium.ChannelKey
							basic1 cesium.ChannelKey
							index2 cesium.ChannelKey
							basic2 cesium.ChannelKey
							basic3 cesium.ChannelKey
						)
						BeforeEach(func() {
							index1 = GenerateCesiumChannelKey()
							basic1 = GenerateCesiumChannelKey()
							index2 = GenerateCesiumChannelKey()
							basic2 = GenerateCesiumChannelKey()
							basic3 = GenerateCesiumChannelKey()

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
									telem.NewSeriesV[float64](100.00, 101.03, 102.06, 103.00, 104.00, 105.80),
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
								g.Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[float64](100.00, 101.03, 102.06, 103.00, 104.00, 105.80).Data))
							}).Should(Succeed())

							By("Writing more telemetry")
							ok = w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(20, 22, 23, 24),
									telem.NewSeriesV[int64](200, 202, 203, 204),
									telem.NewSecondsTSV(20, 21, 22, 23, 24, 25),
									telem.NewSeriesV[int64](200, 201, 202, 203, 204, 205),
									telem.NewSeriesV[float64](200.00, 201.03, 202.06, 203.00, 204.00, 205.80),
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
								g.Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[float64](100, 101.03, 102.06, 103.00, 104.00, 105.80, 200.00, 201.03, 202.06, 203.00, 204.00, 205.80).Data))
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
									telem.NewSeriesV[float64](100.00, 101.03, 102.06, 103.00, 104.00, 105.80),
								},
							))
							Expect(ok).To(BeTrue())
							Expect(w.Error()).ToNot(HaveOccurred())
							Expect(w.Close()).To(Succeed())

							By("Writing telemetry that would collide with previous domains")
							w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
								Channels:         []cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								Start:            9 * telem.SecondTS,
								Mode:             cesium.WriterPersistStream,
								EnableAutoCommit: config.True(),
							}))
							ok = w.Write(cesium.NewFrame(
								[]cesium.ChannelKey{index1, basic1, index2, basic2, basic3},
								[]telem.Series{
									telem.NewSecondsTSV(9, 10, 11),
									telem.NewSeriesV[int64](99, 100, 101),
									telem.NewSecondsTSV(9, 10, 11, 12),
									telem.NewSeriesV[int64](99, 100, 101, 102),
									telem.NewSeriesV[float64](0.99, 1, 1.01, 1.02),
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
							Expect(f.Get(basic3)[0].Data).To(Equal(telem.NewSeriesV[float64](100.00, 101.03, 102.06, 103.00, 104.00, 105.80).Data))

							By("Resolving the error to be commit error")
							err := w.Error()
							Expect(err).To(MatchError(validate.Error))
							Expect(err).To(MatchError(ContainSubstring("domain overlaps with an existing domain")))

							By("Closing the writer")
							Expect(w.Close()).To(Succeed())
						})

						Describe("Auto-Persist", func() {
							It("Should auto persist on every commit when set to always auto persist", func() {
								By("Opening a writer")
								w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
									Channels:            []cesium.ChannelKey{index1, basic1},
									Start:               10 * telem.SecondTS,
									Mode:                cesium.WriterPersistStream,
									AutoPersistInterval: cesium.AlwaysAutoPersist,
									EnableAutoCommit:    config.True(),
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
								_, err := f.Read(buf)
								Expect(err).ToNot(HaveOccurred())
								Expect(f.Close()).To(Succeed())
								Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
								Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(12*telem.SecondTS + 1)))
								Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(16)))

								f = MustSucceed(fs.Open(channelKeyToPath(basic1)+"/index.domain", os.O_RDONLY))
								buf = make([]byte, 26)
								_, err = f.Read(buf)
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
								_, err = f.Read(buf)
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
									Channels:            []cesium.ChannelKey{index1, basic1},
									Start:               10 * telem.SecondTS,
									Mode:                cesium.WriterPersistStream,
									EnableAutoCommit:    config.True(),
									AutoPersistInterval: 200 * telem.Millisecond,
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
								Expect(err).ToNot(HaveOccurred())

								By("Asserting that the telemetry has been persisted")
								Eventually(func(g Gomega) {
									f := MustSucceed(fs.Open(channelKeyToPath(index1)+"/index.domain", os.O_RDONLY))
									buf := make([]byte, 26)
									_, err = f.Read(buf)
									g.Expect(err).ToNot(HaveOccurred())
									g.Expect(f.Close()).To(Succeed())
									g.Expect(binary.LittleEndian.Uint64(buf[0:8])).To(Equal(uint64(10 * telem.SecondTS)))
									g.Expect(binary.LittleEndian.Uint64(buf[8:16])).To(Equal(uint64(43*telem.SecondTS + 1)))
									g.Expect(binary.LittleEndian.Uint32(buf[22:26])).To(Equal(uint32(13 * 8)))

									f = MustSucceed(fs.Open(channelKeyToPath(basic1)+"/index.domain", os.O_RDONLY))
									buf = make([]byte, 26)
									_, err = f.Read(buf)
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

					By("Using SetMode to change the mode to persist")
					Expect(w.SetMode(cesium.WriterPersistStream)).To(BeTrue())
					ok = w.Write(cesium.NewFrame(
						[]cesium.ChannelKey{basic1Index, basic1},
						[]telem.Series{
							telem.NewSecondsTSV(10, 11, 12, 13),
							telem.NewSeriesV[int64](1, 2, 3, 4),
						}),
					)
					Expect(ok).To(BeTrue())
					end, ok = w.Commit()
					Expect(ok).To(BeTrue())
					Expect(end).To(Equal(13*telem.SecondTS + 1))

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
					Expect(err).To(MatchError(core.ChannelNotFound))
				})
				Specify("Encounters channel that does not exist after already successfully creating some writers", func() {
					key1 := GenerateCesiumChannelKey()
					key2 := GenerateCesiumChannelKey()

					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: key1, DataType: telem.Int64T, Rate: 1 * telem.Hz},
						cesium.Channel{Key: key2, DataType: telem.Int64T, Rate: 1 * telem.Hz},
					)).To(Succeed())

					_, err := db.OpenWriter(ctx, cesium.WriterConfig{Channels: []cesium.ChannelKey{88888}, Start: 10 * telem.SecondTS})
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
					key := GenerateCesiumChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{key}, Start: 10 * telem.SecondTS}))
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

			Describe("Close", func() {
				It("Should close properly with a control setup", func() {
					k1, k2, k3, k4 := GenerateCesiumChannelKey(), GenerateCesiumChannelKey(), GenerateCesiumChannelKey(), GenerateCesiumChannelKey()
					Expect(db.ConfigureControlUpdateChannel(ctx, k1)).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k2, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k3, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k4, DataType: telem.TimeStampT, IsIndex: true})).To(Succeed())

					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{k2, k3, k4}, Start: 10 * telem.SecondTS}))
					Expect(w.Close()).To(Succeed())
				})
				It("Should not allow operations on a closed writer", func() {
					k := GenerateCesiumChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: k, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						w = MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{Channels: []core.ChannelKey{k}, Start: 10 * telem.SecondTS}))
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
