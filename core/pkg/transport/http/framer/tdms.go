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
	"fmt"
	"io"
	"math"
	"math/bits"
	"slices"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// TDMSEncoder encodes a read as an NI TDMS file. Each index channel heads a group named
// after it, followed by the float channels it timestamps. A float channel holds NaN,
// which NI tools read as no value, where it has no sample. TDMS has no missing value
// for other types, so each other channel gets its own group: the timestamps of its
// samples, then the samples. A channel whose name another channel of the read shares
// takes its name and key, such as "temp (1048577)". Each group sets wf_xcolumns to mark
// its first channel as its time channel. UUID, bytes, and JSON samples become TDMS
// strings in their text form. A read of a channel whose index it does not include, or a
// read under a limit reduction, fails before the first byte. A read that fails after
// the first byte truncates the body.
var TDMSEncoder tdmsEncoder

type tdmsEncoder struct{}

var _ xhttp.Encoder = tdmsEncoder{}

// ContentType implements http.Encoder.
func (tdmsEncoder) ContentType() string { return "application/x-tdms" }

// Encode implements encoding.Encoder. It holds the whole read in memory, so prefer
// EncodeStream.
func (e tdmsEncoder) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeStream implements encoding.Encoder. value must be a ReadResponse.
func (tdmsEncoder) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	res, err := readResponse(value)
	if err != nil {
		return err
	}
	f, err := newTDMSFile(res)
	if err != nil {
		return err
	}
	columns := f.columns()
	if err = openColumns(ctx, res, columns); err != nil {
		return err
	}
	return errors.Combine(f.write(w), closeColumns(columns))
}

const (
	tdmsVersion = 4713
	// tdmsLeadInSize is the byte size of the lead-in that opens every segment.
	tdmsLeadInSize = 28
	// tdmsNoRawData is the raw data index of an object with no data in a segment.
	tdmsNoRawData uint32 = 0xFFFFFFFF
	// tdmsSegmentSamples is the number of index samples after which a segment ends.
	tdmsSegmentSamples = 100_000
)

// Flags of a segment's table of contents.
const (
	tdmsTocMetaData   uint32 = 1 << 1
	tdmsTocNewObjList uint32 = 1 << 2
	tdmsTocRawData    uint32 = 1 << 3
)

// tdmsDataType is a TDMS data type code.
type tdmsDataType uint32

const (
	tdmsI8        tdmsDataType = 0x01
	tdmsI16       tdmsDataType = 0x02
	tdmsI32       tdmsDataType = 0x03
	tdmsI64       tdmsDataType = 0x04
	tdmsU8        tdmsDataType = 0x05
	tdmsU16       tdmsDataType = 0x06
	tdmsU32       tdmsDataType = 0x07
	tdmsU64       tdmsDataType = 0x08
	tdmsF32       tdmsDataType = 0x09
	tdmsF64       tdmsDataType = 0x0A
	tdmsString    tdmsDataType = 0x20
	tdmsBoolean   tdmsDataType = 0x21
	tdmsTimeStamp tdmsDataType = 0x44
)

// tdmsFile writes the segments of a file, reusing its buffers across them.
type tdmsFile struct {
	groups  []tdmsGroup
	objects uint32
	meta    []byte
	raw     [][]byte
}

// tdmsGroup holds an index channel and the channels it timestamps.
type tdmsGroup struct {
	index column
	// path is the object path of the group the index heads.
	path      string
	indexPath string
	// times holds the TDMS timestamps of the segment's index samples.
	times []byte
	// channels holds the channels that share the index's timestamps, then the channels
	// with groups of their own.
	channels []tdmsChannel
}

// tdmsChannel is a data channel of the file.
type tdmsChannel struct {
	column
	// groupPath is the object path of the channel's own group, or empty when the
	// channel shares the timestamps of its index.
	groupPath string
	path      string
	// timePath is the object path of the channel's own time channel.
	timePath string
	tdmsType tdmsDataType
	// nan is the sample a channel that shares its index's timestamps holds where it has
	// none.
	nan []byte
	// count is the number of samples the segment holds for the channel.
	count int
	// data holds the segment's samples, or the end offset of each string.
	data []byte
	// text holds the segment's strings.
	text []byte
	// times holds the TDMS timestamps of the segment's samples, for a channel with its
	// own time channel.
	times []byte
}

func newTDMSFile(res ReadResponse) (*tdmsFile, error) {
	indexGroups, err := groupByIndex(res)
	if err != nil {
		return nil, err
	}
	names, err := tdmsNames(indexGroups)
	if err != nil {
		return nil, err
	}
	f := &tdmsFile{groups: make([]tdmsGroup, len(indexGroups))}
	for i, channels := range indexGroups {
		indexName := names[channels[0].Key()]
		g := &f.groups[i]
		g.index, g.path = newColumn(channels[0]), tdmsPath(indexName)
		g.indexPath = tdmsPath(indexName, indexName)
		var own []tdmsChannel
		for _, ch := range channels[1:] {
			name := names[ch.Key()]
			t, err := newTDMSDataType(ch.DataType)
			if err != nil {
				return nil, err
			}
			c := tdmsChannel{
				column:   newColumn(ch),
				tdmsType: t,
				nan:      tdmsNaN(t),
			}
			if c.nan != nil {
				c.path = tdmsPath(indexName, name)
				g.channels = append(g.channels, c)
				continue
			}
			c.groupPath, c.path = tdmsPath(name), tdmsPath(name, name)
			c.timePath = tdmsPath(name, indexName)
			own = append(own, c)
		}
		g.channels = append(g.channels, own...)
	}
	return f, nil
}

// tdmsNames returns the name each channel of groups takes in the file: its own name, or
// its name and key when another channel shares the name. It returns a validation error
// when two channels still take the same name.
func tdmsNames(groups [][]channel.Channel) (map[channel.Key]string, error) {
	counts := make(map[string]int)
	for _, group := range groups {
		for _, ch := range group {
			counts[ch.Name]++
		}
	}
	var (
		names = make(map[channel.Key]string, len(counts))
		taken = make(map[string]channel.Key, len(counts))
	)
	for _, group := range groups {
		for _, ch := range group {
			name := ch.Name
			if counts[name] > 1 {
				name = fmt.Sprintf("%s (%s)", name, ch.Key())
			}
			if other, ok := taken[name]; ok {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"channels %s and %s both take the name %s in a TDMS file",
					other,
					ch.Key(),
					name,
				)
			}
			names[ch.Key()], taken[name] = name, ch.Key()
		}
	}
	return names, nil
}

// columns returns a pointer to every column of the file.
func (f *tdmsFile) columns() []*column {
	var columns []*column
	for i := range f.groups {
		g := &f.groups[i]
		columns = append(columns, &g.index)
		for j := range g.channels {
			columns = append(columns, &g.channels[j].column)
		}
	}
	return columns
}

func newTDMSDataType(dt telem.DataType) (tdmsDataType, error) {
	switch dt {
	case telem.Int8T:
		return tdmsI8, nil
	case telem.Int16T:
		return tdmsI16, nil
	case telem.Int32T:
		return tdmsI32, nil
	case telem.Int64T:
		return tdmsI64, nil
	case telem.Uint8T:
		return tdmsU8, nil
	case telem.Uint16T:
		return tdmsU16, nil
	case telem.Uint32T:
		return tdmsU32, nil
	case telem.Uint64T:
		return tdmsU64, nil
	case telem.Float32T:
		return tdmsF32, nil
	case telem.Float64T:
		return tdmsF64, nil
	case telem.BooleanT:
		return tdmsBoolean, nil
	case telem.TimestampT:
		return tdmsTimeStamp, nil
	case telem.StringT, telem.JSONT, telem.UUIDT, telem.BytesT:
		return tdmsString, nil
	default:
		return 0, errors.Newf("data type %s has no TDMS form", dt)
	}
}

// tdmsNaN returns the NaN sample of t, or nil when t is not a float type.
func tdmsNaN(t tdmsDataType) []byte {
	switch t {
	case tdmsF32:
		return telem.ByteOrder.AppendUint32(nil, math.Float32bits(float32(math.NaN())))
	case tdmsF64:
		return telem.ByteOrder.AppendUint64(nil, math.Float64bits(math.NaN()))
	default:
		return nil
	}
}

// tdmsPath returns the object path of the given names, quoting each one.
func tdmsPath(names ...string) string {
	var b strings.Builder
	for _, name := range names {
		b.WriteString("/'")
		b.WriteString(strings.ReplaceAll(name, "'", "''"))
		b.WriteByte('\'')
	}
	return b.String()
}

// write writes the file: a segment that declares its groups and channels, then the
// segments of each group's samples.
func (f *tdmsFile) write(w io.Writer) error {
	if err := f.writeHeader(w); err != nil {
		return err
	}
	for i := range f.groups {
		g := &f.groups[i]
		for {
			n, err := g.read()
			if err != nil {
				return err
			}
			if n == 0 {
				break
			}
			if err = f.writeSegment(w, g, n); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *tdmsFile) writeHeader(w io.Writer) error {
	f.reset()
	f.appendDeclaration("/", false)
	for i := range f.groups {
		g := &f.groups[i]
		f.appendDeclaration(g.path, true)
		for j := range g.channels {
			if c := &g.channels[j]; c.groupPath != "" {
				f.appendDeclaration(c.groupPath, true)
			}
		}
	}
	for i := range f.groups {
		g := &f.groups[i]
		f.appendDeclaration(g.indexPath, false)
		for j := range g.channels {
			c := &g.channels[j]
			if c.groupPath != "" {
				f.appendDeclaration(c.timePath, false)
			}
			f.appendDeclaration(c.path, false)
		}
	}
	return f.flush(w, tdmsTocMetaData|tdmsTocNewObjList)
}

// appendDeclaration appends an object with no raw data. A group's object marks its
// first channel as its time channel.
func (f *tdmsFile) appendDeclaration(path string, group bool) {
	f.objects++
	f.meta = appendTDMSString(f.meta, path)
	f.meta = telem.ByteOrder.AppendUint32(f.meta, tdmsNoRawData)
	if !group {
		f.meta = telem.ByteOrder.AppendUint32(f.meta, 0)
		return
	}
	f.meta = telem.ByteOrder.AppendUint32(f.meta, 1)
	f.meta = appendTDMSString(f.meta, "wf_xcolumns")
	f.meta = telem.ByteOrder.AppendUint32(f.meta, uint32(tdmsString))
	f.meta = appendTDMSString(f.meta, "one")
}

// writeSegment writes a segment of the n index samples g read and the channel samples
// at their alignments.
func (f *tdmsFile) writeSegment(w io.Writer, g *tdmsGroup, n int) error {
	f.reset()
	f.appendData(g.indexPath, tdmsTimeStamp, n, g.times, nil)
	for i := range g.channels {
		c := &g.channels[i]
		if c.count == 0 {
			continue
		}
		if c.groupPath != "" {
			f.appendData(c.timePath, tdmsTimeStamp, c.count, c.times, nil)
		}
		f.appendData(c.path, c.tdmsType, c.count, c.data, c.text)
	}
	return f.flush(w, tdmsTocMetaData|tdmsTocNewObjList|tdmsTocRawData)
}

// appendData appends an object that holds count samples of type t in the segment,
// stored as data followed by text.
func (f *tdmsFile) appendData(
	path string,
	t tdmsDataType,
	count int,
	data, text []byte,
) {
	f.objects++
	f.meta = appendTDMSString(f.meta, path)
	indexSize := uint32(20)
	if t == tdmsString {
		indexSize = 28
	}
	f.meta = telem.ByteOrder.AppendUint32(f.meta, indexSize)
	f.meta = telem.ByteOrder.AppendUint32(f.meta, uint32(t))
	f.meta = telem.ByteOrder.AppendUint32(f.meta, 1)
	f.meta = telem.ByteOrder.AppendUint64(f.meta, uint64(count))
	if t == tdmsString {
		f.meta = telem.ByteOrder.AppendUint64(f.meta, uint64(len(data)+len(text)))
	}
	f.meta = telem.ByteOrder.AppendUint32(f.meta, 0)
	f.raw = append(f.raw, data, text)
}

// read fills the group's buffers with its next index samples and the channel samples at
// their alignments. It stops after the series that reaches tdmsSegmentSamples, and
// returns the number of index samples read.
func (g *tdmsGroup) read() (int, error) {
	g.times = g.times[:0]
	for i := range g.channels {
		c := &g.channels[i]
		c.count, c.data, c.text, c.times = 0, c.data[:0], c.text[:0], c.times[:0]
	}
	var n int
	for n < tdmsSegmentSamples {
		if _, size := g.index.sample(); size == 0 {
			break
		}
		var (
			start = g.index.alignment()
			step  = g.index.pending[0].Multiple()
			k     = g.index.remaining()
			data  = g.index.pending[0].Data[g.index.offset:]
			from  = len(g.times)
		)
		for i := range k {
			ts := telem.TimeStamp(telem.ByteOrder.Uint64(data[8*i:]))
			g.times = appendTDMSTimeStamp(g.times, ts)
		}
		g.index.skip(k)
		for i := range g.channels {
			if err := g.channels[i].read(start, step, k, g.times[from:]); err != nil {
				return 0, err
			}
		}
		n += k
	}
	return n, nil
}

// read appends the channel's samples at the n index alignments from start, step apart
// and in one domain. times holds the TDMS timestamps of those alignments.
func (c *tdmsChannel) read(
	start telem.Alignment,
	step uint32,
	n int,
	times []byte,
) error {
	for i := 0; i < n; {
		if _, size := c.sample(); size == 0 {
			c.appendMissing(n - i)
			return nil
		}
		at, a := start.AddSamples(uint32(i)*step), c.alignment()
		if a < at {
			return c.unindexedError()
		}
		if a > at {
			gap := n - i
			if a.DomainIndex() == at.DomainIndex() {
				d := uint64(a.SampleIndex()-at.SampleIndex()) + uint64(step) - 1
				gap = min(gap, int(d/uint64(step)))
			}
			c.appendMissing(gap)
			i += gap
			continue
		}
		k := 1
		if c.size > 0 {
			k = min(n-i, c.remaining())
		}
		if err := c.appendSamples(k); err != nil {
			return err
		}
		if c.groupPath != "" {
			c.times = append(c.times, times[16*i:16*(i+k)]...)
		}
		i += k
	}
	return nil
}

// appendMissing fills k index samples the channel has no sample for. Only a channel
// that shares its index's timestamps holds anything there.
func (c *tdmsChannel) appendMissing(k int) {
	if c.groupPath != "" {
		return
	}
	c.count += k
	start := len(c.data)
	c.data = slices.Grow(c.data, k*len(c.nan))[:start+k*len(c.nan)]
	filled := copy(c.data[start:], c.nan)
	for filled < k*len(c.nan) {
		filled += copy(c.data[start+filled:], c.data[start:start+filled])
	}
}

// appendSamples appends the next k samples of the series the channel points at.
func (c *tdmsChannel) appendSamples(k int) error {
	c.count += k
	switch c.tdmsType {
	case tdmsString:
		for range k {
			sample, n := c.sample()
			if err := c.appendText(sample); err != nil {
				return err
			}
			c.advance(n)
		}
	case tdmsTimeStamp:
		data := c.pending[0].Data[c.offset:]
		for i := range k {
			ts := telem.TimeStamp(telem.ByteOrder.Uint64(data[8*i:]))
			c.data = appendTDMSTimeStamp(c.data, ts)
		}
		c.skip(k)
	default:
		c.data = append(c.data, c.pending[0].Data[c.offset:c.offset+k*c.size]...)
		c.skip(k)
	}
	return nil
}

// appendText appends the text form of sample to a string channel.
func (c *tdmsChannel) appendText(sample []byte) error {
	var err error
	if c.text, err = telem.AppendSampleText(c.text, c.dataType, sample); err != nil {
		return err
	}
	if len(c.text) > math.MaxUint32 {
		return errors.Newf(
			"text of a TDMS segment exceeds the %d bytes its offsets can address",
			uint32(math.MaxUint32),
		)
	}
	c.data = telem.ByteOrder.AppendUint32(c.data, uint32(len(c.text)))
	return nil
}

// tdmsEpochOffset is the number of seconds from the TDMS epoch, 1904-01-01 UTC, to the
// Unix epoch.
const tdmsEpochOffset = 2082844800

// appendTDMSTimeStamp appends ts as a TDMS timestamp: the fraction of the second in
// units of 2^-64 seconds, then the whole seconds since the TDMS epoch. The fraction
// rounds up, so a reader that rounds it down to nanoseconds recovers ts exactly.
func appendTDMSTimeStamp(dst []byte, ts telem.TimeStamp) []byte {
	secs, nanos := int64(ts)/1e9, int64(ts)%1e9
	if nanos < 0 {
		secs, nanos = secs-1, nanos+1e9
	}
	fraction, rem := bits.Div64(uint64(nanos), 0, 1e9)
	if rem != 0 {
		fraction++
	}
	dst = telem.ByteOrder.AppendUint64(dst, fraction)
	return telem.ByteOrder.AppendUint64(dst, uint64(secs+tdmsEpochOffset))
}

func appendTDMSString(dst []byte, v string) []byte {
	dst = telem.ByteOrder.AppendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}

func (f *tdmsFile) reset() {
	f.objects, f.meta, f.raw = 0, append(f.meta[:0], 0, 0, 0, 0), f.raw[:0]
}

// flush writes the lead-in, the metadata, and the raw data of a segment.
func (f *tdmsFile) flush(w io.Writer, toc uint32) error {
	telem.ByteOrder.PutUint32(f.meta, f.objects)
	size := len(f.meta)
	for _, b := range f.raw {
		size += len(b)
	}
	var leadIn [tdmsLeadInSize]byte
	copy(leadIn[:4], "TDSm")
	telem.ByteOrder.PutUint32(leadIn[4:], toc)
	telem.ByteOrder.PutUint32(leadIn[8:], tdmsVersion)
	telem.ByteOrder.PutUint64(leadIn[12:], uint64(size))
	telem.ByteOrder.PutUint64(leadIn[20:], uint64(len(f.meta)))
	if _, err := w.Write(leadIn[:]); err != nil {
		return err
	}
	if _, err := w.Write(f.meta); err != nil {
		return err
	}
	for _, b := range f.raw {
		if len(b) == 0 {
			continue
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return nil
}
