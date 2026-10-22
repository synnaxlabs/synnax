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
// no sample there. A channel is excluded when the read does not include its index. A
// read that fails after the first byte truncates the body.
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
	// position is the index of the next sample within the first pending series.
	position uint32
	// stride is the alignment distance between consecutive samples of a series.
	stride uint32
	// exhausted is true once the iterator returns a frame without the channel.
	exhausted bool
}

// newCSVGroups builds the columns of the response, grouped by index channel and
// ordered by the first channel each group timestamps.
func newCSVGroups(res ReadResponse) []csvGroup {
	var (
		all    = slices.Concat(res.Channels, res.Indexes)
		stride = max(res.DownsampleFactor, 1)
	)
	records := make(map[channel.Key]channel.Channel, len(all))
	for _, ch := range all {
		records[ch.Key()] = ch
	}
	position := make(map[channel.Key]int, len(all))
	groups := make([]csvGroup, 0, len(all))
	for _, ch := range res.Channels {
		index, ok := records[ch.Index()]
		if !ok {
			continue
		}
		i, ok := position[index.Key()]
		if !ok {
			i, position[index.Key()] = len(groups), len(groups)
			groups = append(groups, csvGroup{
				columns: []csvColumn{newCSVColumn(index, stride)},
			})
		}
		if !ch.IsIndex {
			groups[i].columns = append(groups[i].columns, newCSVColumn(ch, stride))
		}
	}
	return groups
}

func newCSVColumn(ch channel.Channel, stride uint32) csvColumn {
	c := csvColumn{key: ch.Key(), name: ch.Name, dataType: ch.DataType, stride: stride}
	if !ch.DataType.IsVariable() {
		c.size = int(ch.DataType.Density())
	}
	return c
}

// push stages the series fr holds for each of the group's columns.
func (g *csvGroup) push(fr Frame) {
	for i := range g.columns {
		c := &g.columns[i]
		series := fr.Get(c.key).Series
		c.pending = append(c.pending, series...)
		c.exhausted = len(series) == 0
	}
	for _, s := range fr.Get(g.columns[0].key).Series {
		if s.Len() > 0 {
			g.watermark = s.ValueAt[telem.TimeStamp](-1)
		}
	}
}

// ready reports whether the group can write its next row. The index must hold a sample,
// and unless flush is set, each data channel must hold a sample or have none left.
func (g *csvGroup) ready(flush bool) bool {
	for i := range g.columns {
		c := &g.columns[i]
		if _, n := c.sample(); n == 0 && (i == 0 || (!flush && !c.exhausted)) {
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
	if g.columns[0].exhausted {
		return telem.TimeStampMax
	}
	return g.watermark + 1
}

// appendRow appends the cells of the group's next row and moves its columns past the
// samples it writes. A data channel's cell is empty unless its next sample shares the
// alignment of the index's. Only valid when ready.
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
			return nil, errors.Newf(
				"[api.Read] channel %s has a sample at alignment %s with no index sample",
				c.name,
				a,
			)
		}
		if dst, err = appendCSVSample(dst, c); err != nil {
			return nil, err
		}
	}
	return dst, nil
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
		c.pending, c.offset, c.position = c.pending[1:], 0, 0
	}
	return nil, 0
}

// alignment returns the alignment of the sample the column points at. Only valid when
// sample returns one.
func (c *csvColumn) alignment() telem.Alignment {
	return c.pending[0].Alignment.AddSamples(c.position * c.stride)
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
			if i > 0 {
				dst = append(dst, ',')
			}
			if g.writable && g.time() == first {
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
func appendCSVSample(dst []byte, c *csvColumn) ([]byte, error) {
	sample, n := c.sample()
	if n == 0 {
		return dst, nil
	}
	c.offset += n
	c.position++
	start := len(dst)
	dst, err := telem.AppendSampleText(dst, c.dataType, sample)
	if err != nil {
		return nil, err
	}
	// Only variable-length text can hold a character that needs quoting.
	if c.size != 0 {
		return dst, nil
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
