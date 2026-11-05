// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"sync"

	"github.com/synnaxlabs/x/telem"
)

const (
	// window is how long a Clock remembers the delay of each frame. The least delay in
	// the window estimates the fixed delay of the network.
	window = 10 * telem.Second
	// maxJump is how far a time may fall behind the time before it before a Clock
	// treats the stream as restarted.
	maxJump = telem.Second
	// ntpEpoch is the number of seconds from the NTP epoch, 1900, to the Unix epoch.
	ntpEpoch = 2208988800
)

// Clock times the frames of one RTP stream. By default, a frame's time is the media
// time of its RTP timestamp, plus the least delay between media time and arrival in
// the last 10 s. This removes the jitter of the network. With Capture, a frame's time
// comes from the RTCP sender reports of the stream instead. Each time is Offset later,
// and always later than the time before it. A Clock is safe for concurrent use.
type Clock struct {
	// Rate is the RTP clock rate of the stream, in hertz.
	Rate uint32
	// Offset is added to each time.
	Offset telem.TimeSpan
	// Capture takes times from sender reports. Frames before the first report, and
	// after a restart of the stream until the next report, have no time.
	Capture bool
	mu      struct {
		sync.Mutex
		started bool
		// rtp and ticks are the RTP timestamp of the last frame, and that timestamp
		// unwrapped to 64 bits.
		rtp   uint32
		ticks int64
		// delays holds the delays of the window in ascending order, so the first is
		// the least.
		delays []delay
		// report maps the ticks of the last sender report to its NTP time.
		report struct {
			valid bool
			ntp   telem.TimeStamp
			ticks int64
		}
		last telem.TimeStamp
	}
}

type delay struct {
	arrival telem.TimeStamp
	// base is the arrival time minus the media time of the frame.
	base telem.TimeStamp
}

// Report records the RTCP sender report of the stream with the 64-bit NTP time ntp and
// the RTP timestamp rtp.
func (c *Clock) Report(ntp uint64, rtp uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := &c.mu.report
	r.valid = true
	r.ntp = telem.TimeStamp(int64(ntp>>32-ntpEpoch)*int64(telem.Second) +
		int64((ntp&0xFFFFFFFF)*uint64(telem.Second)>>32))
	r.ticks = c.unwrap(rtp)
}

// Stamp returns the time of the frame with the RTP timestamp rtp that arrived at
// arrival. It returns false when a Capture clock has no sender report to time the
// frame by.
func (c *Clock) Stamp(arrival telem.TimeStamp, rtp uint32) (telem.TimeStamp, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ticks := c.unwrap(rtp)
	c.mu.rtp, c.mu.ticks = rtp, ticks
	media := c.span(ticks)
	var t telem.TimeStamp
	if c.Capture {
		r := c.mu.report
		if !r.valid {
			return 0, false
		}
		t = r.ntp.Add(media - c.span(r.ticks))
		if t.Add(c.Offset) < c.mu.last.Add(-maxJump) {
			c.mu.report.valid = false
			return 0, false
		}
	} else {
		t = c.smooth(arrival, media)
	}
	t = max(t.Add(c.Offset), c.mu.last+1)
	c.mu.last = t
	return t, true
}

// smooth returns the media time plus the least delay of the window.
func (c *Clock) smooth(arrival telem.TimeStamp, media telem.TimeSpan) telem.TimeStamp {
	d := delay{arrival: arrival, base: arrival.Add(-media)}
	t := c.push(d)
	if t.Add(c.Offset) < c.mu.last.Add(-maxJump) {
		c.mu.delays = c.mu.delays[:0]
		t = c.push(d)
	}
	return t
}

// push adds d to the window and returns the time that the least delay gives d.
func (c *Clock) push(d delay) telem.TimeStamp {
	delays := c.mu.delays
	for len(delays) > 0 && delays[len(delays)-1].base >= d.base {
		delays = delays[:len(delays)-1]
	}
	delays = append(delays, d)
	expired := 0
	for delays[expired].arrival < d.arrival.Add(-window) {
		expired++
	}
	c.mu.delays = delays[expired:]
	return d.arrival - d.base + c.mu.delays[0].base
}

// unwrap returns rtp as ticks since the first frame or report, which never wrap.
func (c *Clock) unwrap(rtp uint32) int64 {
	if !c.mu.started {
		c.mu.started = true
		c.mu.rtp = rtp
		return 0
	}
	return c.mu.ticks + int64(int32(rtp-c.mu.rtp))
}

// span returns the media time of ticks.
func (c *Clock) span(ticks int64) telem.TimeSpan {
	rate := int64(c.Rate)
	return telem.TimeSpan(ticks/rate)*telem.Second +
		telem.TimeSpan((ticks%rate)*int64(telem.Second)/rate)
}
