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
	"os"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
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

// openCrashEngine returns an Open function that reads each Cesium commit as ack.
func openCrashEngine(
	ack crash.Ack,
) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
		db, err := cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithInstrumentation(PanicLogger()),
		)
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
	if err := commitFrame(ctx, e.db, start, frame); err != nil {
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

// run runs the harness on the adapter with one reading of a commit, one fault profile,
// and one run key. As Durable, a commit promises to survive a crash. As Visible, it
// promises only that a read sees it.
func run(
	ctx context.Context,
	ack crash.Ack,
	profile crash.Profile,
	runKey uint64,
	ordered bool,
) error {
	_, err := crash.Run(ctx, crash.Config{
		RunKey:         runKey,
		Open:           openCrashEngine(ack),
		Commits:        8,
		Stride:         1,
		Profile:        profile,
		EntriesOrdered: ordered,
	})
	return err
}

// Each entry is what the harness finds in Cesium today, and why.
var _ = DescribeTable("Crash harness",
	func(
		ctx SpecContext,
		ack crash.Ack,
		profile crash.Profile,
		runKey uint64,
		rule crash.Rule,
		cause string,
	) {
		var v *crash.Violation
		err := run(ctx, ack, profile, runKey, false)
		Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
		Expect(v.Rule).To(Equal(rule))
		Expect(v.Detail).To(ContainSubstring(cause))
	},
	// No commit syncs a file, so a crash loses commits that Cesium acked.
	Entry("Durable, Strict: R1", crash.Durable, crash.Strict, uint64(0), crash.R1,
		"durable commit 1 is missing"),
	Entry("Durable, SyncErrors: R1", crash.Durable, crash.SyncErrors, uint64(0),
		crash.R1, "durable commit 1 is missing"),
	Entry("Durable, FullDisk: R1", crash.Durable, crash.FullDisk, uint64(0), crash.R1,
		"durable commit 1 is missing"),
	Entry("Durable, OldCorruption: R1", crash.Durable, crash.OldCorruption, uint64(0),
		crash.R1, "durable commit 1 is missing"),
	Entry("Durable, NewCorruption: R1", crash.Durable, crash.NewCorruption, uint64(0),
		crash.R1, "durable commit 1 is missing"),
	// A channel directory that a crash left with no meta file stops the open of the
	// whole database.
	Entry("Durable, Partial: R4", crash.Durable, crash.Partial, uint64(0), crash.R4,
		"data_type: required"),
	Entry("Visible, Partial: R4", crash.Visible, crash.Partial, uint64(0), crash.R4,
		"data_type: required"),
	Entry("Durable, Mixed: R4", crash.Durable, crash.Mixed, uint64(0), crash.R4,
		"data_type: required"),
	Entry("Visible, Mixed: R4", crash.Visible, crash.Mixed, uint64(0), crash.R4,
		"data_type: required"),
	// A crash image holds no file to read, so the read faults change nothing.
	Entry("Durable, OneTimeReadFaults: R1", crash.Durable, crash.OneTimeReadFaults,
		uint64(0), crash.R1, "durable commit 1 is missing"),
	Entry("Durable, RepeatedReadFaults: R1", crash.Durable, crash.RepeatedReadFaults,
		uint64(0), crash.R1, "durable commit 1 is missing"),
)

// With the directory changes in order, as ext4, XFS, APFS, and NTFS journals keep them,
// Cesium breaks the same rules: it syncs no directory, so a crash still keeps a channel
// directory and loses its meta file.
var _ = DescribeTable("Crash harness, ordered entries",
	func(
		ctx SpecContext,
		ack crash.Ack,
		profile crash.Profile,
		rule crash.Rule,
		cause string,
	) {
		var v *crash.Violation
		err := run(ctx, ack, profile, 0, true)
		Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
		Expect(v.Rule).To(Equal(rule))
		Expect(v.Detail).To(ContainSubstring(cause))
	},
	Entry("Durable, Partial: R4", crash.Durable, crash.Partial, crash.R4,
		"data_type: required"),
	Entry("Visible, Partial: R4", crash.Visible, crash.Partial, crash.R4,
		"data_type: required"),
	Entry("Durable, Mixed: R4", crash.Durable, crash.Mixed, crash.R4,
		"data_type: required"),
	Entry("Visible, Mixed: R4", crash.Visible, crash.Mixed, crash.R4,
		"data_type: required"),
)

// Each entry holds only because Cesium syncs nothing: each crash image is empty, so the
// harness checks only R0.
var _ = DescribeTable("Crash harness, holding",
	func(ctx SpecContext, ack crash.Ack, profile crash.Profile, runKey uint64) {
		Expect(run(ctx, ack, profile, runKey, false)).To(Succeed())
	},
	Entry("Visible, Strict", crash.Visible, crash.Strict, uint64(0)),
	Entry("Visible, SyncErrors", crash.Visible, crash.SyncErrors, uint64(0)),
	Entry("Visible, FullDisk", crash.Visible, crash.FullDisk, uint64(0)),
	Entry("Visible, OldCorruption", crash.Visible, crash.OldCorruption, uint64(0)),
	Entry("Visible, NewCorruption", crash.Visible, crash.NewCorruption, uint64(0)),
	Entry("Visible, OneTimeReadFaults", crash.Visible, crash.OneTimeReadFaults,
		uint64(0)),
	Entry("Visible, RepeatedReadFaults", crash.Visible, crash.RepeatedReadFaults,
		uint64(0)),
	// The disk fills on the first commit, so the engine acks nothing.
	Entry("Durable, FullDisk, run key 1", crash.Durable, crash.FullDisk, uint64(1)),
)

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
		Expect(commitFrame(ctx, e.db, start, telem.MultiFrame(keys, series))).
			To(Succeed())
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
			Expect(commitFrame(ctx, e.db, after, extra)).To(Succeed())
			records := testutil.MustSucceed(e.Recovered(ctx))
			Expect(records).To(HaveLen(1))
			Expect(records[0].ID).To(Equal(uint64(2)))
			Expect(records[0].Payload).To(BeEmpty())
		},
	)

	DescribeTable(
		"Should return no payload for a commit whose clock file a crash changed",
		func(ctx SpecContext, change func(f xfs.File, size int64) error) {
			open := openCrashEngine(crash.Visible)
			disk := testutil.MustSucceed(fault.New(fault.Config{}))
			e := testutil.MustSucceed(open(ctx, disk))
			payload := bytes.Repeat([]byte{7}, 1400)
			for id := uint64(1); id <= 2; id++ {
				testutil.MustSucceed(
					e.Commit(
						ctx,
						crash.Record{ID: id, Payload: payload},
						crash.Visible,
					),
				)
			}
			Expect(e.Close()).To(Succeed())
			image := testutil.MustSucceed(disk.Crash(fault.CrashConfig{
				DataSurvivalRate:  1,
				EntrySurvivalRate: 1,
			}))
			f := testutil.MustSucceed(image.Open("1/1.domain", os.O_RDWR))
			Expect(change(f, testutil.MustSucceed(f.Stat()).Size())).To(Succeed())
			Expect(f.Close()).To(Succeed())
			e = testutil.MustSucceed(open(ctx, image))
			defer func() { Expect(e.Close()).To(Succeed()) }()
			records := testutil.MustSucceed(e.Recovered(ctx))
			Expect(records).To(HaveLen(2))
			Expect(records[0]).To(Equal(crash.Record{ID: 1, Payload: payload}))
			Expect(records[1].ID).To(Equal(uint64(2)))
			Expect(records[1].Payload).To(BeEmpty())
		},
		Entry("a block of zeros", func(f xfs.File, size int64) error {
			_, err := f.WriteAt(make([]byte, 4096), size-4096)
			return err
		}),
		Entry("a lost tail", func(f xfs.File, size int64) error {
			return f.Truncate(size - 4096)
		}),
	)
})

// commitFrame writes fr from start with one writer on the clock of its first channel,
// commits it, and closes the writer.
func commitFrame(
	ctx context.Context,
	db *cesium.DB,
	start telem.TimeStamp,
	fr cesium.Frame,
) error {
	keys := fr.KeysSlice()
	first, err := db.RetrieveChannel(ctx, keys[0])
	if err != nil {
		return err
	}
	w, err := db.OpenWriter(ctx, cesium.WriterConfig{
		Channels: keys,
		Clock:    first.Clock(),
		Start:    start,
	})
	if err != nil {
		return err
	}
	if _, _, err = w.Write(ctx, fr, nil); err == nil {
		_, _, err = w.Commit(ctx, nil)
	}
	return errors.Join(err, w.Close())
}
