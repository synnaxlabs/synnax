// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Breaker node", func() {
	// pushRaw sets the output of source to values at the raw stamps, with alignment.
	pushRaw := func(
		j *joinSetup,
		source string,
		alignment telem.Alignment,
		values []float64,
		stamps ...telem.TimeStamp,
	) {
		src := j.state.Node(source)
		data := telem.NewSeries(values)
		t := telem.NewSeries(stamps)
		data.Alignment, t.Alignment = alignment, alignment
		*src.Output(0), *src.OutputTime(0) = data, t
		src.MarkFresh(0)
	}

	// The outputs take the time range of the joined primary, which for a run that ends
	// at TimeStampMax is [stamp, TimeStampMax].
	It("Should not wrap the time range of a run that ends at the largest stamp", func(
		ctx SpecContext,
	) {
		j := openJoin(ctx, false)
		pushRaw(j, "b", 0, []float64{10}, telem.TimeStampMax-1)
		pushRaw(j, "a", 0, []float64{1}, telem.TimeStampMax)
		Expect(j.next(ctx)).To(ConsistOf(0, 1))
		a, _ := j.output(0)
		Expect(a.TimeRange.End).To(BeNumerically(">=", a.TimeRange.Start))
	})

	// The scheduler runs the node on a fresh input or on MarkSelfChanged, so an input
	// that arrives while runs remain must still reach an output.
	It("Should emit an input that arrives while runs remain", func(ctx SpecContext) {
		j := openJoin(ctx, false)
		var stamps []telem.TimeStamp
		// run runs the node as the scheduler does: once, then again on each cycle for
		// which it called MarkSelfChanged.
		run := func() {
			for {
				self := j.self
				if len(j.next(ctx)) > 0 {
					_, t := j.output(0)
					stamps = append(stamps, t.Unmarshal[telem.TimeStamp]()...)
				}
				if j.self == self {
					return
				}
			}
		}
		j.push("b", 0, []float64{10, 11, 12, 20}, 0, 1, 2, 20)
		j.push("a", 0, []float64{1, 2, 3, 4, 5}, 1, 2, 10, 11, 21)
		Expect(j.next(ctx)).To(ConsistOf(0, 1))
		_, t := j.output(0)
		stamps = append(stamps, t.Unmarshal[telem.TimeStamp]()...)
		// a's next series arrives on the cycle that replays the second run.
		j.push("a", 5, []float64{6, 7}, 22, 23)
		run()
		Expect(stamps).To(Equal([]telem.TimeStamp{
			1 * telem.SecondTS,
			2 * telem.SecondTS,
			21 * telem.SecondTS,
			22 * telem.SecondTS,
			23 * telem.SecondTS,
		}))
	})
})
