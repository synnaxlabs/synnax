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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = DescribeTableSubtree("Claim move", func(pointerLog bool) {
	var (
		host      testutil.Node
		idx, data channel.Key
		keys      channel.Keys
	)
	BeforeEach(func(ctx SpecContext) {
		host = newNode(ctx, pointerLog)
		idxCh := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
			Name:     "move_time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		}}))[0]
		dataCh := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
			Name:       "move",
			DataType:   telem.Int64T,
			LocalIndex: idxCh.LocalKey,
		}}))[0]
		idx, data = idxCh.Key(), dataCh.Key()
		keys = channel.Keys{idx, data}
	})
	open := func(
		ctx SpecContext,
		subject string,
		keys channel.Keys,
		start telem.TimeStamp,
		authority control.Authority,
	) *writer.Writer {
		GinkgoHelper()
		return MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
			ControlSubject:   control.Subject{Key: subject},
			Keys:             keys,
			Start:            start,
			Authorities:      []control.Authority{authority},
			EnableAutoCommit: new(false),
			Sync:             new(true),
		}))
	}
	// both returns a frame of the index at the stamps of seconds and of the data at
	// values.
	both := func(seconds []telem.TimeStamp, values ...int64) frame.Frame {
		return frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(values),
		})
	}
	only := func(values ...int64) frame.Frame {
		return frame.NewUnary(data, telem.NewSeries(values))
	}
	seconds := func(s ...telem.TimeStamp) []telem.TimeStamp { return s }
	// read returns the stored series of key in tr.
	read := func(
		ctx SpecContext,
		key channel.Key,
		tr telem.TimeRange,
	) telem.MultiSeries {
		GinkgoHelper()
		iter := MustSucceed(host.Framer.OpenIterator(ctx, iterator.Config{
			Keys:   channel.Keys{key},
			Bounds: tr,
		}))
		Expect(iter.SeekFirst()).To(BeTrue())
		Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
		series := iter.Value().Get(key)
		Expect(iter.Close()).To(Succeed())
		return series
	}
	values := func(ctx SpecContext, tr telem.TimeRange) []int64 {
		GinkgoHelper()
		var out []int64
		for _, s := range read(ctx, data, tr).Series {
			out = append(out, s.Unmarshal[int64]()...)
		}
		return out
	}
	stamps := func(ctx SpecContext) []telem.TimeStamp {
		GinkgoHelper()
		var out []telem.TimeStamp
		for _, s := range read(ctx, idx, telem.TimeRangeMax).Series {
			out = append(out, s.Unmarshal[telem.TimeStamp]()...)
		}
		return out
	}

	It(
		"Should move a data channel to the session that controls it at its first "+
			"write, and commit the samples of the old holder once its clock commits",
		func(ctx SpecContext) {
			clock := open(ctx, "clock", keys, 10*telem.SecondTS, 100)
			Expect(clock.Write(both(seconds(10, 11), 1, 2))).To(BeTrue())
			moved := open(ctx, "moved", channel.Keys{data}, 12*telem.SecondTS, 200)
			Expect(clock.Write(both(seconds(12, 13), 3, 4))).To(BeFalse())
			Expect(moved.Write(only(30, 40))).To(BeTrue())
			_, authorized := MustSucceed2(clock.Commit())
			Expect(authorized).To(BeFalse(), "the session lost the data channel")
			Expect(MustCommit(moved)).To(Equal(13*telem.SecondTS + 1))
			Expect(clock.Write(both(seconds(14), 5))).To(BeFalse())
			end, authorized := MustSucceed2(clock.Commit())
			Expect(end).To(Equal(14*telem.SecondTS + 1))
			Expect(authorized).To(BeFalse(), "the session lost the data channel")
			Expect(moved.Close()).To(Succeed())
			Expect(clock.Close()).To(Succeed())
			Expect(stamps(ctx)).To(Equal([]telem.TimeStamp{
				10 * telem.SecondTS,
				11 * telem.SecondTS,
				12 * telem.SecondTS,
				13 * telem.SecondTS,
				14 * telem.SecondTS,
			}))
			Expect(values(ctx, telem.TimeRangeMax)).To(Equal([]int64{1, 2, 30, 40}))
		},
	)

	It(
		"Should give a data channel back to its old session with the samples that the "+
			"move kept for it when the new holder closes",
		func(ctx SpecContext) {
			clock := open(ctx, "clock", keys, 10*telem.SecondTS, 100)
			Expect(clock.Write(both(seconds(10, 11), 1, 2))).To(BeTrue())
			moved := open(ctx, "moved", channel.Keys{data}, 12*telem.SecondTS, 200)
			Expect(clock.Write(both(seconds(12, 13), 3, 4))).To(BeFalse())
			Expect(moved.Write(only(30, 40))).To(BeTrue())
			Expect(moved.Close()).To(Succeed())
			Expect(clock.Write(both(seconds(14), 5))).To(BeTrue())
			end, authorized := MustSucceed2(clock.Commit())
			Expect(end).To(Equal(14*telem.SecondTS + 1))
			Expect(
				authorized,
			).To(BeFalse(), "a write since the last commit was refused")
			Expect(clock.Close()).To(Succeed())
			Expect(stamps(ctx)).To(HaveLen(5))
			Expect(values(ctx, telem.TimeRangeMax)).To(Equal([]int64{1, 2, 5}))
			Expect(values(ctx, (14 * telem.SecondTS).SpanRange(telem.Second))).
				To(Equal([]int64{5}))
		},
	)

	It(
		"Should move a key both ways and start its claim after its stored samples "+
			"when it comes back",
		func(ctx SpecContext) {
			clock := open(ctx, "clock", keys, 10*telem.SecondTS, 100)
			Expect(clock.Write(both(seconds(10, 11), 1, 2))).To(BeTrue())
			Expect(MustCommit(clock)).To(Equal(11*telem.SecondTS + 1))
			moved := open(ctx, "moved", channel.Keys{data}, 12*telem.SecondTS, 200)
			Expect(clock.Write(both(seconds(12, 13), 3, 4))).To(BeFalse())
			end, authorized := MustSucceed2(clock.Commit())
			Expect(end).To(Equal(13*telem.SecondTS + 1))
			Expect(
				authorized,
			).To(BeFalse(), "a write since the last commit was refused")
			Expect(moved.Write(only(30, 40))).To(BeTrue())
			Expect(MustCommit(moved)).To(Equal(13*telem.SecondTS + 1))
			Expect(moved.Close()).To(Succeed())
			Expect(clock.Write(both(seconds(14), 5))).To(BeTrue())
			Expect(MustCommit(clock)).To(Equal(14*telem.SecondTS + 1))
			Expect(clock.Close()).To(Succeed())
			Expect(stamps(ctx)).To(HaveLen(5))
			Expect(values(ctx, telem.TimeRangeMax)).To(Equal([]int64{1, 2, 30, 40, 5}))
			Expect(values(ctx, (12 * telem.SecondTS).Range(14*telem.SecondTS))).
				To(Equal([]int64{30, 40}))
		},
	)

	It(
		"Should keep the run of the clock when the session that holds it loses a "+
			"data channel",
		func(ctx SpecContext) {
			clock := open(ctx, "clock", keys, 10*telem.SecondTS, 100)
			Expect(clock.Write(both(seconds(10, 11), 1, 2))).To(BeTrue())
			Expect(MustCommit(clock)).To(Equal(11*telem.SecondTS + 1))
			moved := open(ctx, "moved", channel.Keys{data}, 12*telem.SecondTS, 200)
			Expect(clock.Write(both(seconds(12, 13), 3, 4))).To(BeFalse())
			Expect(moved.Write(only(30, 40))).To(BeTrue())
			Expect(clock.Write(both(seconds(14), 5))).To(BeFalse())
			_, authorized := MustSucceed2(clock.Commit())
			Expect(authorized).To(BeFalse(), "the session lost the data channel")
			Expect(moved.Close()).To(Succeed())
			Expect(clock.Close()).To(Succeed())
			series := read(ctx, idx, telem.TimeRangeMax).Series
			Expect(series).ToNot(BeEmpty())
			for i := 1; i < len(series); i++ {
				Expect(series[i].Alignment).
					To(Equal(series[i-1].AlignmentBounds().Upper))
			}
		},
	)
}, Entry("old layout", false), Entry("pointer log", true))
