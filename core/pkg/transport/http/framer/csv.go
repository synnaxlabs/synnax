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

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
)

// csvFlushSize is the number of bytes the encoder accumulates before writing them out.
const csvFlushSize = 64 << 10

// CSVEncoder encodes a read as CSV. Channels are grouped by the index channel that
// timestamps them. Each group contributes its index column, then one column per data
// channel, headed by the channel's name. Rows are ordered by timestamp, and a group's
// cells are empty at a timestamp it has no sample for. Channels with no index are
// excluded. A read that fails after the first byte truncates the body.
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
func (csvEncoder) EncodeStream(_ context.Context, w io.Writer, value any) error {
	res, err := readResponse(value)
	if err != nil {
		return err
	}
	groups := newCSVGroups(res)
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
	if err = drain(res, func(fr Frame) error {
		for i := range groups {
			groups[i].push(fr)
		}
		var err error
		buf, err = writeCSVRows(w, buf, groups, false)
		return err
	}); err != nil {
		return err
	}
	if buf, err = writeCSVRows(w, buf, groups, true); err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// csvGroup holds the columns sharing an index channel. The first column is the index.
type csvGroup struct {
	columns []csvColumn
	// watermark is the latest timestamp the iterator has returned for the index.
	watermark telem.TimeStamp
	// exhausted is true once the iterator returns a frame without the index.
	exhausted bool
	// writable holds the result of ready for the row being assembled.
	writable bool
}

// csvColumn walks the samples of one channel across the series returned for it.
type csvColumn struct {
	key      channel.Key
	name     string
	dataType telem.DataType
	// size is the byte size of one sample, or 0 for a variable-length data type.
	size int
	// pending holds the series returned for the channel that are not fully written.
	pending []telem.Series
	// offset is the byte position of the next sample within the first pending series.
	offset int
}

// newCSVGroups builds the columns of the response, grouped by index channel and
// ordered by the first channel each group timestamps.
func newCSVGroups(res ReadResponse) []csvGroup {
	all := slices.Concat(res.Channels, res.Indexes)
	records := make(map[channel.Key]channel.Channel, len(all))
	for _, ch := range all {
		records[ch.Key()] = ch
	}
	position := make(map[channel.Key]int, len(all))
	groups := make([]csvGroup, 0, len(all))
	for _, ch := range res.Channels {
		index := ch.Index()
		if index == 0 {
			continue
		}
		i, ok := position[index]
		if !ok {
			i, position[index] = len(groups), len(groups)
			groups = append(groups, csvGroup{
				columns: []csvColumn{newCSVColumn(records[index])},
			})
		}
		if !ch.IsIndex {
			groups[i].columns = append(groups[i].columns, newCSVColumn(ch))
		}
	}
	return groups
}

func newCSVColumn(ch channel.Channel) csvColumn {
	c := csvColumn{key: ch.Key(), name: ch.Name, dataType: ch.DataType}
	if !ch.DataType.IsVariable() {
		c.size = int(ch.DataType.Density())
	}
	return c
}

// push stages the series fr holds for each of the group's columns.
func (g *csvGroup) push(fr Frame) {
	for i := range g.columns {
		c := &g.columns[i]
		c.pending = append(c.pending, fr.Get(c.key).Series...)
	}
	index := fr.Get(g.columns[0].key).Series
	g.exhausted = len(index) == 0
	for _, s := range index {
		if s.Len() > 0 {
			g.watermark = s.ValueAt[telem.TimeStamp](-1)
		}
	}
}

// ready reports whether the group can write its next row. Every column must hold a
// sample, unless flush is set, in which case the index alone is enough.
func (g *csvGroup) ready(flush bool) bool {
	for i := range g.columns {
		if _, n := g.columns[i].sample(); n == 0 && (i == 0 || !flush) {
			return false
		}
	}
	return true
}

// time returns the timestamp of the group's next row. Only valid when the index holds
// a sample.
func (g *csvGroup) time() telem.TimeStamp {
	sample, _ := g.columns[0].sample()
	return telem.TimeStamp(telem.ByteOrder.Uint64(sample))
}

// bound returns a lower bound on the timestamp of the group's next row.
func (g *csvGroup) bound() telem.TimeStamp {
	if _, n := g.columns[0].sample(); n > 0 {
		return g.time()
	}
	if g.exhausted {
		return telem.TimeStampMax
	}
	return g.watermark + 1
}

// sample discards the series the column has fully written, then returns the sample the
// column points at and the number of bytes it occupies. n is 0 when no sample remains.
func (c *csvColumn) sample() (sample []byte, n int) {
	for len(c.pending) > 0 {
		data := c.pending[0].Data[c.offset:]
		if c.size == 0 {
			sample, n = telem.UnmarshalVariableSample(data)
		} else if len(data) >= c.size {
			sample, n = data[:c.size], c.size
		}
		if n > 0 {
			return sample, n
		}
		c.pending, c.offset = c.pending[1:], 0
	}
	return nil, 0
}

// writeCSVRows appends every row that no later frame can precede or add cells to,
// writing dst to w each time it fills. It returns the rows not yet written. When flush
// is set, no frame is left to wait for, so it appends every remaining row.
func writeCSVRows(
	w io.Writer,
	dst []byte,
	groups []csvGroup,
	flush bool,
) ([]byte, error) {
	for {
		first := telem.TimeStampMax
		for i := range groups {
			g := &groups[i]
			if g.writable = g.ready(flush); g.writable {
				first = min(first, g.time())
			}
		}
		if first == telem.TimeStampMax {
			return dst, nil
		}
		for i := range groups {
			if !flush && !groups[i].writable && groups[i].bound() <= first {
				return dst, nil
			}
		}
		for i := range groups {
			g := &groups[i]
			write := g.writable && g.time() == first
			for j := range g.columns {
				if i > 0 || j > 0 {
					dst = append(dst, ',')
				}
				if !write {
					continue
				}
				var err error
				if dst, err = appendCSVSample(dst, &g.columns[j]); err != nil {
					return dst, err
				}
			}
		}
		dst = append(dst, '\n')
		if len(dst) < csvFlushSize {
			continue
		}
		if _, err := w.Write(dst); err != nil {
			return dst, err
		}
		dst = dst[:0]
	}
}

// appendCSVSample appends the sample the column points at, if it holds one, and moves
// the column past it.
func appendCSVSample(dst []byte, c *csvColumn) ([]byte, error) {
	sample, n := c.sample()
	if n == 0 {
		return dst, nil
	}
	c.offset += n
	start := len(dst)
	dst, err := telem.AppendSampleText(dst, c.dataType, sample)
	// Only variable-length text can hold a character that needs quoting.
	if err != nil || c.size != 0 {
		return dst, err
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
