// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/errors"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

func inlineField(dt telem.DataType, entries ...payload.EnumEntry) v0.InlineField {
	return v0.InlineField{
		Key:        "f1",
		Name:       "level",
		Channel:    7,
		Pointer:    "/level",
		DataType:   dt,
		EnumValues: entries,
	}
}

// as returns v as a T, and fails the spec when v is not a T.
func as[T any](v any) T {
	GinkgoHelper()
	t, ok := v.(T)
	Expect(ok).To(BeTrue())
	return t
}

// jsonFields returns the fields of the JSON message msg.
func jsonFields(msg v0.Message) []v0.JSONField {
	GinkgoHelper()
	return as[v0.JSONPayload](msg.Payload.Variant).Fields
}

var _ = Describe("LiftMessage", func() {
	var (
		found   map[channel.Key]v0.Channel
		changes []string
		paths   []string
	)
	BeforeEach(func() { found, changes, paths = nil, nil, nil })
	note := func(path, change string) {
		paths = append(paths, path)
		changes = append(changes, change)
	}

	// lift lifts a message that holds old alone, and returns the field of old in the
	// message and the read of the message.
	lift := func(old v0.InlineField, index bool) (v0.JSONField, v0.ReadMessage) {
		GinkgoHelper()
		var indexKey string
		if index {
			indexKey = old.Key
		}
		msg, read := v0.LiftMessage([]v0.InlineField{old}, indexKey, found, "e", note)
		return jsonFields(msg)[0], read
	}

	DescribeTable(
		"Should pick the kind of the stored data type",
		func(dt telem.DataType, kind payload.Kind) {
			f, _ := lift(inlineField(dt), false)
			Expect(f.Type()).To(Equal(kind))
			Expect(changes).To(BeEmpty())
		},
		Entry("float32", telem.Float32T, payload.KindFloat),
		Entry("int16", telem.Int16T, payload.KindInteger),
		Entry("uint8", telem.Uint8T, payload.KindUnsigned),
		Entry("boolean", telem.BooleanT, payload.KindBoolean),
		Entry("string", telem.StringT, payload.KindString),
		Entry("timestamp", telem.TimestampT, payload.KindTimestamp),
		Entry("json", telem.JSONT, payload.KindJSON),
		Entry("uuid", telem.UUIDT, payload.KindUUID),
		Entry("bytes", telem.BytesT, payload.KindBytes),
	)

	It("Should return an unnamed message that its read reads", func() {
		msg, read := v0.LiftMessage(
			[]v0.InlineField{inlineField(telem.Float64T)}, "", found, "e", note,
		)
		Expect(msg.Name).To(BeEmpty())
		Expect(msg.Key).To(Equal(read.Message))
	})

	It("Should keep the name and the pointer of the field", func() {
		f, read := lift(inlineField(telem.Float64T), false)
		base := f.Base()
		Expect(base.Name).To(Equal("level"))
		Expect(base.Pointer).To(Equal("/level"))
		Expect(base.Scale).To(Equal(1.0))
		Expect(
			read.Bindings,
		).To(Equal([]v0.ReadBinding{{Field: base.Key, DataType: telem.Float64T}}))
	})

	It("Should keep the stored data type as the data type of a new channel", func() {
		f, read := lift(inlineField(telem.Int16T), false)
		Expect(read.Bindings).To(Equal([]v0.ReadBinding{
			{Field: f.Base().Key, DataType: telem.Int16T},
		}))
	})

	It("Should bind the channel of a field and take the kind of the channel", func() {
		found = map[channel.Key]v0.Channel{7: {Name: "level", DataType: telem.Int32T}}
		f, read := lift(inlineField(
			telem.Float64T,
			payload.EnumEntry{Label: "ON", Value: 1},
			payload.EnumEntry{Label: "HALF", Value: 0.5},
		), false)
		integer := as[v0.IntegerJSONField](f.Variant)
		Expect(integer.EnumValues).To(Equal(
			[]payload.IntegerEntry{{Label: "ON", Value: 1}},
		))
		Expect(read.Bindings).To(Equal([]v0.ReadBinding{
			{Field: f.Base().Key, Channel: 7},
		}))
		Expect(changes).To(Equal([]string{
			"data type float64 became int32, the data type of its channel",
			`enum label "HALF" was dropped: 0.5 is not integer`,
		}))
	})

	It("Should lift a field with a time format on a timestamp channel", func() {
		found = map[channel.Key]v0.Channel{
			7: {Name: "time", DataType: telem.TimestampT, Index: true},
		}
		old := inlineField("")
		old.TimeFormat = new(payload.TimeFormatUnixMs)
		f, _ := lift(old, false)
		stamp := as[v0.TimestampJSONField](f.Variant)
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
	})

	It("Should convert whole entries and drop the others on an integer kind", func() {
		f, _ := lift(inlineField(
			telem.Int32T,
			payload.EnumEntry{Label: "OFF", Value: 0},
			payload.EnumEntry{Label: "ON", Value: 1},
			payload.EnumEntry{Label: "HALF", Value: 0.5},
			payload.EnumEntry{Label: "", Value: 2},
		), false)
		integer := as[v0.IntegerJSONField](f.Variant)
		Expect(integer.EnumValues).To(Equal([]payload.IntegerEntry{
			{Label: "OFF", Value: 0},
			{Label: "ON", Value: 1},
		}))
		Expect(changes).To(ConsistOf(
			`enum label "HALF" was dropped: 0.5 is not integer`,
			`enum label "" was dropped: it is empty`,
		))
	})

	It("Should keep the first entry of a repeated label", func() {
		f, _ := lift(inlineField(
			telem.BooleanT,
			payload.EnumEntry{Label: "ON", Value: 1},
			payload.EnumEntry{Label: "ON", Value: 0},
		), false)
		boolean := as[v0.BooleanJSONField](f.Variant)
		Expect(boolean.EnumValues).To(Equal(
			[]payload.BooleanEntry{{Label: "ON", Value: true}},
		))
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: it repeats an earlier label`,
		))
	})

	It("Should drop a negative entry on an unsigned kind", func() {
		f, _ := lift(inlineField(
			telem.Uint16T,
			payload.EnumEntry{Label: "LOW", Value: -1},
			payload.EnumEntry{Label: "HIGH", Value: 3},
		), false)
		unsigned := as[v0.UnsignedJSONField](f.Variant)
		Expect(unsigned.EnumValues).To(Equal(
			[]payload.UnsignedEntry{{Label: "HIGH", Value: 3}},
		))
		Expect(
			changes,
		).To(ConsistOf(`enum label "LOW" was dropped: -1 is not unsigned`))
	})

	It("Should keep every finite entry on a float kind", func() {
		f, _ := lift(inlineField(
			telem.Float64T,
			payload.EnumEntry{Label: "HALF", Value: 0.5},
		), false)
		float := as[v0.FloatJSONField](f.Variant)
		Expect(float.EnumValues).To(Equal(
			[]payload.FloatEntry{{Label: "HALF", Value: 0.5}},
		))
		Expect(changes).To(BeEmpty())
	})

	It("Should map a boolean entry to whether its value is not 0", func() {
		f, _ := lift(inlineField(
			telem.BooleanT,
			payload.EnumEntry{Label: "OFF", Value: 0},
			payload.EnumEntry{Label: "ON", Value: 2},
		), false)
		boolean := as[v0.BooleanJSONField](f.Variant)
		Expect(boolean.EnumValues).To(Equal([]payload.BooleanEntry{
			{Label: "OFF", Value: false},
			{Label: "ON", Value: true},
		}))
	})

	It("Should drop the entries of a string field", func() {
		f, _ := lift(inlineField(
			telem.StringT,
			payload.EnumEntry{Label: "ON", Value: 1},
		), false)
		str := as[v0.StringJSONField](f.Variant)
		Expect(str.EnumValues).To(BeEmpty())
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: a string field takes no numeric entries`,
		))
	})

	It("Should give a timestamp field with no time format ISO 8601", func() {
		f, _ := lift(inlineField(telem.TimestampT), false)
		stamp := as[v0.TimestampJSONField](f.Variant)
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
	})

	It("Should keep the time format of a field stored with another data type", func() {
		old := inlineField(telem.Float64T)
		old.TimeFormat = new(payload.TimeFormatUnixMs)
		f, read := lift(old, false)
		stamp := as[v0.TimestampJSONField](f.Variant)
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		Expect(read.Bindings[0].DataType).To(BeEmpty())
		Expect(changes).To(ConsistOf(
			"data type float64 became the timestamp kind of an index or time format",
		))
	})

	It("Should give ISO 8601 to a field whose time format is not valid", func() {
		old := inlineField(telem.Float64T)
		old.TimeFormat = new(payload.TimeFormat("unix"))
		f, _ := lift(old, false)
		stamp := as[v0.TimestampJSONField](f.Variant)
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
		Expect(stamp.Validate()).To(Succeed())
		Expect(changes).To(ConsistOf(
			"data type float64 became the timestamp kind of an index or time format",
			`time format "unix" is not valid and became iso8601`,
		))
	})

	It("Should keep the kind and entries of a typed field with a time format", func() {
		old := inlineField(telem.Uint8T, payload.EnumEntry{Label: "ON", Value: 1})
		old.TimeFormat = new(payload.TimeFormatUnixSec)
		f, _ := lift(old, false)
		unsigned := as[v0.UnsignedJSONField](f.Variant)
		Expect(unsigned.EnumValues).To(Equal(
			[]payload.UnsignedEntry{{Label: "ON", Value: 1}},
		))
		Expect(changes).To(ConsistOf(
			"time format unix_sec was dropped, because the field is unsigned",
		))
	})

	It("Should lift an index field to the timestamp kind and point the index at it",
		func() {
			f, read := lift(inlineField(""), true)
			stamp := as[v0.TimestampJSONField](f.Variant)
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
			Expect(read.Index).To(HaveValue(Equal(f.Base().Key)))
			Expect(read.Bindings).To(Equal([]v0.ReadBinding{{Field: f.Base().Key}}))
			Expect(changes).To(ConsistOf(
				"data type float64 became the timestamp kind of an index or time format",
			))
		},
	)

	It("Should lift an unknown data type to the float kind with no data type", func() {
		f, read := lift(inlineField("complex128"), false)
		Expect(f.Type()).To(Equal(payload.KindFloat))
		Expect(read.Bindings[0].DataType).To(BeEmpty())
		Expect(changes).To(ConsistOf(
			`unknown data type "complex128" became the float kind`,
		))
	})

	It("Should lift an unknown data type with a time format to the timestamp kind",
		func() {
			old := inlineField("complex128")
			old.TimeFormat = new(payload.TimeFormatUnixSec)
			f, _ := lift(old, false)
			stamp := as[v0.TimestampJSONField](f.Variant)
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixSec))
			Expect(changes).To(ConsistOf(
				`unknown data type "complex128" became the float kind`,
				"data type complex128 became the timestamp kind of an index or time format",
			))
		},
	)

	It("Should take float64 for an empty data type", func() {
		f, read := lift(inlineField(""), false)
		Expect(f.Type()).To(Equal(payload.KindFloat))
		Expect(read.Bindings[0].DataType).To(Equal(telem.Float64T))
		Expect(changes).To(BeEmpty())
	})

	It("Should keep a disabled field in the message without a binding", func() {
		disabled := inlineField(telem.Float64T)
		disabled.Key, disabled.Name, disabled.Disabled = "f2", "spare", true
		msg, read := v0.LiftMessage(
			[]v0.InlineField{inlineField(telem.Float64T), disabled},
			"",
			found,
			"e",
			note,
		)
		Expect(jsonFields(msg)).To(HaveLen(2))
		Expect(read.Bindings).To(HaveLen(1))
		Expect(read.Bindings[0].Field).To(Equal(jsonFields(msg)[0].Base().Key))
	})

	It("Should name the fields of a message apart from each other", func() {
		inline := make([]v0.InlineField, 3)
		for i, f := range [][2]string{{"", "/out/flow"}, {"level", "/a"}, {"level", "/b"}} {
			inline[i] = inlineField(telem.Float64T)
			inline[i].Name, inline[i].Pointer = f[0], f[1]
		}
		msg, _ := v0.LiftMessage(inline, "", found, "e", note)
		names := make([]string, 0, len(inline))
		for _, f := range jsonFields(msg) {
			names = append(names, f.Base().Name)
		}
		Expect(names).To(Equal([]string{"out_flow", "level", "level_2"}))
	})

	It("Should note each change at the path of its field", func() {
		unknown := inlineField("complex128")
		unknown.Key, unknown.Pointer = "f2", "/flow"
		v0.LiftMessage(
			[]v0.InlineField{inlineField(telem.Float64T), unknown},
			"gone",
			found,
			"entries.2",
			note,
		)
		Expect(paths).To(Equal([]string{"entries.2.fields.1", "entries.2.index"}))
		Expect(changes).To(Equal([]string{
			`unknown data type "complex128" became the float kind`,
			"index field gone does not exist and was dropped",
		}))
	})

	Describe("Overlapping pointers", func() {
		// lift lifts the field level at /level, then the field flow at pointer.
		lift := func(pointer string) {
			flow := inlineField(telem.Float64T)
			flow.Key, flow.Name, flow.Pointer = "f2", "flow", pointer
			v0.LiftMessage(
				[]v0.InlineField{
					inlineField(telem.Float64T),
					flow,
				},
				"",
				found,
				"e",
				note,
			)
		}

		DescribeTable("Should note a field that reads the value of an earlier field",
			func(pointer, reads string) {
				lift(pointer)
				Expect(paths).To(Equal([]string{"e.fields.1"}))
				Expect(changes).To(Equal([]string{
					`field "flow" reads ` + reads + ` field "level", so keep one of ` +
						"them before you edit the ICD",
				}))
			},
			Entry("the same pointer", "/level", "the same value as"),
			Entry("a pointer into its value", "/level/raw", "a value inside"),
			Entry("a pointer that holds its value", "", "a value that holds"),
		)

		DescribeTable("Should not note a field whose value is apart",
			func(pointer string) {
				lift(pointer)
				Expect(changes).To(BeEmpty())
			},
			Entry("a sibling pointer", "/flow"),
			Entry("a pointer that extends a token", "/levels"),
			Entry("a malformed pointer", "/level~2"),
		)
	})

	It("Should drop an index that names no field", func() {
		_, read := v0.LiftMessage(
			[]v0.InlineField{inlineField(telem.Float64T)}, "gone", found, "e", note,
		)
		Expect(read.Index).To(BeNil())
	})
})

var _ = Describe("FieldName", func() {
	// names returns the names that FieldName gives fields of each name and pointer,
	// apart from each other.
	names := func(fields ...[2]string) []string {
		taken := make(set.Set[string])
		out := make([]string, len(fields))
		for i, f := range fields {
			out[i] = v0.FieldName(taken, f[0], f[1])
		}
		return out
	}

	DescribeTable("Should name a field with no name by its pointer",
		func(pointer, name string) {
			Expect(names([2]string{"", pointer})).To(Equal([]string{name}))
		},
		Entry("a top-level pointer", "/level", "level"),
		Entry("a nested pointer", "/motor/speed", "motor_speed"),
		Entry("escaped tokens", "/a~1b/c~0d", "a/b_c~d"),
		Entry("the whole document", "", "value"),
		Entry("a pointer to an empty key", "/", "value"),
		Entry("a malformed pointer", "/a~2", "a~2"),
	)

	It("Should keep the name of a named field", func() {
		Expect(names([2]string{"flow", "/level"})).To(Equal([]string{"flow"}))
	})

	It("Should number each name that is taken and add each name to taken", func() {
		taken := set.New("level")
		Expect(v0.FieldName(taken, "level", "/a")).To(Equal("level_2"))
		Expect(v0.FieldName(taken, "", "")).To(Equal("value"))
		Expect(v0.FieldName(taken, "value", "/x")).To(Equal("value_2"))
		Expect(taken).To(Equal(set.New("level", "level_2", "value", "value_2")))
	})

	It("Should number a name that is taken as a channel name", func() {
		Expect(names(
			[2]string{"a b", ""},
			[2]string{"a_b", ""},
			[2]string{"", "/1st"},
			[2]string{"_1st", ""},
		)).To(Equal([]string{"a b", "a_b_2", "1st", "_1st_2"}))
		taken := make(set.Set[string])
		v0.FieldName(taken, "a b", "")
		Expect(taken).To(Equal(set.New("a_b")))
	})
})

var _ = Describe("UniqueName", func() {
	It("Should keep a free name and suffix a taken one", func() {
		taken := set.New("level")
		Expect(v0.UniqueName(taken, "flow")).To(Equal("flow"))
		Expect(v0.UniqueName(taken, "level")).To(Equal("level (2)"))
		Expect(v0.UniqueName(taken, "level")).To(Equal("level (3)"))
		Expect(taken).To(Equal(set.New("level", "flow", "level (2)", "level (3)")))
	})
})

var _ = Describe("Channels", func() {
	Describe("Of", func() {
		var asked [][]channel.Key
		BeforeEach(func() { asked = nil })
		channels := v0.Channels(func(
			_ context.Context,
			_ gorp.Snapshot,
			keys []channel.Key,
		) (map[channel.Key]v0.Channel, error) {
			asked = append(asked, keys)
			return map[channel.Key]v0.Channel{7: {Name: "level"}}, nil
		})
		withChannel := func(ch channel.Key) v0.InlineField {
			f := inlineField(telem.Float64T)
			f.Channel = ch
			return f
		}

		It("Should look up each channel of the fields once", func(ctx SpecContext) {
			Expect(channels.Of(ctx, nil, []v0.InlineField{
				withChannel(7), withChannel(0), withChannel(9), withChannel(7),
			})).To(Equal(map[channel.Key]v0.Channel{7: {Name: "level"}}))
			Expect(asked).To(Equal([][]channel.Key{{7, 9}}))
		})

		It("Should look up nothing when no field has a channel", func(ctx SpecContext) {
			Expect(channels.Of(ctx, nil, []v0.InlineField{withChannel(0)})).To(BeNil())
			Expect(asked).To(BeEmpty())
		})

		It("Should return the error of the lookup", func(ctx SpecContext) {
			failed := errors.New("lookup failed")
			failing := v0.Channels(func(
				context.Context,
				gorp.Snapshot,
				[]channel.Key,
			) (map[channel.Key]v0.Channel, error) {
				return nil, failed
			})
			Expect(failing.Of(ctx, nil, []v0.InlineField{withChannel(7)})).
				Error().To(MatchError(failed))
		})
	})
})

var _ = Describe("LiftKind", func() {
	var changes []string
	BeforeEach(func() { changes = nil })
	note := func(format string, args ...any) {
		changes = append(changes, fmt.Sprintf(format, args...))
	}
	base := v0.JSONBaseField{Name: "level", Scale: 1, Pointer: "/level"}

	It("Should keep the entries that the kind can hold and note the others", func() {
		f := v0.LiftKind(v0.InlineField{EnumValues: []payload.EnumEntry{
			{Label: "OFF", Value: 0},
			{Label: "HALF", Value: 0.5},
		}}, payload.KindInteger, base, note)
		integer := as[v0.IntegerJSONField](f.Variant)
		Expect(integer.JSONBaseField).To(Equal(base))
		Expect(integer.EnumValues).To(Equal([]payload.IntegerEntry{
			{Label: "OFF", Value: 0},
		}))
		Expect(
			changes,
		).To(ConsistOf(`enum label "HALF" was dropped: 0.5 is not integer`))
	})

	It("Should keep the time format of a timestamp kind", func() {
		format := payload.TimeFormatUnixMs
		f := v0.LiftKind(
			v0.InlineField{TimeFormat: &format},
			payload.KindTimestamp,
			base,
			note,
		)
		Expect(as[v0.TimestampJSONField](f.Variant).TimeFormat).To(Equal(format))
		Expect(changes).To(BeEmpty())
	})

	It("Should drop the entries of a kind that takes none", func() {
		f := v0.LiftKind(v0.InlineField{EnumValues: []payload.EnumEntry{
			{Label: "ON", Value: 1},
		}}, payload.KindString, base, note)
		Expect(f.Variant).To(Equal(v0.StringJSONField{JSONBaseField: base}))
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: a string field takes no numeric entries`,
		))
	})
})
