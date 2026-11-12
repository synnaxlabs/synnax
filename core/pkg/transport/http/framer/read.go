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

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	// recordData holds a frame encoded by the frame codec.
	recordData byte = 0
	// recordTerminator ends the body. Its payload is empty when the read completed and
	// a JSON error payload when it did not.
	recordTerminator byte = 1
)

const recordHeaderSize = 5

// NewFrameEncoder returns an encoder that writes a read as a sequence of records, each
// a one-byte kind, a four-byte little-endian length, and that many bytes of payload.
// Data records hold the requested channels, and only those, encoded by the frame codec.
// The last record is a terminator: empty when the read completed, a JSON error payload
// when it did not. A body that ends without a terminator was cut short in transit. ctx
// bounds the iterators the encoder opens.
func NewFrameEncoder(ctx context.Context) xhttp.Encoder {
	return frameEncoder{ctx: ctx}
}

type frameEncoder struct{ ctx context.Context }

// ContentType implements http.Encoder.
func (frameEncoder) ContentType() string { return frameContentType }

// Encode implements encoding.Encoder. It holds the whole read in memory, so prefer
// EncodeStream.
func (e frameEncoder) Encode(value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeStream implements encoding.Encoder. value must be a ReadResponse.
func (e frameEncoder) EncodeStream(w io.Writer, value any) error {
	res, err := readResponse(value)
	if err != nil {
		return err
	}
	keys, dataTypes := keysAndDataTypes(res.Channels)
	iter, err := res.OpenIterator(e.ctx, keys, res.Bounds)
	if err != nil {
		return err
	}
	c := codec.NewStatic(keys, dataTypes)
	var buf bytes.Buffer
	write := func(fr Frame) error {
		buf.Reset()
		if err := c.EncodeStream(e.ctx, &buf, fr); err != nil {
			return err
		}
		return writeRecord(w, recordData, buf.Bytes())
	}
	readErr := drainer(res)(iter, write)
	var payload []byte
	if readErr != nil {
		fErr := errors.Encode(e.ctx, readErr, false)
		if payload, err = json.Codec.Encode(fErr); err != nil {
			return err
		}
	}
	return writeRecord(w, recordTerminator, payload)
}

func writeRecord(w io.Writer, kind byte, payload []byte) error {
	var header [recordHeaderSize]byte
	header[0] = kind
	telem.ByteOrder.PutUint32(header[1:], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// drain seeks iter to the start of its bounds and calls f with every frame it returns,
// stopping at the first error. It always closes iter, and it returns the read error
// that Close reports.
func drain(iter *svcframer.Iterator, f func(Frame) error) error {
	var err error
	if iter.SeekFirst() {
		for iter.Next(IteratorAutoSpan) {
			if err = f(iter.Value()); err != nil {
				break
			}
		}
	}
	return errors.Combine(err, iter.Close())
}

// drainLatest seeks iter to the end of its bounds and calls f with the frame it reads
// back, which holds up to the chunk size of iter for each channel. It always closes
// iter, and it returns the read error that Close reports.
func drainLatest(iter *svcframer.Iterator, f func(Frame) error) error {
	var err error
	if iter.SeekLast() && iter.Prev(IteratorAutoSpan) {
		err = f(iter.Value())
	}
	return errors.Combine(err, iter.Close())
}

// drainEarliest seeks iter to the start of its bounds and calls f with the frame it
// reads, which holds up to the chunk size of iter for each channel. It always closes
// iter, and it returns the read error that Close reports.
func drainEarliest(iter *svcframer.Iterator, f func(Frame) error) error {
	var err error
	if iter.SeekFirst() && iter.Next(IteratorAutoSpan) {
		err = f(iter.Value())
	}
	return errors.Combine(err, iter.Close())
}

// drainer returns the drain that reads the samples res asks for.
func drainer(res ReadResponse) func(*svcframer.Iterator, func(Frame) error) error {
	switch {
	case res.Latest > 0:
		return drainLatest
	case res.Earliest > 0:
		return drainEarliest
	default:
		return drain
	}
}

func readResponse(value any) (ReadResponse, error) {
	res, ok := value.(ReadResponse)
	if !ok {
		return res, errors.Newf("[api.Read] cannot encode a value of type %T", value)
	}
	return res, nil
}

// groupByIndex groups the channels of res by the index channel that timestamps them,
// ordered by the first channel each group timestamps. Each group starts with its index.
// It returns a validation error when res leaves out the index of a channel or carries a
// limit reduction.
func groupByIndex(res ReadResponse) ([][]channel.Channel, error) {
	if _, ok := res.Reduction.Variant.(telem.LimitReduction); ok {
		// A limit leaves non-numeric channels raw beside their reduced index, so their
		// samples have no row.
		return nil, errors.Wrap(
			validate.ErrValidation,
			"a CSV or TDMS read cannot take a limit reduction: use a stride",
		)
	}
	all := slices.Concat(res.Channels, res.Indexes)
	records := make(map[channel.Key]channel.Channel, len(all))
	for _, ch := range all {
		records[ch.Key()] = ch
	}
	position := make(map[channel.Key]int, len(all))
	groups := make([][]channel.Channel, 0, len(all))
	for _, ch := range res.Channels {
		index, ok := records[ch.Index()]
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"channel %s has no index in the read: add its index to keys or set "+
					"timestamps_included",
				ch.Name,
			)
		}
		i, ok := position[index.Key()]
		if !ok {
			i, position[index.Key()] = len(groups), len(groups)
			groups = append(groups, []channel.Channel{index})
		}
		if !ch.IsIndex {
			groups[i] = append(groups[i], ch)
		}
	}
	return groups, nil
}

// column walks the samples of one channel. It reads the channel through its own
// iterator, one chunk at a time, so it never holds more than one chunk ahead of the
// samples written.
type column struct {
	key channel.Key
	// index is the key of the index channel that timestamps the column.
	index    channel.Key
	name     string
	dataType telem.DataType
	// size is the byte size of one sample, or 0 for a variable-length data type.
	size int
	iter *svcframer.Iterator
	// done is true once iter has no samples left.
	done bool
	// pending holds the series read for the channel that are not fully written.
	pending []telem.Series
	// offset is the byte position of the next sample within the first pending series.
	offset int
	// position is the index of the next sample within the first pending series.
	position uint32
}

func newColumn(ch channel.Channel) column {
	c := column{
		key:      ch.Key(),
		index:    ch.Index(),
		name:     ch.Name,
		dataType: ch.DataType,
	}
	if !ch.DataType.IsVariable() {
		c.size = int(ch.DataType.Density())
	}
	return c
}

// openColumns opens an iterator for every column. On failure, it closes the iterators
// it opened.
func openColumns(ctx context.Context, res ReadResponse, columns []*column) error {
	bounds := make(map[channel.Key]telem.TimeRange)
	for _, c := range columns {
		b, ok := bounds[c.index]
		if !ok {
			var err error
			if b, err = rowBounds(ctx, res, c.index); err != nil {
				return errors.Combine(err, closeColumns(columns))
			}
			bounds[c.index] = b
		}
		iter, err := res.OpenIterator(ctx, channel.Keys{c.key}, b)
		if err != nil {
			return errors.Combine(err, closeColumns(columns))
		}
		c.iter, c.done = iter, !iter.SeekFirst()
	}
	return nil
}

// rowBounds returns the bounds of the rows of index: the bounds of res, narrowed to the
// newest or oldest rows when res asks for them. The columns of an index read the same
// bounds, so a channel that lags its index leaves its cells empty.
func rowBounds(
	ctx context.Context,
	res ReadResponse,
	index channel.Key,
) (telem.TimeRange, error) {
	if res.Latest == 0 && res.Earliest == 0 {
		return res.Bounds, nil
	}
	iter, err := res.OpenIterator(ctx, channel.Keys{index}, res.Bounds)
	if err != nil {
		return telem.TimeRange{}, err
	}
	bounds := res.Bounds
	err = drainer(res)(iter, func(fr Frame) error {
		first, last := telem.TimeStampMax, telem.TimeStampMin
		for _, s := range fr.Get(index).Series {
			first = min(first, s.ValueAt[telem.TimeStamp](0))
			last = max(last, s.ValueAt[telem.TimeStamp](-1))
		}
		bounds = telem.TimeRange{Start: first, End: last.Add(telem.Nanosecond)}
		return nil
	})
	if err != nil {
		return telem.TimeRange{}, err
	}
	return bounds, nil
}

// closeColumns closes the iterator of every column and returns their read errors.
func closeColumns(columns []*column) error {
	var err error
	for _, c := range columns {
		if c.iter != nil {
			err = errors.Combine(err, c.iter.Close())
			c.iter = nil
		}
	}
	return err
}

// sample discards the series the column has fully written, reading the next chunk when
// none is left, then returns the sample the column points at and the number of bytes it
// occupies. n is 0 when no sample remains.
func (c *column) sample() (sample []byte, n int) {
	for {
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
		if c.done {
			return nil, 0
		}
		if c.done = !c.iter.Next(IteratorAutoSpan); !c.done {
			c.pending = c.iter.Value().Get(c.key).Series
		}
	}
}

// alignment returns the alignment of the sample the column points at. Only valid when
// sample returns one.
func (c *column) alignment() telem.Alignment {
	return c.pending[0].Alignment.AddSamples(c.position * c.pending[0].Multiple())
}

// advance moves the column past the sample it points at, which occupies n bytes.
func (c *column) advance(n int) {
	c.offset += n
	c.position++
}

// remaining returns the number of samples left in the series the column points at. Only
// valid for a fixed-size data type, and when sample returns one.
func (c *column) remaining() int {
	return (len(c.pending[0].Data) - c.offset) / c.size
}

// skip moves the column past the next k samples of its series. Only valid for a
// fixed-size data type, and when the series holds them.
func (c *column) skip(k int) {
	c.offset += k * c.size
	c.position += uint32(k)
}

// unindexedError returns the error for a sample the column points at that sits at no
// alignment of its index.
func (c *column) unindexedError() error {
	return errors.Newf(
		"[api.Read] channel %s has a sample at alignment %s with no index sample",
		c.name,
		c.alignment(),
	)
}

func keysAndDataTypes(channels []channel.Channel) (channel.Keys, []telem.DataType) {
	keys := make(channel.Keys, len(channels))
	dataTypes := make([]telem.DataType, len(channels))
	for i, ch := range channels {
		keys[i], dataTypes[i] = ch.Key(), ch.DataType
	}
	return keys, dataTypes
}
