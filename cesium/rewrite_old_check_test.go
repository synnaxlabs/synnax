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
	"encoding/binary"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/testutil/oldlayout"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// oldReadout is what a read gives for one series.
type oldReadout struct {
	Values    []int64
	Alignment telem.Alignment
	TimeRange telem.TimeRange
}

var _ = Describe("Check: the rewrite of a checked version 0 segment", func() {
	It("Should read every sample as before once it rewrites an old file whose "+
		"pointers are out of time order and whose last block is partial",
		func(ctx SpecContext) {
			const (
				idx, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
				// The late run lies first in each old file and the early run after
				// it, as a backfill leaves them. 25,000 samples of 8 bytes end inside
				// block 3.
				late, early = 20000, 5000
				lateAt      = 100000 * telem.SecondTS
				earlyAt     = 10 * telem.SecondTS
			)
			stamps := make([]telem.TimeStamp, 0, late+early)
			values := make([]int64, 0, late+early)
			for i := range late {
				stamps = append(stamps, lateAt+telem.TimeStamp(i)*telem.SecondTS)
				values = append(values, int64(i))
			}
			for i := range early {
				stamps = append(stamps, earlyAt+telem.TimeStamp(i)*telem.SecondTS)
				values = append(values, int64(late+i))
			}
			// index.domain holds its records in time order.
			pointers := []oldlayout.Pointer{
				{
					TimeRange: telem.TimeRange{
						Start: earlyAt,
						End:   stamps[late+early-1] + 1,
					},
					File:   1,
					Offset: 8 * late,
					Size:   8 * early,
				},
				{
					TimeRange: telem.TimeRange{Start: lateAt, End: stamps[late-1] + 1},
					File:      1,
					Size:      8 * late,
				},
			}
			disk := xfs.NewMem()
			Expect(oldlayout.Write(
				disk,
				oldlayout.Key{
					Channel: channel.Channel{
						Key:      idx,
						Name:     "time",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					Counter:  1,
					Pointers: pointers,
					Files:    map[uint16][]byte{1: telem.NewSeries(stamps).Data},
				},
				oldlayout.Key{
					Channel: channel.Channel{
						Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
					},
					Counter:  1,
					Pointers: pointers,
					Files:    map[uint16][]byte{1: telem.NewSeries(values).Data},
				},
			)).To(Succeed())
			keys := []cesium.ChannelKey{idx, data}
			open := func(ctx context.Context) (*cesium.DB, func()) {
				GinkgoHelper()
				db := MustSucceed(cesium.Open(
					ctx,
					"",
					cesium.WithFS(disk),
					cesium.WithPointerLog(),
					cesium.WithVersion0Rewritable(),
					cesium.WithRandom(rand.NewPCG(1, 2)),
					cesium.WithIdentity(rand.NewPCG(1, 2).Uint64()),
				))
				stop := StartRun(db.Run)
				return db, func() {
					GinkgoHelper()
					stop()
					Expect(db.Close()).To(Succeed())
				}
			}
			ranges := []telem.TimeRange{
				telem.TimeRangeMax,
				{Start: earlyAt, End: earlyAt + 100*telem.SecondTS},
				{
					Start: earlyAt + 4990*telem.SecondTS,
					End:   lateAt + 15010*telem.SecondTS,
				},
				{Start: lateAt + 19990*telem.SecondTS, End: telem.TimeStampMax},
			}
			readAll := func(ctx context.Context, db *cesium.DB) [][]oldReadout {
				GinkgoHelper()
				out := make([][]oldReadout, len(ranges))
				for i, tr := range ranges {
					fr := MustSucceed(db.Read(ctx, tr, keys...))
					for _, k := range keys {
						for _, s := range fr.Get(k).Series {
							r := oldReadout{
								Alignment: s.Alignment,
								TimeRange: s.TimeRange,
							}
							for j := 0; j+8 <= len(s.Data); j += 8 {
								r.Values = append(
									r.Values,
									int64(binary.LittleEndian.Uint64(s.Data[j:])),
								)
							}
							out[i] = append(out[i], r)
						}
					}
				}
				return out
			}
			// rewritten reports whether the directory of key holds no old data file and
			// one segment file.
			rewritten := func(key cesium.ChannelKey) bool {
				GinkgoHelper()
				var old, segs int
				for _, info := range MustSucceed(disk.List(
					strconv.FormatUint(uint64(key), 10),
				)) {
					switch name := info.Name(); {
					case name == "1.domain":
						old++
					case strings.HasSuffix(name, ".seg"):
						segs++
					}
				}
				return old == 0 && segs == 1
			}
			db, closeDB := open(ctx)
			Eventually(func() (map[cesium.ChannelKey]int, error) {
				return checkFiles(disk, keys...)
			}).WithTimeout(time.Minute).Should(Equal(map[cesium.ChannelKey]int{
				idx: 1, data: 1,
			}))
			// The delete leaves 10,000 of the 25,000 samples of each old file live.
			Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRange{
				Start: lateAt,
				End:   lateAt + 15000*telem.SecondTS,
			})).To(Succeed())
			before := readAll(ctx, db)
			var got []int64
			for _, s := range MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).
				Get(data).Series {
				for j := 0; j+8 <= len(s.Data); j += 8 {
					got = append(got, int64(binary.LittleEndian.Uint64(s.Data[j:])))
				}
			}
			want := append(slices.Clone(values[late:]), values[15000:late]...)
			Expect(got).To(Equal(want))
			Eventually(func() bool {
				return rewritten(idx) && rewritten(data)
			}).WithTimeout(time.Minute).Should(BeTrue())
			Expect(readAll(ctx, db)).To(Equal(before))
			closeDB()
			db, closeDB = open(ctx)
			Expect(readAll(ctx, db)).To(Equal(before))
			closeDB()
		})
})
