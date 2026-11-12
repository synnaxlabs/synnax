// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"encoding/json/jsontext"

	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
)

// column accumulates one field's samples as series data, in arrival order. Its buffers
// survive across polls.
type column struct {
	dataType telem.DataType
	decoder  *xjson.Decoder
	// density is the size of a sample, zero when samples vary in size.
	density int
	data    []byte
	// ends[i] is the end of sample i in data. It is only set when density is zero.
	ends []int
	// fillable is true when an absent value writes fill, or skips the record when the
	// column has no fill.
	fillable bool
	// filled is true when the column has a fill.
	filled bool
	// emptyLabeled is true when an enum entry has an empty label, so an empty string
	// is that entry's value rather than no value.
	emptyLabeled bool
	// fill is the sample an absent value writes, with no length prefix.
	fill []byte
}

func newDecoderColumn(dec *xjson.Decoder, dt telem.DataType) column {
	c := column{dataType: dt, decoder: dec}
	if !dt.IsVariable() {
		c.density = int(dt.Density())
	}
	return c
}

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

func (c *column) truncate(n int) {
	if c.density != 0 {
		c.data = c.data[:n*c.density]
		return
	}
	c.ends = c.ends[:n]
	end := 0
	if n > 0 {
		end = c.ends[n-1]
	}
	c.data = c.data[:end]
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

// absent returns true when raw is no value: missing, or for a fillable column, null or
// an empty string with no enum entry on a channel that does not hold strings.
func (c *column) absent(raw jsontext.Value) bool {
	if raw == nil {
		return true
	}
	if !c.fillable {
		return false
	}
	return raw.Kind() == 'n' ||
		(c.dataType != telem.StringT && !c.emptyLabeled && string(raw) == `""`)
}

// appendFill appends the fill of the column, which must have one.
func (c *column) appendFill() {
	if c.density == 0 {
		c.data = telem.ByteOrder.AppendUint32(c.data, uint32(len(c.fill)))
	}
	c.data = append(c.data, c.fill...)
	if c.density == 0 {
		c.ends = append(c.ends, len(c.data))
	}
}

// append converts raw into a sample and appends it. It returns an error wrapping
// validate.ErrValidation when raw does not convert, and then appends nothing.
func (c *column) append(raw jsontext.Value) error {
	data, err := c.decoder.Append(c.data, raw)
	if err != nil {
		return err
	}
	c.data = data
	if c.density == 0 {
		c.ends = append(c.ends, len(c.data))
	}
	return nil
}
