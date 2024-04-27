// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	"encoding/binary"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"os"
)

func extractPointer(f fs.File) (p struct {
	telem.TimeRange
	fileKey uint16
	offset  uint32
	length  uint32
}) {
	var b = make([]byte, 26)
	_, err := f.ReadAt(b, 0)
	Expect(err).ToNot(HaveOccurred())
	p.TimeRange.Start = telem.TimeStamp(binary.LittleEndian.Uint64(b[0:8]))
	p.TimeRange.End = telem.TimeStamp(binary.LittleEndian.Uint64(b[8:16]))
	p.fileKey = binary.LittleEndian.Uint16(b[16:18])
	p.offset = binary.LittleEndian.Uint32(b[18:22])
	p.length = binary.LittleEndian.Uint32(b[22:26])

	return
}

var _ = Describe("Writer Behavior", func() {
	for fsName, fs := range fileSystems {
		fs := fs()
		Context("FS: "+fsName, func() {
			Describe("Index", func() {
				var db *unary.DB
				BeforeEach(func() {
					db = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(rootPath + "/writer_test/happy")),
						Channel: core.Channel{
							Key:      2,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(fs.Remove(rootPath)).To(Succeed())
				})
				Specify("Happy Path", func() {
					w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   telem.TimeStamp(0),
						Subject: control.Subject{Key: "foo"},
					}))
					Expect(t.Occurred()).To(BeTrue())
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(telem.Alignment(6)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(11*telem.SecondTS + 1))
					t = MustSucceed(w.Close(ctx))
					Expect(t.Occurred()).To(BeTrue())
					Expect(db.LeadingControlState()).To(BeNil())
				})

				Describe("Auto-Commit", func() {
					It("Should automatically commit each write with an approximate timestamp when there is no end on the writer", func() {
						By("Opening a writer")
						w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:      telem.TimeStamp(0),
							Subject:    control.Subject{Key: "foo"},
							AutoCommit: config.True(),
						}))

						By("Writing telemetry to the writer")
						Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))

						By("Checking that the telemetry has been committed")
						i := db.OpenIterator(unary.IterRange(telem.TimeRangeMax))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.Next(ctx, 10*telem.Second)).To(BeTrue())
						Expect(i.Value().Series[0].TimeRange).To(Equal((0 * telem.SecondTS).Range(5*telem.SecondTS + 1)))
						Expect(i.Value().Series[0].Data).To(Equal(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5).Data))

						By("Writing more data to the writer")
						Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(10, 11, 12, 13)))).To(Equal(telem.Alignment(6)))
						Expect(i.SeekFirst(ctx)).To(BeTrue())
						Expect(i.Next(ctx, 20*telem.Second)).To(BeTrue())
						Expect(i.Value().Series[0].TimeRange).To(Equal((0 * telem.SecondTS).Range(13*telem.SecondTS + 1)))
						Expect(i.Value().Series[0].Data).To(Equal(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5, 10, 11, 12, 13).Data))

						By("Closing resources")
						_, err := w.Close(ctx)
						Expect(err).ToNot(HaveOccurred())
						Expect(i.Close()).To(Succeed())
					})
				})

				It("Should automatically commit each write with an exact timestamp when there is an end on the writer", func() {
					By("Opening a writer")
					w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:      telem.TimeStamp(0),
						End:        20 * telem.SecondTS,
						Subject:    control.Subject{Key: "foo"},
						AutoCommit: config.True(),
					}))

					By("Writing telemetry to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))

					By("Checking that the telemetry has been committed")
					i := db.OpenIterator(unary.IterRange(telem.TimeRangeMax))
					Expect(i.SeekFirst(ctx)).To(BeTrue())
					Expect(i.Next(ctx, 50*telem.Second)).To(BeTrue())
					Expect(i.Value().Series[0].TimeRange).To(Equal((0 * telem.SecondTS).Range(20 * telem.SecondTS)))
					Expect(i.Value().Series[0].Data).To(Equal(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5).Data))

					By("Writing more data to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(10, 11, 12, 13)))).To(Equal(telem.Alignment(6)))
					Expect(i.SeekFirst(ctx)).To(BeTrue())
					Expect(i.Next(ctx, 50*telem.Second)).To(BeTrue())
					Expect(i.Value().Series[0].TimeRange).To(Equal((0 * telem.SecondTS).Range(20 * telem.SecondTS)))
					Expect(i.Value().Series[0].Data).To(Equal(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5, 10, 11, 12, 13).Data))

					By("Writing data beyond the end")
					_, err := w.Write(ctx, telem.NewSecondsTSV(18, 19, 20, 21, 22))
					Expect(err).To(MatchError(ContainSubstring("commit timestamp cannot be greater than preset end")))

					By("Closing resources")
					_, err = w.Close(ctx)
					Expect(err).ToNot(HaveOccurred())
					Expect(i.Close()).To(Succeed())
				})

				It("Should automatically commit and persist the index each time when the threshold is not set", func() {
					By("Opening a writer")
					w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:      telem.TimeStamp(0),
						Subject:    control.Subject{Key: "foo"},
						AutoCommit: config.True(),
					}))

					By("Writing telemetry to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))

					By("Checking that the telemetry has been persisted")
					f := MustSucceed(db.FS.Open("index.domain", os.O_RDONLY))
					p := extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.Start).To(Equal(0 * telem.SecondTS))
					Expect(p.End).To(Equal(5*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(db.Channel.DataType.Density().Size(6))))

					By("Writing more data to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(10, 11, 12, 13)))).To(Equal(telem.Alignment(6)))
					f = MustSucceed(db.FS.Open("index.domain", os.O_RDONLY))
					p = extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.Start).To(Equal(0 * telem.SecondTS))
					Expect(p.End).To(Equal(13*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(db.Channel.DataType.Density().Size(10))))

					By("Closing resources")
					_, err := w.Close(ctx)
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should automatically commit and persist the index each time the threshold is reached", func() {
					By("Opening a writer")
					w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:                telem.TimeStamp(0),
						Subject:              control.Subject{Key: "foo"},
						AutoCommit:           config.True(),
						AutoPersistThreshold: int64(8),
					}))

					By("Writing telemetry to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))

					By("Checking that the telemetry has not been persisted")
					fi := MustSucceed(db.FS.Stat("index.domain"))
					Expect(fi.Size()).To(Equal(int64(0)))

					By("Writing more telemetry to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(8)))).To(Equal(telem.Alignment(6)))

					By("Checking that the telemetry has not been persisted")
					fi = MustSucceed(db.FS.Stat("index.domain"))
					Expect(fi.Size()).To(Equal(int64(0)))

					By("Writing more telemetry to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(10, 13, 15, 16)))).To(Equal(telem.Alignment(7)))

					By("Checking that the telemetry has been persisted")
					f := MustSucceed(db.FS.Open("index.domain", os.O_RDONLY))
					p := extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.Start).To(Equal(0 * telem.SecondTS))
					Expect(p.End).To(Equal(16*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(db.Channel.DataType.Density().Size(11))))

					By("Writing more data to the writer")
					Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(20, 21, 22, 23)))).To(Equal(telem.Alignment(11)))

					By("Checking that the telemetry has not been persisted")
					f = MustSucceed(db.FS.Open("index.domain", os.O_RDONLY))
					p = extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.Start).To(Equal(0 * telem.SecondTS))
					Expect(p.End).To(Equal(16*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(db.Channel.DataType.Density().Size(11))))

					By("Closing resources")
					_, err := w.Close(ctx)
					Expect(err).ToNot(HaveOccurred())

					By("Checking that the stranded data have been committed")
					f = MustSucceed(db.FS.Open("index.domain", os.O_RDONLY))
					p = extractPointer(f)
					Expect(f.Close()).To(Succeed())
					Expect(p.Start).To(Equal(0 * telem.SecondTS))
					Expect(p.End).To(Equal(23*telem.SecondTS + 1))
					Expect(p.length).To(Equal(uint32(db.Channel.DataType.Density().Size(15))))
				})

				Context("Commit stranded data on close", func() {
					It("Should be successful", func() {
						By("Opening a writer")
						w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:                telem.TimeStamp(0),
							Subject:              control.Subject{Key: "foo"},
							AutoCommit:           config.True(),
							AutoPersistThreshold: int64(8),
						}))

						By("Writing telemetry to the writer")
						Expect(MustSucceed(w.Write(ctx, telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))

						By("Checking that the telemetry has not been persisted")
						fi := MustSucceed(db.FS.Stat("index.domain"))
						Expect(fi.Size()).To(Equal(int64(0)))

						By("Closing resources")
						_, err := w.Close(ctx)
						Expect(err).ToNot(HaveOccurred())

						By("Checking that the stranded data have been persisted")
						f := MustSucceed(db.FS.Open("index.domain", os.O_RDONLY))
						p := extractPointer(f)
						Expect(f.Close()).To(Succeed())
						Expect(p.Start).To(Equal(0 * telem.SecondTS))
						Expect(p.End).To(Equal(5*telem.SecondTS + 1))
						Expect(p.length).To(Equal(uint32(db.Channel.DataType.Density().Size(6))))
					})
				})

			})
			Describe("Channel Indexed", func() {
				var (
					db        *unary.DB
					indexDB   *unary.DB
					index     uint32 = 1
					data      uint32 = 2
					indexPath        = rootPath + "/writer_test/index"
					dataPath         = rootPath + "/writer_test/data"
				)
				BeforeEach(func() {
					indexDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(indexPath)),
						Channel: core.Channel{
							Key:      index,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
					db = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(dataPath)),
						Channel: core.Channel{
							Key:      data,
							DataType: telem.Int64T,
							Index:    index,
						},
					}))
					db.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(indexDB.Close()).To(Succeed())
					Expect(fs.Remove(rootPath)).To(Succeed())
				})
				Specify("Happy Path", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20))).To(Succeed())
					w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					}))
					By("Taking control of the DB")
					Expect(db.LeadingControlState().Subject).To(Equal(control.Subject{Key: "foo"}))
					Expect(t.Occurred()).To(BeTrue())
					Expect(MustSucceed(w.Write(ctx, telem.NewSeries([]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})))).To(Equal(telem.Alignment(0)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(20*telem.SecondTS + 1))
					t = MustSucceed(w.Close(ctx))
					Expect(t.Occurred()).To(BeTrue())
					By("Releasing control of the DB")
					Expect(db.LeadingControlState()).To(BeNil())
				})
			})
			Describe("Control", func() {
				Describe("Index", func() {
					var db *unary.DB
					BeforeEach(func() {
						db = MustSucceed(unary.Open(unary.Config{
							FS: MustSucceed(fs.Sub(rootPath + "/writer_test/control")),
							Channel: core.Channel{
								Key:      2,
								DataType: telem.TimeStampT,
								IsIndex:  true,
							},
						}))
					})
					AfterEach(func() {
						Expect(db.Close()).To(Succeed())
						Expect(fs.Remove(rootPath)).To(Succeed())
					})
					Specify("Control Handoff", func() {
						w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.Absolute - 1,
							Subject:   control.Subject{Key: "foo"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w1.Write(ctx, telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.Alignment(0)))
						w2, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.Absolute,
							Subject:   control.Subject{Key: "bar"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w2.Write(ctx, telem.NewSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(telem.Alignment(6)))
						a, err := w1.Write(ctx, telem.NewSecondsTSV(12, 13, 14, 15, 16, 17))
						Expect(err).To(MatchError(control.Unauthorized))
						Expect(a).To(Equal(telem.Alignment(0)))
						_, err = w1.Commit(ctx)
						Expect(err).To(MatchError(control.Unauthorized))
						t = MustSucceed(w2.Close(ctx))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w1.Write(ctx, telem.NewSecondsTSV(12, 13, 14, 15, 16, 17)))).To(Equal(telem.Alignment(12)))
						Expect(MustSucceed(w1.Commit(ctx))).To(Equal(17*telem.SecondTS + 1))
						t = MustSucceed(w1.Close(ctx))
						Expect(t.Occurred()).To(BeTrue())
					})
				})
			})

			Describe("Close", Ordered, func() {
				var db *unary.DB
				BeforeEach(func() {
					db = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(rootPath + "/close_test")),
						Channel: core.Channel{
							Key:      100,
							DataType: telem.TimeStampT,
						},
					}))
				})

				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(fs.Remove(rootPath)).To(Succeed())
				})

				It("Should not allow operations on a closed writer", func() {
					var (
						w, t = MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:   10 * telem.SecondTS,
							Subject: control.Subject{Key: "foo"}},
						))
						e = core.EntityClosed("unary.writer")
					)
					Expect(t.Occurred()).To(BeTrue())
					_, err := w.Close(ctx)
					Expect(err).ToNot(HaveOccurred())
					_, err = w.Commit(ctx)
					Expect(err).To(MatchError(e))
					_, err = w.Write(ctx, telem.Series{Data: []byte{1, 2, 3}})
					Expect(err).To(MatchError(e))
					_, err = w.Close(ctx)
					Expect(err).To(BeNil())
				})
			})
		})
	}
})
