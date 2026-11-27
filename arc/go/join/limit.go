// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join

import "github.com/synnaxlabs/x/telem"

// lastIntervals is the number of an input's last intervals that set its hold limit.
const lastIntervals = 16

// intervals holds an input's last intervals, in a fixed ring, and caches their hold
// limit and 3 times their median.
type intervals struct {
	ring       [lastIntervals]telem.TimeSpan
	n          int
	next       int
	limitSpan  telem.TimeSpan
	withinSpan telem.TimeSpan
	fresh      bool
}

func (iv *intervals) add(span telem.TimeSpan) {
	iv.ring[iv.next] = span
	iv.next = (iv.next + 1) % lastIntervals
	iv.n = min(iv.n+1, lastIntervals)
	iv.fresh = false
}

// limit returns the hold limit of the intervals, or 0 when there is none.
func (iv *intervals) limit() telem.TimeSpan {
	iv.update()
	return iv.limitSpan
}

// unsteady reports whether there are intervals but no steady rate, so no hold limit.
func (iv *intervals) unsteady() bool { return iv.n > 0 && iv.limit() == 0 }

// within returns 3 times the median of the intervals, or 0 when there are none.
func (iv *intervals) within() telem.TimeSpan {
	iv.update()
	return iv.withinSpan
}

func (iv *intervals) update() {
	if iv.fresh {
		return
	}
	iv.fresh = true
	n := iv.n
	if n == 0 {
		iv.limitSpan, iv.withinSpan = 0, 0
		return
	}
	var sorted [lastIntervals]telem.TimeSpan
	copy(sorted[:n], iv.ring[:n])
	for i := 1; i < n; i++ {
		for k := i; k > 0 && sorted[k] < sorted[k-1]; k-- {
			sorted[k], sorted[k-1] = sorted[k-1], sorted[k]
		}
	}
	// The median is the mean of the two middle intervals, which are one for an odd
	// count.
	lo, hi := sorted[(n-1)/2], sorted[n/2]
	iv.withinSpan = sum(scale(lo, 3), scale(hi-lo, 3)/2)
	iv.limitSpan = 0
	if steady(sorted[:n], sum(lo, hi)) {
		iv.limitSpan = iv.withinSpan
	}
}

// steady reports whether at least 3 in 4 of sorted lie within a factor of 2 of their
// median, given as twice the median.
func steady(sorted []telem.TimeSpan, twiceMedian telem.TimeSpan) bool {
	near := 0
	for _, span := range sorted {
		if span <= twiceMedian && twiceMedian <= scale(span, 4) {
			near++
		}
	}
	return 4*near >= 3*len(sorted)
}

// scale returns k times span, or TimeSpanMax when that overflows.
func scale(span, k telem.TimeSpan) telem.TimeSpan {
	if span > telem.TimeSpanMax/k {
		return telem.TimeSpanMax
	}
	return k * span
}

// sum returns a plus b, or TimeSpanMax when that overflows.
func sum(a, b telem.TimeSpan) telem.TimeSpan {
	if a > telem.TimeSpanMax-b {
		return telem.TimeSpanMax
	}
	return a + b
}
