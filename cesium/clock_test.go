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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Clock", func() {
	It("Should start a writer that opens with no start at the time of the clock", func(
		ctx SpecContext,
	) {
		clock := timetestutil.NewClock(time.Unix(500, 0))
		db := DeferClose(MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(xfs.NewMem()),
			cesium.WithClock(clock),
		)))
		StartRun(db.Run)
		idx := GenerateChannelKey()
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
		keys := []cesium.ChannelKey{idx}
		live := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
		}))
		fill := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    400 * telem.SecondTS,
			Bounded:  true,
		}))
		mustWrite(ctx, fill, telem.UnaryFrame(idx, telem.NewSeriesSecondsTSV(400, 550)))
		// The live writer holds the time from the clock's reading, 500 s, on.
		claimed := SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"cannot write to time range 1970-01-01T00:08:20Z",
			)),
		)
		Expect(fill.Commit(ctx, nil)).Error().To(claimed)
		Expect(fill.Close()).To(claimed)
		Expect(live.Close()).To(Succeed())
	})
})
