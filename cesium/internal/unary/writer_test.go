// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func filterDataFiles(l []os.FileInfo) []os.FileInfo {
	return lo.Filter(l, func(item os.FileInfo, _ int) bool {
		return item.Name() != "counter.domain" && item.Name() != "index.domain" && item.Name() != "tombstone.domain" && item.Name() != "meta.json"
	})
}

var _ = Describe("Writer Behavior", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			Describe("Index", func() {
				var (
					db      *unary.DB
					fs      fs.FS
					cleanUp func() error
				)
				BeforeEach(func() {
					fs, cleanUp = makeFS()
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        fs,
						MetaCodec: codec,
						Channel: channel.Channel{
							Name:     "Conrad",
							Key:      2,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				Specify("Happy Path", func() {
					w, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
						Start:   telem.TimeStamp(0),
						Subject: control.Subject{Key: "foo"},
					}))
					Expect(t.Occurred()).To(BeTrue())
					v, err := w.Write(telem.NewSeriesSecondsTSV(0, 1, 2, 3, 4, 5))
					Expect(err).ToNot(HaveOccurred())
					Expect(v).To(Equal(alignment.Leading(1, 0)))
					Expect(MustSucceed(w.Write(telem.NewSeriesSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(alignment.Leading(1, 6)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(11*telem.SecondTS + 1))
					t = MustSucceed(w.Close())
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
						Expect(err).To(MatchError(ContainSubstring("subject.key: required")))
					})
				})
			})
			Describe("Channel Indexed", func() {
				var (
					dataDB  *unary.DB
					dataFS  fs.FS
					indexDB *unary.DB
					indexFS fs.FS
					index   = testutil.GenerateChannelKey()
					data    = testutil.GenerateChannelKey()
					cleanUp func() error
				)
				BeforeEach(func() {
					var fs fs.FS
					fs, cleanUp = makeFS()
					indexFS = MustSucceed(fs.Sub("index"))
					indexDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        indexFS,
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      index,
							Name:     "Cayley",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						FileSize:        telem.Size(10*telem.TimeStampT.Density()) * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					dataFS = MustSucceed(fs.Sub("data"))
					dataDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        dataFS,
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      data,
							Name:     "Maxwell",
							DataType: telem.Int64T,
							Index:    index,
						},
						FileSize:        telem.Size(5*telem.Int64T.Density()) * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					dataDB.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(dataDB.Close()).To(Succeed())
					Expect(indexDB.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				Specify("Happy Path", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20))).To(Succeed())
					w, t := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					}))
					By("Taking control of the DB")
					Expect(dataDB.LeadingControlState().Subject).To(Equal(control.Subject{Key: "foo"}))
					Expect(t.Occurred()).To(BeTrue())
					Expect(MustSucceed(w.Write(telem.NewSeries([]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})))).To(Equal(alignment.Leading(1, 0)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(20*telem.SecondTS + 1))
					t = MustSucceed(w.Close())
					Expect(t.Occurred()).To(BeTrue())
					By("Releasing control of the DB")
					Expect(dataDB.LeadingControlState()).To(BeNil())
				})
				Specify("Open Writer domain overlap", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20))).To(Succeed())
					w, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					}))
					Expect(MustSucceed(w.Write(telem.NewSeries([]int64{0, 1, 2, 3, 4})))).To(Equal(alignment.Leading(1, 0)))
					Expect(MustSucceed(w.Commit(ctx))).To(Equal(14*telem.SecondTS + 1))
					_, err := w.Close()
					Expect(err).ToNot(HaveOccurred())

					_, _, err = dataDB.OpenWriter(ctx, unary.WriterConfig{
						Start:   12 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"},
					})
					Expect(err).To(MatchError(ContainSubstring("channel [Maxwell]<%d>", data)))
					Expect(err).To(MatchError(ContainSubstring("overlaps")))
				})
				Describe("Auto file switch", func() {
					Specify("File cutoff on commit with no preset end", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSeriesSecondsTSV(10, 11, 13, 14, 15, 16, 17))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that both writers have still written to one file")
						l := filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						l = filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSeriesSecondsTSV(23, 29, 31, 37, 41, 43))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](203, 209, 301, 307, 401, 403))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the second writer is now writing to a different file")
						l = filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(13 * telem.Int64T.Density())))

						l = filterDataFiles(MustSucceed(dataFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(7*telem.Int64T.Density()), int64(6*telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSeriesSecondsTSV(47, 53))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the first writer is now writing to a different file")
						l = filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(13*telem.Int64T.Density()), int64(2*telem.Int64T.Density())))

						Expect(w1.Commit(ctx)).To(Equal(53*telem.SecondTS + 1))

						_, err = w1.Close()
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close()
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i, _ := dataDB.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.SeriesAt(0)).To(telem.MatchSeriesDataV[int64](100, 101, 103, 104, 105, 106, 107))
							Expect(f.SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(17*telem.SecondTS + 1)))
							Expect(f.SeriesAt(1)).To(telem.MatchSeriesDataV[int64](203, 209, 301, 307, 401, 403))
							Expect(f.SeriesAt(1).TimeRange).To(Equal((17*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
							Expect(i.Close()).To(Succeed())
						})
					})
					Specify("Special case where the file exceeds limit by 1", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSeriesSecondsTSV(10, 11, 13, 14, 15, 16, 17))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that both writers have still written to one file")
						l := filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						l = filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(7 * telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(17*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSeriesSecondsTSV(23, 29, 31, 37, 41, 43))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](203, 209, 301, 307, 401, 403))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the second writer is now writing to a different file")
						l = filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(1))
						Expect(l[0].Size()).To(Equal(int64(13 * telem.Int64T.Density())))

						l = filterDataFiles(MustSucceed(dataFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(7*telem.Int64T.Density()), int64(6*telem.Int64T.Density())))

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSeriesSecondsTSV(47, 53))
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the first writer is now writing to a different file")
						l = filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(13*telem.Int64T.Density()), int64(2*telem.Int64T.Density())))

						Expect(w1.Commit(ctx)).To(Equal(53*telem.SecondTS + 1))

						_, err = w1.Close()
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close()
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i, _ := dataDB.OpenIterator(unary.IterRange(telem.TimeRangeMax))
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.SeriesAt(0).Len()).To(Equal(int64(7)))
							Expect(f.SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(17*telem.SecondTS + 1)))
							Expect(f.SeriesAt(1).Len()).To(Equal(int64(6)))
							Expect(f.SeriesAt(1).TimeRange).To(Equal((17*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
							Expect(f.Get(data).Data()).To(Equal(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 203, 209, 301, 307, 401, 403).Data))
							Expect(i.Close()).To(Succeed())
						})
					})
					Specify("File cutoff on commit with preset end", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSeriesSecondsTSV(10, 11, 13, 14, 15, 16, 17, 18, 20, 23, 25))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 108, 200, 203, 205))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(25*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(25*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSeriesSecondsTSV(29, 31, 37, 41, 43))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](209, 301, 307, 401, 403))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(43*telem.SecondTS + 1))

						By("Asserting that both writers are now writing to a different file")
						l := filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(11*telem.Int64T.Density()), int64(5*telem.Int64T.Density())))

						l = filterDataFiles(MustSucceed(dataFS.List("")))
						Expect(l).To(HaveLen(3))
						Expect([]int64{l[0].Size(), l[1].Size(), l[2].Size()}).To(ConsistOf(int64(11*telem.Int64T.Density()), int64(5*telem.Int64T.Density()), int64(0)))

						_, err = w1.Close()
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close()
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i, _ := dataDB.OpenIterator(unary.IterRange(telem.TimeRangeMax))
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.SeriesAt(0).Len()).To(Equal(int64(11)))
							Expect(f.SeriesAt(1).Len()).To(Equal(int64(5)))
							Expect(f.Get(data).Data()).To(Equal(telem.NewSeriesV[int64](100, 101, 103, 104, 105, 106, 107, 108, 200, 203, 205, 209, 301, 307, 401, 403).Data))
							Expect(f.SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(25*telem.SecondTS + 1)))
							Expect(f.SeriesAt(1).TimeRange).To(Equal((25*telem.SecondTS + 1).Range(43*telem.SecondTS + 1)))
							Expect(i.Close()).To(Succeed())
						})
					})

					Specify("Just enough data to switch files", func() {
						w1, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "foo"}}))
						w2, _ := MustSucceed2(dataDB.OpenWriter(ctx, unary.WriterConfig{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS, Subject: control.Subject{Key: "moo"}}))

						By("Writing telemetry")
						_, err := w1.Write(telem.NewSeriesSecondsTSV(10))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](100))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(10*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(10*telem.SecondTS + 1))

						By("Writing more telemetry")
						_, err = w1.Write(telem.NewSeriesSecondsTSV(11, 12, 13, 14, 15, 16, 17, 18, 19))
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Write(telem.NewSeriesV[int64](101, 102, 103, 104, 105, 106, 107, 108, 109))
						Expect(err).ToNot(HaveOccurred())

						By("Committing the data")
						Expect(w1.Commit(ctx)).To(Equal(19*telem.SecondTS + 1))
						Expect(w2.Commit(ctx)).To(Equal(19*telem.SecondTS + 1))

						By("Asserting that both writers have only written to one file")
						l := filterDataFiles(MustSucceed(indexFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(10*telem.Int64T.Density()), int64(0)))

						l = filterDataFiles(MustSucceed(dataFS.List("")))
						Expect(l).To(HaveLen(2))
						Expect([]int64{l[0].Size(), l[1].Size()}).To(ConsistOf(int64(10*telem.Int64T.Density()), int64(0)))

						_, err = w1.Close()
						Expect(err).ToNot(HaveOccurred())
						_, err = w2.Close()
						Expect(err).ToNot(HaveOccurred())

						By("Asserting that the data is correct", func() {
							i, _ := dataDB.OpenIterator(unary.IteratorConfig{Bounds: telem.TimeRangeMax})
							Expect(i.SeekFirst(ctx)).To(BeTrue())
							Expect(i.Next(ctx, telem.TimeSpanMax)).To(BeTrue())
							f := i.Value()
							Expect(f.SeriesAt(0).Len()).To(Equal(int64(10)))
							Expect(f.SeriesAt(0).Data).To(Equal(telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 106, 107, 108, 109).Data))
							Expect(f.SeriesAt(0).TimeRange).To(Equal((10 * telem.SecondTS).Range(19*telem.SecondTS + 1)))
							Expect(i.Close()).To(Succeed())
						})
					})
				})
			})

			Describe("Control", func() {
				var (
					db      *unary.DB
					fs      fs.FS
					cleanUp func() error
				)
				BeforeEach(func() {
					fs, cleanUp = makeFS()
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        fs,
						MetaCodec: codec,
						Channel: channel.Channel{
							Name:     "Frederick",
							Key:      2,
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				Describe("Index", func() {
					Specify("Control Handoff", func() {
						w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.AuthorityAbsolute - 1,
							Subject:   control.Subject{Key: "foo"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(0, 1, 2, 3, 4, 5)))).To(Equal(alignment.Leading(1, 0)))
						w2, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:     10 * telem.SecondTS,
							Authority: control.AuthorityAbsolute,
							Subject:   control.Subject{Key: "bar"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w2.Write(telem.NewSeriesSecondsTSV(6, 7, 8, 9, 10, 11)))).To(Equal(alignment.Leading(1, 6)))
						a, err := w1.Write(telem.NewSeriesSecondsTSV(12, 13, 14, 15, 16, 17))
						Expect(err).To(MatchError(control.ErrUnauthorized))
						Expect(a).To(Equal(telem.Alignment(0)))
						_, err = w1.Commit(ctx)
						Expect(err).To(MatchError(control.ErrUnauthorized))
						t = MustSucceed(w2.Close())
						Expect(t.Occurred()).To(BeTrue())
						Expect(MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(12, 13, 14, 15, 16, 17)))).To(Equal(alignment.Leading(1, 12)))
						Expect(MustSucceed(w1.Commit(ctx))).To(Equal(17*telem.SecondTS + 1))
						t = MustSucceed(w1.Close())
						Expect(t.Occurred()).To(BeTrue())
					})
				})
				Describe("ErrOnUnauthorizedOpen", func() {
					It("Should return an error if the write does not acquire control", func() {
						w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:                 10 * telem.SecondTS,
							Authority:             control.AuthorityAbsolute,
							Subject:               control.Subject{Key: "foo"},
							ErrOnUnauthorizedOpen: config.True(),
						}))
						Expect(t.Occurred()).To(BeTrue())
						w2, t, err := db.OpenWriter(ctx, unary.WriterConfig{
							Start:                 10 * telem.SecondTS,
							Authority:             control.AuthorityAbsolute - 1,
							Subject:               control.Subject{Key: "bar"},
							ErrOnUnauthorizedOpen: config.True(),
						})
						Expect(t.Occurred()).To(BeFalse())
						Expect(err).To(HaveOccurredAs(control.ErrUnauthorized))
						Expect(w2).To(BeNil())
						t, err = w1.Close()
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.IsRelease()).To(BeTrue())
						Expect(err).ToNot(HaveOccurred())
					})
				})
				Describe("Control Subjects", func() {
					It("Should return an error when attempting to open a writer with a duplicate control subject key", func() {
						w1, t := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:   10 * telem.SecondTS,
							Subject: control.Subject{Key: "foo"},
						}))
						Expect(t.Occurred()).To(BeTrue())
						w2, t, err := db.OpenWriter(ctx, unary.WriterConfig{
							Start:   10 * telem.SecondTS,
							Subject: control.Subject{Key: "foo"},
						})
						Expect(t.Occurred()).To(BeFalse())
						Expect(err).To(MatchError(ContainSubstring("already registered in the region")))
						Expect(w2).To(BeNil())
						MustSucceed(w1.Close())
					})
				})
			})

			Describe("Close", Ordered, func() {
				var (
					db      *unary.DB
					fs      fs.FS
					cleanUp func() error
					key     = testutil.GenerateChannelKey()
				)
				BeforeEach(func() {
					fs, cleanUp = makeFS()
					db = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        fs,
						MetaCodec: codec,
						Channel: channel.Channel{
							Key:      key,
							Name:     "gauss",
							DataType: telem.TimeStampT,
							IsIndex:  true,
						},
						Instrumentation: PanicLogger(),
					}))
				})
				AfterEach(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				It("Should not allow operations on a closed writer", func() {
					var (
						w, t = MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
							Start:   10 * telem.SecondTS,
							Subject: control.Subject{Key: "foo"}},
						))
						e = channel.NewErrResourceClosed("unary.writer")
					)
					Expect(t.Occurred()).To(BeTrue())
					_, err := w.Close()
					Expect(err).ToNot(HaveOccurred())
					_, err = w.Commit(ctx)
					Expect(err).To(HaveOccurredAs(e))
					Expect(err).To(MatchError(ContainSubstring("channel [gauss]<%d>", key)))
					_, err = w.Write(telem.Series{Data: []byte{1, 2, 3}})
					Expect(err).To(HaveOccurredAs(e))
					_, err = w.Close()
					Expect(err).ToNot(HaveOccurred())
				})
				It("Should not open a writer on a closed database", func() {
					Expect(db.Close()).To(Succeed())
					_, _, err := db.OpenWriter(ctx, unary.WriterConfig{
						Start:   10 * telem.SecondTS,
						Subject: control.Subject{Key: "foo"}},
					)
					Expect(err).To(HaveOccurredAs(channel.NewErrResourceClosed("unary.db")))
					Expect(err).To(MatchError(ContainSubstring("channel [gauss]<%d>", key)))
				})
				It("Should not write on a closed database", func() {
					Expect(db.Close()).To(Succeed())
					Expect(unary.Write(ctx, db, 0, telem.NewSeriesV[int64](0, 1, 2))).To(HaveOccurredAs(channel.NewErrResourceClosed("unary.db")))
				})
			})
		})
	}
})
