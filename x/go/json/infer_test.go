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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
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

// presentPointer returns the pointer of the present field of shape, or "none".
func presentPointer(shape xjson.Shape) string {
	if shape.Present < 0 {
		return "none"
	}
	return shape.Fields[shape.Present].Pointer.String()
}

// narrow and wide return a document with a seconds timestamp offset s from t0, where
// the fields of narrow are all in wide.
func narrow(s int64) string { return fmt.Sprintf(`{"ts":%d,"v":1}`, t0Seconds+s) }

func wide(s int64) string { return fmt.Sprintf(`{"ts":%d,"v":1,"w":2}`, t0Seconds+s) }

// distinct returns n documents of n sets of fields, with seconds timestamps that start
// at offset s from t0 and rise.
func distinct(n int, s int64) []string {
	docs := make([]string, n)
	for i := range docs {
		docs[i] = fmt.Sprintf(`{"ts":%d,"x%d":1}`, t0Seconds+s+int64(i), i)
	}
	return docs
}

// stamped returns n documents of source with a seconds timestamp at /ts and the fields
// that fields returns for each index.
func stamped(source string, n int, fields func(i int) string) []xjson.Document {
	return documents(source, n, func(i int) string {
		return fmt.Sprintf(`{"ts":%d,%s}`, t0Seconds+int64(i), fields(i))
	})
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
			Kind:       payload.KindTimestamp,
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
		docs := make([]xjson.Document, 0, 10)
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

	It("Should give a shape to each cluster of documents of one source", func() {
		clusters := func(i int) string {
			ms := t0Milliseconds + int64(i)
			if i%2 == 0 {
				return fmt.Sprintf(`{"time":%d,"pressure":%d.5,"flow":2}`, ms, i)
			}
			return fmt.Sprintf(`{"time":%d,"state":"run","faults":0}`, ms)
		}
		docs := documents("line-1", 6, clusters)
		docs = append(docs, documents("line-2", 2, func(i int) string {
			return clusters(i * 2)
		})...)
		shapes := xjson.Infer(docs)
		Expect(shapes).To(HaveLen(2))
		Expect(shapes[0].Sources).To(Equal([]string{"line-1", "line-2"}))
		Expect(shapes[0].Count).To(Equal(5))
		Expect(shapes[0].Fields).To(HaveLen(3))
		Expect(fieldAt(shapes[0], "/pressure").DataType).To(Equal(telem.Float64T))
		Expect(indexPointer(shapes[0])).To(Equal("/time"))
		Expect(shapes[1].Sources).To(Equal([]string{"line-1"}))
		Expect(shapes[1].Count).To(Equal(3))
		Expect(fieldAt(shapes[1], "/state").Type).To(Equal(xjson.String))
		Expect(indexPointer(shapes[1])).To(Equal("/time"))
	})

	It("Should join a cluster whose fields are all in one wider cluster", func() {
		shapes := xjson.Infer(documents("tank", 4, func(i int) string {
			flow := "2"
			if i%2 == 1 {
				flow = "null"
			}
			return fmt.Sprintf(
				`{"ts":%d,"pressure":1,"flow":%s}`, t0Seconds+int64(i), flow,
			)
		}))
		Expect(shapes).To(HaveLen(1))
		Expect(shapes[0].Count).To(Equal(4))
		Expect(fieldAt(shapes[0], "/flow").Count).To(Equal(2))
		Expect(indexPointer(shapes[0])).To(Equal("/ts"))
	})

	It("Should keep apart a cluster whose fields are in two wider clusters", func() {
		shapes := xjson.Infer(documents("line", 3, func(i int) string {
			return [...]string{
				`{"t":1,"pressure":1}`,
				`{"t":2,"state":"run"}`,
				`{"t":3}`,
			}[i]
		}))
		Expect(shapes).To(HaveLen(3))
		Expect(shapes[2].Fields).To(HaveLen(1))
		Expect(shapes[2].Count).To(Equal(1))
	})

	It("Should give documents with no fields to no cluster of two", func() {
		shapes := xjson.Infer(documents("line", 4, func(i int) string {
			return [...]string{
				`{"t":1,"pressure":1}`,
				`{"t":2,"state":"run"}`,
				`{}`,
				`{"faults":[],"flow":null}`,
			}[i]
		}))
		Expect(shapes).To(HaveLen(2))
		Expect(shapes[0].Count).To(Equal(1))
		Expect(shapes[1].Count).To(Equal(1))
	})

	It(
		"Should give documents with no fields to the one cluster of their source",
		func() {
			shapes := xjson.Infer(documents("tank", 2, func(i int) string {
				return [...]string{`{"pressure":1}`, `{}`}[i]
			}))
			Expect(shapes).To(HaveLen(1))
			Expect(shapes[0].Count).To(Equal(2))
			Expect(fieldAt(shapes[0], "/pressure").Count).To(Equal(1))
		},
	)

	DescribeTable("Should split a source into clusters only up to 16 clusters",
		func(sets, shapes, fields int) {
			inferred := xjson.Infer(documents("devices", sets, func(i int) string {
				return fmt.Sprintf(`{"device-%d":1}`, i)
			}))
			Expect(inferred).To(HaveLen(shapes))
			Expect(inferred[0].Fields).To(HaveLen(fields))
			Expect(inferred[0].Count).To(Equal(fields))
		},
		Entry("16 clusters", 16, 16, 1),
		Entry("17 clusters", 17, 1, 17),
	)

	DescribeTable("Should count clusters after joining up to 256 sets of fields",
		func(sets, shapes int) {
			docs := documents("line", sets-1, func(i int) string {
				fields := []string{`"t":1`, `"pressure":1`}
				for f := range 8 {
					if i&(1<<f) == 0 {
						fields = append(fields, fmt.Sprintf(`"v%d":1`, f))
					}
				}
				return "{" + strings.Join(fields, ",") + "}"
			})
			docs = append(docs, documents("line", 1, func(int) string {
				return `{"t":1,"state":"run"}`
			})...)
			Expect(xjson.Infer(docs)).To(HaveLen(shapes))
		},
		Entry("256 sets", 256, 2),
		Entry("257 sets", 257, 1),
	)

	It(
		"Should mark a shape by its first field that only its documents all hold",
		func() {
			shapes := xjson.Infer(stamped("pump", 3, func(i int) string {
				return [...]string{`"note":"x","rpm":1`, `"fault":3`, `"rpm":2`}[i]
			}))
			Expect(shapes).To(HaveLen(2))
			Expect(presentPointer(shapes[0])).To(Equal("/rpm"))
			Expect(presentPointer(shapes[1])).To(Equal("/fault"))
		},
	)

	DescribeTable(
		"Should not mark a shape by a field that nests with a field of another",
		func(kinds, present []string) {
			shapes := xjson.Infer(documents("pump", len(kinds), func(i int) string {
				return kinds[i]
			}))
			pointers := make([]string, len(shapes))
			for i, s := range shapes {
				pointers[i] = presentPointer(s)
			}
			Expect(pointers).To(Equal(present))
		},
		Entry("an object and a string at one pointer",
			[]string{`{"status":{"code":1}}`, `{"status":"ok"}`},
			[]string{"none"},
		),
		Entry("a string and an object at one pointer, with a shared field",
			[]string{`{"t":1,"status":"ok"}`, `{"t":1,"status":{"code":3}}`},
			[]string{"none"},
		),
		Entry("an object and a string at one pointer, with fields that differ",
			[]string{`{"status":{"code":1},"fault":2}`, `{"status":"ok","rpm":3}`},
			[]string{"/fault", "/rpm"},
		),
	)

	It("Should mark a shape by a value field before its timestamp", func() {
		shapes := xjson.Infer(documents("pump", 4, func(i int) string {
			if i%2 == 0 {
				return fmt.Sprintf(`{"ts":%d,"rpm":%d}`, t0Seconds+int64(i), i)
			}
			return fmt.Sprintf(`{"time":%d,"code":%d}`, t0Seconds+int64(i), i)
		}))
		Expect(shapes).To(HaveLen(2))
		Expect(presentPointer(shapes[0])).To(Equal("/rpm"))
		Expect(presentPointer(shapes[1])).To(Equal("/code"))
	})

	It(
		"Should mark a shape by its timestamp when no value field sets it apart",
		func() {
			shapes := xjson.Infer(documents("pump", 4, func(i int) string {
				if i%2 == 0 {
					return fmt.Sprintf(`{"ts":%d,"v":%d}`, t0Seconds+int64(i), i)
				}
				return fmt.Sprintf(`{"time":%d,"v":%d}`, t0Seconds+int64(i), i)
			}))
			Expect(shapes).To(HaveLen(2))
			Expect(presentPointer(shapes[0])).To(Equal("/ts"))
			Expect(presentPointer(shapes[1])).To(Equal("/time"))
		},
	)

	It("Should not mark a shape by the whole document", func() {
		shapes := xjson.Infer(documents("pump", 2, func(i int) string {
			return [...]string{`1.5`, `{"rpm":1}`}[i]
		}))
		Expect(shapes).To(HaveLen(2))
		Expect(presentPointer(shapes[0])).To(Equal("none"))
		Expect(presentPointer(shapes[1])).To(Equal("/rpm"))
	})

	It("Should give no present field to the only shape of its source", func() {
		shapes := xjson.Infer(stamped("pump", 2, func(i int) string {
			return fmt.Sprintf(`"rpm":%d`, i)
		}))
		Expect(shapes).To(HaveLen(1))
		Expect(presentPointer(shapes[0])).To(Equal("none"))
	})

	It("Should merge the clusters of a source that no field tells apart", func() {
		shapes := xjson.Infer(stamped("line", 5, func(i int) string {
			return [...]string{
				`"a":1,"b":1`, `"b":1,"c":1`, `"a":1,"b":1`, `"a":1,"c":1`, `"d":1`,
			}[i]
		}))
		Expect(shapes).To(HaveLen(2))
		Expect(shapes[0].Count).To(Equal(4))
		Expect(shapes[0].Fields).To(HaveLen(4))
		Expect(presentPointer(shapes[0])).To(Equal("none"))
		Expect(indexPointer(shapes[0])).To(Equal("/ts"))
		Expect(presentPointer(shapes[1])).To(Equal("/d"))
	})

	It("Should mark a shape of two sources by a field no other shape of either holds",
		func() {
			docs := stamped("pump-1", 2, func(i int) string {
				return [...]string{`"a":1,"b":1`, `"a":1,"c":1`}[i]
			})
			docs = append(docs, stamped("pump-2", 2, func(i int) string {
				return [...]string{`"a":1,"b":1`, `"b":1,"d":1`}[i]
			})...)
			shapes := xjson.Infer(docs)
			Expect(shapes).To(HaveLen(3))
			Expect(shapes[0].Sources).To(Equal([]string{"pump-1", "pump-2"}))
			Expect(presentPointer(shapes[0])).To(Equal("none"))
			Expect(presentPointer(shapes[1])).To(Equal("/c"))
			Expect(presentPointer(shapes[2])).To(Equal("/d"))
		},
	)

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
			Kind:     payload.KindBoolean,
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

	It("Should give an enum label of 64 bytes", func() {
		label := strings.Repeat("x", 64)
		shapes := xjson.Infer(documents("a", 2, func(int) string {
			return fmt.Sprintf(`{"state":%q}`, label)
		}))
		Expect(fieldAt(shapes[0], "/state").Enum).To(Equal([]string{label}))
	})

	It("Should not give an enum to a string field with a value over 64 bytes", func() {
		shapes := xjson.Infer(documents("a", 3, func(i int) string {
			if i == 2 {
				return fmt.Sprintf(`{"state":%q}`, strings.Repeat("x", 65))
			}
			return `{"state":"open"}`
		}))
		Expect(fieldAt(shapes[0], "/state").Enum).To(BeNil())
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

	DescribeTable("Should index on a timestamp that rises across the sets of a cluster",
		func(index string, docs ...string) {
			shapes := xjson.Infer(documents("pump", len(docs), func(i int) string {
				return docs[i]
			}))
			Expect(shapes).To(HaveLen(1))
			Expect(indexPointer(shapes[0])).To(Equal(index))
		},
		Entry("rising across the sets", "/ts", wide(0), narrow(1), wide(2)),
		Entry(
			"older than the last value of the other set",
			"none",
			wide(0),
			narrow(2),
			wide(1),
		),
		Entry(
			"two sets that each rise and alternate with older values",
			"none",
			narrow(10), wide(5), narrow(20), wide(15), narrow(30), wide(25),
		),
	)

	It(
		"Should index each cluster on a timestamp that is out of order across them",
		func() {
			shapes := xjson.Infer(documents("pump", 4, func(i int) string {
				s := [...]int64{10, 5, 20, 15}[i]
				return fmt.Sprintf(
					`{"ts":%d,"%s":1}`,
					t0Seconds+s,
					[...]string{"rpm", "fault"}[i%2],
				)
			}))
			Expect(shapes).To(HaveLen(2))
			Expect(indexPointer(shapes[0])).To(Equal("/ts"))
			Expect(indexPointer(shapes[1])).To(Equal("/ts"))
		},
	)

	DescribeTable("Should index on a timestamp after the sets of a source pass 256",
		func(index string, docs []string) {
			shapes := xjson.Infer(documents("pump", len(docs), func(i int) string {
				return docs[i]
			}))
			Expect(shapes).To(HaveLen(1))
			Expect(indexPointer(shapes[0])).To(Equal(index))
		},
		Entry("rising", "/ts", distinct(257, 0)),
		Entry(
			"out of order across two sets before",
			"none",
			append([]string{wide(0), narrow(2), wide(1)}, distinct(256, 3)...),
		),
		Entry(
			"older in the 257th set than the newest value before",
			"none",
			append(distinct(256, 0), narrow(100)),
		),
	)

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
		times := []string{`"2026-01-02T03:04:05Z"`, strconv.FormatInt(t0Seconds, 10)}
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
			`{"pointer":"/a~1b/ts","data_type":"timestamp","kind":"timestamp",` +
				`"type":"number","time_format":"unix_ns","enum":[],"count":2,` +
				`"mismatched":0}`,
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
			`{"pointer":"/v","data_type":"int64","kind":"integer","type":"number",` +
				`"enum":[],"count":1,"mismatched":0}`,
		))
	})

	DescribeTable(
		"Should give each field the kind of a payload field that holds its values",
		func(values []string, dataType telem.DataType, kind payload.Kind) {
			shapes := xjson.Infer(documents("a", len(values), func(i int) string {
				return fmt.Sprintf(`{"v":%s}`, values[i])
			}))
			v := fieldAt(shapes[0], "/v")
			Expect(v.DataType).To(Equal(dataType))
			Expect(v.Kind).To(Equal(kind))
		},
		Entry(
			"signed integers",
			[]string{"1", "-2"},
			telem.Int64T,
			payload.KindInteger,
		),
		Entry(
			"an integer above the int64 range",
			[]string{"1", "18446744073709551615"},
			telem.Uint64T,
			payload.KindUnsigned,
		),
		Entry("a fraction", []string{"1", "2.5"}, telem.Float64T, payload.KindFloat),
		Entry("booleans", []string{"true", "false"}, telem.Uint8T, payload.KindBoolean),
		Entry("strings", []string{`"a"`, `"b"`}, telem.StringT, payload.KindString),
	)
})

var _ = Describe("Inferrer", func() {
	It("Should infer one shape from every document added", func() {
		inf := xjson.NewInferrer()
		for i := range 3 {
			ts := t0 + telem.TimeStamp(i)*telem.SecondTS
			inf.Add(
				fmt.Appendf(nil, `{"ts":%d,"v":%d}`, int64(ts)/1e6, i),
				ts,
			)
		}
		inf.Add(
			fmt.Appendf(nil, `{"ts":%d,"v":1.5}`, t0Milliseconds+3000),
			t0+3*telem.SecondTS,
		)
		shapes := inf.Shapes()
		Expect(shapes).To(HaveLen(1))
		shape := shapes[0]
		Expect(shape.Sources).To(BeEmpty())
		Expect(shape.Count).To(Equal(4))
		Expect(presentPointer(shape)).To(Equal("none"))
		Expect(indexPointer(shape)).To(Equal("/ts"))
		ts := fieldAt(shape, "/ts")
		Expect(ts.Kind).To(Equal(payload.KindTimestamp))
		Expect(ts.TimeFormat).To(HaveValue(Equal(xjson.UnixMillisecond)))
		Expect(fieldAt(shape, "/v").Kind).To(Equal(payload.KindFloat))
	})

	It("Should add nothing for data that is not JSON", func() {
		inf := xjson.NewInferrer()
		inf.Add([]byte("not json"), t0)
		inf.Add([]byte(`{"v":1} {"v":2}`), t0)
		Expect(inf.Shapes()).To(BeEmpty())
	})

	It("Should give no index when a timestamp falls", func() {
		inf := xjson.NewInferrer()
		inf.Add(fmt.Appendf(nil, `{"ts":%d}`, t0Milliseconds+1000), t0)
		inf.Add(fmt.Appendf(nil, `{"ts":%d}`, t0Milliseconds), t0+telem.SecondTS)
		shapes := inf.Shapes()
		Expect(shapes).To(HaveLen(1))
		Expect(fieldAt(shapes[0], "/ts").Kind).To(Equal(payload.KindTimestamp))
		Expect(shapes[0].Index).To(Equal(-1))
	})

	DescribeTable("Should give one shape past 1,024 fields across its sets of fields",
		func(second, shapes, fields int) {
			inf := xjson.NewInferrer()
			for i, n := range []int{600, second} {
				names := make([]string, n)
				for f := range names {
					names[f] = fmt.Sprintf(`"s%d-%d":1`, i, f)
				}
				inf.Add([]byte("{"+strings.Join(names, ",")+"}"), t0)
			}
			inferred := inf.Shapes()
			Expect(inferred).To(HaveLen(shapes))
			Expect(inferred[0].Fields).To(HaveLen(fields))
		},
		Entry("1,024 fields", 424, 2, 600),
		Entry("1,025 fields", 425, 1, 1024),
	)

	DescribeTable(
		"Should infer the shapes that Infer gives the same documents",
		func(docs []xjson.Document) {
			inf := xjson.NewInferrer()
			for _, doc := range docs {
				inf.Add(doc.Data, doc.Time)
			}
			expected := xjson.Infer(docs)
			for i := range expected {
				expected[i].Sources = nil
			}
			Expect(inf.Shapes()).To(Equal(expected))
		},
		Entry("two record kinds", stamped("", 4, func(i int) string {
			return [...]string{`"rpm":1`, `"fault":"x"`, `"rpm":2.5`, `"fault":"y"`}[i]
		})),
		Entry("a set that joins a wider set", documents("", 3, func(i int) string {
			return [...]string{narrow(0), wide(1), narrow(2)}[i]
		})),
		Entry("sets that no field tells apart", documents("", 3, func(i int) string {
			return [...]string{`{"a":1,"b":1}`, `{"a":2,"c":1}`, `{"b":2,"c":2}`}[i]
		})),
		Entry(
			"a timestamp out of order across sets",
			documents("", 3, func(i int) string {
				return [...]string{wide(0), narrow(2), wide(1)}[i]
			}),
		),
		Entry("more than 256 sets", documents("", 257, func(i int) string {
			return append(distinct(256, 0), narrow(100))[i]
		})),
	)
})
