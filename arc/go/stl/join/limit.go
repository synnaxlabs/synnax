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
	ring  [window]telem.TimeSpan
	n     int
	next  int
	held  telem.TimeSpan
	fresh bool
}

func (iv *intervals) add(span telem.TimeSpan) {
	iv.ring[iv.next] = span
	iv.next = (iv.next + 1) % window
	iv.n = min(iv.n+1, window)
	iv.fresh = false
}

// limit returns the hold limit of the intervals.
func (iv *intervals) limit() telem.TimeSpan {
	if !iv.fresh {
		iv.held, iv.fresh = holdLimit(iv.ring[:iv.n]), true
	}
	return iv.held
}

// holdLimit returns 3 times the median of spans, or 0 for no limit: when spans is
// empty, or when the input has no steady rate, as fewer than 3 in 4 of spans lie within
// a factor of 2 of their median.
func holdLimit(spans []telem.TimeSpan) telem.TimeSpan {
	n := len(spans)
	if n == 0 {
		return 0
	}
	var sorted [window]telem.TimeSpan
	copy(sorted[:n], spans)
	for i := 1; i < n; i++ {
		for k := i; k > 0 && sorted[k] < sorted[k-1]; k-- {
			sorted[k], sorted[k-1] = sorted[k-1], sorted[k]
		}
	}
	median := sorted[n/2]
	if n%2 == 0 {
		median = sorted[n/2-1] + (median-sorted[n/2-1])/2
	}
	steady := 0
	for _, span := range sorted[:n] {
		if span-median <= median && median-span <= span {
			steady++
		}
	}
	if 4*steady < 3*n {
		return 0
	}
	if median > telem.TimeSpanMax/3 {
		return telem.TimeSpanMax
	}
	return 3 * median
}
