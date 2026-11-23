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

// column places the samples of one binding in the data its decoder appends to, in
// arrival order. Its buffers survive across polls.
type column struct {
	dataType telem.DataType
	// density is the size of a sample, zero when samples vary in size.
	density int
	// ends[i] is the end of sample i in the data. It is only set when density is zero.
	ends []int
}

func newColumn(dt telem.DataType) column {
	c := column{dataType: dt}
	if !dt.IsVariable() {
		c.density = int(dt.Density())
	}
	return c
}

// sample returns sample i of data.
func (c *column) sample(data []byte, i int) []byte {
	if c.density != 0 {
		return data[i*c.density : (i+1)*c.density]
	}
	start := 0
	if i > 0 {
		start = c.ends[i-1]
	}
	return data[start:c.ends[i]]
}

// appended records that a new sample ends data.
func (c *column) appended(data []byte) {
	if c.density == 0 {
		c.ends = append(c.ends, len(data))
	}
}

func (c *column) size(data []byte, rows []row) int {
	if c.density != 0 {
		return len(rows) * c.density
	}
	n := 0
	for _, r := range rows {
		n += len(c.sample(data, r.pos))
	}
	return n
}
