// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"math"
	"slices"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

const (
	// maxEnumLabels is the most distinct strings of a field that infers as an enum.
	maxEnumLabels = 16
	// maxInferredFields is the most fields that one shape holds.
	maxInferredFields = 1024
)

// Document is one sampled JSON document.
type Document struct {
	// Source is where the document came from, such as a record key or a topic. The
	// sources whose documents hold the same fields share one shape.
	Source string
	// Data is the JSON text of the document.
	Data []byte
	// Time is when the document was written. Zero selects the time of inference.
	Time telem.TimeStamp
}

// InferredField is a scalar field that inference found in a set of documents.
type InferredField struct {
	// Pointer locates the field in a document.
	Pointer Pointer `json:"pointer" msgpack:"pointer"`
	// DataType is the data type of a channel that takes the values of the field.
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// Type is the JSON type of most values of the field.
	Type Type `json:"type" msgpack:"type"`
	// TimeFormat is the format of the values of a timestamp field. Nil for any other
	// field.
	TimeFormat *TimeFormat `json:"time_format,omitempty" msgpack:"time_format,omitempty"`
	// Enum holds the distinct values, sorted, of a string field that repeats a few.
	Enum []string `json:"enum" msgpack:"enum"`
	// Count is the number of documents that hold a value of the field.
	Count int `json:"count" msgpack:"count"`
	// Mismatched is the number of values that a channel of DataType does not take.
	Mismatched int `json:"mismatched" msgpack:"mismatched"`
}

// Shape is the layout that the documents of one or more sources share.
type Shape struct {
	// Sources are the sources of the documents, in the order first seen.
	Sources []string `json:"sources" msgpack:"sources"`
	// Fields are the scalar fields of the documents, in the order first seen.
	Fields []InferredField `json:"fields" msgpack:"fields"`
	// Index is the position in Fields of the timestamp that indexes the documents, or
	// -1 when no field can.
	Index int `json:"index" msgpack:"index"`
	// Count is the number of documents of the shape.
	Count int `json:"count" msgpack:"count"`
}

// Infer returns the shapes of docs, which are in the order that each source wrote them.
//   - A field is a JSON pointer to a scalar. Arrays and nulls give no field.
//   - Types widen across documents: int64, then uint64, then float64. A boolean field
//     gives uint8. The JSON type of most values wins, and the rest are mismatched.
//   - A string field that repeats a few values gives an enum.
//   - A field whose name suggests a time and whose values all fit one format near the
//     document time gives a timestamp. Unix units come from the magnitude.
//   - The index is the timestamp nearest the document time that rises in each source.
//
// A document that is not JSON gives no fields, and a source with no JSON document gives
// no shape.
func Infer(docs []Document) []Shape {
	now := telem.Now()
	var (
		sources []*inferSource
		byName  = make(map[string]*inferSource)
	)
	for _, doc := range docs {
		leaves, err := parseLeaves(doc.Data)
		if err != nil {
			continue
		}
		src, ok := byName[doc.Source]
		if !ok {
			src = &inferSource{name: doc.Source, fields: make(set.Set[string])}
			byName[doc.Source] = src
			sources = append(sources, src)
		}
		ref := doc.Time
		if ref <= 0 {
			ref = now
		}
		src.docs = append(src.docs, parsedDocument{leaves: leaves, time: ref})
		for _, l := range leaves {
			if !src.fields.Contains(l.key) {
				src.fields.Add(l.key)
				src.order = append(src.order, l.key)
			}
		}
	}
	var (
		groups      []*inferGroup
		bySignature = make(map[string]*inferGroup)
	)
	for _, src := range sources {
		sig := slices.Clone(src.order)
		slices.Sort(sig)
		key := strings.Join(sig, "\x00")
		g, ok := bySignature[key]
		if !ok {
			g = &inferGroup{fields: make(map[string]*fieldStats)}
			bySignature[key] = g
			groups = append(groups, g)
		}
		g.add(src)
	}
	shapes := make([]Shape, len(groups))
	for i, g := range groups {
		shapes[i] = g.shape()
	}
	return shapes
}

type inferSource struct {
	name   string
	docs   []parsedDocument
	fields set.Set[string]
	order  []string
}

type parsedDocument struct {
	leaves []leaf
	time   telem.TimeStamp
}

// leaf is one scalar of a document.
type leaf struct {
	pointer Pointer
	// key is the text form of pointer.
	key  string
	kind jsontext.Kind
	raw  []byte
}

// parseLeaves returns the scalars of data in document order. It returns an error when
// data is not one JSON value.
func parseLeaves(data []byte) ([]leaf, error) {
	var leaves []leaf
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	if err := walkLeaves(dec, nil, &leaves); err != nil {
		return nil, err
	}
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the JSON value")
	}
	return leaves, nil
}

func walkLeaves(dec *jsontext.Decoder, pointer Pointer, leaves *[]leaf) error {
	switch dec.PeekKind() {
	case '{':
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		for dec.PeekKind() != '}' {
			name, err := dec.ReadToken()
			if err != nil {
				return err
			}
			child := append(slices.Clip(pointer), name.String())
			if err = walkLeaves(dec, child, leaves); err != nil {
				return err
			}
		}
		_, err := dec.ReadToken()
		return err
	case '"', '0', 't', 'f':
		value, err := dec.ReadValue()
		if err != nil {
			return err
		}
		*leaves = append(*leaves, leaf{
			pointer: pointer,
			key:     pointer.String(),
			kind:    value.Kind(),
			raw:     bytes.Clone(value),
		})
		return nil
	default:
		return dec.SkipValue()
	}
}

type inferGroup struct {
	sources []string
	fields  map[string]*fieldStats
	order   []*fieldStats
	count   int
}

func (g *inferGroup) add(src *inferSource) {
	g.sources = append(g.sources, src.name)
	for _, doc := range src.docs {
		g.count++
		for _, l := range doc.leaves {
			s, ok := g.fields[l.key]
			if !ok {
				if len(g.order) == maxInferredFields {
					continue
				}
				s = newFieldStats(l.pointer)
				g.fields[l.key] = s
				g.order = append(g.order, s)
			}
			s.add(src.name, l, doc.time)
		}
	}
}

func (g *inferGroup) shape() Shape {
	shape := Shape{
		Sources: g.sources,
		Fields:  make([]InferredField, len(g.order)),
		Index:   -1,
		Count:   g.count,
	}
	nearest := math.Inf(1)
	for i, s := range g.order {
		shape.Fields[i] = s.field()
		// An index must stamp every document.
		if !s.timestamp() || !s.time.rising || s.count < g.count {
			continue
		}
		if d := s.time.distance / float64(s.count); d < nearest {
			shape.Index, nearest = i, d
		}
	}
	return shape
}

// fieldStats accumulates the values of one field.
type fieldStats struct {
	pointer                    Pointer
	count                      int
	numbers, strings, booleans int
	// unreadable is the number of numbers that do not fit a float64.
	unreadable int
	// floating is true when a number has a fraction or does not fit a 64-bit integer.
	floating, unsigned, negative bool
	labels                       set.Set[string]
	time                         timeStats
}

// timeStats tracks whether the values of a field are timestamps.
type timeStats struct {
	// candidate is true while every value fits format near its document time.
	candidate bool
	format    TimeFormat
	// formatted is true once a value sets format.
	formatted bool
	// rising is true while no value is older than the value before it in its source.
	rising bool
	last   map[string]telem.TimeStamp
	// distance is the sum of the distances of the values from their document times.
	distance float64
}

func newFieldStats(pointer Pointer) *fieldStats {
	s := &fieldStats{pointer: pointer, labels: make(set.Set[string])}
	if len(pointer) > 0 && timeName(pointer[len(pointer)-1]) {
		s.time = timeStats{
			candidate: true,
			rising:    true,
			last:      make(map[string]telem.TimeStamp),
		}
	}
	return s
}

func (s *fieldStats) add(source string, l leaf, ref telem.TimeStamp) {
	s.count++
	switch l.kind {
	case '0':
		s.numbers++
		switch n, err := parseNumber(l.raw); {
		case err != nil:
			s.unreadable++
		case n.kind == floatNumber:
			s.floating = true
		case n.kind == uintNumber:
			s.unsigned = true
		case n.i < 0:
			s.negative = true
		}
	case '"':
		s.strings++
		if len(s.labels) <= maxEnumLabels {
			if text, err := jsontext.AppendUnquote(nil, l.raw); err == nil {
				s.labels.Add(string(text))
			}
		}
	default:
		s.booleans++
	}
	s.addTime(source, l, ref)
}

func (s *fieldStats) addTime(source string, l leaf, ref telem.TimeStamp) {
	t := &s.time
	if !t.candidate {
		return
	}
	ts, format, ok := timeOf(l, ref)
	if !ok || (t.formatted && format != t.format) {
		t.candidate = false
		return
	}
	t.format, t.formatted = format, true
	if last, seen := t.last[source]; seen && ts < last {
		t.rising = false
	}
	t.last[source] = ts
	t.distance += math.Abs(float64(ts) - float64(ref))
}

func (s *fieldStats) timestamp() bool { return s.time.candidate && s.time.formatted }

func (s *fieldStats) field() InferredField {
	f := InferredField{Pointer: s.pointer, Count: s.count}
	numeric := s.numbers + s.booleans
	switch {
	case s.numbers > 0 && numeric >= s.strings:
		f.Type, f.DataType, f.Mismatched = Number, s.numberType(), s.strings+s.unreadable
	case s.numbers == 0 && s.booleans > s.strings:
		f.Type, f.DataType, f.Mismatched = Boolean, telem.Uint8T, s.strings
	default:
		f.Type, f.DataType, f.Mismatched = String, telem.StringT, numeric
	}
	if s.timestamp() {
		f.DataType, f.TimeFormat, f.Mismatched = telem.TimestampT, new(s.time.format), 0
		return f
	}
	if f.Type == String && len(s.labels) <= maxEnumLabels && len(s.labels) < s.strings {
		f.Enum = make([]string, 0, len(s.labels))
		for label := range s.labels {
			f.Enum = append(f.Enum, label)
		}
		slices.Sort(f.Enum)
	}
	return f
}

func (s *fieldStats) numberType() telem.DataType {
	switch {
	case s.floating, s.unsigned && s.negative:
		return telem.Float64T
	case s.unsigned:
		return telem.Uint64T
	default:
		return telem.Int64T
	}
}

// timeName reports whether a field name suggests that the field holds a time.
func timeName(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "t", "ts", "epoch":
		return true
	}
	return strings.Contains(lower, "time") ||
		strings.Contains(lower, "date") ||
		strings.Contains(lower, "stamp") ||
		strings.HasSuffix(lower, "_at") ||
		strings.HasSuffix(lower, "_ts") ||
		strings.HasSuffix(name, "At") ||
		strings.HasSuffix(name, "Ts")
}

// unixFormats are the numeric time formats, in rising order of resolution.
var unixFormats = [...]TimeFormat{
	UnixSecond,
	UnixMillisecond,
	UnixMicrosecond,
	UnixNanosecond,
}

// timeOf parses l as a timestamp. A string holds ISO 8601 or a Unix number, and a
// number is Unix in the unit that puts it nearest ref. It reports false when the value
// is not a timestamp within a factor of two of ref.
func timeOf(l leaf, ref telem.TimeStamp) (telem.TimeStamp, TimeFormat, bool) {
	text := l.raw
	switch l.kind {
	case '"':
		unquoted, err := jsontext.AppendUnquote(nil, l.raw)
		if err != nil {
			return 0, 0, false
		}
		if ts, err := parseISO8601(bytes.Clone(unquoted)); err == nil {
			return ts, ISO8601, nearTime(ts, ref)
		}
		text = unquoted
	case '0':
	default:
		return 0, 0, false
	}
	n, err := parseNumber(text)
	if err != nil {
		return 0, 0, false
	}
	format, nearest := UnixSecond, math.Inf(1)
	for _, f := range unixFormats {
		d := math.Abs(n.float()*math.Pow10(f.nanosExponent()) - float64(ref))
		if d < nearest {
			format, nearest = f, d
		}
	}
	ts, err := parseUnix(text, format)
	if err != nil {
		return 0, 0, false
	}
	return ts, format, nearTime(ts, ref)
}

// nearTime reports whether ts is within a factor of two of ref, both measured from the
// Unix epoch.
func nearTime(ts, ref telem.TimeStamp) bool {
	t, r := float64(ts), float64(ref)
	return t >= r/2 && t <= r*2
}
