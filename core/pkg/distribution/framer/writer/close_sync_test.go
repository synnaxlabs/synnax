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

var _ = DescribeTableSubtree("Close sync", func(pointerLog bool) {
	const interval = 2 * telem.Second
	var (
		db           *cesium.DB
		clock        *timetestutil.Clock
		svc          *writer.Service
		idx          = channel.NewKey(1, 1)
		data         = channel.NewKey(1, 2)
		stamps, vals *writer.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		db = MustOpen(cesium.Open(ctx, "db", append(
			layout(pointerLog),
			cesium.WithFS(xfs.NewMem()),
		)...))
		StartRun(db.Run)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      uint32(idx),
				Name:     "close_sync_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      uint32(data),
				Name:     "close_sync",
				Index:    uint32(idx),
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		clock = timetestutil.NewClock(time.Unix(0, 0))
		svc = MustSucceed(writer.NewService(writer.ServiceConfig{
			TS:                db,
			Writes:            confluence.NewStream[relay.Response](100),
			Clock:             clock,
			PointerLogEnabled: new(pointerLog),
		}))
	})
	// open opens the writers of the stamps and of the values, and writes stamps 10 s to
	// 12 s and values 1 to 3 on them.
	open := func(ctx SpecContext) {
		GinkgoHelper()
		openOne := func(subject string, key channel.Key, persist telem.TimeSpan) *writer.Writer {
			GinkgoHelper()
			return MustSucceed(svc.Open(ctx, writer.Config{
				ControlSubject:           control.Subject{Key: subject},
				Keys:                     channel.Keys{key},
				Start:                    10 * telem.SecondTS,
				EnableAutoCommit:         new(true),
				AutoIndexPersistInterval: persist,
				Sync:                     new(true),
			}))
		}
		// The clock syncs no commit until the spec moves the clock past an hour.
		stamps = openOne("stamps", idx, telem.Hour)
		vals = openOne("values", data, interval)
		Expect(stamps.Write(frame.NewUnary(idx, telem.NewSeriesSecondsTSV(10)))).
			To(BeTrue())
		Expect(vals.Write(frame.NewUnary(data, telem.NewSeriesV[int64](1)))).
			To(BeTrue())
		Expect(stamps.Write(frame.NewUnary(idx, telem.NewSeriesSecondsTSV(11, 12)))).
			To(BeTrue())
		Expect(vals.Write(frame.NewUnary(data, telem.NewSeriesV[int64](2, 3)))).
			To(BeTrue())
	}
	stored := func(ctx SpecContext) []int64 {
		GinkgoHelper()
		series := MustSucceed(db.Read(ctx, telem.TimeRangeMax, uint32(data))).
			Get(uint32(data))
		out := make([]int64, 0, series.Len())
		for _, s := range series.Series {
			out = append(out, s.Unmarshal[int64]()...)
		}
		return out
	}

	It(
		"Should keep the last commits of a session that writes data against stored "+
			"stamps when the clock syncs while the session closes",
		func(ctx SpecContext) {
			open(ctx)
			closed := make(chan error, 1)
			go func() { closed <- vals.Close() }()
			if pointerLog {
				Consistently(closed, interval.Duration()/2).ShouldNot(Receive())
			}
			clock.Advance(2 * time.Hour)
			Expect(stamps.Write(frame.NewUnary(idx, telem.NewSeriesSecondsTSV(13)))).
				To(BeTrue())
			Eventually(closed).Should(Receive(BeNil()))
			Expect(stamps.Close()).To(Succeed())
			Expect(stored(ctx)).To(Equal([]int64{1, 2, 3}))
		},
	)

	It(
		"Should return within one persist interval when the clock never syncs",
		func(ctx SpecContext) {
			open(ctx)
			start := time.Now()
			if pointerLog {
				Expect(vals.Close()).To(MatchError(ContainSubstring(
					"has not made the stamps from 10000000000 to 12000000000 durable",
				)))
				Expect(time.Since(start)).To(BeNumerically("<", 2*interval.Duration()))
				Expect(stored(ctx)).To(BeEmpty())
			} else {
				Expect(vals.Close()).To(Succeed())
				Expect(stored(ctx)).To(Equal([]int64{1, 2, 3}))
			}
			Expect(stamps.Close()).To(Succeed())
		},
	)
}, Entry("old layout", false), Entry("pointer log", true))
