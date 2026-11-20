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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer Commit with a skip", func() {
	It(
		"Should commit every data channel that skip does not name",
		func(ctx SpecContext) {
			db := mustOpenDBOnFS(ctx, xfs.NewMem())
			// Each round has a new group, so the order of its data channels is new.
			for round := range 4 {
				index := GenerateChannelKey()
				data := make([]cesium.ChannelKey, 8)
				channels := []cesium.Channel{{
					Key:      index,
					Name:     fmt.Sprintf("index-%d", round),
					IsIndex:  true,
					DataType: telem.TimestampT,
				}}
				for i := range data {
					data[i] = GenerateChannelKey()
					channels = append(channels, cesium.Channel{
						Key:      data[i],
						Name:     fmt.Sprintf("data-%d-%d", round, i),
						Index:    index,
						DataType: telem.Int64T,
					})
				}
				Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: append([]cesium.ChannelKey{index}, data...),
					Start:    10 * telem.SecondTS,
				}))
				series := []telem.Series{telem.NewSeriesSecondsTSV(10, 11, 12, 13)}
				for range data {
					series = append(series, telem.NewSeriesV[int64](1, 2, 3, 4))
				}
				mustWrite(ctx, w, telem.MultiFrame(
					append([]cesium.ChannelKey{index}, data...),
					series,
				))

				skipped := data[0]
				end, refused := MustSucceed2(w.Commit(
					ctx,
					func(k cesium.ChannelKey) bool { return k == skipped },
				))
				Expect(refused).To(BeTrue())
				Expect(end).To(BeZero())
				Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, skipped)).Count()).
					To(BeZero())
				for _, k := range data[1:] {
					fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, k))
					Expect(fr.Count()).To(Equal(1), "round %d, channel %d", round, k)
				}
				Expect(w.Close()).To(Succeed())
			}
		},
	)
})

var _ = Describe("Writer Write with a skip", func() {
	It("Should allocate nothing for a write that skips every channel", func(
		ctx SpecContext,
	) {
		db := mustOpenDBOnFS(ctx, xfs.NewMem())
		index, data := GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      index,
				Name:     "index",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      data,
				Name:     "data",
				Index:    index,
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{index, data},
			Start:    10 * telem.SecondTS,
		}))
		fr := telem.MultiFrame([]cesium.ChannelKey{index, data}, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11),
			telem.NewSeriesV[int64](1, 2),
		})
		skipAll := func(cesium.ChannelKey) bool { return true }
		var (
			skipped bool
			err     error
		)
		allocs := testing.AllocsPerRun(100, func() {
			_, skipped, err = w.Write(ctx, fr, skipAll)
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(skipped).To(BeTrue())
		Expect(allocs).To(BeZero())
		Expect(w.Close()).To(Succeed())
	})
})
