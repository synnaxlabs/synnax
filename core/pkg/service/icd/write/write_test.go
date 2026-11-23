// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package write_test

import (
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func jsonField(name string, v icd.JSONFieldVariant) icd.JSONField {
	f := icd.JSONField{Variant: v}
	f.SetBase(
		icd.JSONBaseField{Key: uuid.New(), Name: name, Scale: 1, Pointer: "/" + name},
	)
	return f
}

// withMax returns f with limit as its max.
func withMax(f icd.JSONField, limit float64) icd.JSONField {
	b := f.Base()
	b.Max = &limit
	f.SetBase(b)
	return f
}

func jsonMessage(name string, fields ...icd.JSONField) icd.Message {
	return icd.Message{
		Key:     uuid.New(),
		Name:    name,
		Payload: icd.Payload{Variant: icd.JSONPayload{Fields: fields}},
	}
}

var (
	temp     = jsonField("temp", icd.FloatJSONField{})
	limit    = jsonField("limit", icd.FloatJSONField{Default: new(payload.Float(1))})
	mode     = jsonField("mode", icd.StringJSONField{Default: new("auto")})
	id       = jsonField("id", icd.UuidJSONField{})
	setpoint = jsonMessage("Setpoint", temp, mode, limit)
	tagged   = jsonMessage("Tagged", id)
	binary   = icd.Message{
		Key:     uuid.New(),
		Name:    "Frame",
		Payload: icd.Payload{Variant: icd.BinaryPayload{}},
	}
	plant = icd.ICD{Name: "Plant", Messages: []icd.Message{setpoint, tagged, binary}}
)

const valueKey channel.Key = 2

func channelOf(key icd.FieldKey, ch channel.Key, dt *telem.DataType) icd.WriteBinding {
	return icd.WriteBinding{Variant: icd.ChannelWriteBinding{
		WriteBaseBinding: icd.WriteBaseBinding{Field: key},
		Channel:          ch,
		DataType:         dt,
	}}
}

func valueOf(key icd.FieldKey, v icd.WriteValueVariant) icd.WriteBinding {
	return icd.WriteBinding{Variant: icd.ValueWriteBinding{
		WriteBaseBinding: icd.WriteBaseBinding{Field: key},
		Value:            icd.WriteValue{Variant: v},
	}}
}

func bind(f icd.JSONField) icd.WriteBinding {
	return channelOf(f.Base().Key, valueKey, nil)
}

func fixed(f icd.JSONField, v icd.WriteValueVariant) icd.WriteBinding {
	return valueOf(f.Base().Key, v)
}

func writeOf(m icd.Message, bindings ...icd.WriteBinding) icd.WriteMessage {
	return icd.WriteMessage{Message: m.Key, Bindings: bindings}
}

var _ = Describe("NewDocuments", func() {
	held := map[channel.Key]channel.Channel{valueKey: {DataType: telem.Float64T}}
	documents := func(wm icd.WriteMessage, ic icd.ICD) (*write.Documents, error) {
		return write.NewDocuments(wm, ic, held, "target")
	}
	// send returns the one document that the Documents of wm in ic builds from command,
	// a frame of the channel with valueKey.
	send := func(wm icd.WriteMessage, ic icd.ICD, command telem.Series) string {
		GinkgoHelper()
		d := MustSucceed(documents(wm, ic))
		skips := driver.NewSkips("command", driver.SkipHold)
		var docs []string
		for doc := range d.From(frame.NewUnary(valueKey, command), skips) {
			docs = append(docs, string(doc))
		}
		Expect(docs).To(HaveLen(1))
		return docs[0]
	}

	It("Should send the default of each field with no binding", func() {
		Expect(send(writeOf(setpoint, bind(temp)), plant, telem.NewSeriesV(20.0))).
			To(MatchJSON(`{"temp":20,"mode":"auto","limit":1}`))
	})

	DescribeTable("Should send the fixed value of a binding",
		func(f icd.JSONField, v icd.WriteValueVariant, expected string) {
			m := jsonMessage("Fixed", temp, f)
			ic := icd.ICD{Name: "Plant", Messages: []icd.Message{m}}
			wm := writeOf(m, bind(temp), fixed(f, v))
			Expect(send(wm, ic, telem.NewSeriesV(20.0))).
				To(MatchJSON(`{"temp":20,"v":` + expected + `}`))
		},
		Entry("a float",
			jsonField("v", icd.FloatJSONField{}),
			icd.FloatWriteValue{Value: 2.5},
			`2.5`,
		),
		Entry("an integer",
			jsonField("v", icd.IntegerJSONField{}),
			icd.IntegerWriteValue{Value: -3},
			`-3`,
		),
		Entry("an unsigned integer",
			jsonField("v", icd.UnsignedJSONField{}),
			icd.UnsignedWriteValue{Value: 7},
			`7`,
		),
		Entry("a string in place of the field's default",
			jsonField("v", icd.StringJSONField{Default: new("auto")}),
			icd.StringWriteValue{Value: "manual"},
			`"manual"`,
		),
		Entry("a boolean",
			jsonField("v", icd.BooleanJSONField{}),
			icd.BooleanWriteValue{Value: true},
			`true`,
		),
		Entry(
			"a timestamp",
			jsonField(
				"v",
				icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixMs},
			),
			icd.TimestampWriteValue{Value: telem.TimeStamp(5 * telem.Millisecond)},
			`5`,
		),
		Entry("a JSON value",
			jsonField("v", icd.JSONJSONField{}),
			icd.JSONWriteValue{Value: `{"max":10}`},
			`{"max":10}`,
		),
		Entry("a UUID",
			jsonField("v", icd.UuidJSONField{}),
			icd.UuidWriteValue{Value: uuid.MustParse(
				"6f1c3a52-9a4e-4c1b-8d2e-0b7a1f3c5d90",
			)},
			`"6f1c3a52-9a4e-4c1b-8d2e-0b7a1f3c5d90"`,
		),
		Entry("bytes",
			jsonField("v", icd.BytesJSONField{}),
			icd.BytesWriteValue{Value: "AQI="},
			`"AQI="`,
		),
	)

	DescribeTable("Should refuse a fixed value that its field cannot send",
		func(v icd.JSONField, value icd.WriteValueVariant, expected string) {
			m := jsonMessage("Fixed", temp, v)
			ic := icd.ICD{Name: "Plant", Messages: []icd.Message{m}}
			wm := writeOf(m, bind(temp), fixed(v, value))
			Expect(documents(wm, ic)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(expected)),
			))
		},
		Entry("JSON text that is not valid",
			jsonField("v", icd.JSONJSONField{}),
			icd.JSONWriteValue{Value: "{"},
			"bindings.1.value: field v: must be valid JSON",
		),
		Entry("text that is not base64",
			jsonField("v", icd.BytesJSONField{}),
			icd.BytesWriteValue{Value: "AQI"},
			"bindings.1.value: field v: must be base64",
		),
		Entry("a float that is not finite",
			jsonField("v", icd.FloatJSONField{}),
			icd.FloatWriteValue{Value: payload.Float(math.Inf(1))},
			"bindings.1.value: field v: must be finite",
		),
		Entry("a float above the max",
			withMax(jsonField("v", icd.FloatJSONField{}), 100),
			icd.FloatWriteValue{Value: 120},
			"bindings.1.value: field v: 120 is above the max 100",
		),
	)

	It("Should accept a fixed value for a field that has no default", func() {
		wm := writeOf(
			setpoint,
			bind(limit),
			fixed(temp, icd.FloatWriteValue{Value: 5}),
		)
		Expect(send(wm, plant, telem.NewSeriesV(20.0))).
			To(MatchJSON(`{"temp":5,"mode":"auto","limit":20}`))
	})

	It("Should take the data type of a channel that exists over its binding's",
		func() {
			wm := writeOf(
				setpoint,
				channelOf(temp.Base().Key, valueKey, new(telem.Int32T)),
			)
			Expect(send(wm, plant, telem.NewSeriesV(20.5))).
				To(MatchJSON(`{"temp":20.5,"mode":"auto","limit":1}`))
		},
	)

	DescribeTable("Should refuse a write the message cannot send",
		func(wm icd.WriteMessage, expected string) {
			Expect(documents(wm, plant)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(expected)),
			))
		},
		Entry("a binding with no type",
			writeOf(setpoint, bind(temp), icd.WriteBinding{}),
			"bindings.1.type: required",
		),
		Entry("a value with no type",
			writeOf(setpoint, bind(temp), valueOf(limit.Base().Key, nil)),
			"bindings.1.value.type: required",
		),
		Entry("a message the ICD lacks",
			icd.WriteMessage{Message: uuid.New()},
			"does not exist in ICD Plant",
		),
		Entry("a message that is not JSON",
			writeOf(binary, bind(temp)),
			"message Frame is not a JSON message",
		),
		Entry("a message with no bindings",
			writeOf(setpoint),
			"message Setpoint binds no channels",
		),
		Entry("a message whose bindings all have values",
			writeOf(
				setpoint,
				fixed(temp, icd.FloatWriteValue{Value: 5}),
				fixed(limit, icd.FloatWriteValue{Value: 9}),
			),
			"message Setpoint binds no channels",
		),
		Entry("a field with no binding and no default",
			writeOf(setpoint, bind(limit)),
			"field temp: needs a channel or a default",
		),
		Entry("a value of another kind than its field",
			writeOf(setpoint, bind(limit), fixed(temp, icd.StringWriteValue{})),
			"bindings.1.value: field temp of kind float needs a value of its kind",
		),
		Entry(
			"a value for a field that does not exist",
			writeOf(
				setpoint,
				bind(limit),
				valueOf(uuid.Nil(), icd.FloatWriteValue{Value: 5}),
			),
			"bindings.1.field: field 00000000-0000-0000-0000-000000000000 does not exist",
		),
		Entry("a value for a field that a channel fills",
			writeOf(
				setpoint,
				bind(temp),
				bind(limit),
				fixed(temp, icd.FloatWriteValue{Value: 5}),
			),
			"bindings.2.field: field temp has more than one binding",
		),
	)
})
