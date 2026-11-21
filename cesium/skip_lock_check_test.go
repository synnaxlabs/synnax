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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Skip check under the claim lock", func() {
	// flips returns a skip that is false on its first call and true after, as when a
	// writer loses control between the check with no lock and the claim lock.
	flips := func() func(cesium.ChannelKey) bool {
		calls := 0
		return func(cesium.ChannelKey) bool {
			calls++
			return calls > 1
		}
	}
	open := func(ctx SpecContext) (*cesium.DB, *cesium.Writer, cesium.ChannelKey) {
		db := mustOpenDBOnFS(ctx, xfs.NewMem())
		index := GenerateChannelKey()
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: index, Name: "index", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{index},
			Start:    10 * telem.SecondTS,
		}))
		return db, w, index
	}
	frame := func(index cesium.ChannelKey) cesium.Frame {
		return telem.MultiFrame(
			[]cesium.ChannelKey{index},
			[]telem.Series{telem.NewSeriesSecondsTSV(10, 11)},
		)
	}

	It("Should skip a write when skip turns true under the claim lock", func(
		ctx SpecContext,
	) {
		db, w, index := open(ctx)
		written, skipped := MustSucceed2(w.Write(ctx, frame(index), flips()))
		Expect(skipped).To(BeTrue())
		Expect(written.Count()).To(BeZero())
		MustSucceed2(w.Commit(ctx, nil))
		Expect(
			MustSucceed(db.Read(ctx, telem.TimeRangeMax, index)).Count(),
		).To(BeZero())
		Expect(w.Close()).To(Succeed())
	})

	It("Should skip a commit when skip turns true under the claim lock", func(
		ctx SpecContext,
	) {
		db, w, index := open(ctx)
		MustSucceed2(w.Write(ctx, frame(index), nil))
		end, refused := MustSucceed2(w.Commit(ctx, flips()))
		Expect(refused).To(BeTrue())
		Expect(end).To(BeZero())
		Expect(
			MustSucceed(db.Read(ctx, telem.TimeRangeMax, index)).Count(),
		).To(BeZero())
		Expect(w.Close()).To(Succeed())
	})
})
