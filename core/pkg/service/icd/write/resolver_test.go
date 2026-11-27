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
	"context"
	"math"
	"slices"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const modeKey channel.Key = 3

// missingField is the key of a field that no message holds.
var missingField = uuid.New()

var (
	gain = icd.BinaryField{Variant: icd.FloatBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: "gain", Scale: 1},
			BitLength: 32,
		},
	}}
	unit = icd.BinaryField{Variant: icd.UnsignedBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: "unit", Scale: 1},
			StartBit:  32,
			BitLength: 8,
		},
		Default: new(payload.Unsigned(1)),
	}}
	stuck = icd.BinaryField{Variant: icd.UnsignedBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: "stuck"},
			StartBit:  40,
			BitLength: 8,
		},
		Default: new(payload.Unsigned(0)),
	}}
	counts = icd.Message{
		Key:  uuid.New(),
		Name: "Counts",
		Payload: icd.Payload{Variant: icd.BinaryPayload{
			Fields: []icd.BinaryField{gain, unit, stuck},
		}},
	}
	volts = textField("volts", nil)
	trim  = textField("trim", new(payload.Float(0)))
	line  = icd.Message{
		Key:  uuid.New(),
		Name: "Line",
		Payload: icd.Payload{Variant: icd.TextPayload{
			Delimiter: ",",
			Fields:    []icd.TextField{volts, trim},
		}},
	}
)

// actuator sends an opening or a rate, as its selector picks, and names itself with
// id.
var (
	actuatorID = icd.BinaryField{Variant: icd.UnsignedBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: "id", Scale: 1},
			BitLength: 8,
		},
	}}
	selector = icd.BinaryField{Variant: icd.UnsignedBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: "selector", Scale: 1},
			StartBit:  8,
			BitLength: 8,
		},
	}}
	opening = icd.BinaryField{Variant: icd.UnsignedBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField:       icd.BaseField{Key: uuid.New(), Name: "opening", Scale: 1},
			StartBit:        16,
			BitLength:       8,
			Multiplexor:     new(selector.Base().Key),
			MultiplexValues: []int32{0},
		},
	}}
	rate = icd.BinaryField{Variant: icd.FloatBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{
				Key: uuid.New(), Name: "rate", Scale: 2, Max: new(100.0),
			},
			StartBit:        16,
			BitLength:       32,
			Multiplexor:     new(selector.Base().Key),
			MultiplexValues: []int32{1},
		},
	}}
	actuator = icd.Message{
		Key:  uuid.New(),
		Name: "Actuator",
		Payload: icd.Payload{Variant: icd.BinaryPayload{
			Fields: []icd.BinaryField{actuatorID, selector, opening, rate},
			Identifier: &icd.Identifier{Variant: icd.FieldIdentifier{
				Field: actuatorID.Base().Key, Value: 7,
			}},
		}},
	}
)

// unheldKey is the key of a channel that the channel lookup does not find.
const unheldKey channel.Key = 9

func textField(name string, fallback *payload.Float) icd.TextField {
	return icd.TextField{Variant: icd.DelimitedTextField{
		TextBaseField: icd.TextBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: name, Scale: 1},
			Default:   fallback,
		},
	}}
}

// framed returns w as a write that frames its message on a byte stream.
func framed(w write.Write) write.Write {
	w.Framed = true
	return w
}

// writeAt returns the write of wm of the ICD with key, at targets.i of a config.
func writeAt(i int, key icd.Key, wm icd.WriteMessage) write.Write {
	return write.Write{
		Path:    []string{"targets", strconv.Itoa(i)},
		ICD:     key,
		Message: wm,
	}
}

var _ = Describe("Resolver", func() {
	var (
		resolver write.Resolver
		// held holds the channels that the channel lookup finds.
		held map[channel.Key]icdv0.Channel
		// asked holds the keys of each channel lookup.
		asked [][]channel.Key
	)
	BeforeEach(func() {
		held = map[channel.Key]icdv0.Channel{
			valueKey: {Name: "value", DataType: telem.Float64T},
		}
		asked = nil
		resolver = write.NewResolver(icdSvc, func(
			_ context.Context,
			_ gorp.Snapshot,
			keys []channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			asked = append(asked, keys)
			return held, nil
		})
	})

	Describe("Check", func() {
		given := plant
		given.Key = uuid.New()
		given.Messages = append(slices.Clone(plant.Messages), counts, line, actuator)
		// check checks writes with given as the only ICD that the caller holds.
		check := func(ctx context.Context, writes ...write.Write) error {
			GinkgoHelper()
			return resolver.Check(
				ctx,
				db,
				writes,
				map[icd.Key]icd.ICD{given.Key: given},
			)
		}

		It("Should accept writes whose messages build", func(ctx SpecContext) {
			Expect(check(
				ctx,
				writeAt(0, given.Key, writeOf(setpoint, bind(temp))),
				writeAt(
					1,
					given.Key,
					writeOf(tagged, channelOf(id.Base().Key, modeKey, nil)),
				),
			)).To(Succeed())
		})

		It("Should return the error of the first write that does not build, at the "+
			"path of its message", func(ctx SpecContext) {
			Expect(check(
				ctx,
				writeAt(0, given.Key, writeOf(setpoint, bind(temp))),
				writeAt(1, given.Key, writeOf(tagged)),
				writeAt(2, given.Key, writeOf(binary)),
			)).To(MatchError("targets.1.message: message Tagged binds no channels"))
		})

		Describe("A field with a zero scale", func() {
			flat := withZeroScale(
				jsonField("flat", icd.FloatJSONField{Default: new(payload.Float(3))}),
			)
			valve := jsonMessage("Valve", temp, flat)
			valves := icd.ICD{
				Key: uuid.New(), Name: "Valves", Messages: []icd.Message{valve},
			}
			checkValve := func(ctx context.Context, wm icd.WriteMessage) error {
				GinkgoHelper()
				return resolver.Check(
					ctx,
					db,
					[]write.Write{writeAt(0, valves.Key, wm)},
					map[icd.Key]icd.ICD{valves.Key: valves},
				)
			}

			It("Should refuse a write that binds it", func(ctx SpecContext) {
				wm := writeOf(valve, bind(temp), bind(flat))
				Expect(checkValve(ctx, wm)).To(MatchError(
					"targets.0.message: message Valve: field flat: " +
						"scale must be non-zero: validation error",
				))
			})

			It("Should accept a write that leaves it to its default", func(
				ctx SpecContext,
			) {
				Expect(checkValve(ctx, writeOf(valve, bind(temp)))).To(Succeed())
			})
		})

		It("Should take the data type of each channel that the lookup finds", func(
			ctx SpecContext,
		) {
			w := writeAt(0, given.Key, writeOf(
				setpoint, bind(temp), channelOf(mode.Base().Key, modeKey, nil),
			))
			Expect(check(ctx, w)).To(Succeed())
			held[modeKey] = icdv0.Channel{Name: "mode", DataType: telem.Float64T}
			Expect(check(ctx, w)).To(MatchError(
				"targets.0.message: message Setpoint: field mode: cannot write float64 " +
					"channels to string fields: validation error",
			))
		})

		// scaled returns f with scale and offset.
		scaled := func(f icd.JSONField, scale, offset float64) icd.JSONField {
			b := f.Base()
			b.Scale, b.Offset = scale, offset
			f.SetBase(b)
			return f
		}

		DescribeTable("Should give a binding whose channel the lookup does not find "+
			"its own data type, or else one from its field",
			func(
				ctx SpecContext,
				f icd.JSONField,
				dt *telem.DataType,
				matcher types.GomegaMatcher,
			) {
				m := jsonMessage("Command", f)
				ic := icd.ICD{
					Key:      uuid.New(),
					Name:     "Plant",
					Messages: []icd.Message{m},
				}
				w := writeAt(
					0,
					ic.Key,
					writeOf(m, channelOf(f.Base().Key, modeKey, dt)),
				)
				Expect(resolver.Check(ctx, db, []write.Write{w}, map[icd.Key]icd.ICD{
					ic.Key: ic,
				})).To(matcher)
			},
			Entry(
				"its own data type",
				jsonField("mode", icd.StringJSONField{}),
				new(telem.Int32T),
				MatchError(
					"targets.0.message: message Command: field mode: cannot write "+
						"int32 channels to string fields: validation error",
				),
			),
			Entry("the default of its field's kind",
				jsonField("id", icd.UuidJSONField{}),
				nil,
				Succeed(),
			),
			Entry("float64 for a scaled field",
				scaled(jsonField("raw", icd.IntegerJSONField{}), 0.5, 0),
				nil,
				Succeed(),
			),
			Entry("float64 for an offset field",
				scaled(jsonField("raw", icd.IntegerJSONField{}), 1, 4),
				nil,
				Succeed(),
			),
		)

		It("Should accept framed writes of binary and text messages", func(
			ctx SpecContext,
		) {
			Expect(check(
				ctx,
				framed(writeAt(0, given.Key, writeOf(
					counts,
					channelOf(gain.Base().Key, valueKey, nil),
					valueOf(unit.Base().Key, icd.UnsignedWriteValue{Value: 2}),
				))),
				framed(writeAt(1, given.Key, writeOf(
					line, channelOf(volts.Base().Key, valueKey, nil),
				))),
			)).To(Succeed())
		})

		It("Should not ask a binding of the identifier or of a field that a fixed "+
			"multiplexor never selects", func(ctx SpecContext) {
			Expect(check(ctx, framed(writeAt(0, given.Key, writeOf(
				actuator,
				channelOf(opening.Base().Key, valueKey, nil),
				valueOf(selector.Base().Key, icd.UnsignedWriteValue{Value: 0}),
			))))).To(Succeed())
		})

		It("Should take the data type of a framed binding from the channel that the "+
			"lookup finds", func(ctx SpecContext) {
			held[modeKey] = icdv0.Channel{Name: "mode", DataType: telem.StringT}
			Expect(check(ctx, framed(writeAt(0, given.Key, writeOf(
				counts, channelOf(gain.Base().Key, modeKey, new(telem.Float64T)),
			))))).To(MatchError(
				"targets.0.message.bindings.0: field gain: cannot write string " +
					"channels to float fields: validation error",
			))
		})

		It("Should refuse a write of a binary message that is not framed", func(
			ctx SpecContext,
		) {
			w := writeAt(0, given.Key, writeOf(
				counts, channelOf(gain.Base().Key, valueKey, nil),
			))
			Expect(check(ctx, w)).To(MatchError(
				"targets.0.message: message Counts is not a JSON message",
			))
		})

		It("Should build the encoder of a framed write of a JSON message", func(
			ctx SpecContext,
		) {
			held[modeKey] = icdv0.Channel{Name: "mode", DataType: telem.Float64T}
			w := framed(writeAt(0, given.Key, writeOf(
				setpoint, bind(temp), channelOf(mode.Base().Key, modeKey, nil),
			)))
			Expect(check(ctx, w)).To(MatchError(
				"targets.0.message: message Setpoint: field mode: cannot write float64 " +
					"channels to string fields: validation error",
			))
		})

		DescribeTable("Should refuse a framed write of a binary or text message at "+
			"the path of the problem",
			func(ctx SpecContext, wm icd.WriteMessage, expected string) {
				Expect(check(ctx, framed(writeAt(0, given.Key, wm)))).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(expected),
				))
			},
			Entry("no channel binding",
				writeOf(counts, valueOf(unit.Base().Key, icd.UnsignedWriteValue{})),
				"targets.0.message: message Counts binds no channels",
			),
			Entry("a binding to a field that the message lacks",
				writeOf(counts, channelOf(missingField, valueKey, nil)),
				"targets.0.message.bindings.0.field: field "+missingField.String()+
					" does not exist",
			),
			Entry("a field with two bindings",
				writeOf(
					counts,
					channelOf(gain.Base().Key, valueKey, nil),
					channelOf(gain.Base().Key, modeKey, nil),
				),
				"targets.0.message.bindings.1.field: field gain has more than one "+
					"binding",
			),
			Entry("a value of another kind than its field",
				writeOf(
					counts,
					channelOf(gain.Base().Key, valueKey, nil),
					valueOf(unit.Base().Key, icd.FloatWriteValue{Value: 2}),
				),
				"targets.0.message.bindings.1.value: field unit of kind unsigned "+
					"needs a value of its kind",
			),
			Entry("a field with no binding and no default",
				writeOf(line, channelOf(trim.Base().Key, valueKey, nil)),
				"targets.0.message: field volts has no binding and no default",
			),
			Entry("a channel on a field with a zero scale",
				writeOf(
					counts,
					channelOf(gain.Base().Key, valueKey, nil),
					channelOf(stuck.Base().Key, valueKey, nil),
				),
				"targets.0.message.bindings.1: field stuck: scale must be non-zero",
			),
			Entry(
				"a channel data type that cannot write its field",
				writeOf(
					line,
					channelOf(volts.Base().Key, unheldKey, new(telem.StringT)),
				),
				"targets.0.message.bindings.0: field volts: cannot write string "+
					"channels to float fields: validation error",
			),
			Entry("a channel that is not a float on a scaled field",
				writeOf(
					actuator,
					valueOf(selector.Base().Key, icd.UnsignedWriteValue{Value: 1}),
					channelOf(rate.Base().Key, unheldKey, new(telem.Int32T)),
				),
				"targets.0.message.bindings.1: field rate: a scaled field requires "+
					"a float channel, not int32: validation error",
			),
			Entry("a value that the bits of its field cannot hold",
				writeOf(
					counts,
					channelOf(gain.Base().Key, valueKey, nil),
					valueOf(unit.Base().Key, icd.UnsignedWriteValue{Value: 300}),
				),
				"targets.0.message.bindings.1.value: field unit: 300 does not fit "+
					"the field: validation error",
			),
			Entry("an infinite value",
				writeOf(
					line,
					channelOf(volts.Base().Key, valueKey, nil),
					valueOf(trim.Base().Key, icd.FloatWriteValue{
						Value: payload.Float(math.Inf(1)),
					}),
				),
				"targets.0.message.bindings.1.value: field trim: cannot be "+
					"infinite: validation error",
			),
			Entry("a value that decodes outside the min and max of its field",
				writeOf(
					actuator,
					channelOf(selector.Base().Key, valueKey, nil),
					valueOf(rate.Base().Key, icd.FloatWriteValue{Value: 60}),
				),
				"targets.0.message.bindings.1.value: field rate: 60 decodes to 120, "+
					"which is above the max 100: validation error",
			),
			Entry("a binding to the field that holds the identifier",
				writeOf(actuator, channelOf(actuatorID.Base().Key, valueKey, nil)),
				"targets.0.message.bindings.0.field: field id holds the identifier, "+
					"so it takes no binding",
			),
			Entry("a channel on a field that a fixed multiplexor never selects",
				writeOf(
					actuator,
					valueOf(selector.Base().Key, icd.UnsignedWriteValue{Value: 1}),
					channelOf(opening.Base().Key, valueKey, nil),
				),
				"targets.0.message.bindings.1: field opening takes a channel, but "+
					"multiplexor selector never selects it",
			),
			Entry("no binding or default for a field that a multiplexor set by a "+
				"channel can select",
				writeOf(
					actuator,
					channelOf(selector.Base().Key, valueKey, nil),
					channelOf(opening.Base().Key, valueKey, nil),
				),
				"targets.0.message: field rate has no binding and no default",
			),
		)

		It("Should retrieve an ICD that the given ICDs lack", func(ctx SpecContext) {
			stored := icd.ICD{Name: "Stored", Messages: []icd.Message{tagged}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &stored)
			})).To(Succeed())
			Expect(check(ctx, writeAt(0, stored.Key, writeOf(setpoint, bind(temp))))).
				To(MatchError(
					"targets.0.message: message " + setpoint.Key.String() +
						" does not exist in ICD Stored",
				))
		})

		It("Should skip a write whose ICD does not exist", func(ctx SpecContext) {
			Expect(check(ctx, writeAt(0, uuid.New(), writeOf(tagged)))).To(Succeed())
		})

		It("Should skip a write with no bindings of a message with no fields", func(
			ctx SpecContext,
		) {
			Expect(check(ctx, writeAt(0, given.Key, writeOf(empty)))).To(Succeed())
		})

		It("Should refuse a write with no bindings of a message with fields", func(
			ctx SpecContext,
		) {
			Expect(check(ctx, writeAt(0, given.Key, writeOf(tagged)))).
				To(MatchError("targets.0.message: message Tagged binds no channels"))
		})

		It("Should refuse a write that binds a field of a message with no fields", func(
			ctx SpecContext,
		) {
			key := temp.Base().Key.String()
			Expect(check(ctx, writeAt(0, given.Key, writeOf(empty, bind(temp))))).
				To(MatchError("targets.0.message: message Empty: field " + key +
					" does not exist: validation error"))
		})

		It("Should look up each channel that a binding names once", func(
			ctx SpecContext,
		) {
			Expect(check(
				ctx,
				writeAt(0, given.Key, writeOf(setpoint, bind(temp))),
				writeAt(1, given.Key, writeOf(
					setpoint,
					bind(temp),
					channelOf(mode.Base().Key, modeKey, nil),
					fixed(limit, icd.FloatWriteValue{Value: 2}),
				)),
				writeAt(2, uuid.New(), writeOf(setpoint, bind(temp))),
			)).To(Succeed())
			Expect(asked).To(HaveExactElements(ConsistOf(valueKey, modeKey)))
		})

		It("Should not look up channels when no binding names one", func(
			ctx SpecContext,
		) {
			Expect(check(ctx, writeAt(0, uuid.New(), writeOf(tagged)))).To(Succeed())
			Expect(asked).To(BeEmpty())
		})

		It("Should return the error of the channel lookup", func(ctx SpecContext) {
			resolver = write.NewResolver(icdSvc, func(
				context.Context,
				gorp.Snapshot,
				[]channel.Key,
			) (map[channel.Key]icdv0.Channel, error) {
				return nil, errors.New("no channels")
			})
			Expect(check(ctx, writeAt(0, given.Key, writeOf(setpoint, bind(temp))))).
				To(MatchError("no channels"))
		})
	})

	Describe("Hash", func() {
		var stored icd.ICD
		BeforeEach(func(ctx SpecContext) {
			stored = icd.ICD{Name: "Plant", Messages: []icd.Message{setpoint, tagged}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &stored)
			})).To(Succeed())
		})
		sum := func(ctx context.Context, writes ...write.Write) string {
			GinkgoHelper()
			h := content.NewHash()
			Expect(resolver.Hash(ctx, db, &h, writes)).To(Succeed())
			return h.Sum()
		}
		// setPointer sets the pointer of f, a field of m, to pointer.
		setPointer := func(
			ctx context.Context,
			m icd.Message,
			f icd.JSONField,
			pointer string,
		) {
			GinkgoHelper()
			path := []string{
				"messages", m.Key.String(),
				"payload", "fields", f.Base().Key.String(),
				"pointer",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Apply(ctx, stored.Key, icd.NewSetAction(
					icd.SetPayload{Path: path, Value: pointer},
				))
			})).To(Succeed())
		}

		// reframed creates a copy of stored, under a new key, that frames with framing.
		reframed := func(ctx context.Context, framing icd.Framing) icd.Key {
			GinkgoHelper()
			other := stored
			other.Key = uuid.New()
			other.Framing = &framing
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			return other.Key
		}

		It("Should cover the framing of the ICD of a framed write", func(
			ctx SpecContext,
		) {
			wm := writeOf(setpoint, bind(temp))
			slip := reframed(ctx, icd.Framing{Variant: icd.SlipFraming{}})
			Expect(sum(ctx, framed(writeAt(0, slip, wm)))).
				ToNot(Equal(sum(ctx, framed(writeAt(0, stored.Key, wm)))))
		})

		It("Should not cover the framing of the ICD of a write", func(
			ctx SpecContext,
		) {
			wm := writeOf(setpoint, bind(temp))
			slip := reframed(ctx, icd.Framing{Variant: icd.SlipFraming{}})
			Expect(sum(ctx, writeAt(0, slip, wm))).
				To(Equal(sum(ctx, writeAt(0, stored.Key, wm))))
		})

		It("Should give the same sum for the same writes", func(ctx SpecContext) {
			w := writeAt(0, stored.Key, writeOf(setpoint, bind(temp)))
			Expect(sum(ctx, w)).To(Equal(sum(ctx, w)))
		})

		It("Should change when the payload of a message that a write sends changes",
			func(ctx SpecContext) {
				w := writeAt(0, stored.Key, writeOf(setpoint, bind(temp)))
				before := sum(ctx, w)
				setPointer(ctx, setpoint, temp, "/temperature")
				Expect(sum(ctx, w)).ToNot(Equal(before))
			},
		)

		It("Should not change when a message that no write sends changes", func(
			ctx SpecContext,
		) {
			w := writeAt(0, stored.Key, writeOf(setpoint, bind(temp)))
			before := sum(ctx, w)
			setPointer(ctx, tagged, id, "/uuid")
			Expect(sum(ctx, w)).To(Equal(before))
		})

		It("Should mark a write of a message that the ICD lacks", func(
			ctx SpecContext,
		) {
			lacked := writeAt(0, stored.Key, writeOf(binary))
			Expect(sum(ctx, lacked)).ToNot(Equal(sum(ctx)))
		})

		It("Should fail for an ICD that does not exist", func(ctx SpecContext) {
			h := content.NewHash()
			missing := writeAt(0, uuid.New(), writeOf(setpoint, bind(temp)))
			Expect(resolver.Hash(ctx, db, &h, []write.Write{missing})).
				To(MatchError(query.ErrNotFound))
		})
	})
})

var _ = Describe("Targets", func() {
	It("Should give the framed write of each target at its path", func() {
		first, second := uuid.New(), uuid.New()
		wm := writeOf(counts, channelOf(gain.Base().Key, valueKey, nil))
		Expect(write.Targets([]icd.WriteTarget{
			{ICD: first, Message: wm},
			{ICD: second, Message: wm},
		})).To(Equal([]write.Write{
			{Path: []string{"targets", "0"}, ICD: first, Message: wm, Framed: true},
			{Path: []string{"targets", "1"}, ICD: second, Message: wm, Framed: true},
		}))
	})
})
