// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package serial_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/serial"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// binaryField returns an unsigned binary field of one byte at byte start, with
// fallback as its default.
func binaryField(
	name string,
	start uint16,
	fallback *payload.Unsigned,
) icd.BinaryField {
	return icd.BinaryField{Variant: icd.UnsignedBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: name, Scale: 1},
			StartBit:  start * 8,
			BitLength: 8,
			ByteOrder: icd.ByteOrderLittleEndian,
		},
		Default: fallback,
	}}
}

// missing is the key of a field that no message holds.
var missing = uuid.New()

var _ = Describe("Service", func() {
	var (
		svc   *serial.Service
		tasks *tasktestutil.Tasks
		// ic holds a Reading message of a level and a Command message of a valve and a
		// unit with a default.
		ic                    icd.ICD
		reading, command      icd.Message
		levelField, unitField icd.BinaryField
		valveField            icd.BinaryField
		level, valve          channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(serial.OpenService(ctx, serial.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
			ICD:     icdSvc,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
			Stores: svc.Stores(),
		}))
		levelField = binaryField("level", 0, nil)
		valveField = binaryField("valve", 0, nil)
		unitField = binaryField("unit", 1, new(payload.Unsigned(1)))
		reading = icd.Message{
			Key:  uuid.New(),
			Name: "Reading",
			Payload: icd.Payload{Variant: icd.BinaryPayload{
				Fields: []icd.BinaryField{levelField},
			}},
		}
		command = icd.Message{
			Key:  uuid.New(),
			Name: "Command",
			Payload: icd.Payload{Variant: icd.BinaryPayload{
				Fields: []icd.BinaryField{valveField, unitField},
			}},
		}
		ic = icd.ICD{
			Name:     "Pump",
			Messages: []icd.Message{reading, command},
			Framing:  &icd.Framing{Variant: icd.CobsFraming{}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return icdSvc.NewWriter(tx).Create(ctx, &ic)
		})).To(Succeed())
		idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
		level = createChannel(ctx, telem.Float64T, idx)
		valve = createChannel(ctx, telem.Uint8T, idx)
	})

	// set applies value at path of the ICD.
	set := func(ctx context.Context, path []string, value any) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return icdSvc.NewWriter(tx).Apply(ctx, ic.Key, icd.NewSetAction(
				icd.SetPayload{Path: path, Value: value},
			))
		})).To(Succeed())
	}

	Describe("Read", func() {
		// entry returns an entry that reads the level of reading into the level
		// channel, with poll when it is not nil.
		entry := func(poll map[string]any) map[string]any {
			e := map[string]any{
				"icd": ic.Key,
				"messages": []any{map[string]any{
					"message": reading.Key,
					"bindings": []any{map[string]any{
						"field": levelField.Base().Key, "channel": level.Key(),
					}},
				}},
			}
			if poll != nil {
				e["poll"] = poll
			}
			return e
		}

		It("Should save an entry that polls a binary message of its ICD", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"entries": []any{entry(map[string]any{"query": "AQ==", "rate": 5})},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["entries"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("icd", ic.Key.String()),
				HaveKeyWithValue("poll", SatisfyAll(
					HaveKeyWithValue("query", "AQ=="),
					HaveKeyWithValue("rate", BeEquivalentTo(5)),
				)),
			)))
		})

		It("Should change the hash when a message that an entry reads changes", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"entries": []any{entry(nil)},
			}))
			before := MustSucceed(svc.Read.Hash(ctx, db, key))
			set(ctx, []string{
				"messages", reading.Key.String(),
				"payload", "fields", levelField.Base().Key.String(),
				"start_bit",
			}, 8)
			Expect(svc.Read.Hash(ctx, db, key)).ToNot(Equal(before))
		})

		It("Should decode an entry that a save refuses", func(ctx SpecContext) {
			e := entry(nil)
			gone := uuid.New()
			e["messages"] = []any{map[string]any{"message": gone}}
			data := MustSucceed(svc.Read.Normalize(
				ctx, db, svc.Read.Version(), msgpack.EncodedJSON{"entries": []any{e}},
			))
			Expect(data["entries"]).To(HaveExactElements(HaveKeyWithValue(
				"messages",
				HaveExactElements(HaveKeyWithValue("message", gone.String())),
			)))
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "Pump", data)).Error().
				To(MatchError(ContainSubstring(
					"entries.0.messages.0.message: message " + gone.String() +
						" does not exist in ICD Pump",
				)))
		})

		DescribeTable("Should refuse an entry at the path of the problem",
			func(ctx SpecContext, e func() map[string]any, expected string) {
				Expect(svc.Read.Prepare(
					ctx, db, uuid.New(), "Pump", msgpack.EncodedJSON{
						"entries": []any{e()},
					},
				)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(expected),
				))
			},
			Entry("a poll with an empty query",
				func() map[string]any {
					return entry(map[string]any{"query": "", "rate": 1})
				},
				"entries.0.poll.query: must be at least 1 bytes long",
			),
			Entry("a poll with no rate",
				func() map[string]any {
					return entry(map[string]any{"query": "AQ==", "rate": 0})
				},
				"entries.0.poll.rate: must be greater than 0",
			),
			Entry("a binding to a field that the message lacks",
				func() map[string]any {
					e := entry(nil)
					e["messages"] = []any{map[string]any{
						"message": reading.Key,
						"bindings": []any{map[string]any{
							"field": missing, "channel": level.Key(),
						}},
					}}
					return e
				},
				"entries.0.messages.0.bindings.0.field: field "+missing.String()+
					" does not exist in message Reading",
			),
		)
	})

	Describe("Write", func() {
		// target returns a target that sends command with the valve field bound to
		// the valve channel, then each of bindings, and period when it is not nil.
		target := func(period any, bindings ...any) map[string]any {
			t := map[string]any{
				"icd": ic.Key,
				"message": map[string]any{
					"message": command.Key,
					"bindings": append([]any{map[string]any{
						"type":    "channel",
						"field":   valveField.Base().Key,
						"channel": valve.Key(),
					}}, bindings...),
				},
			}
			if period != nil {
				t["period"] = period
			}
			return t
		}

		It("Should save a target that sends a binary message on a period", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"targets": []any{target(100 * telem.Millisecond)},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["targets"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("icd", ic.Key.String()),
				HaveKeyWithValue("period", "100000000"),
			)))
		})

		It("Should change the hash when the message of a target changes", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"targets": []any{target(nil)},
			}))
			before := MustSucceed(svc.Write.Hash(ctx, db, key))
			set(ctx, []string{
				"messages", command.Key.String(),
				"payload", "fields", unitField.Base().Key.String(),
				"start_bit",
			}, 16)
			Expect(svc.Write.Hash(ctx, db, key)).ToNot(Equal(before))
		})

		DescribeTable("Should refuse a target at the path of the problem",
			func(ctx SpecContext, t func() map[string]any, expected string) {
				Expect(svc.Write.Prepare(
					ctx, db, uuid.New(), "Pump", msgpack.EncodedJSON{
						"targets": []any{t()},
					},
				)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(expected),
				))
			},
			Entry("a period of zero",
				func() map[string]any { return target(0) },
				"targets.0.period: must be greater than 0s",
			),
			Entry("a field with two bindings",
				func() map[string]any {
					return target(nil, map[string]any{
						"type":    "channel",
						"field":   valveField.Base().Key,
						"channel": valve.Key(),
					})
				},
				"targets.0.message.bindings.1.field: field valve has more than one "+
					"binding",
			),
		)
	})
})
