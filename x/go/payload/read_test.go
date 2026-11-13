// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload_test

import (
	"encoding/json/jsontext"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Read", func() {
	nan := payload.Float(math.NaN())
	floatRead := func(fill *payload.Float, entries ...payload.FloatEntry) payload.Read {
		return payload.FromFloat(payload.FloatRead{EnumValues: entries, Fill: fill})
	}

	Describe("Reconcile", func() {
		DescribeTable(
			"should set the kind that the channel takes",
			func(r payload.Read, dt telem.DataType, expected payload.Read) {
				out, _ := r.Reconcile(dt, false)
				Expect(out).To(Equal(expected))
			},
			Entry("float to float", floatRead(nil), telem.Float32T,
				payload.Read{Kind: payload.KindFloat}),
			Entry("float to integer", floatRead(nil), telem.Int16T,
				payload.Read{Kind: payload.KindInteger}),
			Entry("float to unsigned", floatRead(nil), telem.Uint8T,
				payload.Read{Kind: payload.KindUnsigned}),
			Entry("float to string", floatRead(nil), telem.StringT,
				payload.Read{Kind: payload.KindString}),
			Entry("float to boolean", floatRead(nil), telem.BooleanT,
				payload.Read{Kind: payload.KindBoolean}),
			Entry("float to timestamp", floatRead(nil), telem.TimestampT,
				payload.Read{
					Kind: payload.KindTimestamp,
					Timestamp: payload.TimestampRead{
						TimeFormat: payload.TimeFormatISO8601,
					},
				}),
			Entry("string to json", payload.Read{Kind: payload.KindString},
				telem.JSONT, payload.Read{Kind: payload.KindJSON}),
			Entry("json to uuid", payload.Read{Kind: payload.KindJSON},
				telem.UUIDT, payload.Read{Kind: payload.KindUUID}),
			Entry("uuid to bytes", payload.Read{Kind: payload.KindUUID},
				telem.BytesT, payload.Read{Kind: payload.KindBytes}),
			Entry("timestamp keeps its format",
				payload.Read{
					Kind: payload.KindTimestamp,
					Timestamp: payload.TimestampRead{
						TimeFormat: payload.TimeFormatUnixMs,
					},
				},
				telem.TimestampT,
				payload.Read{
					Kind: payload.KindTimestamp,
					Timestamp: payload.TimestampRead{
						TimeFormat: payload.TimeFormatUnixMs,
					},
				}),
		)

		It("Should report no change for a field that agrees", func() {
			r := payload.FromInteger(payload.IntegerRead{
				EnumValues: []payload.IntegerEntry{{Label: "ON", Value: 1}},
			})
			out, changes := r.Reconcile(telem.Int8T, false)
			Expect(out).To(Equal(r))
			Expect(changes).To(BeEmpty())
		})

		It("Should clear the data type", func() {
			dt := telem.Int64T
			out, changes := payload.FromInteger(
				payload.IntegerRead{DataType: &dt},
			).Reconcile(telem.Int64T, false)
			Expect(out.Number.DataType).To(BeNil())
			Expect(changes).To(ConsistOf("data type int64 was cleared"))
		})

		It("Should convert the entries and the fill between numeric kinds", func() {
			fill := payload.Float(3)
			out, changes := floatRead(
				&fill,
				payload.FloatEntry{Label: "ON", Value: 1},
				payload.FloatEntry{Label: "OFF", Value: 0},
			).Reconcile(telem.Uint8T, false)
			Expect(out.Unsigned()).To(Equal(payload.UnsignedRead{
				EnumValues: []payload.UnsignedEntry{
					{Label: "ON", Value: 1},
					{Label: "OFF", Value: 0},
				},
				Fill: new(payload.Unsigned(3)),
			}))
			Expect(changes).To(ConsistOf("kind float became unsigned"))
		})

		It("Should drop the entries that the new kind does not hold", func() {
			out, changes := floatRead(
				nil,
				payload.FloatEntry{Label: "HALF", Value: 0.5},
				payload.FloatEntry{Label: "NEG", Value: -1},
				payload.FloatEntry{Label: "BIG", Value: 300},
				payload.FloatEntry{Label: "NAN", Value: nan},
				payload.FloatEntry{Label: "ON", Value: 1},
			).Reconcile(telem.Uint8T, false)
			Expect(out.Unsigned().EnumValues).To(Equal(
				[]payload.UnsignedEntry{{Label: "ON", Value: 1}},
			))
			Expect(changes).To(ConsistOf(
				"kind float became unsigned",
				`enum label "HALF" was dropped: 0.5 does not fit uint8`,
				`enum label "NEG" was dropped: -1 does not fit uint8`,
				`enum label "BIG" was dropped: 300 does not fit uint8`,
				`enum label "NAN" was dropped: NaN does not fit uint8`,
			))
		})

		It("Should drop a non-finite fill on an integer kind", func() {
			out, changes := floatRead(&nan).Reconcile(telem.Int32T, false)
			Expect(out.Number.Fill).To(BeNil())
			Expect(
				changes,
			).To(ContainElement("fill NaN was dropped: it does not fit int32"))
		})

		It("Should drop a fill past the range of float32", func() {
			fill := payload.Float(1e39)
			out, changes := floatRead(&fill).Reconcile(telem.Float32T, false)
			Expect(out.Number.Fill).To(BeNil())
			Expect(
				changes,
			).To(ConsistOf("fill 1e+39 was dropped: it does not fit float32"))
		})

		It("Should keep the first entry of a repeated label", func() {
			out, changes := payload.Read{
				Kind: payload.KindString,
				String: payload.StringRead{EnumValues: []payload.StringEntry{
					{Label: "A", Value: "first"},
					{Label: "A", Value: "second"},
				}},
			}.Reconcile(telem.StringT, false)
			Expect(out.String.EnumValues).To(Equal(
				[]payload.StringEntry{{Label: "A", Value: "first"}},
			))
			Expect(changes).To(ConsistOf(`enum label "A" was dropped as a repeat`))
		})

		It("Should drop an empty label on a boolean field", func() {
			out, changes := payload.Read{
				Kind: payload.KindBoolean,
				Boolean: payload.BooleanRead{EnumValues: []payload.BooleanEntry{
					{Label: "", Value: false},
					{Label: "ON", Value: true},
				}},
			}.Reconcile(telem.BooleanT, false)
			Expect(out.Boolean.EnumValues).To(Equal(
				[]payload.BooleanEntry{{Label: "ON", Value: true}},
			))
			Expect(changes).To(ConsistOf("an empty enum label was dropped"))
		})

		It("Should drop an empty label on a numeric field", func() {
			out, changes := floatRead(
				nil, payload.FloatEntry{Label: "", Value: 0},
			).Reconcile(telem.Float64T, false)
			Expect(out.Number.EnumValues).To(BeEmpty())
			Expect(changes).To(ConsistOf("an empty enum label was dropped"))
		})

		It("Should keep the entries of one value under two labels", func() {
			r := payload.Read{
				Kind: payload.KindBoolean,
				Boolean: payload.BooleanRead{EnumValues: []payload.BooleanEntry{
					{Label: "ON", Value: true},
					{Label: "HIGH", Value: true},
				}},
			}
			out, changes := r.Reconcile(telem.BooleanT, false)
			Expect(out).To(Equal(r))
			Expect(changes).To(BeEmpty())
		})

		It("Should drop the entries and the fill on a change to another kind", func() {
			fill := payload.Float(1)
			out, changes := floatRead(
				&fill, payload.FloatEntry{Label: "ON", Value: 1},
			).Reconcile(telem.BooleanT, false)
			Expect(out).To(Equal(payload.Read{Kind: payload.KindBoolean}))
			Expect(changes).To(ConsistOf(
				"kind float became boolean",
				"1 enum entries were dropped",
				"the fill was dropped",
			))
		})

		It("Should drop the fill of an index channel", func() {
			fill := telem.SecondTS
			out, changes := payload.Read{
				Kind: payload.KindTimestamp,
				Timestamp: payload.TimestampRead{
					TimeFormat: payload.TimeFormatUnixSec,
					Fill:       &fill,
				},
			}.Reconcile(telem.TimestampT, true)
			Expect(out.Timestamp).To(Equal(payload.TimestampRead{
				TimeFormat: payload.TimeFormatUnixSec,
			}))
			Expect(changes).To(ConsistOf(
				"the fill was dropped, because an index channel takes none",
			))
		})

		It("Should leave a field unchanged for an unknown data type", func() {
			r := floatRead(nil, payload.FloatEntry{Label: "A", Value: 1})
			out, changes := r.Reconcile(telem.UnknownT, false)
			Expect(out).To(Equal(r))
			Expect(changes).To(BeEmpty())
		})
	})

	Describe("Check", func() {
		It("Should accept a field that agrees with its channel", func() {
			fill := payload.Float(math.Inf(1))
			Expect(floatRead(
				&fill, payload.FloatEntry{Label: "ON", Value: 1},
			).Check(telem.Float32T, false)).To(Succeed())
		})

		DescribeTable(
			"should reject a field that does not agree with its channel",
			func(r payload.Read, dt telem.DataType, index bool, msg string) {
				Expect(r.Check(dt, index)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("a kind the channel does not take",
				floatRead(nil), telem.Int8T, false,
				"a field of kind float cannot read into a channel of data type int8"),
			Entry("an entry past the width",
				payload.FromInteger(payload.IntegerRead{
					EnumValues: []payload.IntegerEntry{{Label: "BIG", Value: 200}},
				}), telem.Int8T, false,
				`enum label "BIG": 200 does not fit int8`),
			Entry("a fill past the width",
				payload.FromUnsigned(payload.UnsignedRead{
					Fill: new(payload.Unsigned(70000)),
				}), telem.Uint16T, false,
				"fill: 70000 does not fit uint16"),
			Entry("a fill on an index channel",
				payload.Read{
					Kind: payload.KindTimestamp,
					Timestamp: payload.TimestampRead{
						TimeFormat: payload.TimeFormatISO8601,
						Fill:       new(telem.TimeStamp(0)),
					},
				}, telem.TimestampT, true,
				"an index channel cannot take a fill"),
		)
	})

	Describe("Validate", func() {
		It("Should reject a repeated label", func() {
			Expect(floatRead(
				nil,
				payload.FloatEntry{Label: "ON", Value: 1},
				payload.FloatEntry{Label: "ON", Value: 2},
			).Validate()).To(MatchError(ContainSubstring(`enum label "ON" is repeated`)))
		})

		It("Should reject an empty label on a numeric field", func() {
			Expect(floatRead(nil, payload.FloatEntry{Label: "", Value: 0}).Validate()).
				To(MatchError(ContainSubstring("cannot be empty on a float field")))
		})

		It("Should accept an empty label on a string field", func() {
			Expect(payload.Read{
				Kind: payload.KindString,
				String: payload.StringRead{
					EnumValues: []payload.StringEntry{{Label: "", Value: "none"}},
				},
			}.Validate()).To(Succeed())
		})

		It("Should accept a value under two labels", func() {
			Expect(floatRead(
				nil,
				payload.FloatEntry{Label: "ON", Value: 1},
				payload.FloatEntry{Label: "HIGH", Value: 1},
			).Validate()).To(Succeed())
		})
	})

	Describe("Field", func() {
		decode := func(r payload.Read, dt telem.DataType, raw string) []byte {
			GinkgoHelper()
			dec := MustSucceed(xjson.NewDecoder(MustSucceed(r.Field(dt))))
			return MustSucceed(dec.Append(nil, jsontext.Value(raw)))
		}

		It("Should read a label as its integer value", func() {
			r := payload.FromInteger(payload.IntegerRead{
				EnumValues: []payload.IntegerEntry{
					{Label: "MAX", Value: math.MaxInt64},
				},
			})
			Expect(decode(r, telem.Int64T, `"MAX"`)).To(Equal(
				telem.NewSeriesV[int64](math.MaxInt64).Data,
			))
		})

		It("Should read a label as its boolean value", func() {
			r := payload.Read{
				Kind: payload.KindBoolean,
				Boolean: payload.BooleanRead{
					EnumValues: []payload.BooleanEntry{{Label: "ON", Value: true}},
				},
			}
			Expect(decode(r, telem.BooleanT, `"ON"`)).To(Equal([]byte{1}))
		})

		It("Should read a timestamp in its format", func() {
			r := payload.Read{
				Kind: payload.KindTimestamp,
				Timestamp: payload.TimestampRead{
					TimeFormat: payload.TimeFormatUnixSec,
				},
			}
			Expect(decode(r, telem.TimestampT, `2`)).To(Equal(
				telem.NewSeriesV(2 * telem.SecondTS).Data,
			))
		})
	})

	Describe("Fill", func() {
		It("Should return no fill for a required field", func() {
			_, ok := floatRead(nil).Fill(telem.Float64T)
			Expect(ok).To(BeFalse())
		})

		DescribeTable(
			"should return the fill as one sample",
			func(r payload.Read, dt telem.DataType, expected []byte) {
				Expect(MustBeOk(r.Fill(dt))).To(Equal(expected))
			},
			Entry("float", floatRead(new(payload.Float(1.5))), telem.Float32T,
				telem.NewSeriesV[float32](1.5).Data),
			Entry("string",
				payload.Read{
					Kind:   payload.KindString,
					String: payload.StringRead{Fill: new("off")},
				},
				telem.StringT, []byte{3, 0, 0, 0, 'o', 'f', 'f'}),
			Entry("boolean",
				payload.Read{
					Kind:    payload.KindBoolean,
					Boolean: payload.BooleanRead{Fill: new(true)},
				},
				telem.BooleanT, []byte{1}),
			Entry("timestamp",
				payload.Read{
					Kind: payload.KindTimestamp,
					Timestamp: payload.TimestampRead{
						Fill: new(telem.SecondTS),
					},
				},
				telem.TimestampT, telem.NewSeriesV(telem.SecondTS).Data),
		)
	})

	DescribeTable(
		"Absent",
		func(raw jsontext.Value, dt telem.DataType, absent bool) {
			Expect(payload.Absent(raw, dt)).To(Equal(absent))
		},
		Entry("no value", jsontext.Value(nil), telem.Float64T, true),
		Entry("null", jsontext.Value("null"), telem.StringT, true),
		Entry("an empty string on a number", jsontext.Value(`""`), telem.Int8T, true),
		Entry(
			"an empty string on a string",
			jsontext.Value(`""`),
			telem.StringT,
			false,
		),
		Entry("zero", jsontext.Value("0"), telem.Float64T, false),
		Entry("null on JSON", jsontext.Value("null"), telem.JSONT, false),
		Entry("an empty string on JSON", jsontext.Value(`""`), telem.JSONT, false),
		Entry("an empty string on bytes", jsontext.Value(`""`), telem.BytesT, false),
		Entry("null on bytes", jsontext.Value("null"), telem.BytesT, true),
		Entry("an empty string on a UUID", jsontext.Value(`""`), telem.UUIDT, true),
	)
})
