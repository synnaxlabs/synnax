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
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

var _ = Describe("Persist", func() {
	var (
		fs    *xfs.MemFS
		db    *cesium.DB
		clock *timetestutil.Clock
		svc   *writer.Service
		idx   = channel.NewKey(1, 1)
		data  = channel.NewKey(1, 2)
		keys  = channel.Keys{idx, data}
	)
	BeforeEach(func(ctx SpecContext) {
		fs = xfs.NewMem()
		db = MustOpen(cesium.Open(ctx, "db", cesium.WithFS(fs)))
		StartRun(db.Run)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      uint32(idx),
				Name:     "persist_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      uint32(data),
				Name:     "persist",
				Index:    uint32(idx),
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		clock = timetestutil.NewClock(time.Unix(0, 0))
		svc = MustSucceed(writer.NewService(writer.ServiceConfig{
			TS:     db,
			Writes: confluence.NewStream[relay.Response](100),
			Clock:  clock,
		}))
	})
	open := func(
		ctx SpecContext,
		autoCommit bool,
		interval telem.TimeSpan,
	) *writer.Writer {
		GinkgoHelper()
		return MustSucceed(svc.Open(ctx, writer.Config{
			Keys:                     keys,
			Start:                    10 * telem.SecondTS,
			EnableAutoCommit:         new(autoCommit),
			AutoIndexPersistInterval: interval,
			Sync:                     new(true),
		}))
	}
	write := func(w *writer.Writer, seconds ...telem.TimeStamp) {
		GinkgoHelper()
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(make([]int64, len(seconds))),
		}))).To(BeTrue())
	}
	// visible returns the number of samples of the data channel that reads see.
	visible := func(ctx SpecContext) int64 {
		GinkgoHelper()
		return MustSucceed(db.Read(ctx, telem.TimeRangeMax, uint32(data))).
			Get(uint32(data)).Len()
	}
	// durable returns the number of samples of the data channel that a DB opened on the
	// same files sees, as a restart after a crash would.
	durable := func(ctx SpecContext) int64 {
		GinkgoHelper()
		probe := MustSucceed(cesium.Open(ctx, "db", cesium.WithFS(fs)))
		defer func() { Expect(probe.Close()).To(Succeed()) }()
		return MustSucceed(probe.Read(ctx, telem.TimeRangeMax, uint32(data))).
			Get(uint32(data)).Len()
	}

	It(
		"Should make every commit durable when the writer does not auto commit",
		func(ctx SpecContext) {
			w := open(ctx, false, 0)
			write(w, 10, 11)
			Expect(visible(ctx)).To(BeZero())
			MustCommit(w)
			Expect(durable(ctx)).To(Equal(int64(2)))
			Expect(w.Close()).To(Succeed())
		},
	)

	It(
		"Should make an auto commit durable once the persist interval passes",
		func(ctx SpecContext) {
			w := open(ctx, true, 200*telem.Millisecond)
			write(w, 10)
			Expect(visible(ctx)).To(Equal(int64(1)))
			Expect(durable(ctx)).To(BeZero())
			clock.Advance(199 * time.Millisecond)
			write(w, 11)
			Expect(durable(ctx)).To(BeZero())
			clock.Advance(time.Millisecond)
			write(w, 12)
			Expect(durable(ctx)).To(Equal(int64(3)))
			write(w, 13)
			Expect(visible(ctx)).To(Equal(int64(4)))
			Expect(durable(ctx)).To(Equal(int64(3)))
			Expect(w.Close()).To(Succeed())
		},
	)

	It(
		"Should make every auto commit durable when the interval is AlwaysIndexPersistOnAutoCommit",
		func(ctx SpecContext) {
			w := open(ctx, true, writer.AlwaysIndexPersistOnAutoCommit)
			write(w, 10)
			Expect(durable(ctx)).To(Equal(int64(1)))
			write(w, 11)
			Expect(durable(ctx)).To(Equal(int64(2)))
			Expect(w.Close()).To(Succeed())
		},
	)

	It(
		"Should default the persist interval of an auto commit to one second",
		func(ctx SpecContext) {
			w := MustSucceed(svc.Open(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			write(w, 10)
			clock.Advance(999 * time.Millisecond)
			write(w, 11)
			Expect(durable(ctx)).To(BeZero())
			clock.Advance(time.Millisecond)
			write(w, 12)
			Expect(durable(ctx)).To(Equal(int64(3)))
			Expect(w.Close()).To(Succeed())
		},
	)

	It(
		"Should make every commit durable when the writer closes",
		func(ctx SpecContext) {
			w := open(ctx, true, telem.Hour)
			write(w, 10, 11)
			Expect(durable(ctx)).To(BeZero())
			Expect(w.Close()).To(Succeed())
			Expect(durable(ctx)).To(Equal(int64(2)))
		},
	)
})
