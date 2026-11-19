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

// NewTDMSEncoder returns an encoder that writes a read as an NI TDMS file. Each index
// channel heads a group named after it, followed by the float channels it timestamps. A
// float channel holds NaN, which NI tools read as no value, where it has no sample.
// TDMS has no missing value for other types, so each other channel gets its own group:
// the timestamps of its samples, then the samples. A channel whose name another channel
// of the read shares takes its name and key, such as "temp (1048577)". A channel gets a
// column per data type of its samples. The column of its declared type keeps the name;
// each other type takes a group of its own, such as "temp (int32)". When the first
// sample of a channel has another type, every column of the channel takes a group of
// its own. Each group sets wf_xcolumns to mark its first channel as its time channel.
// UUID, bytes, and JSON samples become TDMS strings in their text form. A latest or
// earliest read holds only the newest or oldest rows of each index. A read of a channel
// whose index it does not include, or a read under a limit reduction, fails before the
// first byte. A read that fails after the first byte truncates the body. ctx bounds the
// iterators the encoder opens.
func NewTDMSEncoder(ctx context.Context) xhttp.Encoder { return tdmsEncoder{ctx: ctx} }

type tdmsEncoder struct{ ctx context.Context }

// ContentType implements http.Encoder.
func (tdmsEncoder) ContentType() string { return "application/x-tdms" }

// Encode implements encoding.Encoder. It holds the whole read in memory, so prefer
// EncodeStream.
func (e tdmsEncoder) Encode(value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeStream implements encoding.Encoder. value must be a ReadResponse.
func (e tdmsEncoder) EncodeStream(w io.Writer, value any) error {
	res, err := readResponse(value)
	if err != nil {
		return err
	}
	f, err := newTDMSFile(res)
	if err != nil {
		return err
	}
	columns := f.columns()
	if err = openColumns(e.ctx, res, columns); err != nil {
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

// tdmsChannel is a data channel of the file, with a column of the file for each data
// type of its samples.
type tdmsChannel struct {
	column
	// fileName is the name the channel takes in the file.
	fileName  string
	indexName string
	declared  telem.DataType
	// tdmsType is the TDMS type of the declared type.
	tdmsType tdmsDataType
	// shared is true when a column of the declared type shares the timestamps of the
	// index.
	shared bool
	// outputs holds a column for each data type, in the order of their first sample.
	// Only the first can share the timestamps of the index.
	outputs []tdmsOutput
}

// tdmsOutput is the column of the file that holds the samples of one data type of a
// channel.
type tdmsOutput struct {
	dataType telem.DataType
	tdmsType tdmsDataType
	// groupPath is the object path of the column's own group, or empty when the column
	// shares the timestamps of its index.
	groupPath string
	path      string
	// timePath is the object path of the column's own time channel.
	timePath string
	// nan is the sample a shared column holds where the channel has none.
	nan []byte
	// groupDeclared is true once a segment declares the column's group.
	groupDeclared bool
	// count is the number of samples the segment holds for the column.
	count int
	// data holds the segment's samples, or the end offset of each string.
	data []byte
	// text holds the segment's strings.
	text []byte
	// times holds the TDMS timestamps of the segment's samples, for a column with its
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
			t, err := newTDMSDataType(ch.DataType)
			if err != nil {
				return nil, err
			}
			c := tdmsChannel{
				column:    newColumn(ch),
				fileName:  names[ch.Key()],
				indexName: indexName,
				declared:  ch.DataType,
				tdmsType:  t,
				shared:    tdmsNaN(t) != nil,
			}
			if c.shared {
				g.channels = append(g.channels, c)
			} else {
				own = append(own, c)
			}
		}
		g.channels = append(g.channels, own...)
	}
	return f, nil
}

// start gives the channel the column of its declared type when its first sample has
// that type, or when it has no samples. Each other column joins the file at its first
// sample, in a group of its own, so the file holds a column of the declared type only
// when the read does.
func (c *tdmsChannel) start() {
	if _, n := c.sample(); n > 0 && c.dataType != c.declared {
		return
	}
	o := tdmsOutput{dataType: c.declared, tdmsType: c.tdmsType, groupDeclared: true}
	if c.shared {
		o.path, o.nan = tdmsPath(c.indexName, c.fileName), tdmsNaN(c.tdmsType)
	} else {
		o.own(c.fileName, c.indexName)
	}
	c.outputs = append(c.outputs, o)
}

// output returns the column for the samples the channel points at, adding one in a
// group of its own when the channel has none of their type. A column of a type other
// than the declared one takes the name "<name> (<type>)".
func (c *tdmsChannel) output() (*tdmsOutput, error) {
	for i := range c.outputs {
		if c.outputs[i].dataType == c.dataType {
			return &c.outputs[i], nil
		}
	}
	t, err := newTDMSDataType(c.dataType)
	if err != nil {
		return nil, err
	}
	name := c.fileName
	if c.dataType != c.declared {
		name = typedName(name, c.dataType)
	}
	o := tdmsOutput{dataType: c.dataType, tdmsType: t}
	o.own(name, c.indexName)
	c.outputs = append(c.outputs, o)
	return &c.outputs[len(c.outputs)-1], nil
}

// own puts the column in a group of its own, named name, with the time channel
// indexName.
func (o *tdmsOutput) own(name, indexName string) {
	o.groupPath, o.path = tdmsPath(name), tdmsPath(name, name)
	o.timePath = tdmsPath(name, indexName)
}

// tdmsNames returns the name each channel of groups takes in the file: its own name, or
// its name and key when another channel shares the name. It returns a validation error
// when two channels still take the same name, or when a channel takes the name of a
// typed column of another, so a read fails before the first byte.
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
	for _, group := range groups {
		for _, ch := range group[1:] {
			for code := telem.DataTypeCodeUint8; code <= telem.DataTypeCodeBoolean; code++ {
				if code.DataType() == ch.DataType {
					continue
				}
				name := typedName(names[ch.Key()], code.DataType())
				if other, ok := taken[name]; ok {
					return nil, errors.Wrapf(
						validate.ErrValidation,
						"channels %s and %s both take the name %s in a TDMS file",
						other,
						ch.Key(),
						name,
					)
				}
			}
		}
	}
	return names, nil
}

// typedName returns the name of the column of a channel's samples of type dt.
func typedName(name string, dt telem.DataType) string {
	return fmt.Sprintf("%s (%s)", name, dt)
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
	case telem.UnknownT:
	}
	return 0, errors.Newf("data type %s has no TDMS form", dt)
}

// tdmsNaN returns the NaN sample of t, or nil when t is not a float type.
func tdmsNaN(t tdmsDataType) []byte {
	switch t {
	case tdmsF32:
		return telem.ByteOrder.AppendUint32(nil, math.Float32bits(float32(math.NaN())))
	case tdmsF64:
		return telem.ByteOrder.AppendUint64(nil, math.Float64bits(math.NaN()))
	case tdmsI8, tdmsI16, tdmsI32, tdmsI64, tdmsU8, tdmsU16, tdmsU32, tdmsU64,
		tdmsString, tdmsBoolean, tdmsTimeStamp:
		return nil
	default:
		panic(fmt.Sprintf("unknown TDMS data type %#x", uint32(t)))
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
	for i := range f.groups {
		for j := range f.groups[i].channels {
			f.groups[i].channels[j].start()
		}
	}
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
			for _, o := range g.channels[j].outputs {
				if o.groupPath != "" {
					f.appendDeclaration(o.groupPath, true)
				}
			}
		}
	}
	for i := range f.groups {
		g := &f.groups[i]
		f.appendDeclaration(g.indexPath, false)
		for j := range g.channels {
			for _, o := range g.channels[j].outputs {
				if o.groupPath != "" {
					f.appendDeclaration(o.timePath, false)
				}
				f.appendDeclaration(o.path, false)
			}
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
// at their alignments. It declares the group of each column that has its first samples
// in the segment.
func (f *tdmsFile) writeSegment(w io.Writer, g *tdmsGroup, n int) error {
	f.reset()
	f.appendData(g.indexPath, tdmsTimeStamp, n, g.times, nil)
	for i := range g.channels {
		for j := range g.channels[i].outputs {
			o := &g.channels[i].outputs[j]
			if o.count == 0 {
				continue
			}
			if !o.groupDeclared {
				f.appendDeclaration(o.groupPath, true)
				o.groupDeclared = true
			}
			if o.groupPath != "" {
				f.appendData(o.timePath, tdmsTimeStamp, o.count, o.times, nil)
			}
			f.appendData(o.path, o.tdmsType, o.count, o.data, o.text)
		}
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
		for j := range g.channels[i].outputs {
			o := &g.channels[i].outputs[j]
			o.count, o.data, o.text, o.times = 0, o.data[:0], o.text[:0], o.times[:0]
		}
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
			err := g.channels[i].read(&g.index, start, step, k, g.times[from:])
			if err != nil {
				return 0, err
			}
		}
		n += k
	}
	return n, nil
}

// read appends the channel's samples at the n alignments of index from start, step
// apart and in one domain. times holds the TDMS timestamps of those alignments. It
// fails when the channel's samples are on another clock.
func (c *tdmsChannel) read(
	index *column,
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
		if err := c.clockError(index); err != nil {
			return err
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
		o, err := c.output()
		if err != nil {
			return err
		}
		if err = c.appendSamples(o, k); err != nil {
			return err
		}
		if o.groupPath != "" {
			o.times = append(o.times, times[16*i:16*(i+k)]...)
			c.appendMissing(k)
		}
		i += k
	}
	return nil
}

// appendMissing fills k index samples that the channel's shared column has no sample
// for. A channel with no shared column holds nothing there.
func (c *tdmsChannel) appendMissing(k int) {
	if len(c.outputs) == 0 || c.outputs[0].groupPath != "" {
		return
	}
	o := &c.outputs[0]
	o.count += k
	start := len(o.data)
	o.data = slices.Grow(o.data, k*len(o.nan))[:start+k*len(o.nan)]
	filled := copy(o.data[start:], o.nan)
	for filled < k*len(o.nan) {
		filled += copy(o.data[start+filled:], o.data[start:start+filled])
	}
}

// appendSamples appends the next k samples of the series the channel points at to o.
func (c *tdmsChannel) appendSamples(o *tdmsOutput, k int) error {
	o.count += k
	switch o.tdmsType {
	case tdmsString:
		for range k {
			sample, n := c.sample()
			if err := c.appendText(o, sample); err != nil {
				return err
			}
			c.advance(n)
		}
	case tdmsTimeStamp:
		data := c.pending[0].Data[c.offset:]
		for i := range k {
			ts := telem.TimeStamp(telem.ByteOrder.Uint64(data[8*i:]))
			o.data = appendTDMSTimeStamp(o.data, ts)
		}
		c.skip(k)
	case tdmsI8, tdmsI16, tdmsI32, tdmsI64, tdmsU8, tdmsU16, tdmsU32, tdmsU64, tdmsF32,
		tdmsF64, tdmsBoolean:
		o.data = append(o.data, c.pending[0].Data[c.offset:c.offset+k*c.size]...)
		c.skip(k)
	default:
		return errors.Newf("unknown TDMS data type %#x", uint32(o.tdmsType))
	}
	return nil
}

// appendText appends the text form of sample to the string column o.
func (c *tdmsChannel) appendText(o *tdmsOutput, sample []byte) error {
	var err error
	if o.text, err = telem.AppendSampleText(o.text, c.dataType, sample); err != nil {
		return err
	}
	if len(o.text) > math.MaxUint32 {
		return errors.Newf(
			"text of a TDMS segment exceeds the %d bytes its offsets can address",
			uint32(math.MaxUint32),
		)
	}
	o.data = telem.ByteOrder.AppendUint32(o.data, uint32(len(o.text)))
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
