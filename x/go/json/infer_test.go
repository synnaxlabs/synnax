// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"encoding/json/v2"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// t0 is the write time of the first sampled document.
const t0 = telem.TimeStamp(1767323045123456789)

// t0 in Unix seconds, milliseconds, and microseconds.
const (
	t0Seconds      = int64(t0) / 1e9
	t0Milliseconds = int64(t0) / 1e6
	t0Microseconds = int64(t0) / 1e3
)

// documents returns n documents of source, written a second apart from t0, with the
// text that data returns for each index.
func documents(source string, n int, data func(i int) string) []xjson.Document {
	docs := make([]xjson.Document, n)
	for i := range docs {
		docs[i] = xjson.Document{
			Source: source,
			Data:   []byte(data(i)),
			Time:   t0 + telem.TimeStamp(i)*telem.SecondTS,
		}
	}
	return docs
}

// fieldAt returns the field of shape at pointer.
func fieldAt(shape xjson.Shape, pointer string) xjson.InferredField {
	GinkgoHelper()
	for _, f := range shape.Fields {
		if f.Pointer.String() == pointer {
			return f
		}
	}
	Fail(fmt.Sprintf("shape has no field %q", pointer))
	return xjson.InferredField{}
}

// indexPointer returns the pointer of the index of shape, or "none".
func indexPointer(shape xjson.Shape) string {
	if shape.Index < 0 {
		return "none"
	}
	return shape.Fields[shape.Index].Pointer.String()
}

var _ = Describe("Infer", func() {
	It("Should find an ISO 8601 index and an enum", func() {
		states := []string{"open", "closed", "open", "open"}
		shapes := xjson.Infer(documents("valves", 4, func(i int) string {
			return fmt.Sprintf(
				`{"time":"2026-01-02T03:04:%02dZ","state":%q,"position":%d.5}`,
				i, states[i], i,
			)
		}))
		Expect(shapes).To(HaveLen(1))
		Expect(shapes[0].Sources).To(Equal([]string{"valves"}))
		Expect(shapes[0].Count).To(Equal(4))
		Expect(indexPointer(shapes[0])).To(Equal("/time"))
		Expect(fieldAt(shapes[0], "/time")).To(Equal(xjson.InferredField{
			Pointer:    xjson.Pointer{"time"},
			DataType:   telem.TimestampT,
			Type:       xjson.String,
			TimeFormat: new(xjson.ISO8601),
			Count:      4,
		}))
		state := fieldAt(shapes[0], "/state")
		Expect(state.DataType).To(Equal(telem.StringT))
		Expect(state.Enum).To(Equal([]string{"closed", "open"}))
		position := fieldAt(shapes[0], "/position")
		Expect(position.DataType).To(Equal(telem.Float64T))
		Expect(position.TimeFormat).To(BeNil())
	})

	It("Should give one shape to the sources that hold the same fields", func() {
		pressure := func(i int) string {
			return fmt.Sprintf(`{"ts":%d,"pressure":%d}`, t0Milliseconds+int64(i), i)
		}
		docs := documents("tank-1", 3, pressure)
		docs = append(docs, documents("pump-1", 3, func(i int) string {
			return fmt.Sprintf(
				`{"ts":%d,"rpm":%d.25,"running":true}`,
				t0Milliseconds+int64(i),
				i,
			)
		})...)
		docs = append(docs, documents("tank-2", 3, pressure)...)
		shapes := xjson.Infer(docs)
		Expect(shapes).To(HaveLen(2))
		Expect(shapes[0].Sources).To(Equal([]string{"tank-1", "tank-2"}))
		Expect(shapes[0].Count).To(Equal(6))
		Expect(fieldAt(shapes[0], "/ts").TimeFormat).
			To(HaveValue(Equal(xjson.UnixMillisecond)))
		Expect(fieldAt(shapes[0], "/pressure").DataType).To(Equal(telem.Int64T))
		Expect(shapes[1].Sources).To(Equal([]string{"pump-1"}))
		Expect(fieldAt(shapes[1], "/running").DataType).To(Equal(telem.Uint8T))
	})

	It("Should give one shape to many sources of one value and one time", func() {
		var docs []xjson.Document
		for ch := range 5 {
			docs = append(docs, documents(
				fmt.Sprintf("sensor-%d", ch),
				2,
				func(i int) string {
					return fmt.Sprintf(
						`{"value":%d.5,"time":%d.25}`, i, t0Seconds+int64(i),
					)
				},
			)...)
		}
		shapes := xjson.Infer(docs)
		Expect(shapes).To(HaveLen(1))
		Expect(shapes[0].Sources).To(HaveLen(5))
		Expect(shapes[0].Fields).To(HaveLen(2))
		Expect(indexPointer(shapes[0])).To(Equal("/time"))
		Expect(fieldAt(shapes[0], "/time").TimeFormat).
			To(HaveValue(Equal(xjson.UnixSecond)))
	})

	It("Should keep every digit of a nanosecond index and read a boolean", func() {
		shapes := xjson.Infer(documents("valve", 3, func(i int) string {
			return fmt.Sprintf(`{"timestamp":%d,"open":%t}`, int64(t0)+int64(i), i == 1)
		}))
		Expect(indexPointer(shapes[0])).To(Equal("/timestamp"))
		Expect(fieldAt(shapes[0], "/timestamp").TimeFormat).
			To(HaveValue(Equal(xjson.UnixNanosecond)))
		Expect(fieldAt(shapes[0], "/open")).To(Equal(xjson.InferredField{
			Pointer:  xjson.Pointer{"open"},
			DataType: telem.Uint8T,
			Type:     xjson.Boolean,
			Count:    3,
		}))
	})

	It("Should count the values that are not of the type of most values", func() {
		readings := []string{"1", "2", `"N/A"`, "4"}
		shapes := xjson.Infer(documents("meter", 4, func(i int) string {
			return fmt.Sprintf(
				`{"t":%d,"reading":%s}`, t0Microseconds+int64(i), readings[i],
			)
		}))
		reading := fieldAt(shapes[0], "/reading")
		Expect(reading.DataType).To(Equal(telem.Int64T))
		Expect(reading.Type).To(Equal(xjson.Number))
		Expect(reading.Mismatched).To(Equal(1))
		Expect(fieldAt(shapes[0], "/t").TimeFormat).
			To(HaveValue(Equal(xjson.UnixMicrosecond)))
	})

	It("Should count the numbers that do not fit a float64", func() {
		values := []string{"1.5", "1e400", "2.5"}
		shapes := xjson.Infer(documents("meter", 3, func(i int) string {
			return fmt.Sprintf(`{"v":%s}`, values[i])
		}))
		v := fieldAt(shapes[0], "/v")
		Expect(v.DataType).To(Equal(telem.Float64T))
		Expect(v.Mismatched).To(Equal(1))
	})

	DescribeTable("Should widen the type of a field across documents",
		func(values []string, expected telem.DataType) {
			shapes := xjson.Infer(documents("a", len(values), func(i int) string {
				return `{"v":` + values[i] + `}`
			}))
			Expect(fieldAt(shapes[0], "/v").DataType).To(Equal(expected))
		},
		Entry("integers", []string{"1", "-2"}, telem.Int64T),
		Entry("an integer and a fraction", []string{"1", "2.5"}, telem.Float64T),
		Entry("an integer beyond int64", []string{"1", "18446744073709551615"},
			telem.Uint64T),
		Entry("a negative and an integer beyond int64",
			[]string{"-1", "18446744073709551615"}, telem.Float64T),
		Entry("booleans and numbers", []string{"true", "3"}, telem.Int64T),
		Entry("strings with no repeat", []string{`"a"`, `"b"`}, telem.StringT),
	)

	It("Should not give an enum to a string field that repeats no value", func() {
		shapes := xjson.Infer(documents("a", 3, func(i int) string {
			return fmt.Sprintf(`{"name":"pump-%d"}`, i)
		}))
		Expect(fieldAt(shapes[0], "/name").Enum).To(BeNil())
	})

	It("Should pick the rising timestamp nearest the write time as the index", func() {
		shapes := xjson.Infer(documents("a", 3, func(i int) string {
			return fmt.Sprintf(
				`{"created_at":"2025-06-01T00:00:00Z","ts":%d}`, t0Seconds+int64(i),
			)
		}))
		Expect(fieldAt(shapes[0], "/created_at").DataType).To(Equal(telem.TimestampT))
		Expect(indexPointer(shapes[0])).To(Equal("/ts"))
	})

	It("Should not index on a timestamp that falls", func() {
		shapes := xjson.Infer(documents("a", 3, func(i int) string {
			return fmt.Sprintf(`{"ts":%d}`, t0Seconds-int64(i))
		}))
		Expect(fieldAt(shapes[0], "/ts").DataType).To(Equal(telem.TimestampT))
		Expect(shapes[0].Index).To(Equal(-1))
	})

	It("Should not index on a timestamp that some documents lack", func() {
		docs := documents("a", 2, func(i int) string {
			return fmt.Sprintf(`{"ts":%d,"v":1}`, t0Seconds+int64(i))
		})
		docs = append(docs, documents("a", 1, func(int) string { return `{"v":2}` })...)
		Expect(xjson.Infer(docs)[0].Index).To(Equal(-1))
	})

	It("Should not take a number far from the write time as a timestamp", func() {
		shapes := xjson.Infer(documents("a", 2, func(i int) string {
			return fmt.Sprintf(`{"uptime":%d}`, 3600+i)
		}))
		Expect(fieldAt(shapes[0], "/uptime").DataType).To(Equal(telem.Int64T))
		Expect(shapes[0].Index).To(Equal(-1))
	})

	It("Should not take a time-named field that mixes formats as a timestamp", func() {
		times := []string{`"2026-01-02T03:04:05Z"`, fmt.Sprint(t0Seconds)}
		shapes := xjson.Infer(documents("a", 2, func(i int) string {
			return `{"time":` + times[i] + `}`
		}))
		time := fieldAt(shapes[0], "/time")
		Expect(time.DataType).To(Equal(telem.Int64T))
		Expect(time.Mismatched).To(Equal(1))
	})

	It("Should read a Unix time in a string", func() {
		shapes := xjson.Infer(documents("a", 2, func(i int) string {
			return fmt.Sprintf(`{"ts":"%d"}`, t0Milliseconds+int64(i))
		}))
		ts := fieldAt(shapes[0], "/ts")
		Expect(ts.Type).To(Equal(xjson.String))
		Expect(ts.TimeFormat).To(HaveValue(Equal(xjson.UnixMillisecond)))
	})

	It("Should take a scalar document as a field at the root", func() {
		shapes := xjson.Infer(documents("temp", 2, func(i int) string {
			return fmt.Sprintf("%d.5", i)
		}))
		Expect(shapes[0].Fields).To(HaveLen(1))
		Expect(shapes[0].Fields[0].Pointer).To(BeEmpty())
		Expect(shapes[0].Fields[0].DataType).To(Equal(telem.Float64T))
	})

	It("Should give no field to arrays and nulls", func() {
		shapes := xjson.Infer(documents("a", 1, func(int) string {
			return `{"list":[1,2],"none":null,"nested":{"v":1}}`
		}))
		Expect(shapes[0].Fields).To(HaveLen(1))
		Expect(shapes[0].Fields[0].Pointer).To(Equal(xjson.Pointer{"nested", "v"}))
	})

	It("Should skip documents that are not JSON", func() {
		docs := documents(
			"a",
			2,
			func(i int) string { return []string{"{", `{"v":1}`}[i] },
		)
		docs = append(
			docs,
			documents("b", 1, func(int) string { return "\x00\x01" })...,
		)
		shapes := xjson.Infer(docs)
		Expect(shapes).To(HaveLen(1))
		Expect(shapes[0].Sources).To(Equal([]string{"a"}))
		Expect(shapes[0].Count).To(Equal(1))
	})

	It("Should write fields in their config form", func() {
		shapes := xjson.Infer(documents("a", 2, func(i int) string {
			return fmt.Sprintf(`{"a/b":{"ts":%d}}`, int64(t0)+int64(i))
		}))
		text := MustSucceed(json.Marshal(shapes[0].Fields[0]))
		Expect(string(text)).To(Equal(
			`{"pointer":"/a~1b/ts","data_type":"timestamp","type":"number",` +
				`"time_format":"unix_ns","enum":[],"count":2,"mismatched":0}`,
		))
		var decoded xjson.InferredField
		Expect(json.Unmarshal(text, &decoded)).To(Succeed())
		Expect(decoded.Pointer).To(Equal(xjson.Pointer{"a/b", "ts"}))
		Expect(decoded.Type).To(Equal(xjson.Number))
		Expect(decoded.TimeFormat).To(HaveValue(Equal(xjson.UnixNanosecond)))
	})

	It("Should omit the time format of a field that is not a timestamp", func() {
		shapes := xjson.Infer(documents("a", 1, func(int) string { return `{"v":1}` }))
		Expect(string(MustSucceed(json.Marshal(shapes[0].Fields[0])))).To(Equal(
			`{"pointer":"/v","data_type":"int64","type":"number","enum":[],` +
				`"count":1,"mismatched":0}`,
		))
	})
})
