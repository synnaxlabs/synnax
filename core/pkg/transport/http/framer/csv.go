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
	"math"
	"slices"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
)

const (
	// csvFlushSize is the number of bytes the encoder accumulates before writing them
	// out.
	csvFlushSize = 64 << 10
	// variablePrefixSize is the size of the length prefix a series writes ahead of each
	// variable-length sample.
	variablePrefixSize = 4
)

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
		if buf = appendCSVRows(buf, groups, false); len(buf) < csvFlushSize {
			return nil
		}
		_, err := w.Write(buf)
		buf = buf[:0]
		return err
	}); err != nil {
		return err
	}
	_, err = w.Write(appendCSVRows(buf, groups, true))
	return err
}

// csvGroup holds the columns sharing an index channel. The first column is the index.
type csvGroup struct{ columns []csvColumn }

// csvColumn walks the samples of one channel across the series returned for it.
type csvColumn struct {
	key      channel.Key
	name     string
	dataType telem.DataType
	// pending holds the series returned for the channel that are not fully written.
	pending []telem.Series
	// sample is the index of the next sample to write within the first pending series.
	sample int64
	// length is the sample count of the first pending series.
	length int64
	// offset is the byte position of the next sample within the first pending series.
	// Only used for variable-length data types.
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
	return csvColumn{key: ch.Key(), name: ch.Name, dataType: ch.DataType}
}

// push stages the series fr holds for each of the group's columns.
func (g *csvGroup) push(fr Frame) {
	for i := range g.columns {
		c := &g.columns[i]
		c.pending = append(c.pending, fr.Get(c.key).Series...)
	}
}

// ready reports whether every column in the group has a sample to write.
func (g *csvGroup) ready() bool {
	for i := range g.columns {
		if !g.columns[i].ok() {
			return false
		}
	}
	return true
}

// time returns the timestamp of the group's next row. Only valid when ready.
func (g *csvGroup) time() telem.TimeStamp {
	c := &g.columns[0]
	return telem.ValueAt[telem.TimeStamp](c.pending[0], int(c.sample))
}

// advance moves every column in the group to its next sample.
func (g *csvGroup) advance() {
	for i := range g.columns {
		g.columns[i].advance()
	}
}

// ok discards the series the column has fully written and reports whether a sample
// remains.
func (c *csvColumn) ok() bool {
	for len(c.pending) > 0 {
		if c.sample == 0 {
			c.length, c.offset = c.pending[0].Len(), 0
		}
		if c.sample < c.length {
			return true
		}
		c.pending, c.sample = c.pending[1:], 0
	}
	return false
}

// value returns the bytes of the sample the column points at. Only valid when the
// column is ok.
func (c *csvColumn) value() []byte {
	data := c.pending[0].Data
	if !c.dataType.IsVariable() {
		size := int(c.dataType.Density())
		return data[int(c.sample)*size:][:size]
	}
	size := int(telem.ByteOrder.Uint32(data[c.offset:]))
	return data[c.offset+variablePrefixSize:][:size]
}

// advance moves the column to its next sample.
func (c *csvColumn) advance() {
	if c.dataType.IsVariable() {
		c.offset += variablePrefixSize + len(c.value())
	}
	c.sample++
}

// appendCSVRows appends every row the staged samples cover. Unless flush is set, it
// holds back the latest timestamp, since a later frame can still add columns to it.
func appendCSVRows(dst []byte, groups []csvGroup, flush bool) []byte {
	for {
		var first, last telem.TimeStamp
		found := false
		for i := range groups {
			if !groups[i].ready() {
				continue
			}
			t := groups[i].time()
			if !found || t < first {
				first = t
			}
			if !found || t > last {
				last = t
			}
			found = true
		}
		if !found || (!flush && first == last) {
			return dst
		}
		for i := range groups {
			g := &groups[i]
			write := g.ready() && g.time() == first
			for j := range g.columns {
				if i > 0 || j > 0 {
					dst = append(dst, ',')
				}
				if write {
					dst = appendCSVSample(dst, &g.columns[j])
				}
			}
			if write {
				g.advance()
			}
		}
		dst = append(dst, '\n')
	}
}

// appendCSVSample appends the sample the column points at. Only valid when the column
// is ok.
func appendCSVSample(dst []byte, c *csvColumn) []byte {
	b := c.value()
	switch c.dataType {
	case telem.Float64T:
		v := math.Float64frombits(telem.ByteOrder.Uint64(b))
		return strconv.AppendFloat(dst, v, 'f', -1, 64)
	case telem.Float32T:
		v := math.Float32frombits(telem.ByteOrder.Uint32(b))
		return strconv.AppendFloat(dst, float64(v), 'f', -1, 32)
	case telem.Int64T, telem.TimeStampT:
		return strconv.AppendInt(dst, int64(telem.ByteOrder.Uint64(b)), 10)
	case telem.Int32T:
		return strconv.AppendInt(dst, int64(int32(telem.ByteOrder.Uint32(b))), 10)
	case telem.Int16T:
		return strconv.AppendInt(dst, int64(int16(telem.ByteOrder.Uint16(b))), 10)
	case telem.Int8T:
		return strconv.AppendInt(dst, int64(int8(b[0])), 10)
	case telem.Uint64T:
		return strconv.AppendUint(dst, telem.ByteOrder.Uint64(b), 10)
	case telem.Uint32T:
		return strconv.AppendUint(dst, uint64(telem.ByteOrder.Uint32(b)), 10)
	case telem.Uint16T:
		return strconv.AppendUint(dst, uint64(telem.ByteOrder.Uint16(b)), 10)
	case telem.Uint8T, telem.BoolT:
		return strconv.AppendUint(dst, uint64(b[0]), 10)
	default:
		return appendCSVText(dst, b)
	}
}

// appendCSVText appends v as a field, quoting it when it holds a comma, quote, or line
// break.
func appendCSVText(dst, v []byte) []byte {
	if !bytes.ContainsAny(v, ",\"\n\r") {
		return append(dst, v...)
	}
	dst = append(dst, '"')
	for _, b := range v {
		if b == '"' {
			dst = append(dst, b)
		}
		dst = append(dst, b)
	}
	return append(dst, '"')
}
