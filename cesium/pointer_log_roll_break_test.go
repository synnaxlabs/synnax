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
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Pointer log roll breaker", func() {
	It("Should read a data channel across a roll of its clock's segment", func(
		ctx SpecContext,
	) {
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(xfs.NewMem()),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		stop := StartRun(db.Run)
		defer func() {
			stop()
			Expect(db.Close()).To(Succeed())
		}()
		idx, data := cesium.ChannelKey(1), cesium.ChannelKey(2)
		keys := []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Uint8T},
		)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    2,
		}))
		// Sample i has stamp 2(i + 1) and value i mod 251. The clock's segment reaches
		// 2^30 bytes, and seals, after 2^27 samples, while the data segment holds 2^27
		// bytes and stays open.
		const chunk = 1 << 22
		stamps := make([]telem.TimeStamp, chunk)
		values := make([]uint8, chunk)
		write := func(from, n int) {
			GinkgoHelper()
			for i := range n {
				stamps[i] = telem.TimeStamp(2 * (from + i + 1))
				values[i] = uint8((from + i) % 251)
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps[:n]),
				telem.NewSeries(values[:n]),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		const roll = 1 << 27
		for from := 0; from < roll; from += chunk {
			write(from, chunk)
		}
		write(roll, 16)
		Expect(w.Close()).To(Succeed())
		tr := telem.TimeRange{
			Start: telem.TimeStamp(2 * (roll - 4 + 1)),
			End:   telem.TimeStamp(2*(roll+4+1) + 1),
		}
		fr := MustSucceed(db.Read(ctx, tr, data))
		var got []uint8
		for _, s := range fr.Get(data).Series {
			got = append(got, s.Data...)
		}
		want := make([]uint8, 0, 9)
		for i := roll - 4; i <= roll+4; i++ {
			want = append(want, uint8(i%251))
		}
		Expect(got).To(Equal(want))
	})
})

var _ = Describe("Pointer log seek breaker", func() {
	DescribeTable("Should seek at or before the largest stamp",
		func(ctx SpecContext, opts ...cesium.Option) {
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				append([]cesium.Option{
					cesium.WithFS(xfs.NewMem()),
					cesium.WithRandom(rand.NewPCG(1, 2)),
				}, opts...)...,
			))
			stop := StartRun(db.Run)
			defer func() {
				stop()
				Expect(db.Close()).To(Succeed())
			}()
			idx, data := cesium.ChannelKey(1), cesium.ChannelKey(2)
			keys := []cesium.ChannelKey{idx, data}
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
				},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Clock:    idx,
				Start:    10 * telem.SecondTS,
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(
				keys,
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11),
					telem.NewSeriesV[int64](10, 11),
				},
			), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
				Channels: []cesium.ChannelKey{data},
				Bounds:   telem.TimeRangeMax,
			}))
			defer func() { Expect(iter.Close()).To(Succeed()) }()
			Expect(iter.SeekLE(telem.TimeStampMax)).To(BeTrue())
		},
		Entry("old layout"),
		Entry("pointer log", cesium.WithPointerLog()),
	)
})
