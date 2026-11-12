// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	"fmt"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/index"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TruncateToIndex", func() {
	for fsName, openFS := range FileSystems {
		Context("FS:"+fsName, func() {
			var (
				fs      *FaultyFS
				indexDB *unary.DB
				tr      = (10 * telem.SecondTS).Range(20 * telem.SecondTS)
			)
			BeforeEach(func(ctx SpecContext) {
				fs = WrapFaultyFS(openFS())
				indexDB = MustOpen(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("index")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "index",
						DataType: telem.TimestampT,
						IsIndex:  true,
					},
					Instrumentation: PanicLogger(),
				}))
			})
			// openData opens a data channel that is indexed by indexDB and writes the
			// given samples to it, with the index stamps at 10 s, 11 s, and so on.
			openData := func(ctx SpecContext, data telem.Series) *unary.DB {
				GinkgoHelper()
				db := MustOpen(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("data")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "data",
						DataType: data.DataType,
						Index:    indexDB.Channel().Key,
					},
					Instrumentation: PanicLogger(),
				}))
				db.SetIndex(indexDB.Index())
				stamps := make([]telem.TimeStamp, data.Len())
				for i := range stamps {
					stamps[i] = telem.TimeStamp(10+i) * telem.SecondTS
				}
				Expect(unary.Write(
					ctx,
					indexDB,
					10*telem.SecondTS,
					telem.NewSeriesV(stamps...),
				)).To(Succeed())
				Expect(unary.Write(ctx, db, 10*telem.SecondTS, data)).To(Succeed())
				return db
			}

			DescribeTable(
				"Should truncate a domain that holds more samples than its index",
				func(ctx SpecContext, data, expected telem.Series) {
					dataDB := openData(ctx, data)
					Expect(indexDB.Delete(
						ctx,
						(14 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					Expect(ReadUnary(ctx, dataDB, tr)).Error().
						To(MatchError(index.ErrDiscontinuous))
					Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
					frame := MustSucceed(ReadUnary(ctx, dataDB, tr))
					Expect(frame.Count()).To(Equal(1))
					Expect(frame.SeriesAt(0)).To(telem.MatchSeriesData(expected))
					Expect(frame.SeriesAt(0).TimeRange).To(Equal(
						(10 * telem.SecondTS).Range(14 * telem.SecondTS),
					))
				},
				Entry("fixed-density data",
					telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6),
					telem.NewSeriesV[int64](1, 2, 3, 4),
				),
				Entry("variable-density data",
					telem.NewSeriesV("a", "b", "c", "d", "e", "f"),
					telem.NewSeriesV("a", "b", "c", "d"),
				),
			)

			readData := func(name string) []byte {
				GinkgoHelper()
				f := MustSucceed(MustSucceed(fs.Sub("data")).Open(name, os.O_RDONLY))
				defer func() { Expect(f.Close()).To(Succeed()) }()
				return MustSucceed(io.ReadAll(f))
			}
			// quarantined returns the contents of the file that holds the bytes cut
			// from the domain that ends at end, from start on.
			quarantined := func(start, end telem.TimeStamp) []byte {
				GinkgoHelper()
				return readData(fmt.Sprintf("%d-%d.truncated", start, end))
			}

			DescribeTable("Should copy the samples it cuts to a file",
				func(ctx SpecContext, data, cut telem.Series) {
					dataDB := openData(ctx, data)
					Expect(indexDB.Delete(
						ctx,
						(14 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
					Expect(quarantined(14*telem.SecondTS, 15*telem.SecondTS+1)).
						To(Equal(cut.Data))
				},
				Entry("fixed-density data",
					telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6),
					telem.NewSeriesV[int64](5, 6),
				),
				Entry("variable-density data",
					telem.NewSeriesV("a", "b", "c", "d", "e", "f"),
					telem.NewSeriesV("e", "f"),
				),
			)

			It("Should copy a domain that it removes to a file", func(ctx SpecContext) {
				dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3))
				Expect(indexDB.Delete(ctx, tr)).To(Succeed())
				Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
				Expect(quarantined(10*telem.SecondTS, 12*telem.SecondTS+1)).
					To(Equal(telem.NewSeriesV[int64](1, 2, 3).Data))
			})

			It("Should not overwrite an earlier file for the same cut",
				func(ctx SpecContext) {
					dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))
					Expect(indexDB.Delete(
						ctx,
						(14 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					name := fmt.Sprintf(
						"%d-%d",
						14*telem.SecondTS,
						15*telem.SecondTS+1,
					)
					f := MustSucceed(MustSucceed(fs.Sub("data")).Open(
						name+".truncated",
						os.O_CREATE|os.O_WRONLY,
					))
					Expect(f.Write([]byte("earlier"))).To(Equal(7))
					Expect(f.Close()).To(Succeed())
					Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
					Expect(readData(name + ".truncated")).To(Equal([]byte("earlier")))
					Expect(readData(name + ".1.truncated")).
						To(Equal(telem.NewSeriesV[int64](5, 6).Data))
				},
			)

			It("Should copy to a new file when an earlier one of the same size differs",
				func(ctx SpecContext) {
					dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))
					Expect(indexDB.Delete(
						ctx,
						(14 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					name := fmt.Sprintf(
						"%d-%d",
						14*telem.SecondTS,
						15*telem.SecondTS+1,
					)
					earlier := telem.NewSeriesV[int64](7, 8).Data
					f := MustSucceed(MustSucceed(fs.Sub("data")).Open(
						name+".truncated",
						os.O_CREATE|os.O_WRONLY,
					))
					Expect(f.Write(earlier)).To(Equal(len(earlier)))
					Expect(f.Close()).To(Succeed())
					Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
					Expect(readData(name + ".truncated")).To(Equal(earlier))
					Expect(readData(name + ".1.truncated")).
						To(Equal(telem.NewSeriesV[int64](5, 6).Data))
				},
			)

			It(
				"Should reuse the file of a truncation that failed",
				func(ctx SpecContext) {
					dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))
					Expect(indexDB.Delete(
						ctx,
						(14 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					fs.SetOptions(WithFailWriteAt())
					Expect(dataDB.TruncateToIndex(ctx)).To(MatchError(ErrFault))
					fs.SetOptions()
					Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
					Expect(quarantined(14*telem.SecondTS, 15*telem.SecondTS+1)).
						To(Equal(telem.NewSeriesV[int64](5, 6).Data))
					Expect(MustSucceed(fs.Sub("data")).Exists(fmt.Sprintf(
						"%d-%d.1.truncated",
						14*telem.SecondTS,
						15*telem.SecondTS+1,
					))).To(BeFalse())
				},
			)

			It("Should replace a temporary file that an earlier copy left",
				func(ctx SpecContext) {
					dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))
					Expect(indexDB.Delete(
						ctx,
						(14 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					tmp := fmt.Sprintf(
						"%d-%d.truncated.tmp",
						14*telem.SecondTS,
						15*telem.SecondTS+1,
					)
					f := MustSucceed(MustSucceed(fs.Sub("data")).Open(
						tmp,
						os.O_CREATE|os.O_WRONLY,
					))
					Expect(f.Write([]byte("a partial copy that is long"))).To(Equal(27))
					Expect(f.Close()).To(Succeed())
					Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
					Expect(quarantined(14*telem.SecondTS, 15*telem.SecondTS+1)).
						To(Equal(telem.NewSeriesV[int64](5, 6).Data))
					Expect(MustSucceed(fs.Sub("data")).Exists(tmp)).To(BeFalse())
				},
			)

			DescribeTable("Should not truncate when the copy fails",
				func(ctx SpecContext, fault FaultyFSOption) {
					dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6))
					Expect(indexDB.Delete(
						ctx,
						(14 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					fs.SetOptions(fault)
					Expect(dataDB.TruncateToIndex(ctx)).To(MatchError(ErrFault))
					fs.SetOptions()
					Expect(ReadUnary(ctx, dataDB, tr)).Error().
						To(MatchError(index.ErrDiscontinuous))
					name := fmt.Sprintf(
						"%d-%d.truncated",
						14*telem.SecondTS,
						15*telem.SecondTS+1,
					)
					data := MustSucceed(fs.Sub("data"))
					Expect(data.Exists(name)).To(BeFalse())
					Expect(data.Exists(name + ".tmp")).To(BeFalse())
				},
				Entry("on open", WithFailOpen(fmt.Sprintf(
					"%d-%d.truncated.tmp",
					14*telem.SecondTS,
					15*telem.SecondTS+1,
				))),
				Entry("on rename", WithFailRename(fmt.Sprintf(
					"%d-%d.truncated.tmp",
					14*telem.SecondTS,
					15*telem.SecondTS+1,
				))),
				Entry("on stat", WithFailStat(fmt.Sprintf(
					"%d-%d.truncated",
					14*telem.SecondTS,
					15*telem.SecondTS+1,
				))),
				Entry("on write", WithFailWrite()),
			)

			It("Should name the channel and the domain when it fails",
				func(ctx SpecContext) {
					written := openData(ctx, telem.NewSeriesV("a", "b", "c", "d"))
					Expect(written.Close()).To(Succeed())
					Expect(indexDB.Delete(
						ctx,
						(12 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					dataDB := MustOpen(unary.Open(ctx, unary.Config{
						FS:              MustSucceed(fs.Sub("data")),
						MetaCodec:       json.Codec,
						Channel:         written.Channel(),
						Instrumentation: PanicLogger(),
					}))
					dataDB.SetIndex(indexDB.Index())
					fs.SetOptions(WithFailReadAt())
					err := dataDB.TruncateToIndex(ctx)
					Expect(err).To(MatchError(ErrFault))
					Expect(err).To(MatchError(ContainSubstring(
						dataDB.Channel().String(),
					)))
					Expect(err).To(MatchError(ContainSubstring(
						"domain " + (10 * telem.SecondTS).
							Range(13*telem.SecondTS+1).String(),
					)))
				},
			)

			It("Should name the domain when the delete fails", func(ctx SpecContext) {
				dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3, 4))
				Expect(indexDB.Delete(
					ctx,
					(12 * telem.SecondTS).Range(20*telem.SecondTS),
				)).To(Succeed())
				fs.SetOptions(WithFailWriteAt())
				err := dataDB.TruncateToIndex(ctx)
				Expect(err).To(MatchError(ErrFault))
				Expect(err).To(MatchError(ContainSubstring(
					"domain " + (10 * telem.SecondTS).Range(13*telem.SecondTS+1).
						String(),
				)))
				Expect(quarantined(12*telem.SecondTS, 13*telem.SecondTS+1)).
					To(Equal(telem.NewSeriesV[int64](3, 4).Data))
			})

			It("Should name the domain when reading the index fails",
				func(ctx SpecContext) {
					dataDB := MustOpen(unary.Open(ctx, unary.Config{
						FS:        MustSucceed(fs.Sub("data")),
						MetaCodec: json.Codec,
						Channel: channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "data",
							DataType: telem.Int64T,
							Index:    indexDB.Channel().Key,
						},
						Instrumentation: PanicLogger(),
					}))
					dataDB.SetIndex(indexDB.Index())
					Expect(unary.Write(
						ctx,
						indexDB,
						10*telem.SecondTS,
						telem.NewSeriesSecondsTSV(10, 11, 12, 13),
					)).To(Succeed())
					Expect(unary.Write(
						ctx,
						dataDB,
						12*telem.SecondTS,
						telem.NewSeriesV[int64](1, 2),
					)).To(Succeed())
					Expect(indexDB.Delete(
						ctx,
						(13 * telem.SecondTS).Range(20*telem.SecondTS),
					)).To(Succeed())
					fs.SetOptions(WithFailReadAt())
					err := dataDB.TruncateToIndex(ctx)
					Expect(err).To(MatchError(ErrFault))
					Expect(err).To(MatchError(ContainSubstring(
						"domain " + (12 * telem.SecondTS).
							Range(13*telem.SecondTS+1).String(),
					)))
				},
			)

			It("Should remove a domain whose start no index domain holds",
				func(ctx SpecContext) {
					dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3))
					Expect(indexDB.Delete(ctx, tr)).To(Succeed())
					Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
					Expect(dataDB.HasDataFor(ctx, telem.TimeRangeMax)).To(BeFalse())
				},
			)

			It("Should keep a domain that its index covers", func(ctx SpecContext) {
				dataDB := openData(ctx, telem.NewSeriesV[int64](1, 2, 3))
				Expect(dataDB.TruncateToIndex(ctx)).To(Succeed())
				frame := MustSucceed(ReadUnary(ctx, dataDB, tr))
				Expect(frame.Count()).To(Equal(1))
				Expect(frame.SeriesAt(0)).To(
					telem.MatchSeriesData(telem.NewSeriesV[int64](1, 2, 3)),
				)
			})

			It("Should keep the samples of an index channel", func(ctx SpecContext) {
				Expect(unary.Write(
					ctx,
					indexDB,
					10*telem.SecondTS,
					telem.NewSeriesSecondsTSV(10, 11, 12),
				)).To(Succeed())
				Expect(indexDB.TruncateToIndex(ctx)).To(Succeed())
				frame := MustSucceed(ReadUnary(ctx, indexDB, tr))
				Expect(frame.SeriesAt(0)).To(
					telem.MatchSeriesData(telem.NewSeriesSecondsTSV(10, 11, 12)),
				)
			})
		})
	}
})
