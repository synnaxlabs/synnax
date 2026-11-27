// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import "github.com/synnaxlabs/x/telem"

// column holds the samples of one channel in arrival order. Its buffers survive across
// polls.
type column struct {
	dataType telem.DataType
	// density is the size of a sample, zero when samples vary in size.
	density int
	// data holds the current poll's samples.
	data []byte
	// ends[i] is the end of sample i in data. It is only set when density is zero.
	ends []int
}

func newColumn(dt telem.DataType) column {
	c := column{dataType: dt}
	if !dt.IsVariable() {
		c.density = int(dt.Density())
	}
	return c
}

// append adds sample to the end of the column.
func (c *column) append(sample []byte) {
	c.data = append(c.data, sample...)
	if c.density == 0 {
		c.ends = append(c.ends, len(c.data))
	}
}

// sample returns sample i.
func (c *column) sample(i int) []byte {
	if c.density != 0 {
		return c.data[i*c.density : (i+1)*c.density]
	}
	start := 0
	if i > 0 {
		start = c.ends[i-1]
	}
	return c.data[start:c.ends[i]]
}

// reset empties the column and keeps its buffers.
func (c *column) reset() {
	c.data = c.data[:0]
	c.ends = c.ends[:0]
}

func (c *column) size(rows []row) int {
	if c.density != 0 {
		return len(rows) * c.density
	}
	n := 0
	for _, r := range rows {
		n += len(c.sample(r.pos))
	}
	return n
}
