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
	"math/rand/v2"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete on the pointer log, broken", func() {
	var (
		fs        xfs.FS
		db        *cesium.DB
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		StartRun(db.Run)
	}
	BeforeEach(func(ctx SpecContext) {
		fs = xfs.NewMem()
		open(ctx)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	write := func(ctx context.Context, seconds ...telem.TimeStamp) {
		GinkgoHelper()
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			values[i] = int64(s)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    seconds[0] * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(values),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	values := func(ctx context.Context, key cesium.ChannelKey) []int64 {
		GinkgoHelper()
		series := MustSucceed(db.Read(ctx, telem.TimeRangeMax, key)).Get(key).Series
		out := make([]int64, 0, len(series))
		for _, s := range series {
			out = append(out, s.Unmarshal[int64]()...)
		}
		return out
	}
	span := func(start, end telem.TimeStamp) telem.TimeRange {
		return (start * telem.SecondTS).Range(end * telem.SecondTS)
	}
	segments := func(key cesium.ChannelKey) []string {
		GinkgoHelper()
		var out []string
		for _, e := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
			if strings.HasSuffix(e.Name(), ".seg") {
				out = append(out, e.Name())
			}
		}
		return out
	}

	It("Should read a pinned segment after a delete empties it, and unlink it after "+
		"the iterator moves off it", func(ctx SpecContext) {
		write(ctx, 10, 11)
		write(ctx, 20, 21, 22)
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{data},
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, span(0, 15))).
			To(Succeed())
		Expect(segments(data)).To(HaveLen(2))
		Expect(iter.Next(cesium.AutoSpan)).To(BeTrue())
		Expect(iter.Value().Get(data).Series[0].Unmarshal[int64]()).
			To(Equal([]int64{10, 11}))
		Expect(iter.SeekGE(100 * telem.SecondTS)).To(BeFalse())
		Expect(segments(data)).To(HaveLen(1))
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(iter.Next(cesium.AutoSpan)).To(BeTrue())
		Expect(iter.Value().Get(data).Series[0].Unmarshal[int64]()).
			To(Equal([]int64{20, 21, 22}))
		Expect(iter.Close()).To(Succeed())
		iter2 := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds:   telem.TimeRangeMax,
			Channels: []cesium.ChannelKey{data},
		}))
		Expect(iter2.SeekFirst()).To(BeTrue())
		Expect(iter2.Close()).To(Succeed())
	})

	It("Should refuse a range that starts after it ends, as the old layout does",
		func(ctx SpecContext) {
			old := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(xfs.NewMem())))
			stop := StartRun(old.Run)
			DeferCleanup(func() {
				stop()
				Expect(old.Close()).To(Succeed())
			})
			Expect(old.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
				},
			)).To(Succeed())
			inverted := telem.TimeRange{
				Start: 12 * telem.SecondTS,
				End:   11 * telem.SecondTS,
			}
			Expect(old.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, inverted)).
				To(MatchError(ContainSubstring("cannot be after delete end")))
			write(ctx, 10, 11, 12)
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, inverted)).
				To(MatchError(ContainSubstring("cannot be after delete end")))
		})

	It("Should unlink a segment that a delete left empty once a later writer seals it",
		func(ctx SpecContext) {
			write(ctx, 10, 11)
			Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{data}, span(0, 15))).
				To(Succeed())
			write(ctx, 20, 21)
			Expect(values(ctx, data)).To(Equal([]int64{20, 21}))
			Expect(segments(data)).To(HaveLen(1))
		})
})
