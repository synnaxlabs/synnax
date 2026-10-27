// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alignment

import (
	"math"
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ZeroLeading is the start of the domain indexes reserved for written data that has not
// yet been persisted. Persisted data uses the domain indexes below it, so new data
// always orders after persisted data.
const ZeroLeading uint32 = 1 << 31

// Leading returns an Alignment in the leading domain domainIdx, counted from
// ZeroLeading, with the provided sample index.
func Leading(domainIdx, sampleIdx uint32) telem.Alignment {
	return telem.NewAlignment(ZeroLeading+domainIdx, sampleIdx)
}

// ErrExhausted is returned by Counter.Next when no domain index is left above the
// counter's start.
var ErrExhausted = errors.New("alignment domains exhausted, restart to reset")

// Counter hands out alignment domains in increasing order. It is safe for concurrent
// use.
type Counter struct{ last atomic.Uint64 }

// NewCounter returns a Counter whose first domain index is start+1.
func NewCounter(start uint32) *Counter {
	c := &Counter{}
	c.last.Store(uint64(start))
	return c
}

// Next returns the alignment of the first sample in the next domain. It returns
// ErrExhausted once the domain index would pass math.MaxUint32.
func (c *Counter) Next() (telem.Alignment, error) {
	domainIdx := c.last.Add(1)
	if domainIdx > math.MaxUint32 {
		return 0, ErrExhausted
	}
	return telem.NewAlignment(uint32(domainIdx), 0), nil
}
