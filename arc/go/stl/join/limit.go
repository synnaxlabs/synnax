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

// window is the number of an input's last intervals that set its hold limit.
const window = 16

// intervals holds an input's last intervals, in a fixed ring, and caches their hold
// limit.
type intervals struct {
	ring   [window]telem.TimeSpan
	n      int
	next   int
	held   telem.TimeSpan
	spread telem.TimeSpan
	fresh  bool
}

func (iv *intervals) add(span telem.TimeSpan) {
	iv.ring[iv.next] = span
	iv.next = (iv.next + 1) % window
	iv.n = min(iv.n+1, window)
	iv.fresh = false
}

// limit returns the hold limit of the intervals.
func (iv *intervals) limit() telem.TimeSpan {
	iv.update()
	return iv.held
}

// within returns 3 times the median of the intervals, or 0 when there are none.
func (iv *intervals) within() telem.TimeSpan {
	iv.update()
	return iv.spread
}

func (iv *intervals) update() {
	if iv.fresh {
		return
	}
	var sorted [window]telem.TimeSpan
	n := iv.n
	copy(sorted[:n], iv.ring[:n])
	for i := 1; i < n; i++ {
		for k := i; k > 0 && sorted[k] < sorted[k-1]; k-- {
			sorted[k], sorted[k-1] = sorted[k-1], sorted[k]
		}
	}
	iv.held, iv.spread, iv.fresh = holdLimit(
		sorted[:n],
	), thrice(
		median(sorted[:n]),
	), true
}

// median returns the median of sorted, or 0 when it is empty.
func median(sorted []telem.TimeSpan) telem.TimeSpan {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	m := sorted[n/2]
	if n%2 == 0 {
		m = sorted[n/2-1] + (m-sorted[n/2-1])/2
	}
	return m
}

// thrice returns 3 times span, or TimeSpanMax when that overflows.
func thrice(span telem.TimeSpan) telem.TimeSpan {
	if span > telem.TimeSpanMax/3 {
		return telem.TimeSpanMax
	}
	return 3 * span
}

// holdLimit returns 3 times the median of sorted, or 0 for no limit: when sorted is
// empty, when its median is 0, or when the input has no steady rate, as fewer than 3 in
// 4 of sorted lie within a factor of 2 of their median.
func holdLimit(sorted []telem.TimeSpan) telem.TimeSpan {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	median := median(sorted)
	steady := 0
	for _, span := range sorted[:n] {
		if span-median <= median && median-span <= span {
			steady++
		}
	}
	if 4*steady < 3*n {
		return 0
	}
	return thrice(median)
}
