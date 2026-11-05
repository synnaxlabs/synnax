// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/video"
)

const (
	rate = 90000
	// tick is one frame at 25 fps in 90 kHz RTP ticks.
	tick = 3600
)

// frameSpan is one frame at 25 fps.
var frameSpan = 40 * telem.Millisecond

// start is an arbitrary arrival time for the first frame.
var start = 1000 * telem.SecondTS

// ntp encodes t as a 64-bit NTP time.
func ntp(t telem.TimeStamp) uint64 {
	sec := uint64(t/telem.SecondTS) + 2208988800
	frac := uint64(t%telem.SecondTS) << 32 / uint64(telem.Second)
	return sec<<32 | frac
}

// stamp returns the time c gives a frame, and expects it to have one.
func stamp(c *video.Clock, arrival telem.TimeStamp, rtp uint32) telem.TimeStamp {
	GinkgoHelper()
	t, ok := c.Stamp(arrival, rtp)
	Expect(ok).To(BeTrue())
	return t
}

var _ = Describe("Clock", func() {
	Describe("Arrival", func() {
		It("Should remove the jitter of arrival times", func() {
			c := video.Clock{Rate: rate}
			jitter := []telem.TimeSpan{0, 15, 4, 30, 9, 22}
			for i, j := range jitter {
				arrival := start.Add(telem.TimeSpan(i)*frameSpan + j*telem.Millisecond)
				Expect(
					stamp(&c, arrival, uint32(i*tick)),
				).To(Equal(start.Add(telem.TimeSpan(i) * frameSpan)))
			}
		})

		It("Should add the offset", func() {
			c := video.Clock{Rate: rate, Offset: -200 * telem.Millisecond}
			Expect(stamp(&c, start, 0)).To(Equal(start.Add(-200 * telem.Millisecond)))
		})

		It("Should follow a lower delay at once", func() {
			c := video.Clock{Rate: rate}
			Expect(stamp(&c, start.Add(50*telem.Millisecond), 0)).
				To(Equal(start.Add(50 * telem.Millisecond)))
			Expect(stamp(&c, start.Add(2*frameSpan), 2*tick)).
				To(Equal(start.Add(2 * frameSpan)))
		})

		It("Should forget a delay older than the window", func() {
			c := video.Clock{Rate: rate}
			_, _ = c.Stamp(start, 0)
			// Every frame after the first arrives 40 ms late.
			late := 40 * telem.Millisecond
			var t telem.TimeStamp
			for i := 1; i <= 330; i++ {
				arrival := start.Add(telem.TimeSpan(i)*frameSpan + late)
				t, _ = c.Stamp(arrival, uint32(i*tick))
			}
			Expect(t).To(Equal(start.Add(330*frameSpan + late)))
		})

		It("Should keep times continuous across a wrap of the RTP timestamp", func() {
			c := video.Clock{Rate: rate}
			first := uint32(math.MaxUint32 - tick + 1)
			_, _ = c.Stamp(start, first)
			t, _ := c.Stamp(start.Add(frameSpan), first+tick)
			Expect(t).To(Equal(start.Add(frameSpan)))
		})

		It(
			"Should restart at the arrival time when the RTP timestamps restart",
			func() {
				c := video.Clock{Rate: rate}
				_, _ = c.Stamp(start, 900000)
				arrival := start.Add(frameSpan)
				Expect(stamp(&c, arrival, 0)).To(Equal(arrival))
			},
		)

		It("Should always return a later time than the one before", func() {
			c := video.Clock{Rate: rate}
			first, _ := c.Stamp(start, 0)
			second, _ := c.Stamp(start, 0)
			Expect(second).To(Equal(first + 1))
		})
	})

	Describe("Capture", func() {
		It("Should not time frames before the first sender report", func() {
			c := video.Clock{Rate: rate, Capture: true}
			_, ok := c.Stamp(start, 0)
			Expect(ok).To(BeFalse())
		})

		It("Should time frames from the last sender report", func() {
			c := video.Clock{Rate: rate, Capture: true}
			captured := 500 * telem.SecondTS
			c.Report(ntp(captured), 10*tick)
			Expect(stamp(&c, start, 12*tick)).To(Equal(captured.Add(2 * frameSpan)))
			Expect(stamp(&c, start, 13*tick)).To(Equal(captured.Add(3 * frameSpan)))
		})

		It("Should time a frame captured before the report", func() {
			c := video.Clock{Rate: rate, Capture: true}
			captured := 500 * telem.SecondTS
			c.Report(ntp(captured), 10*tick)
			Expect(stamp(&c, start, 9*tick)).To(Equal(captured.Add(-frameSpan)))
		})

		It("Should drop frames after a restart until the next report", func() {
			c := video.Clock{Rate: rate, Capture: true}
			captured := 500 * telem.SecondTS
			c.Report(ntp(captured), 900000)
			_, ok := c.Stamp(start, 900000)
			Expect(ok).To(BeTrue())
			_, ok = c.Stamp(start, 0)
			Expect(ok).To(BeFalse())
			c.Report(ntp(captured.Add(telem.Second)), 0)
			Expect(stamp(&c, start, tick)).
				To(Equal(captured.Add(telem.Second + frameSpan)))
		})
	})
})
