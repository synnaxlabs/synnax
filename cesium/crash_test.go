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
	"bytes"
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

const (
	crashClock cesium.ChannelKey = iota + 1
	crashData
	crashMirror
)

// crashEngine adapts Cesium to the crash harness. One record is one DB.Write of a frame
// at the second of its ID: one timestamp for each payload byte on the clock, the
// payload on a data channel, and the payload in reverse on a second data channel.
type crashEngine struct {
	db   *cesium.DB
	stop func() error
	// ack is the promise that the adapter reads into a commit of Cesium.
	ack crash.Ack
}

// openCrashEngine returns an Open function that reads each Cesium commit as ack, and
// opens Cesium with opts.
func openCrashEngine(
	ack crash.Ack,
	opts ...func() cesium.Option,
) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
		options := make([]cesium.Option, 0, 2+len(opts))
		options = append(
			options,
			cesium.WithFS(fs),
			cesium.WithInstrumentation(PanicLogger()),
		)
		for _, opt := range opts {
			options = append(options, opt())
		}
		db, err := cesium.Open(ctx, "", options...)
		if err != nil {
			return nil, err
		}
		e := &crashEngine{db: db, stop: testutil.GoRun(ctx, db.Run), ack: ack}
		for _, ch := range []cesium.Channel{
			{Key: crashClock, Name: "clock", IsIndex: true, DataType: telem.TimestampT},
			{Key: crashData, Name: "data", Index: crashClock, DataType: telem.Uint8T},
			{Key: crashMirror, Name: "mirror", Index: crashClock, DataType: telem.Uint8T},
		} {
			_, err = db.RetrieveChannel(ctx, ch.Key)
			if errors.Is(err, cesium.ErrChannelNotFound) {
				err = db.CreateChannel(ctx, ch)
			}
			if err != nil {
				return nil, errors.Combine(err, e.Close())
			}
		}
		return e, nil
	}
}

func reversed(payload []byte) []byte {
	out := slices.Clone(payload)
	slices.Reverse(out)
	return out
}

// clock returns the timestamps of the record id with n payload bytes: one for each
// byte, from the second of id.
func clock(id uint64, n int) telem.Series {
	start := telem.TimeStamp(id) * telem.SecondTS
	stamps := make([]telem.TimeStamp, n)
	for i := range stamps {
		stamps[i] = start + telem.TimeStamp(i)
	}
	return telem.NewSeries(stamps)
}

func (e *crashEngine) Commit(
	ctx context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	frame := telem.MultiFrame(
		[]cesium.ChannelKey{crashClock, crashData, crashMirror},
		[]telem.Series{
			clock(r.ID, len(r.Payload)),
			telem.NewSeries(r.Payload),
			telem.NewSeries(reversed(r.Payload)),
		},
	)
	start := telem.TimeStamp(r.ID) * telem.SecondTS
	if err := e.db.Write(ctx, start, frame); err != nil {
		return 0, err
	}
	return e.ack, nil
}

// Recovered reads all three channels. A commit whose channels do not agree comes back
// with no payload, so the harness sees a wrong payload.
func (e *crashEngine) Recovered(ctx context.Context) ([]crash.Record, error) {
	frame, err := e.db.Read(ctx, telem.TimeRangeMax, crashClock, crashData, crashMirror)
	if err != nil {
		return nil, err
	}
	parts := map[uint64]map[cesium.ChannelKey][]byte{}
	for i, key := range frame.KeysSlice() {
		s := frame.SeriesAt(i)
		id := uint64(s.TimeRange.Start / telem.SecondTS)
		if parts[id] == nil {
			parts[id] = map[cesium.ChannelKey][]byte{}
		}
		parts[id][key] = append(parts[id][key], s.Data...)
	}
	records := make([]crash.Record, 0, len(parts))
	for id, p := range parts {
		r := crash.Record{ID: id}
		data := p[crashData]
		whole := bytes.Equal(p[crashClock], clock(id, len(data)).Data) &&
			bytes.Equal(p[crashMirror], reversed(data))
		if whole {
			r.Payload = data
		}
		records = append(records, r)
	}
	slices.SortFunc(records, func(a, b crash.Record) int {
		return int(a.ID) - int(b.ID)
	})
	return records, nil
}

func (e *crashEngine) Close() error { return errors.Combine(e.stop(), e.db.Close()) }

var _ = Describe("Crash adapter", func() {
	// open returns an adapter on an empty disk in memory.
	open := func(ctx SpecContext) *crashEngine {
		GinkgoHelper()
		e := testutil.MustSucceed(openCrashEngine(crash.Durable)(ctx, xfs.NewMem()))
		engine, ok := e.(*crashEngine)
		Expect(ok).To(BeTrue())
		return engine
	}
	// writeParts writes the record id to the channels of keys alone, as a crash that
	// keeps some of its files would leave it.
	writeParts := func(
		ctx SpecContext,
		e *crashEngine,
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
		Expect(e.db.Write(ctx, start, telem.MultiFrame(keys, series))).To(Succeed())
	}

	DescribeTable("Should return no payload for a commit whose channels do not agree",
		func(ctx SpecContext, keys []cesium.ChannelKey) {
			e := open(ctx)
			defer func() { Expect(e.Close()).To(Succeed()) }()
			payload := []byte("tacocat burrito")
			testutil.MustSucceed(
				e.Commit(ctx, crash.Record{ID: 1, Payload: payload}, crash.Durable),
			)
			writeParts(ctx, e, 2, payload, keys)
			records := testutil.MustSucceed(e.Recovered(ctx))
			Expect(records).To(HaveLen(2))
			Expect(records[0]).To(Equal(crash.Record{ID: 1, Payload: payload}))
			Expect(records[1].ID).To(Equal(uint64(2)))
			Expect(records[1].Payload).To(BeEmpty())
		},
		Entry("no mirror", []cesium.ChannelKey{crashClock, crashData}),
		Entry("no data", []cesium.ChannelKey{crashClock, crashMirror}),
	)

	It("Should return no payload for a commit whose clock is longer than its data",
		func(ctx SpecContext) {
			e := open(ctx)
			defer func() { Expect(e.Close()).To(Succeed()) }()
			payload := []byte("tacocat burrito")
			testutil.MustSucceed(
				e.Commit(ctx, crash.Record{ID: 2, Payload: payload}, crash.Durable),
			)
			// One more stamp in the same second, as a clock file that kept more than
			// its data files would read.
			after := 2*telem.SecondTS + telem.TimeStamp(len(payload))
			extra := telem.MultiFrame(
				[]cesium.ChannelKey{crashClock},
				[]telem.Series{telem.NewSeries([]telem.TimeStamp{after})},
			)
			Expect(e.db.Write(ctx, after, extra)).To(Succeed())
			records := testutil.MustSucceed(e.Recovered(ctx))
			Expect(records).To(HaveLen(1))
			Expect(records[0].ID).To(Equal(uint64(2)))
			Expect(records[0].Payload).To(BeEmpty())
		},
	)
})
