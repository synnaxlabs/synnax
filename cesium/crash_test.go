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

// openCrashEngine returns an Open function that reads each Cesium commit as ack.
func openCrashEngine(
	ack crash.Ack,
) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
		db, err := cesium.Open(ctx, "", cesium.WithFS(fs))
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

func (e *crashEngine) Commit(
	ctx context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	start := telem.TimeStamp(r.ID) * telem.SecondTS
	stamps := make([]telem.TimeStamp, len(r.Payload))
	for i := range stamps {
		stamps[i] = start + telem.TimeStamp(i)
	}
	frame := telem.MultiFrame(
		[]cesium.ChannelKey{crashClock, crashData, crashMirror},
		[]telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(r.Payload),
			telem.NewSeries(reversed(r.Payload)),
		},
	)
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
		whole := len(p[crashClock]) == 8*len(data) &&
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

// holds is the rule of a run that breaks no rule.
const holds crash.Rule = ""

// Each entry says what the harness finds in Cesium for one reading of a commit and one
// fault profile, at run key 0. As Durable, a commit promises to survive a crash. As
// Visible, it promises only that a read sees it.
var _ = DescribeTable("Crash harness",
	func(ctx SpecContext, ack crash.Ack, profile crash.Profile, rule crash.Rule) {
		err := crash.Run(ctx, crash.Config{
			Open:    openCrashEngine(ack),
			Commits: 8,
			Stride:  1,
			Profile: profile,
		})
		if rule == holds {
			Expect(err).To(Succeed())
			return
		}
		var v *crash.Violation
		Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
		Expect(v.Rule).To(Equal(rule))
	},
	// No commit syncs a file, so a crash loses commits that Cesium acked.
	Entry("Durable, Strict", crash.Durable, crash.Strict, crash.R1),
	Entry("Durable, SyncErrors", crash.Durable, crash.SyncErrors, crash.R1),
	Entry("Durable, FullDisk", crash.Durable, crash.FullDisk, crash.R1),
	Entry("Durable, OldCorruption", crash.Durable, crash.OldCorruption, crash.R1),
	Entry("Durable, NewCorruption", crash.Durable, crash.NewCorruption, crash.R1),
	// A channel directory with a torn or missing meta file stops the open of the whole
	// database.
	Entry("Durable, Partial", crash.Durable, crash.Partial, crash.R4),
	Entry("Durable, Mixed", crash.Durable, crash.Mixed, crash.R4),
	Entry("Visible, Partial", crash.Visible, crash.Partial, crash.R4),
	Entry("Visible, Mixed", crash.Visible, crash.Mixed, crash.R4),
	// One bad read of a meta file stops the open: no checksum, and no second read.
	Entry("Durable, OneTimeReadFaults",
		crash.Durable, crash.OneTimeReadFaults, crash.R4),
	Entry("Visible, OneTimeReadFaults",
		crash.Visible, crash.OneTimeReadFaults, crash.R4),
	Entry("Visible, Strict", crash.Visible, crash.Strict, holds),
	Entry("Visible, SyncErrors", crash.Visible, crash.SyncErrors, holds),
	Entry("Visible, FullDisk", crash.Visible, crash.FullDisk, holds),
	Entry("Visible, OldCorruption", crash.Visible, crash.OldCorruption, holds),
	Entry("Visible, NewCorruption", crash.Visible, crash.NewCorruption, holds),
)
