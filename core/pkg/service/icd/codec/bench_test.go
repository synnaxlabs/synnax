// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
)

// benchKind is one field kind of a benchmark: the variant of the field, the data type
// of its channel, the JSON text of a value, and the sample that the text decodes to or
// encodes from.
type benchKind struct {
	name          string
	variant       icd.JSONFieldVariant
	scale, offset float64
	min, max      *float64
	dataType      telem.DataType
	text          string
	sample        []byte
}

var onOff = []payload.IntegerEntry{{Label: "OFF", Value: 0}, {Label: "ON", Value: 1}}

var (
	benchFloat = telem.NewSeriesV(1234.5678).Data
	benchUUID  = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	benchStamp = telem.NewSeriesV(telem.TimeStamp(1767323045123456789)).Data
	benchBytes = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
)

// scaled returns the sample of v after scale and offset.
func scaled(v, scale, offset float64) []byte {
	return telem.NewSeriesV(v*scale + offset).Data
}

var decodeKinds = []benchKind{
	{
		name:     "float",
		variant:  icd.FloatJSONField{},
		dataType: telem.Float64T,
		text:     "1234.5678",
		sample:   benchFloat,
	},
	{
		name: "float_scaled", variant: icd.FloatJSONField{}, scale: 0.5, offset: 10,
		dataType: telem.Float64T, text: "1234.5678",
		sample: scaled(1234.5678, 0.5, 10),
	},
	{
		name: "float_string", variant: icd.FloatJSONField{}, dataType: telem.Float64T,
		text: `"1234.5678"`, sample: benchFloat,
	},
	{
		name: "integer", variant: icd.IntegerJSONField{}, dataType: telem.Int64T,
		text: "-123456789", sample: telem.NewSeriesV[int64](-123456789).Data,
	},
	{
		name: "integer_label", variant: icd.IntegerJSONField{EnumValues: onOff},
		dataType: telem.Int64T, text: `"ON"`, sample: telem.NewSeriesV[int64](1).Data,
	},
	{
		name:     "unsigned",
		variant:  icd.UnsignedJSONField{},
		dataType: telem.Uint32T,
		text:     "123456",
		sample:   telem.NewSeriesV[uint32](123456).Data,
	},
	{
		name:     "boolean",
		variant:  icd.BooleanJSONField{},
		dataType: telem.BooleanT,
		text:     "true",
		sample:   []byte{1},
	},
	{
		name:     "string",
		variant:  icd.StringJSONField{},
		dataType: telem.StringT,
		text:     `"pump running"`,
		sample:   telem.MarshalVariableSample([]byte("pump running")),
	},
	{
		name:     "timestamp_iso8601",
		variant:  icd.TimestampJSONField{TimeFormat: payload.TimeFormatISO8601},
		dataType: telem.TimestampT,
		text:     `"2026-01-02T03:04:05.123456789Z"`,
		sample:   benchStamp,
	},
	{
		name:     "timestamp_unix_ns",
		variant:  icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixNs},
		dataType: telem.TimestampT,
		text:     "1767323045123456789",
		sample:   benchStamp,
	},
	{
		name:     "json",
		variant:  icd.JSONJSONField{},
		dataType: telem.JSONT,
		text:     `{"a":1,"b":[true,null]}`,
		sample:   telem.MarshalVariableSample([]byte(`{"a":1,"b":[true,null]}`)),
	},
	{
		name: "uuid", variant: icd.UuidJSONField{}, dataType: telem.UUIDT,
		text: `"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`, sample: benchUUID[:],
	},
	{
		name:     "bytes",
		variant:  icd.BytesJSONField{},
		dataType: telem.BytesT,
		text:     `"AAECAwQFBgcICQoLDA0ODw=="`,
		sample:   telem.MarshalVariableSample(benchBytes),
	},
}

var encodeKinds = []benchKind{
	{
		name:     "float",
		variant:  icd.FloatJSONField{},
		dataType: telem.Float64T,
		text:     "1234.5678",
		sample:   benchFloat,
	},
	{
		name: "float_scaled", variant: icd.FloatJSONField{}, scale: 0.5, offset: 10,
		dataType: telem.Float64T, text: "2449.1356", sample: benchFloat,
	},
	{
		name:     "float_bounded",
		variant:  icd.FloatJSONField{},
		min:      new(-1e6),
		max:      new(1e6),
		dataType: telem.Float64T,
		text:     "1234.5678",
		sample:   benchFloat,
	},
	{
		name: "integer", variant: icd.IntegerJSONField{}, dataType: telem.Int64T,
		text: "-123456789", sample: telem.NewSeriesV[int64](-123456789).Data,
	},
	{
		name: "integer_from_float", variant: icd.IntegerJSONField{},
		dataType: telem.Float64T, text: "1235", sample: benchFloat,
	},
	{
		name: "integer_label", variant: icd.IntegerJSONField{EnumValues: onOff},
		dataType: telem.Int64T, text: `"ON"`, sample: telem.NewSeriesV[int64](1).Data,
	},
	{
		name: "unsigned", variant: icd.UnsignedJSONField{}, dataType: telem.Uint32T,
		text: "123456", sample: telem.NewSeriesV[uint32](123456).Data,
	},
	{
		name: "boolean", variant: icd.BooleanJSONField{}, dataType: telem.BooleanT,
		text: "true", sample: []byte{1},
	},
	{
		name: "string", variant: icd.StringJSONField{}, dataType: telem.StringT,
		text: `"pump running"`, sample: []byte("pump running"),
	},
	{
		name:     "timestamp_iso8601",
		variant:  icd.TimestampJSONField{TimeFormat: payload.TimeFormatISO8601},
		dataType: telem.TimestampT,
		text:     `"2026-01-02T03:04:05.123456789Z"`,
		sample:   benchStamp,
	},
	{
		name:     "timestamp_unix_ns",
		variant:  icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixNs},
		dataType: telem.TimestampT,
		text:     "1767323045123456789",
		sample:   benchStamp,
	},
	{
		name: "json", variant: icd.JSONJSONField{}, dataType: telem.JSONT,
		text: `{"a":1,"b":[true,null]}`, sample: []byte(`{"a":1,"b":[true,null]}`),
	},
	{
		name:     "uuid",
		variant:  icd.UuidJSONField{},
		dataType: telem.UUIDT,
		text:     `"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`,
		sample:   benchUUID[:],
	},
	{
		name: "bytes", variant: icd.BytesJSONField{}, dataType: telem.BytesT,
		text: `"AAECAwQFBgcICQoLDA0ODw=="`, sample: benchBytes,
	},
}

var benchFloatKind = benchKind{
	name: "float", variant: icd.FloatJSONField{}, dataType: telem.Float64T,
	text: "1234.5678", sample: benchFloat,
}

// benchFields is the number of fields of each kind benchmark.
const benchFields = 10

var benchSizes = []int{1, 10, 100, 1000}

// field returns the field of kind k at pointer.
func (k benchKind) field(name, pointer string) icd.JSONField {
	f := icd.JSONField{Variant: k.variant}
	scale := k.scale
	if scale == 0 {
		scale = 1
	}
	f.SetBase(icd.JSONBaseField{
		Key:     uuid.New(),
		Name:    name,
		Scale:   scale,
		Offset:  k.offset,
		Min:     k.min,
		Max:     k.max,
		Pointer: pointer,
	})
	return f
}

// benchMessage returns a message of n fields of kind k at /f0 to /f(n-1), each bound
// to a channel of the data type of k, and a document that holds a value for each.
func benchMessage(k benchKind, n int) (codec.JSONMessage, []byte) {
	m := codec.JSONMessage{Name: k.name}
	var doc strings.Builder
	doc.WriteByte('{')
	for i := range n {
		name := "f" + strconv.Itoa(i)
		f := k.field(name, "/"+name)
		m.Payload.Fields = append(m.Payload.Fields, f)
		m.Bindings = append(
			m.Bindings, codec.Binding{Field: f.Base().Key, DataType: k.dataType},
		)
		if i > 0 {
			doc.WriteByte(',')
		}
		fmt.Fprintf(&doc, "%q:%s", name, k.text)
	}
	doc.WriteByte('}')
	return m, []byte(doc.String())
}

func newBenchDecoder(b *testing.B, messages ...codec.JSONMessage) *codec.JSON {
	b.Helper()
	dec, err := codec.NewJSON(messages)
	if err != nil {
		b.Fatal(err)
	}
	return dec
}

// runDecode decodes doc with dec once per iteration. doc must decode as message want
// with no failed field, and with each sample of the message equal to sample.
func runDecode(
	b *testing.B,
	dec *codec.JSON,
	samples [][][]byte,
	doc []byte,
	want int,
	sample []byte,
) {
	b.Helper()
	decode := func() {
		for _, s := range samples {
			for i := range s {
				s[i] = s[i][:0]
			}
		}
		message, ok, failures, err := dec.Decode(doc, samples)
		if err != nil || !ok || failures != nil || message != want {
			b.Fatalf("decode: message %d, ok %t, failures %v, err %v",
				message, ok, failures, err)
		}
	}
	decode()
	for i, s := range samples[want] {
		if !bytes.Equal(s, sample) {
			b.Fatalf("decode: sample %d is %v, not %v", i, s, sample)
		}
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(doc)))
	for b.Loop() {
		decode()
	}
}

// nestedMessage returns a message of floats at /a/b/f0 to /a/b/f9, and a document that
// holds a value for each.
func nestedMessage() (codec.JSONMessage, []byte) {
	m := codec.JSONMessage{Name: "nested"}
	var doc strings.Builder
	doc.WriteString(`{"a":{"b":{`)
	for i := range benchFields {
		name := "f" + strconv.Itoa(i)
		f := benchFloatKind.field(name, "/a/b/"+name)
		m.Payload.Fields = append(m.Payload.Fields, f)
		m.Bindings = append(
			m.Bindings,
			codec.Binding{Field: f.Base().Key, DataType: telem.Float64T},
		)
		if i > 0 {
			doc.WriteByte(',')
		}
		fmt.Fprintf(&doc, "%q:1234.5678", name)
	}
	doc.WriteString(`}}}`)
	return m, []byte(doc.String())
}

func BenchmarkJSONDecode(b *testing.B) {
	b.Run("kind", func(b *testing.B) {
		for _, k := range decodeKinds {
			b.Run(k.name, func(b *testing.B) {
				m, doc := benchMessage(k, benchFields)
				dec := newBenchDecoder(b, m)
				runDecode(b, dec, newSamples([]codec.JSONMessage{m}), doc, 0, k.sample)
			})
		}
	})
	b.Run("fields", func(b *testing.B) {
		for _, n := range benchSizes {
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				m, doc := benchMessage(benchFloatKind, n)
				dec := newBenchDecoder(b, m)
				runDecode(
					b,
					dec,
					newSamples([]codec.JSONMessage{m}),
					doc,
					0,
					benchFloat,
				)
			})
		}
	})
	b.Run("unbound", func(b *testing.B) {
		// The document holds 100 members, and the message binds the first 10.
		m, _ := benchMessage(benchFloatKind, benchFields)
		_, doc := benchMessage(benchFloatKind, 100)
		dec := newBenchDecoder(b, m)
		runDecode(b, dec, newSamples([]codec.JSONMessage{m}), doc, 0, benchFloat)
	})
	b.Run("nested", func(b *testing.B) {
		m, doc := nestedMessage()
		dec := newBenchDecoder(b, m)
		runDecode(b, dec, newSamples([]codec.JSONMessage{m}), doc, 0, benchFloat)
	})
	b.Run("routed", func(b *testing.B) {
		// Four messages, each with a value rule on /type, and a document of the last.
		messages := make([]codec.JSONMessage, 4)
		for i := range messages {
			messages[i], _ = benchMessage(benchFloatKind, benchFields)
			messages[i].Name = "m" + strconv.Itoa(i)
			messages[i].Payload.Match = &icd.JSONMatch{Variant: icd.StringJSONMatch{
				JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/type"},
				Value:         messages[i].Name,
			}}
		}
		_, doc := benchMessage(benchFloatKind, benchFields)
		doc = append([]byte(`{"type":"m3",`), doc[1:]...)
		dec := newBenchDecoder(b, messages...)
		runDecode(b, dec, newSamples(messages), doc, 3, benchFloat)
	})
}

// benchSamples returns one sample of kind k for each binding of m.
func benchSamples(k benchKind, m codec.JSONMessage) [][]byte {
	samples := make([][]byte, len(m.Bindings))
	for i := range samples {
		samples[i] = k.sample
	}
	return samples
}

// runEncode encodes samples as m once per iteration. The document must equal want,
// unless want is nil.
func runEncode(b *testing.B, m codec.JSONMessage, samples [][]byte, want []byte) {
	b.Helper()
	enc, err := codec.NewJSONEncoder(m)
	if err != nil {
		b.Fatal(err)
	}
	doc, err := enc.Encode(nil, samples)
	if err != nil {
		b.Fatal(err)
	}
	if want != nil && !bytes.Equal(doc, want) {
		b.Fatalf("encode: document is %s, not %s", doc, want)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(doc)))
	for b.Loop() {
		if doc, err = enc.Encode(doc[:0], samples); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONEncode(b *testing.B) {
	b.Run("kind", func(b *testing.B) {
		for _, k := range encodeKinds {
			b.Run(k.name, func(b *testing.B) {
				m, want := benchMessage(k, benchFields)
				runEncode(b, m, benchSamples(k, m), want)
			})
		}
	})
	b.Run("fields", func(b *testing.B) {
		for _, n := range benchSizes {
			b.Run(strconv.Itoa(n), func(b *testing.B) {
				m, want := benchMessage(benchFloatKind, n)
				runEncode(b, m, benchSamples(benchFloatKind, m), want)
			})
		}
	})
	b.Run("defaults", func(b *testing.B) {
		// Ten bound floats, and a constant and a generated default of each text kind.
		m, _ := benchMessage(benchFloatKind, benchFields)
		samples := benchSamples(benchFloatKind, m)
		for i, v := range []icd.JSONFieldVariant{
			icd.StringJSONField{Default: new("synnax")},
			icd.TimestampJSONField{
				TimeFormat: payload.TimeFormatUnixNs,
				Default:    &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
			},
			icd.UuidJSONField{
				Default: &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			},
			icd.FloatJSONField{Default: new(payload.Float(math.Pi))},
		} {
			name := "d" + strconv.Itoa(i)
			m.Payload.Fields = append(
				m.Payload.Fields, benchKind{variant: v}.field(name, "/"+name),
			)
		}
		// The generated defaults change in each document.
		runEncode(b, m, samples, nil)
	})
}
