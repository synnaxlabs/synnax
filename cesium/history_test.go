// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// observeHistory subscribes to db's history changes and returns a channel receiving
// them. Sends are non-blocking, since handlers run under the DB's locks.
func observeHistory(db *cesium.DB) <-chan cesium.HistoryChange {
	GinkgoHelper()
	changes := make(chan cesium.HistoryChange, 100)
	DeferCleanup(db.OnHistoryChange(
		func(_ context.Context, c cesium.HistoryChange) {
			select {
			case changes <- c:
			default:
			}
		},
	))
	return changes
}

func historyChange(
	key cesium.ChannelKey,
	start, end telem.TimeStamp,
) cesium.HistoryChange {
	return cesium.HistoryChange{Key: key, TimeRange: start.Range(end)}
}

var _ = Describe("History", func() {
	for fsName, openFS := range FileSystems {
		Context("FS:"+fsName, Ordered, func() {
			var db *cesium.DB
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				db = mustOpenDBOnFS(
					ctx,
					openFS(),
					cesium.WithFileSizeCap(40*telem.Byte),
				)
			})

			createIndex := func(ctx context.Context) cesium.ChannelKey {
				GinkgoHelper()
				key := GenerateChannelKey()
				Expect(db.CreateChannel(ctx, cesium.Channel{
					Key:      key,
					Name:     "Shackleton",
					IsIndex:  true,
					DataType: telem.TimestampT,
				})).To(Succeed())
				return key
			}

			openWriter := func(
				ctx context.Context,
				key cesium.ChannelKey,
				start telem.TimeStamp,
			) *cesium.Writer {
				GinkgoHelper()
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels:         []cesium.ChannelKey{key},
					Start:            start,
					EnableAutoCommit: new(false),
				}))
				DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
				return w
			}

			writeAndCommit := func(
				w *cesium.Writer,
				key cesium.ChannelKey,
				stamps ...telem.TimeStamp,
			) {
				GinkgoHelper()
				Expect(w.Write(telem.UnaryFrame(
					key,
					telem.NewSeriesSecondsTSV(stamps...),
				))).To(BeTrue())
				MustSucceed(w.Commit())
			}

			It(
				"Should not signal the first commit of a writer at the end",
				func(ctx SpecContext) {
					key := createIndex(ctx)
					Expect(db.WriteSeries(
						ctx,
						key,
						5*telem.SecondTS,
						telem.NewSeriesSecondsTSV(5, 6),
					)).To(Succeed())
					changes := observeHistory(db)
					w := openWriter(ctx, key, 10*telem.SecondTS)
					writeAndCommit(w, key, 10, 11)
					Expect(changes).ToNot(Receive())
				},
			)

			It(
				"Should not signal a commit that extends the last domain",
				func(ctx SpecContext) {
					key := createIndex(ctx)
					w := openWriter(ctx, key, 10*telem.SecondTS)
					writeAndCommit(w, key, 10, 11)
					changes := observeHistory(db)
					writeAndCommit(w, key, 12, 13)
					Expect(changes).ToNot(Receive())
				},
			)

			It(
				"Should signal a domain inserted before existing data",
				func(ctx SpecContext) {
					key := createIndex(ctx)
					Expect(db.WriteSeries(
						ctx,
						key,
						20*telem.SecondTS,
						telem.NewSeriesSecondsTSV(20, 21),
					)).To(Succeed())
					changes := observeHistory(db)
					Expect(db.WriteSeries(
						ctx,
						key,
						10*telem.SecondTS,
						telem.NewSeriesSecondsTSV(10, 11),
					)).To(Succeed())
					Expect(changes).To(Receive(Equal(
						historyChange(key, 10*telem.SecondTS, 11*telem.SecondTS+1),
					)))
					Expect(changes).ToNot(Receive())
				},
			)

			It(
				"Should signal each commit to a domain that is not the last",
				func(ctx SpecContext) {
					key := createIndex(ctx)
					Expect(db.WriteSeries(
						ctx,
						key,
						20*telem.SecondTS,
						telem.NewSeriesSecondsTSV(20, 21),
					)).To(Succeed())
					changes := observeHistory(db)
					w := openWriter(ctx, key, 10*telem.SecondTS)
					writeAndCommit(w, key, 10, 11)
					Expect(changes).To(Receive(Equal(
						historyChange(key, 10*telem.SecondTS, 11*telem.SecondTS+1),
					)))
					writeAndCommit(w, key, 12, 13)
					Expect(changes).To(Receive(Equal(
						historyChange(key, 11*telem.SecondTS+1, 13*telem.SecondTS+1),
					)))
				},
			)

			It(
				"Should not signal the new domain a file rollover starts",
				func(ctx SpecContext) {
					key := createIndex(ctx)
					changes := observeHistory(db)
					w := openWriter(ctx, key, 10*telem.SecondTS)
					// Five timestamps fill the 40 byte file, so this commit rolls over.
					writeAndCommit(w, key, 10, 11, 12, 13, 14)
					writeAndCommit(w, key, 15, 16)
					writeAndCommit(w, key, 17, 18)
					Expect(changes).ToNot(Receive())
				},
			)

			It(
				"Should signal a delete on each channel it removes data from",
				func(ctx SpecContext) {
					idxKey := createIndex(ctx)
					dataKey := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      dataKey,
						Name:     "Wild",
						Index:    idxKey,
						DataType: telem.Int64T,
					})).To(Succeed())
					Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{idxKey, dataKey},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12, 13),
							telem.NewSeriesV[int64](1, 2, 3, 4),
						},
					))).To(Succeed())
					changes := observeHistory(db)
					tr := (11 * telem.SecondTS).Range(13 * telem.SecondTS)
					Expect(db.DeleteTimeRange(
						ctx,
						[]cesium.ChannelKey{idxKey, dataKey},
						tr,
					)).To(Succeed())
					var first, second cesium.HistoryChange
					Expect(changes).To(Receive(&first))
					Expect(changes).To(Receive(&second))
					Expect([]cesium.HistoryChange{first, second}).To(ConsistOf(
						cesium.HistoryChange{Key: idxKey, TimeRange: tr},
						cesium.HistoryChange{Key: dataKey, TimeRange: tr},
					))
				},
			)

			It(
				"Should not signal a delete that removes nothing",
				func(ctx SpecContext) {
					key := createIndex(ctx)
					Expect(db.WriteSeries(
						ctx,
						key,
						10*telem.SecondTS,
						telem.NewSeriesSecondsTSV(10, 11),
					)).To(Succeed())
					changes := observeHistory(db)
					Expect(db.DeleteTimeRange(
						ctx,
						[]cesium.ChannelKey{key},
						(20 * telem.SecondTS).Range(30*telem.SecondTS),
					)).To(Succeed())
					Expect(changes).ToNot(Receive())
				},
			)
		})
	}
})
