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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: a read across the Drop of its key and its index", func() {
	var (
		fs        xfs.FS
		db        *cesium.DB
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		fs = xfs.NewMem()
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		StartRun(db.Run)
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
	// write writes the samples of seconds to both keys in one writer, so each key gets
	// one segment for each call.
	write := func(ctx context.Context, seconds ...telem.TimeStamp) {
		GinkgoHelper()
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			values[i] = int64(s)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    keys[0],
			Start:    seconds[0] * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(values),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}

	DescribeTable("Should finish a read whose bounds cut inside its pointers, and "+
		"unlink the dropped directories after it", func(ctx SpecContext, both bool) {
		write(ctx, 10, 11, 12)
		write(ctx, 20, 21, 22)
		// The bounds cut inside both pointers, so the read finds each cut through the
		// stamps of the index.
		iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Bounds: telem.TimeRange{
				Start: 11 * telem.SecondTS,
				End:   22 * telem.SecondTS,
			},
			Channels: []cesium.ChannelKey{data},
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		dropped := []cesium.ChannelKey{data}
		if both {
			dropped = keys
		}
		Expect(db.Drop(dropped)).To(Succeed())
		var got []int64
		for iter.Next(cesium.AutoSpan) {
			for _, s := range iter.Value().Get(data).Series {
				for i := range int(s.Len()) {
					got = append(got, s.ValueAt[int64](i))
				}
			}
		}
		Expect(iter.Error()).ToNot(HaveOccurred())
		Expect(got).To(Equal([]int64{11, 12, 20, 21}))
		Expect(iter.Close()).To(Succeed())
		for _, k := range dropped {
			if dir := strconv.Itoa(int(k)); MustSucceed(fs.Exists(dir)) {
				Expect(fs.List(dir)).To(BeEmpty(), "key %d", k)
			}
		}
	},
		Entry("in a Drop of the key alone", false),
		Entry("in a Drop of the key and its index", true),
	)
})
