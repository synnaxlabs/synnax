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

// intervals holds an input's last intervals, in a fixed ring. The hold limit is 3 times
// their median, and there is none before the input's second sample.
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

// limit returns 3 times the median of the intervals, or 0 when there are none.
func (iv *intervals) limit() telem.TimeSpan {
	if iv.n == 0 {
		return 0
	}
	if iv.fresh {
		return iv.held
	}
	var sorted [window]telem.TimeSpan
	copy(sorted[:iv.n], iv.ring[:iv.n])
	for i := 1; i < iv.n; i++ {
		for k := i; k > 0 && sorted[k] < sorted[k-1]; k-- {
			sorted[k], sorted[k-1] = sorted[k-1], sorted[k]
		}
	}
	median := sorted[iv.n/2]
	if iv.n%2 == 0 {
		median = (sorted[iv.n/2-1] + median) / 2
	}
	iv.held, iv.fresh = 3*median, true
	return iv.held
}
