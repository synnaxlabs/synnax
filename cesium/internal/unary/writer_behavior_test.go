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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"os"
)

var _ = Describe("Writer Behavior", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		fs, cleanUp := makeFS()
		Context("FS: "+fsName, func() {
			AfterAll(func() {
				Expect(cleanUp()).To(Succeed())
			})
			Describe("Index", func() {
				var db *unary.DB
				BeforeEach(func() {
					db = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub("/writer_test/happy")),
						Channel: core.Channel{
							Key:      2,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
				})
				Specify("Happy Path", func() {
					w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   telem.TimeStamp(0),
						Subject: control.Subject{Key: "foo"},
					}))
					Expect(t.Occurred()).To(BeTrue())
					Expect(MustSucceed(w.Write(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.LeadingAlignment(0)))
					Expect(MustSucceed(w.Write(telem.NewSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(telem.LeadingAlignment(6)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(11*telem.SecondTS + 1))
					t = MustSucceed(w.Close(ctx))
					Expect(t.Occurred()).To(BeTrue())
					Expect(db.LeadingControlState()).To(BeNil())
				})
				Describe("Open", func() {
					It("Should not allow opening a writer if the specified end timestamp is before the start", func() {
						_, _, err := db.OpenWriter(ctx, unary.WriterConfig{Start: 10, End: 3, Subject: control.Subject{Key: "foo"}})
						Expect(err).To(MatchError(ContainSubstring("end timestamp must be after")))
					})

					It("Should not allow opening a writer without a subject key", func() {
						_, _, err := db.OpenWriter(ctx, unary.WriterConfig{Start: 3, End: 10})
						Expect(err).To(MatchError(ContainSubstring("Subject.Key:field must be set")))
					})
				})
			})
			Describe("Channel Indexed", func() {
				var (
					db        *unary.DB
					indexDB   *unary.DB
					index     uint32 = 1
					data      uint32 = 2
					indexPath        = "/writer_test/index"
					dataPath         = "/writer_test/data"
				)
				BeforeEach(func() {
					indexDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(indexPath)),
						Channel: core.Channel{
							Key:      index,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						FileSize: telem.Size(10*telem.TimeStampT.Density()) * telem.ByteSize,
					}))
					db = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(dataPath)),
						Channel: core.Channel{
							Key:      data,
							DataType: telem.Int64T,
							Index:    index,
						},
						FileSize: telem.Size(5*telem.TimeStampT.Density()) * telem.ByteSize,
					}))
					db.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(indexDB.Close()).To(Succeed())
					Expect(fs.Remove(dataPath)).To(Succeed())
					Expect(fs.Remove(indexPath)).To(Succeed())
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
					Expect(MustSucceed(w.Write(telem.NewSeries([]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})))).To(Equal(telem.LeadingAlignment(0)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(20*telem.SecondTS + 1))
					t = MustSucceed(w.Close(ctx))
					Expect(t.Occurred()).To(BeTrue())
					By("Releasing control of the DB")
					Expect(db.LeadingControlState()).To(BeNil())
				})
				Describe("Auto file switch", func() {
					Specify("File cutoff on commit with no preset end", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSecondsTSV(10, 11, 13, 14, 15, 16, 17))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that both writers have still written to one file")
						l := MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						l = MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSecondsTSV(23, 29, 31, 37, 41, 43))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](203, 209, 301, 307, 401, 403))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the second writer is now writing to a different file")
						l = MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(13 * telem.Int64T.Density())))

						l = MustSucceed(db.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(7*telem.Int64T.Density()), int64(6*telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSecondsTSV(47, 53))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the first writer is now writing to a different file")
						l = MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(13*telem.Int64T.Density()), int64(2*telem.Int64T.Density())))

						Expect(w1.Commit(ctx)).To(Equal(53*telem.SecondTS + 1))

						_, err = w1.Close(ctx)
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close(ctx)
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i := db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.Series[0].Len()).To(Equal(int64(7)))
							Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(17*telem.SecondTS + 1)))
							Expect(f.Series[1].Len()).To(Equal(int64(6)))
							Expect(f.Series[1].TimeRange).To(Equal((17*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
							Expect(f.SquashSameKeyData(data)).To(Equal(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 203, 209, 301, 307, 401, 403).Data))
							Expect(i.Close()).To(Succeed())
						})
					})
					Specify("Special case where the file exceeds limit by 1", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSecondsTSV(10, 11, 13, 14, 15, 16, 17))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that both writers have still written to one file")
						l := MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						l = MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSecondsTSV(23, 29, 31, 37, 41, 43))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](203, 209, 301, 307, 401, 403))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the second writer is now writing to a different file")
						l = MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(13 * telem.Int64T.Density())))

						l = MustSucceed(db.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(7*telem.Int64T.Density()), int64(6*telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSecondsTSV(47, 53))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the first writer is now writing to a different file")
						l = MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(13*telem.Int64T.Density()), int64(2*telem.Int64T.Density())))

						Expect(w1.Commit(ctx)).To(Equal(53*telem.SecondTS + 1))

						_, err = w1.Close(ctx)
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close(ctx)
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i := db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.Series[0].Len()).To(Equal(int64(7)))
							Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(17*telem.SecondTS + 1)))
							Expect(f.Series[1].Len()).To(Equal(int64(6)))
							Expect(f.Series[1].TimeRange).To(Equal((17*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
							Expect(f.SquashSameKeyData(data)).To(Equal(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 203, 209, 301, 307, 401, 403).Data))
							Expect(i.Close()).To(Succeed())
						})
					})
					Specify("File cutoff on commit with preset end", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSecondsTSV(10, 11, 13, 14, 15, 16, 17, 18, 20, 23, 25))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 108, 200, 203, 205))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(25*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(25*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSecondsTSV(29, 31, 37, 41, 43))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](209, 301, 307, 401, 403))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

						By("Asserting that both writers are now writing to a different file")
						l := MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(11*telem.Int64T.Density()), int64(5*telem.Int64T.Density())))

						l = MustSucceed(db.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(3))
						Expect([]int64{l[0].Size(), l[1].Size(), l[2].Size()}).To(ConsistOf(int64(11*telem.Int64T.Density()), int64(5*telem.Int64T.Density()), int64(0)))

						_, err = w1.Close(ctx)
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close(ctx)
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i := db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.Series[0].Len()).To(Equal(int64(11)))
							Expect(f.Series[1].Len()).To(Equal(int64(5)))
							Expect(f.SquashSameKeyData(data)).To(Equal(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 108, 200, 203, 205, 209, 301, 307, 401, 403).Data))
							Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(25*telem.SecondTS + 1)))
							Expect(f.Series[1].TimeRange).To(Equal((25*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
							Expect(i.Close()).To(Succeed())
						})
					})

					Specify("Just enough data to switch files", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSecondsTSV(10))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(10*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(10*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSecondsTSV(11, 12, 13, 14, 15, 16, 17, 18, 19))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](101, 102, 103, 104, 105, 106, 107, 108, 109))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(19*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(19*telem.SecondTS + 1))

						By("Asserting that both writers have only written to one file")
						l := MustSucceed(indexDB.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(10*telem.Int64T.Density()), int64(0)))

						l = MustSucceed(db.FS.List(""))
						l = lo.Filter(l, func(item os.FileInfo, _ int) bool {
							return item.Name() != "counter.domain" && item.Name() != "index.domain"
						})
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(10*telem.Int64T.Density()), int64(0)))

						_, err = w1.Close(ctx)
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close(ctx)
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i := db.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.Series[0].Len()).To(Equal(int64(10)))
							Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 106, 107, 108, 109).Data))
							Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(19*telem.SecondTS + 1)))
							Expect(i.Close()).To(Succeed())
						})
					})
				})
			})
			Describe("Control", func() {
				Describe("Index", func() {
					var db *unary.DB
					BeforeEach(func() {
						db = MustSucceed(unary.Open(unary.Config{
							FS: MustSucceed(fs.Sub("/writer_test/control")),
							Channel: core.Channel{
								Key:      2,
								DataType: telem.TimeStampT,
								IsIndex:  true,
							},
						}))
					})
					AfterEach(func() {
						Expect(db.Close()).To(Succeed())
					})
					Specify("Control Handoff", func() {
						w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.Absolute - 1,
							Subject:   control.Subject{Key: "foo"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w1.Write(telem.NewSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(telem.LeadingAlignment(0)))
						w2, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.Absolute,
							Subject:   control.Subject{Key: "bar"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w2.Write(telem.NewSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(telem.LeadingAlignment(6)))
						a, err := w1.Write(telem.NewSecondsTSV(12, 13, 14, 15, 16, 17))
						Expect(err).To(MatchError(control.Unauthorized))
						Expect(a).To(Equal(telem.AlignmentPair(0)))
						_, err = w1.Commit(ctx)
						Expect(err).To(MatchError(control.Unauthorized))
						t = MustSucceed(w2.Close(ctx))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w1.Write(telem.NewSecondsTSV(12, 13, 14, 15, 16, 17)))).To(Equal(telem.LeadingAlignment(12)))
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
						FS: MustSucceed(fs.Sub("/close_test")),
						Channel: core.Channel{
							Key:      100,
							DataType: telem.TimeStampT,
						},
					}))
				})

				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
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
					_, err = w.Write(telem.Series{Data: []byte{1, 2, 3}})
					Expect(err).To(MatchError(e))
					_, err = w.Close(ctx)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	}
})
