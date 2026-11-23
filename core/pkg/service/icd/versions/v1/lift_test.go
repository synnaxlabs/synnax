// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// liftedVariant returns the variant of f, which must be a T.
func liftedVariant[T icd.JSONFieldVariant](f icd.JSONField) T {
	GinkgoHelper()
	v, ok := f.Variant.(T)
	return MustBeOk(v, ok)
}

// liftedFields returns the fields of the JSON message msg.
func liftedFields(msg icd.Message) []icd.JSONField {
	GinkgoHelper()
	p, ok := msg.Payload.Variant.(icd.JSONPayload)
	return MustBeOk(p, ok).Fields
}

// valueChannel returns a channel field at /value that sends channel 7 as a number.
func valueChannel() icd.InlineChannel {
	return icd.InlineChannel{
		Pointer:  "/value",
		JSONType: payload.JSONTypeNumber,
		Channel:  7,
		Name:     "valve_cmd",
		DataType: telem.Float64T,
	}
}

// note is one change that LiftWrite reports.
type note struct{ path, change string }

var _ = Describe("LiftWrite", func() {
	var found map[channel.Key]v0.Channel
	BeforeEach(func() {
		found = map[channel.Key]v0.Channel{
			7: {Name: "valve_cmd", DataType: telem.Float64T},
		}
	})

	// lift lifts the write of ch and fields on topic plant/valve/set at path targets.0,
	// and returns the fields of its message, the write, and each note.
	lift := func(
		ctx SpecContext,
		ch icd.InlineChannel,
		fields ...icd.InlineWriteField,
	) ([]icd.JSONField, icd.WriteMessage, []note) {
		GinkgoHelper()
		var notes []note
		msg, write := MustSucceed2(icd.LiftWrite(
			ctx,
			icd.InlineWrite{
				Target:  "topic plant/valve/set",
				Channel: ch,
				Fields:  fields,
			},
			found,
			"targets.0",
			func(path, change string) { notes = append(notes, note{path, change}) },
		))
		Expect(write.Message).To(Equal(msg.Key))
		Expect(msg.Name).To(BeEmpty())
		return liftedFields(msg), write, notes
	}

	It("Should bind the channel of the channel field", func(ctx SpecContext) {
		f, write, notes := lift(ctx, valueChannel())
		Expect(f).To(HaveLen(1))
		Expect(f[0].Type()).To(Equal(payload.KindFloat))
		Expect(f[0].Base().Name).To(Equal("value"))
		Expect(f[0].Base().Pointer).To(Equal("/value"))
		Expect(f[0].Base().Scale).To(Equal(1.0))
		Expect(write.Bindings).To(Equal([]icd.WriteBinding{{
			Variant: icd.ChannelWriteBinding{Field: f[0].Base().Key, Channel: 7},
		}}))
		Expect(notes).To(BeEmpty())
	})

	It("Should bind no channel and keep the stored data type when the channel is gone",
		func(ctx SpecContext) {
			ch := valueChannel()
			ch.Channel, ch.DataType = 12, telem.Int16T
			f, write, _ := lift(ctx, ch)
			Expect(f[0].Type()).To(Equal(payload.KindInteger))
			Expect(write.Bindings).To(Equal([]icd.WriteBinding{{
				Variant: icd.ChannelWriteBinding{
					Field:    f[0].Base().Key,
					DataType: new(telem.Int16T),
				},
			}}))
		},
	)

	It("Should take float64 for a channel field with no channel or data type",
		func(ctx SpecContext) {
			ch := valueChannel()
			ch.Channel, ch.DataType = 0, ""
			f, write, _ := lift(ctx, ch)
			Expect(f[0].Type()).To(Equal(payload.KindFloat))
			Expect(write.Bindings).To(Equal([]icd.WriteBinding{{
				Variant: icd.ChannelWriteBinding{
					Field:    f[0].Base().Key,
					DataType: new(telem.Float64T),
				},
			}}))
		},
	)

	It("Should note the dropped name of a channel field with no channel",
		func(ctx SpecContext) {
			ch := valueChannel()
			ch.Channel = 0
			_, _, notes := lift(ctx, ch)
			Expect(notes).To(Equal([]note{{
				path: "targets.0.channel",
				change: `channel name "valve_cmd" of field value on topic ` +
					`plant/valve/set was dropped, so a client names the new channel`,
			}}))
		},
	)

	DescribeTable(
		"Should pick the kind that writes the channel as its JSON type",
		func(
			ctx SpecContext,
			dt telem.DataType,
			t payload.JSONType,
			kind payload.Kind,
		) {
			found[7] = v0.Channel{Name: "valve_cmd", DataType: dt}
			ch := valueChannel()
			ch.JSONType = t
			f, _, _ := lift(ctx, ch)
			Expect(f[0].Type()).To(Equal(kind))
		},
		Entry(
			"a float number",
			telem.Float32T,
			payload.JSONTypeNumber,
			payload.KindFloat,
		),
		Entry(
			"an integer number",
			telem.Int64T,
			payload.JSONTypeNumber,
			payload.KindInteger,
		),
		Entry(
			"an unsigned number",
			telem.Uint8T,
			payload.JSONTypeNumber,
			payload.KindUnsigned,
		),
		Entry("a boolean", telem.Uint8T, payload.JSONTypeBoolean, payload.KindBoolean),
		Entry("a string", telem.StringT, payload.JSONTypeString, payload.KindString),
		Entry("a UUID", telem.UUIDT, payload.JSONTypeString, payload.KindUUID),
		Entry("bytes", telem.BytesT, payload.JSONTypeString, payload.KindBytes),
		Entry("JSON", telem.JSONT, payload.JSONTypeString, payload.KindJSON),
	)

	It("Should lift a timestamp channel with its time format", func(ctx SpecContext) {
		found[7] = v0.Channel{Name: "valve_cmd", DataType: telem.TimestampT}
		ch := valueChannel()
		format := payload.TimeFormatUnixMs
		ch.TimeFormat = &format
		f, _, _ := lift(ctx, ch)
		Expect(liftedVariant[icd.TimestampJSONField](f[0]).TimeFormat).
			To(Equal(payload.TimeFormatUnixMs))
	})

	It("Should give a timestamp channel with no time format ISO 8601",
		func(ctx SpecContext) {
			found[7] = v0.Channel{Name: "valve_cmd", DataType: telem.TimestampT}
			f, _, _ := lift(ctx, valueChannel())
			Expect(liftedVariant[icd.TimestampJSONField](f[0]).TimeFormat).
				To(Equal(payload.TimeFormatISO8601))
		},
	)

	It("Should keep the enum values of a number channel sent as a string",
		func(ctx SpecContext) {
			found[7] = v0.Channel{Name: "valve_cmd", DataType: telem.Int32T}
			ch := valueChannel()
			ch.JSONType = payload.JSONTypeString
			ch.EnumValues = []payload.EnumEntry{
				{Label: "OPEN", Value: 1},
				{Label: "HALF", Value: 0.5},
			}
			f, _, _ := lift(ctx, ch)
			Expect(liftedVariant[icd.IntegerJSONField](f[0]).EnumValues).To(Equal(
				[]payload.IntegerEntry{{Label: "OPEN", Value: 1}},
			))
		},
	)

	DescribeTable(
		"Should note each change to what the channel field sends",
		func(
			ctx SpecContext,
			dt telem.DataType,
			t payload.JSONType,
			kind payload.Kind,
			change string,
		) {
			found[7] = v0.Channel{Name: "valve_cmd", DataType: dt}
			ch := valueChannel()
			ch.JSONType = t
			f, _, notes := lift(ctx, ch)
			Expect(f[0].Type()).To(Equal(kind))
			Expect(notes).To(Equal([]note{{path: "targets.0.channel", change: change}}))
		},
		Entry(
			"a number sent as a string",
			telem.Float64T,
			payload.JSONTypeString,
			payload.KindFloat,
			"a float64 channel sent as a JSON string now sends a JSON number",
		),
		Entry(
			"a number sent as a boolean",
			telem.Uint8T,
			payload.JSONTypeBoolean,
			payload.KindBoolean,
			"a uint8 channel sent as a JSON boolean now sends only 0 and 1",
		),
		Entry(
			"JSON sent as a string",
			telem.JSONT,
			payload.JSONTypeString,
			payload.KindJSON,
			"a json channel sent as a JSON string now sends its JSON value",
		),
	)

	It("Should give each static field a constant default", func(ctx SpecContext) {
		f, write, _ := lift(
			ctx,
			valueChannel(),
			icd.StaticInlineField{Pointer: "/mode", Value: "auto"},
			icd.StaticInlineField{Pointer: "/level", Value: 2.5},
			icd.StaticInlineField{Pointer: "/armed", Value: true},
			icd.StaticInlineField{
				Pointer: "/limits",
				Value:   map[string]any{"max": 10.0},
			},
		)
		Expect(f).To(HaveLen(5))
		Expect(*liftedVariant[icd.StringJSONField](f[1]).Default).To(Equal("auto"))
		Expect(
			*liftedVariant[icd.FloatJSONField](f[2]).Default,
		).To(Equal(payload.Float(2.5)))
		Expect(*liftedVariant[icd.BooleanJSONField](f[3]).Default).To(BeTrue())
		Expect(
			*liftedVariant[icd.JSONJSONField](f[4]).Default,
		).To(MatchJSON(`{"max":10}`))
		Expect(f[1].Base().Name).To(Equal("mode"))
		Expect(f[1].Base().Pointer).To(Equal("/mode"))
		Expect(write.Bindings).To(HaveLen(1))
	})

	It("Should give each generated field a generator default", func(ctx SpecContext) {
		format := payload.TimeFormatUnixNs
		f, write, _ := lift(
			ctx,
			valueChannel(),
			icd.GeneratedInlineField{
				Pointer:   "/id",
				Generator: payload.GeneratorTypeUUID,
			},
			icd.GeneratedInlineField{
				Pointer:    "/time",
				Generator:  payload.GeneratorTypeTimestamp,
				TimeFormat: &format,
			},
			icd.GeneratedInlineField{
				Pointer:   "/sent",
				Generator: payload.GeneratorTypeTimestamp,
			},
		)
		Expect(f[1].Variant).To(Equal(icd.UuidJSONField{
			JSONBaseField: f[1].Base(),
			Default:       &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
		}))
		now := &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}}
		Expect(f[2].Variant).To(Equal(icd.TimestampJSONField{
			JSONBaseField: f[2].Base(),
			TimeFormat:    payload.TimeFormatUnixNs,
			Default:       now,
		}))
		Expect(liftedVariant[icd.TimestampJSONField](f[3]).TimeFormat).
			To(Equal(payload.TimeFormatISO8601))
		Expect(write.Bindings).To(HaveLen(1))
	})

	It(
		"Should give a generated timestamp with a time format that is not valid ISO 8601",
		func(ctx SpecContext) {
			format := payload.TimeFormat("weekday")
			f, _, notes := lift(ctx, valueChannel(), icd.GeneratedInlineField{
				Pointer:    "/sent",
				Generator:  payload.GeneratorTypeTimestamp,
				TimeFormat: &format,
			})
			Expect(liftedVariant[icd.TimestampJSONField](f[1]).TimeFormat).
				To(Equal(payload.TimeFormatISO8601))
			Expect(notes).To(Equal([]note{{
				path:   "targets.0.fields.0",
				change: `time format "weekday" is not valid and became iso8601`,
			}}))
		},
	)

	It("Should drop a null field and note it", func(ctx SpecContext) {
		f, _, notes := lift(ctx, valueChannel(), nil)
		Expect(f).To(HaveLen(1))
		Expect(notes).To(Equal([]note{{
			path:   "targets.0.fields.0",
			change: "a null field was dropped",
		}}))
	})

	It("Should name the field of an empty pointer value and keep names unique",
		func(ctx SpecContext) {
			ch := valueChannel()
			ch.Pointer = ""
			f, _, _ := lift(
				ctx,
				ch,
				icd.StaticInlineField{Pointer: "/value", Value: 1.0},
			)
			Expect(f[0].Base().Name).To(Equal("value"))
			Expect(f[0].Base().Pointer).To(Equal(""))
			Expect(f[1].Base().Name).To(Equal("value (2)"))
		},
	)

	It("Should fail on a static value that is not JSON", func(ctx SpecContext) {
		Expect(icd.LiftWrite(
			ctx,
			icd.InlineWrite{
				Channel: valueChannel(),
				Fields: []icd.InlineWriteField{
					icd.StaticInlineField{Pointer: "/bad", Value: make(chan int)},
				},
			},
			found,
			"targets.0",
			func(string, string) {},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("unsupported type")),
		))
	})
})
