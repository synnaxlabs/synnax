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
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Crash adapter", func() {
	// writeParts writes the record id to the channels of keys alone, as a crash that
	// keeps some of its files would leave it.
	writeParts := func(
		ctx SpecContext,
		e crash.Engine,
		id uint64,
		payload []byte,
		keys []cesium.ChannelKey,
	) {
		GinkgoHelper()
		start := telem.TimeStamp(id) * telem.SecondTS
		stamps := make([]telem.TimeStamp, len(payload))
		for i := range stamps {
			stamps[i] = start + telem.TimeStamp(i)
		}
		all := map[cesium.ChannelKey]telem.Series{
			crashClock:  telem.NewSeries(stamps),
			crashData:   telem.NewSeries(payload),
			crashMirror: telem.NewSeries(reversed(payload)),
		}
		series := make([]telem.Series, len(keys))
		for i, k := range keys {
			series[i] = all[k]
		}
		db := e.(*crashEngine).db
		Expect(db.Write(ctx, start, telem.MultiFrame(keys, series))).To(Succeed())
	}

	DescribeTable("Should return no payload for a commit whose channels do not agree",
		func(ctx SpecContext, keys []cesium.ChannelKey) {
			e := MustSucceed(openCrashEngine(crash.Durable)(ctx, xfs.NewMem()))
			defer func() { Expect(e.Close()).To(Succeed()) }()
			payload := []byte("tacocat burrito")
			MustSucceed(
				e.Commit(ctx, crash.Record{ID: 1, Payload: payload}, crash.Durable),
			)
			writeParts(ctx, e, 2, payload, keys)
			records := MustSucceed(e.Recovered(ctx))
			Expect(records).To(HaveLen(2))
			Expect(records[0]).To(Equal(crash.Record{ID: 1, Payload: payload}))
			Expect(records[1].ID).To(Equal(uint64(2)))
			Expect(records[1].Payload).To(BeEmpty())
		},
		Entry("no mirror", []cesium.ChannelKey{crashClock, crashData}),
		Entry("no data", []cesium.ChannelKey{crashClock, crashMirror}),
	)
})
