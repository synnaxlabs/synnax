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
	"fmt"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scratch clock seeds", func() {
	for seed := range 40 {
		It(fmt.Sprintf("Should stamp the clock of every read series, seed %d", seed),
			func(ctx SpecContext) {
				rng := rand.New(rand.NewPCG(uint64(seed), 7))
				db := DeferClose(openDBOnFS(ctx, FileSystems["memFS"]()))
				StartRun(db.Run)
				clockOf := map[cesium.ChannelKey]uint32{}
				type group struct {
					index cesium.ChannelKey
					data  []cesium.ChannelKey
					types []telem.DataType
				}
				groups := make([]group, 1+rng.IntN(3))
				var all []cesium.ChannelKey
				dts := []telem.DataType{
					telem.Float32T, telem.Int64T, telem.Uint8T, telem.StringT,
					telem.Float64T,
				}
				for gi := range groups {
					idx := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key: idx, Name: fmt.Sprint("i", idx), IsIndex: true,
						DataType: telem.TimestampT,
					})).To(Succeed())
					clockOf[idx] = uint32(idx)
					all = append(all, idx)
					g := group{index: idx}
					for range 1 + rng.IntN(3) {
						k := GenerateChannelKey()
						dt := dts[rng.IntN(len(dts))]
						Expect(db.CreateChannel(ctx, cesium.Channel{
							Key: k, Name: fmt.Sprint("d", k), Index: idx, DataType: dt,
						})).To(Succeed())
						clockOf[k] = uint32(idx)
						g.data = append(g.data, k)
						g.types = append(g.types, dt)
						all = append(all, k)
					}
					groups[gi] = g
					t := telem.TimeStamp(rng.IntN(5)) * telem.SecondTS
					for range 1 + rng.IntN(5) {
						n := 1 + rng.IntN(20)
						stamps := make([]telem.TimeStamp, n)
						for j := range n {
							t += telem.TimeStamp(1+rng.IntN(3)) * telem.MillisecondTS
							stamps[j] = t
						}
						keys := []cesium.ChannelKey{idx}
						series := []telem.Series{telem.NewSeriesV(stamps...)}
						for di, k := range g.data {
							keys = append(keys, k)
							switch g.types[di] {
							case telem.Float32T:
								series = append(series, telem.NewSeries(make([]float32, n)))
							case telem.Int64T:
								series = append(series, telem.NewSeries(make([]int64, n)))
							case telem.Uint8T:
								series = append(series, telem.NewSeries(make([]uint8, n)))
							case telem.Float64T:
								series = append(series, telem.NewSeries(make([]float64, n)))
							default:
								strs := make([]string, n)
								for j := range strs {
									strs[j] = "s"
								}
								series = append(series, telem.NewSeries(strs))
							}
						}
						Expect(db.Write(ctx, stamps[0], telem.MultiFrame(keys, series))).
							To(Succeed())
						t += telem.TimeStamp(rng.IntN(1000)) * telem.MillisecondTS
					}
				}
				checked := 0
				for read := range 12 {
					var chs []cesium.ChannelKey
					for _, k := range all {
						if rng.IntN(2) == 0 {
							chs = append(chs, k)
						}
					}
					if len(chs) == 0 {
						chs = all[:1]
					}
					var red telem.Reduction
					switch rng.IntN(5) {
					case 1:
						red = telem.Reduction{Variant: telem.StrideReduction{
							Factor: uint32(2 + rng.IntN(3)),
						}}
					case 2, 3:
						aggs := []telem.Aggregation{
							telem.AggregationMinMax, telem.AggregationAverage,
							telem.AggregationDecimate,
						}
						red = telem.Reduction{Variant: telem.LimitReduction{
							Aggregation: aggs[rng.IntN(len(aggs))],
							PointLimit:  uint32(2 + rng.IntN(10)),
						}}
					}
					it := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Channels:      chs,
						Bounds:        telem.TimeRangeMax,
						Reduction:     red,
						AutoChunkSize: int64(1 + rng.IntN(30)),
					}))
					check := func() {
						for k, s := range it.Value().Entries() {
							checked++
							Expect(s.Clock).To(Equal(clockOf[k]),
								fmt.Sprintf("read %d key %d reduction %+v", read, k, red))
						}
					}
					if rng.IntN(2) == 0 {
						if it.SeekFirst() {
							for it.Next(cesium.AutoSpan) {
								check()
							}
						}
					} else if it.SeekLast() {
						for it.Prev(cesium.AutoSpan) {
							check()
						}
					}
					Expect(it.Close()).To(Succeed())
				}
				Expect(checked).To(BeNumerically(">", 0))
				GinkgoWriter.Printf("seed %d checked %d series\n", seed, checked)
			})
	}
})
