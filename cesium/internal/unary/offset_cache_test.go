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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	cachedSamples = 2000
	// scanRead is the length of each read that a table rebuild makes on a domain larger
	// than it. Each test domain is about 136 kB.
	scanRead = 64 * int(telem.Kilobyte)
	// oneTable holds the table of one test domain but not two.
	oneTable = 12 * telem.Kilobyte
)

var cachedValue = strings.Repeat("v", 64)

var _ = Describe("OffsetCache", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				fs  xfs.FS
				idx *unary.DB
			)
			BeforeEach(func(ctx SpecContext) {
				fs = openFS()
				idx = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("idx")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "idx",
						DataType: telem.TimestampT,
						IsIndex:  true,
					},
				}))
				DeferCleanup(func() { Expect(idx.Close()).To(Succeed()) })
			})

			writeIndex := func(ctx SpecContext, start telem.TimeStamp) telem.TimeRange {
				stamps := make([]telem.TimeStamp, cachedSamples)
				for i := range stamps {
					stamps[i] = start + telem.TimeStamp(i)*telem.SecondTS
				}
				Expect(unary.Write(ctx, idx, start, telem.NewSeriesV(stamps...))).
					To(Succeed())
				return start.Range(stamps[cachedSamples-1] + 1)
			}

			open := func(
				ctx SpecContext,
				ch channel.Channel,
				cache *unary.OffsetCache,
			) (*unary.DB, *Recorder) {
				rec := NewRecorder(MustSucceed(fs.Sub(ch.Name)))
				db := MustSucceed(unary.Open(ctx, unary.Config{
					FS:          rec,
					MetaCodec:   json.Codec,
					Channel:     ch,
					OffsetCache: cache,
				}))
				db.SetIndex(idx.Index())
				return db, rec
			}

			openData := func(
				ctx SpecContext,
				name string,
				cache *unary.OffsetCache,
			) (*unary.DB, *Recorder) {
				db, rec := open(ctx, channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     name,
					DataType: telem.StringT,
					Index:    idx.Channel().Key,
				}, cache)
				DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
				return db, rec
			}

			writeData := func(ctx SpecContext, db *unary.DB, start telem.TimeStamp) {
				values := make([]string, cachedSamples)
				for i := range values {
					values[i] = cachedValue
				}
				Expect(unary.Write(ctx, db, start, telem.NewSeriesV(values...))).
					To(Succeed())
			}

			// scans reads tr from db and reports whether the read rebuilt a table.
			scans := func(
				ctx SpecContext,
				db *unary.DB,
				rec *Recorder,
				tr telem.TimeRange,
			) bool {
				rec.Reset()
				frame := MustSucceed(ReadUnary(ctx, db, tr))
				Expect(frame.Count()).To(BeNumerically(">", 0))
				return rec.Count(MatchOp(OpReadAt), MatchLength(scanRead)) > 0
			}

			It("should rebuild the least recently used table past the budget", func(
				ctx SpecContext,
			) {
				cache := unary.NewOffsetCache(oneTable)
				a, recA := openData(ctx, "a", cache)
				b, recB := openData(ctx, "b", cache)
				tr := writeIndex(ctx, 10*telem.SecondTS)
				writeData(ctx, a, tr.Start)
				writeData(ctx, b, tr.Start)
				Expect(scans(ctx, b, recB, tr)).To(BeFalse())
				Expect(scans(ctx, a, recA, tr)).To(BeTrue())
				Expect(scans(ctx, a, recA, tr)).To(BeFalse())
				Expect(scans(ctx, b, recB, tr)).To(BeTrue())
			})

			It(
				"should not cache a table larger than the budget",
				func(ctx SpecContext) {
					db, rec := openData(
						ctx,
						"big",
						unary.NewOffsetCache(telem.Kilobyte),
					)
					tr := writeIndex(ctx, 10*telem.SecondTS)
					writeData(ctx, db, tr.Start)
					Expect(scans(ctx, db, rec, tr)).To(BeTrue())
					Expect(scans(ctx, db, rec, tr)).To(BeTrue())
				},
			)

			It("should keep the tables of domains that a delete does not touch", func(
				ctx SpecContext,
			) {
				db, rec := openData(ctx, "delete", nil)
				first := writeIndex(ctx, 10*telem.SecondTS)
				second := writeIndex(ctx, 10000*telem.SecondTS)
				writeData(ctx, db, first.Start)
				writeData(ctx, db, second.Start)
				Expect(db.Delete(ctx, (second.Start + 5*telem.SecondTS).Range(
					second.Start+10*telem.SecondTS,
				))).To(Succeed())
				Expect(scans(ctx, db, rec, first)).To(BeFalse())
				Expect(scans(ctx, db, rec, second)).To(BeTrue())
			})

			It(
				"should keep its tables through garbage collection",
				func(ctx SpecContext) {
					db, rec := openData(ctx, "gc", nil)
					tr := writeIndex(ctx, 10*telem.SecondTS)
					writeData(ctx, db, tr.Start)
					Expect(db.GarbageCollect(ctx)).To(Succeed())
					Expect(scans(ctx, db, rec, tr)).To(BeFalse())
				},
			)

			It("should drop the tables of a DB when it closes", func(ctx SpecContext) {
				cache := unary.NewOffsetCache(unary.DefaultOffsetCacheBudget)
				ch := channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     "closed",
					DataType: telem.StringT,
					Index:    idx.Channel().Key,
				}
				db, _ := open(ctx, ch, cache)
				tr := writeIndex(ctx, 10*telem.SecondTS)
				writeData(ctx, db, tr.Start)
				Expect(db.Close()).To(Succeed())
				db, rec := open(ctx, ch, cache)
				DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
				Expect(scans(ctx, db, rec, tr)).To(BeTrue())
			})
		})
	}
})
