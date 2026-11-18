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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
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

	It("Should name a field after its pointer and keep names unique", func() {
		named := func(key, name, pointer string) v0.InlineField {
			f := inlineField(telem.Float64T)
			f.Key, f.Name, f.Pointer = key, name, pointer
			return f
		}
		msg, _ := v0.LiftMessage([]v0.InlineField{
			named("f1", "", "/out/flow"),
			named("f2", "level", "/a"),
			named("f3", "level", "/b"),
			named("f4", "", ""),
		}, "", found, "e", note)
		names := make([]string, 0, 4)
		for _, f := range jsonFields(msg) {
			names = append(names, f.Base().Name)
		}
		Expect(names).To(Equal([]string{"out/flow", "level", "level (2)", "value"}))
	})

	It("Should note each change at the path of its field", func() {
		unknown := inlineField("complex128")
		unknown.Key = "f2"
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

	It("Should drop an index that names no field", func() {
		_, read := v0.LiftMessage(
			[]v0.InlineField{inlineField(telem.Float64T)}, "gone", found, "e", note,
		)
		Expect(read.Index).To(BeNil())
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
