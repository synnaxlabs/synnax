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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Index after a crash", func() {
	It(
		"Should not return more samples than timestamps after a crash loses the tail of "+
			"the index file",
		func(ctx SpecContext) {
			const (
				index cesium.ChannelKey = 1
				data  cesium.ChannelKey = 2
				count                   = 1400
			)
			disk := MustSucceed(fault.New(fault.Config{}))
			db := openDBOnFS(ctx, disk)
			stop := GoRun(ctx, db.Run)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      index,
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      data,
					Name:     "data",
					Index:    index,
					DataType: telem.Uint8T,
				},
			)).To(Succeed())
			for second := range telem.TimeStamp(2) {
				start := (second + 1) * telem.SecondTS
				stamps := make([]telem.TimeStamp, count)
				for i := range stamps {
					stamps[i] = start + telem.TimeStamp(i)
				}
				frame := telem.MultiFrame(
					[]cesium.ChannelKey{index, data},
					[]telem.Series{
						telem.NewSeries(stamps),
						telem.NewSeries(make([]uint8, count)),
					},
				)
				Expect(db.Write(ctx, start, frame)).To(Succeed())
			}
			Expect(stop()).To(Succeed())
			Expect(db.Close()).To(Succeed())
			// The pointer file and the data file are durable, and the newest 4 KiB of
			// the index file is not: the state that ext4 with data=ordered can leave
			// after a power loss, as it orders no data across two files.
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				DataSurvivalRate:  1,
				EntrySurvivalRate: 1,
			}))
			f := MustSucceed(image.Open("1/1.domain", os.O_RDWR))
			Expect(f.Truncate(MustSucceed(f.Stat()).Size() - 4096)).To(Succeed())
			Expect(f.Close()).To(Succeed())
			db = openDBOnFS(ctx, image)
			defer func() { Expect(db.Close()).To(Succeed()) }()
			frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, index, data))
			samples := map[telem.TimeRange][2]int64{}
			for i, key := range frame.KeysSlice() {
				s := frame.SeriesAt(i)
				got := samples[s.TimeRange]
				if key == index {
					got[0] = s.Len()
				} else {
					got[1] = s.Len()
				}
				samples[s.TimeRange] = got
			}
			for tr, got := range samples {
				Expect(got[1]).To(BeNumerically("<=", got[0]), "time range %s", tr)
			}
		},
	)
})
