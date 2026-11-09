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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/index"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TruncateToIndex", func() {
	for fsName, openFS := range FileSystems {
		Context("FS:"+fsName, func() {
			var (
				fs      xfs.FS
				indexDB *unary.DB
				tr      = (10 * telem.SecondTS).Range(20 * telem.SecondTS)
			)
			BeforeEach(func(ctx SpecContext) {
				fs = openFS()
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
