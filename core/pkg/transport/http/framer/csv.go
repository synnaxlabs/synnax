// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"bytes"
	"context"
	"io"
	"slices"

	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
)

// csvFlushSize is the number of bytes the encoder accumulates before writing them out.
const csvFlushSize = 64 << 10

// CSVEncoder encodes a read as CSV. Channels are grouped by the index channel that
// timestamps them. Each group contributes its index column, then one column per data
// channel, headed by the channel's name. Rows are ordered by timestamp, and a group's
// cells are empty at a timestamp it has no sample for. A data channel's cell holds its
// sample at the alignment of the row's index sample, and is empty when the channel has
// no sample there. An empty variable-length sample is written as "", so it differs from
// a missing one. A read of a channel whose index it does not include, or a read under a
// limit reduction, fails before the first byte. A read that fails after the first byte
// truncates the body.
var CSVEncoder csvEncoder

type csvEncoder struct{}

var _ xhttp.Encoder = csvEncoder{}

// ContentType implements http.Encoder.
func (csvEncoder) ContentType() string { return "text/csv" }

// Encode implements encoding.Encoder. It holds the whole read in memory, so prefer
// EncodeStream.
func (e csvEncoder) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeStream implements encoding.Encoder. value must be a ReadResponse.
func (csvEncoder) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	res, err := readResponse(value)
	if err != nil {
		return err
	}
	groups, err := newCSVGroups(res)
	if err != nil {
		return err
	}
	columns := csvColumns(groups)
	if err = openColumns(ctx, res, columns); err != nil {
		return err
	}
	buf := make([]byte, 0, csvFlushSize)
	for i := range groups {
		for j := range groups[i].columns {
			if i > 0 || j > 0 {
				buf = append(buf, ',')
			}
			buf = appendCSVText(buf, []byte(groups[i].columns[j].name))
		}
	}
	buf = append(buf, '\n')
	buf, err = writeCSVRows(w, buf, groups)
	if err = errors.Combine(err, closeColumns(columns)); err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// csvGroup holds the columns sharing an index channel. The first column is the index.
type csvGroup struct{ columns []column }

// newCSVGroups builds the columns of the response, grouped by index channel. It
// returns a validation error when the response leaves out the index of a channel or
// carries a limit reduction.
func newCSVGroups(res ReadResponse) ([]csvGroup, error) {
	indexGroups, err := groupByIndex(res)
	if err != nil {
		return nil, err
	}
	groups := make([]csvGroup, len(indexGroups))
	for i, channels := range indexGroups {
		groups[i].columns = make([]column, len(channels))
		for j, ch := range channels {
			groups[i].columns[j] = newColumn(ch)
		}
	}
	return groups, nil
}

// csvColumns returns a pointer to every column of groups.
func csvColumns(groups []csvGroup) []*column {
	var columns []*column
	for i := range groups {
		for j := range groups[i].columns {
			columns = append(columns, &groups[i].columns[j])
		}
	}
	return columns
}

// time returns the timestamp of the group's next row and reports whether the group has
// one.
func (g *csvGroup) time() (telem.TimeStamp, bool) {
	sample, n := g.columns[0].sample()
	if n == 0 {
		return 0, false
	}
	return telem.TimeStamp(telem.ByteOrder.Uint64(sample)), true
}

// appendRow appends the cells of the group's next row and moves its columns past the
// samples it writes. A data channel's cell is empty unless its next sample shares the
// alignment of the index's. Only valid when the group has a row.
func (g *csvGroup) appendRow(dst []byte) ([]byte, error) {
	index := &g.columns[0]
	alignment := index.alignment()
	dst, err := appendCSVSample(dst, index)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(g.columns); i++ {
		dst = append(dst, ',')
		c := &g.columns[i]
		if _, n := c.sample(); n == 0 {
			continue
		}
		if a := c.alignment(); a > alignment {
			continue
		} else if a < alignment {
			return nil, c.unindexedError()
		}
		if dst, err = appendCSVSample(dst, c); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// writeCSVRows appends the rows of the read in timestamp order, writing dst to w each
// time it fills. It returns the rows not yet written.
func writeCSVRows(w io.Writer, dst []byte, groups []csvGroup) ([]byte, error) {
	for {
		var (
			first telem.TimeStamp
			found bool
		)
		for i := range groups {
			if t, ok := groups[i].time(); ok && (!found || t < first) {
				first, found = t, true
			}
		}
		if !found {
			return dst, nil
		}
		for i := range groups {
			g := &groups[i]
			if i > 0 {
				dst = append(dst, ',')
			}
			if t, ok := g.time(); ok && t == first {
				var err error
				if dst, err = g.appendRow(dst); err != nil {
					return nil, err
				}
				continue
			}
			for range len(g.columns) - 1 {
				dst = append(dst, ',')
			}
		}
		dst = append(dst, '\n')
		if len(dst) < csvFlushSize {
			continue
		}
		if _, err := w.Write(dst); err != nil {
			return nil, err
		}
		dst = dst[:0]
	}
}

// appendCSVSample appends the sample the column points at, if it holds one, and moves
// the column past it.
func appendCSVSample(dst []byte, c *column) ([]byte, error) {
	sample, n := c.sample()
	if n == 0 {
		return dst, nil
	}
	c.advance(n)
	start := len(dst)
	dst, err := telem.AppendSampleText(dst, c.dataType, sample)
	if err != nil {
		return nil, err
	}
	// Only variable-length text can hold a character that needs quoting.
	if c.size != 0 {
		return dst, nil
	}
	if len(dst) == start {
		return append(dst, `""`...), nil
	}
	return quoteCSVField(dst, start), nil
}

// appendCSVText appends v as a field, quoting it when needed.
func appendCSVText(dst, v []byte) []byte {
	return quoteCSVField(append(dst, v...), len(dst))
}

// quoteCSVField quotes the field that starts at dst[start] in place when it holds a
// comma, quote, or line break.
func quoteCSVField(dst []byte, start int) []byte {
	if !bytes.ContainsAny(dst[start:], ",\"\n\r") {
		return dst
	}
	var (
		end    = len(dst)
		quotes = bytes.Count(dst[start:], []byte{'"'})
	)
	dst = slices.Grow(dst, quotes+2)[:end+quotes+2]
	w := len(dst) - 1
	dst[w] = '"'
	for r := end - 1; r >= start; r-- {
		w--
		dst[w] = dst[r]
		if dst[r] == '"' {
			w--
			dst[w] = '"'
		}
	}
	dst[start] = '"'
	return dst
}
