// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

var _ = Describe("Persist with partial control", func() {
	var (
		fs    *xfs.MemFS
		db    *cesium.DB
		svc   *writer.Service
		idxA  = channel.NewKey(1, 1)
		dataA = channel.NewKey(1, 2)
		idxB  = channel.NewKey(1, 3)
		dataB = channel.NewKey(1, 4)
	)
	BeforeEach(func(ctx SpecContext) {
		fs = xfs.NewMem()
		db = MustOpen(cesium.Open(ctx, "db", cesium.WithFS(fs)))
		StartRun(db.Run)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      uint32(idxA),
				Name:     "a_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      uint32(dataA),
				Name:     "a",
				Index:    uint32(idxA),
				DataType: telem.Int64T,
			},
			cesium.Channel{
				Key:      uint32(idxB),
				Name:     "b_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      uint32(dataB),
				Name:     "b",
				Index:    uint32(idxB),
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		svc = MustSucceed(writer.NewService(writer.ServiceConfig{
			TS:     db,
			Writes: confluence.NewStream[relay.Response](100),
			Clock:  timetestutil.NewClock(time.Unix(0, 0)),
		}))
	})
	durable := func(ctx SpecContext, key channel.Key) int64 {
		GinkgoHelper()
		probe := MustSucceed(cesium.Open(ctx, "db", cesium.WithFS(fs)))
		defer func() { Expect(probe.Close()).To(Succeed()) }()
		return MustSucceed(probe.Read(ctx, telem.TimeRangeMax, uint32(key))).
			Get(uint32(key)).Len()
	}
	It(
		"Should make the commit of a controlled index durable when another index lacks control",
		func(ctx SpecContext) {
			keys := channel.Keys{idxA, dataA, idxB, dataB}
			low := MustSucceed(svc.Open(ctx, writer.Config{
				ControlSubject:   control.Subject{Key: "low"},
				Keys:             keys,
				Start:            10 * telem.SecondTS,
				Authorities:      []control.Authority{100},
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			Expect(low.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(10, 11),
				telem.NewSeriesV[int64](1, 2),
				telem.NewSeriesSecondsTSV(10, 11),
				telem.NewSeriesV[int64](3, 4),
			}))).To(BeTrue())
			high := MustSucceed(svc.Open(ctx, writer.Config{
				ControlSubject:   control.Subject{Key: "high"},
				Keys:             channel.Keys{idxB, dataB},
				Start:            10 * telem.SecondTS,
				Authorities:      []control.Authority{200},
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			_, authorized := MustSucceed2(low.Commit())
			Expect(authorized).To(BeFalse())
			Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, uint32(dataA))).
				Get(uint32(dataA)).Len()).To(Equal(int64(2)))
			Expect(durable(ctx, dataA)).To(Equal(int64(2)))
			Expect(high.Close()).To(Succeed())
			Expect(low.Close()).To(Succeed())
		},
	)
})
