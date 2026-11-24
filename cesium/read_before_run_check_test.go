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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: a read before Run", func() {
	It("Should read the samples of a reopened pointer log, or wait for Run, but "+
		"never read it as empty", func(ctx SpecContext) {
		const clock, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
		keys := []cesium.ChannelKey{clock, data}
		fs := xfs.NewMem()
		db, closeDB := MustSucceed2(openLog(ctx, fs))
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      clock,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      data,
				Name:     "data",
				Index:    clock,
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		Expect(write(ctx, db, 10*telem.SecondTS, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11),
			telem.NewSeriesV[int64](1, 2),
		}))).To(Succeed())
		Expect(closeDB()).To(Succeed())
		db = MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		early, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		fr, err := db.Read(early, telem.TimeRangeMax, data)
		cancel()
		if err != nil {
			Expect(err).To(MatchError(context.DeadlineExceeded))
		} else {
			Expect(fr.Get(data).Len()).To(Equal(int64(2)))
		}
		stop := GoRun(ctx, db.Run)
		Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).Get(data).Len()).
			To(Equal(int64(2)))
		Expect(stop()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
})
