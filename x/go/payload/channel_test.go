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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Channel", func() {
	Describe("Field", func() {
		DescribeTable("should write a sample in the JSON type of the kind",
			func(c payload.Channel, dt telem.DataType, expected xjson.Type) {
				Expect(MustSucceed(c.Field(dt)).Type).To(Equal(expected))
			},
			Entry("float", payload.Channel{Kind: payload.KindFloat},
				telem.Float32T, xjson.Number),
			Entry("unsigned", payload.Channel{Kind: payload.KindUnsigned},
				telem.BooleanT, xjson.Number),
			Entry("string", payload.Channel{Kind: payload.KindString},
				telem.Int8T, xjson.String),
			Entry("boolean", payload.Channel{Kind: payload.KindBoolean},
				telem.Uint8T, xjson.Boolean),
			Entry("json", payload.Channel{Kind: payload.KindJSON},
				telem.JSONT, xjson.JSON),
			Entry("integer", payload.Channel{
				Kind:    payload.KindInteger,
				Integer: []payload.IntegerEntry{{Label: "NEG", Value: -1}},
			}, telem.Int8T, xjson.Number),
			Entry("unsigned with entries", payload.Channel{
				Kind:     payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{{Label: "ON", Value: 1}},
			}, telem.Uint8T, xjson.Number),
			Entry("string with integer text", payload.Channel{
				Kind:   payload.KindString,
				String: []payload.StringEntry{{Label: "NEG", Value: "-1"}},
			}, telem.Int8T, xjson.String),
			Entry("string with float text", payload.Channel{
				Kind:   payload.KindString,
				String: []payload.StringEntry{{Label: "HALF", Value: "0.5"}},
			}, telem.Float32T, xjson.String),
			Entry("string with text", payload.Channel{
				Kind:   payload.KindString,
				String: []payload.StringEntry{{Label: "ON", Value: "on"}},
			}, telem.StringT, xjson.String),
		)

		It("should parse the time format of a timestamp", func() {
			f := MustSucceed(payload.Channel{
				Kind:       payload.KindTimestamp,
				TimeFormat: payload.TimeFormatUnixMs,
			}.Field(telem.TimestampT))
			Expect(f.TimeFormat).To(Equal(xjson.UnixMillisecond))
		})

		It("should reject a kind the data type does not allow", func() {
			Expect(payload.Channel{Kind: payload.KindInteger}.Field(telem.Float64T)).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"a float64 channel cannot be written as integer",
				)),
			))
		})

		It("should reject a string entry that is not a value of the data type", func() {
			Expect(payload.Channel{
				Kind:   payload.KindString,
				String: []payload.StringEntry{{Label: "ON", Value: "on"}},
			}.Field(telem.Float64T)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`enum label "ON": "on" is not a float64`)),
			))
		})
	})

	Describe("Check", func() {
		It("should accept entries that fit the data type", func() {
			Expect(payload.Channel{
				Kind:    payload.KindBoolean,
				Boolean: []payload.BooleanEntry{{Label: "ON", Value: true}},
			}.Check(telem.Uint8T)).To(Succeed())
		})

		It("should reject an entry past the data type's range", func() {
			Expect(payload.Channel{
				Kind:     payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{{Label: "BIG", Value: 300}},
			}.Check(telem.Uint8T)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`enum label "BIG"`)),
			))
		})
	})

	Describe("Resolve", func() {
		It("should reject a float entry past the float32 range", func() {
			Expect(payload.Channel{
				Kind:  payload.KindFloat,
				Float: []payload.FloatEntry{{Label: "HUGE", Value: 1e39}},
			}.Resolve(telem.Float32T)).Error().To(MatchError(validate.ErrValidation))
		})

		It("should narrow float entries to a float32 channel", func() {
			c := MustSucceed(payload.Channel{
				Kind:  payload.KindFloat,
				Float: []payload.FloatEntry{{Label: "TENTH", Value: 0.1}},
			}.Resolve(telem.Float32T))
			Expect(float64(c.Float[0].Value)).To(Equal(float64(float32(0.1))))
		})

		It("should reject entries that collide after narrowing", func() {
			Expect(payload.Channel{
				Kind: payload.KindFloat,
				Float: []payload.FloatEntry{
					{Label: "A", Value: 0.1},
					{Label: "B", Value: 0.1000000001},
				},
			}.Resolve(telem.Float32T)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Validate", func() {
		DescribeTable("should reject two entries with one stored value",
			func(c payload.Channel) {
				Expect(c.Validate()).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"enum_values[1]: value is already labeled",
					)),
				))
			},
			Entry("two NaN floats", payload.Channel{
				Kind: payload.KindFloat,
				Float: []payload.FloatEntry{
					{Label: "A", Value: payload.Float(math.NaN())},
					{Label: "B", Value: payload.Float(math.NaN())},
				},
			}),
			Entry("zero and negative zero", payload.Channel{
				Kind: payload.KindFloat,
				Float: []payload.FloatEntry{
					{Label: "A", Value: 0},
					{Label: "B", Value: payload.Float(math.Copysign(0, -1))},
				},
			}),
			Entry("two strings", payload.Channel{
				Kind: payload.KindString,
				String: []payload.StringEntry{
					{Label: "A", Value: "on"},
					{Label: "B", Value: "on"},
				},
			}),
			Entry("two booleans", payload.Channel{
				Kind: payload.KindBoolean,
				Boolean: []payload.BooleanEntry{
					{Label: "A", Value: true},
					{Label: "B", Value: true},
				},
			}),
		)

		It("should accept two values that share a label", func() {
			Expect(payload.Channel{
				Kind: payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{
					{Label: "ON", Value: 1},
					{Label: "ON", Value: 2},
				},
			}.Validate()).To(Succeed())
		})
	})

	Describe("Reconcile", func() {
		DescribeTable("should convert the entries that stay exact",
			func(from payload.Channel, dt telem.DataType, expected payload.Channel) {
				c, _ := from.Reconcile(dt)
				Expect(c).To(Equal(expected))
			},
			Entry("integer to unsigned", payload.Channel{
				Kind: payload.KindInteger,
				Integer: []payload.IntegerEntry{
					{Label: "ONE", Value: 1},
					{Label: "NEG", Value: -1},
				},
			}, telem.Uint8T, payload.Channel{
				Kind:     payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{{Label: "ONE", Value: 1}},
			}),
			Entry("float to unsigned", payload.Channel{
				Kind: payload.KindFloat,
				Float: []payload.FloatEntry{
					{Label: "TWO", Value: 2},
					{Label: "HALF", Value: 0.5},
				},
			}, telem.Uint16T, payload.Channel{
				Kind:     payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{{Label: "TWO", Value: 2}},
			}),
			Entry("unsigned to an integer", payload.Channel{
				Kind: payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{
					{Label: "TWO", Value: 2},
					{Label: "HUGE", Value: math.MaxUint64},
				},
			}, telem.Int64T, payload.Channel{
				Kind:    payload.KindInteger,
				Integer: []payload.IntegerEntry{{Label: "TWO", Value: 2}},
			}),
			Entry("unsigned to a float", payload.Channel{
				Kind: payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{
					{Label: "TWO", Value: 2},
					{Label: "HUGE", Value: 1<<53 + 1},
				},
			}, telem.Float64T, payload.Channel{
				Kind:  payload.KindFloat,
				Float: []payload.FloatEntry{{Label: "TWO", Value: 2}},
			}),
			Entry("an integer to a float", payload.Channel{
				Kind: payload.KindInteger,
				Integer: []payload.IntegerEntry{
					{Label: "NEG", Value: -2},
					{Label: "HUGE", Value: -(1 << 53) - 1},
				},
			}, telem.Float64T, payload.Channel{
				Kind:  payload.KindFloat,
				Float: []payload.FloatEntry{{Label: "NEG", Value: -2}},
			}),
			Entry("a string to json, dropping its entries", payload.Channel{
				Kind:   payload.KindString,
				String: []payload.StringEntry{{Label: "ON", Value: "on"}},
			}, telem.JSONT, payload.Channel{Kind: payload.KindJSON}),
			Entry("a boolean to bytes, dropping its entries", payload.Channel{
				Kind:    payload.KindBoolean,
				Boolean: []payload.BooleanEntry{{Label: "ON", Value: true}},
			}, telem.BytesT, payload.Channel{Kind: payload.KindBytes}),
		)

		It("should drop string entries that do not fit the channel", func() {
			c, changes := payload.Channel{
				Kind: payload.KindString,
				String: []payload.StringEntry{
					{Label: "ONE", Value: "1"},
					{Label: "NEG", Value: "-1"},
				},
			}.Reconcile(telem.Uint8T)
			Expect(c.String).To(HaveExactElements(
				payload.StringEntry{Label: "ONE", Value: "1"},
			))
			Expect(changes).To(HaveLen(1))
		})

		It("should keep a kind the data type allows", func() {
			c := payload.Channel{Kind: payload.KindBoolean}
			Expect(c.Reconcile(telem.Float64T)).To(Equal(c))
		})

		It("should convert whole float entries to an integer channel", func() {
			c, changes := payload.Channel{
				Kind: payload.KindFloat,
				Float: []payload.FloatEntry{
					{Label: "ONE", Value: 1},
					{Label: "HALF", Value: 0.5},
				},
			}.Reconcile(telem.Int32T)
			Expect(c).To(Equal(payload.Channel{
				Kind:    payload.KindInteger,
				Integer: []payload.IntegerEntry{{Label: "ONE", Value: 1}},
			}))
			Expect(changes).To(HaveExactElements(
				"changed the kind from float to integer",
				ContainSubstring(`dropped enum entry "HALF"`),
			))
		})

		It(
			"should keep the entries of a number on a boolean channel as unsigned",
			func() {
				c, _ := payload.Channel{
					Kind: payload.KindFloat,
					Float: []payload.FloatEntry{
						{Label: "OFF", Value: 0},
						{Label: "ON", Value: 1},
					},
				}.Reconcile(telem.BooleanT)
				Expect(c).To(Equal(payload.Channel{
					Kind: payload.KindUnsigned,
					Unsigned: []payload.UnsignedEntry{
						{Label: "OFF", Value: 0},
						{Label: "ON", Value: 1},
					},
				}))
			},
		)

		It("should drop entries on a kind with no numeric entries", func() {
			c, changes := payload.Channel{
				Kind:  payload.KindFloat,
				Float: []payload.FloatEntry{{Label: "ON", Value: 1}},
			}.Reconcile(telem.TimestampT)
			Expect(c).To(Equal(payload.Channel{
				Kind:       payload.KindTimestamp,
				TimeFormat: payload.TimeFormatISO8601,
			}))
			Expect(changes).To(ContainElement("dropped the enum entries"))
		})

		It("should keep the first entry that fits the channel", func() {
			c, changes := payload.Channel{
				Kind: payload.KindUnsigned,
				Unsigned: []payload.UnsignedEntry{
					{Label: "A", Value: 1},
					{Label: "B", Value: 2},
				},
			}.Reconcile(telem.BooleanT)
			Expect(c.Unsigned).To(HaveExactElements(
				payload.UnsignedEntry{Label: "A", Value: 1},
			))
			Expect(changes).To(HaveLen(1))
		})

		It("should keep a known time format when it converts to a timestamp", func() {
			c, _ := payload.Channel{
				Kind:       payload.KindString,
				TimeFormat: payload.TimeFormatUnixMs,
			}.Reconcile(telem.TimestampT)
			Expect(c).To(Equal(payload.Channel{
				Kind:       payload.KindTimestamp,
				TimeFormat: payload.TimeFormatUnixMs,
			}))
		})

		It("should keep a channel of an unknown data type", func() {
			c := payload.Channel{Kind: payload.KindFloat}
			Expect(c.Reconcile(telem.UnknownT)).To(Equal(c))
		})
	})
})
