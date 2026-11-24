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
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: Handoff after a skip", func() {
	var (
		db          *cesium.DB
		clock, data cesium.ChannelKey
		keys        []cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		db = mustOpenDBOnFS(ctx, fs.NewMem())
		clock, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{clock, data}
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
	})
	config := func(
		start telem.TimeStamp,
		keys ...cesium.ChannelKey,
	) cesium.WriterConfig {
		return cesium.WriterConfig{Channels: keys, Clock: clock, Start: start}
	}
	isData := func(key cesium.ChannelKey) bool { return key == data }

	// Each entry commits 1 to 4 at 10 to 13 s, then writes 5 and 6 at 14 and 15 s and
	// loses the data channel at the write or at the commit. A handoff drops the
	// channel, another writer can then write 50 and 60 at 14 and 15 s, and a handoff
	// adopts or commits the Pending, adds the channel back, and writes 7 at 16 s.
	DescribeTable("Should keep each stamp of the data channel once and readable",
		func(ctx SpecContext, atCommit, other, adopt bool) {
			w := MustSucceed(db.OpenWriter(ctx, config(10*telem.SecondTS, keys...)))
			mustWrite(ctx, w, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12, 13),
				telem.NewSeriesV[int64](1, 2, 3, 4),
			}))
			MustSucceed2(w.Commit(ctx, nil))
			writeSkip := isData
			if atCommit {
				writeSkip = nil
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(14, 15),
				telem.NewSeriesV[int64](5, 6),
			}), writeSkip))
			MustSucceed2(w.Commit(ctx, isData))
			after := 15*telem.SecondTS + 1
			w, p := MustSucceed2(
				db.Handoff(ctx, w, config(after, clock), nil, cesium.SameRun),
			)
			MustSucceed2(w.Commit(ctx, nil))
			if other {
				o := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{data},
					Clock:    clock,
					Start:    14 * telem.SecondTS,
				}))
				mustWrite(
					ctx,
					o,
					telem.UnaryFrame(data, telem.NewSeriesV[int64](50, 60)),
				)
				MustSucceed2(o.Commit(ctx, nil))
				Expect(o.Close()).To(Succeed())
			}
			var adopted *cesium.Pending
			if adopt {
				adopted = p
			} else if p != nil {
				_, committed := MustSucceed2(p.Commit(ctx))
				Expect(committed).To(BeTrue())
			}
			w, p = MustSucceed2(
				db.Handoff(ctx, w, config(after, keys...), adopted, cesium.SameRun),
			)
			Expect(p).To(BeNil())
			mustWrite(ctx, w, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(16),
				telem.NewSeriesV[int64](7),
			}))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, keys...))
			var stamps []byte
			for _, s := range fr.Get(clock).Series {
				stamps = append(stamps, s.Data...)
			}
			Expect(stamps).To(Equal(
				telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16).Data,
			))
			values := make([]int64, 0, 7)
			for _, s := range fr.Get(data).Series {
				values = append(values, s.Unmarshal[int64]()...)
			}
			if other {
				Expect(values).To(Equal([]int64{1, 2, 3, 4, 50, 60, 7}))
				return
			}
			// A commit that skips the channel ends 5 and 6, or a later commit keeps
			// them.
			Expect(values).To(BeElementOf(
				[]int64{1, 2, 3, 4, 7},
				[]int64{1, 2, 3, 4, 5, 6, 7},
			))
		},
		Entry("skip at the write, adopt", false, false, true),
		Entry("skip at the write, another writer, adopt", false, true, true),
		Entry("skip at the commit, adopt", true, false, true),
		Entry("skip at the commit, commit the Pending", true, false, false),
		Entry("skip at the commit, another writer, adopt", true, true, true),
		Entry(
			"skip at the commit, another writer, commit the Pending",
			true,
			true,
			false,
		),
	)
})
