// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package read_test

import (
	"context"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/telem"
)

// payloadOf returns the payload of m, which is a P.
func payloadOf[P icd.PayloadVariant](m icd.Message) P {
	GinkgoHelper()
	p, ok := m.Payload.Variant.(P)
	Expect(ok).To(BeTrue())
	return p
}

// edited returns ic with each of ms in place of the message with its key.
func edited(ic icd.ICD, ms ...icd.Message) icd.ICD {
	ic.Messages = slices.Clone(ic.Messages)
	for _, m := range ms {
		i := slices.IndexFunc(ic.Messages, func(o icd.Message) bool {
			return o.Key == m.Key
		})
		ic.Messages[i] = m
	}
	return ic
}

// without returns ic with no message with the key of m.
func without(ic icd.ICD, m icd.Message) icd.ICD {
	ic.Messages = slices.DeleteFunc(
		slices.Clone(ic.Messages),
		func(o icd.Message) bool { return o.Key == m.Key },
	)
	return ic
}

// withFields returns m with a JSON payload of fields.
func withFields(m icd.Message, fields ...icd.JSONField) icd.Message {
	m.Payload = icd.Payload{Variant: icd.JSONPayload{Fields: fields}}
	return m
}

// withBinaryFields returns m with a binary payload of fields.
func withBinaryFields(m icd.Message, fields ...icd.BinaryField) icd.Message {
	m.Payload = icd.Payload{Variant: icd.BinaryPayload{Fields: fields}}
	return m
}

// as returns a field with the base of f and the variant v.
func as(f icd.JSONField, v icd.JSONFieldVariant) icd.JSONField {
	out := icd.JSONField{Variant: v}
	out.SetBase(f.Base())
	return out
}

// jsonRebased returns f with the base that edit makes of its base.
func jsonRebased(f icd.JSONField, edit func(*icd.JSONBaseField)) icd.JSONField {
	b := f.Base()
	edit(&b)
	f.SetBase(b)
	return f
}

// binaryRebased returns f with the base that edit makes of its base.
func binaryRebased(
	f icd.BinaryField,
	edit func(*icd.BinaryBaseField),
) icd.BinaryField {
	b := f.Base()
	edit(&b)
	f.SetBase(b)
	return f
}

func typed(b icd.ReadBinding, dt telem.DataType) icd.ReadBinding {
	b.DataType = dt
	return b
}

func lossy(description string) icd.Change {
	return icd.Change{Description: description, Lossy: true}
}

// wordKey is a data channel of type int16.
const wordKey channel.Key = 10

var _ = Describe("Repair", func() {
	channels := lo.Assign(held, map[channel.Key]icdv0.Channel{
		wordKey: {Name: "word", DataType: telem.Int16T, IndexKey: timeKey},
	})
	var resolver read.Resolver
	BeforeEach(func() {
		resolver = read.NewResolver(icdSvc, func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			return channels, nil
		})
	})
	// repairFrom repairs rd, which the save check accepts against prev, for next, and
	// expects the save check to accept the repaired read against next.
	repairFrom := func(
		ctx context.Context,
		rd read.Read,
		prev, next icd.ICD,
	) ([]icd.ReadMessage, []icd.Change) {
		GinkgoHelper()
		before := map[icd.Key]icd.ICD{prev.Key: prev}
		Expect(resolver.Check(ctx, db, []read.Read{rd}, before)).To(Succeed())
		messages, changes := read.Repair(rd, prev, next, channels)
		rd.Messages = messages
		after := map[icd.Key]icd.ICD{next.Key: next}
		Expect(resolver.Check(ctx, db, []read.Read{rd}, after)).To(Succeed())
		return messages, changes
	}
	repair := func(
		ctx context.Context,
		rd read.Read,
		next icd.ICD,
	) ([]icd.ReadMessage, []icd.Change) {
		GinkgoHelper()
		return repairFrom(ctx, rd, plant, next)
	}
	// indexedRead reads ts as the index of status, with temp and ticks.
	indexedRead := indexed(ts, readOf(
		status,
		bind(ts, timeKey),
		bind(temp, valueKey),
		bind(ticks, levelKey),
	))
	statusRead := readAt(0, plant.Key, indexedRead)
	statusFields := payloadOf[icd.JSONPayload](status).Fields
	// statusWithout returns plant with status lacking the field f.
	statusWithout := func(f icd.JSONField) icd.ICD {
		return edited(plant, withFields(status, slices.DeleteFunc(
			slices.Clone(statusFields),
			func(o icd.JSONField) bool { return o.Base().Key == f.Base().Key },
		)...))
	}
	// statusWith returns plant with f in place of the field of status with its key.
	statusWith := func(f icd.JSONField) icd.ICD {
		fields := slices.Clone(statusFields)
		i := slices.IndexFunc(fields, func(o icd.JSONField) bool {
			return o.Base().Key == f.Base().Key
		})
		fields[i] = f
		return edited(plant, withFields(status, fields...))
	}

	It("Should keep the messages of a read of another ICD", func() {
		rd := readAt(0, uuid.New(), indexedRead)
		messages, changes := read.Repair(rd, plant, without(plant, status), channels)
		Expect(messages).To(Equal(rd.Messages))
		Expect(changes).To(BeEmpty())
	})

	It("Should make no change for a rename or a new pointer", func(ctx SpecContext) {
		renamed := jsonRebased(ts, func(b *icd.JSONBaseField) {
			b.Name, b.Pointer = "time", "/time"
		})
		messages, changes := repair(ctx, statusRead, statusWith(renamed))
		Expect(messages).To(Equal(statusRead.Messages))
		Expect(changes).To(BeEmpty())
	})

	It("Should drop the binding of a deleted field", func(ctx SpecContext) {
		messages, changes := repair(ctx, statusRead, statusWithout(temp))
		Expect(messages).To(Equal([]icd.ReadMessage{indexed(ts, readOf(
			status,
			bind(ts, timeKey),
			bind(ticks, levelKey),
		))}))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Status, the task stops reading field temp, which no longer " +
				"exists. Channel value stops getting its values.",
		)}))
	})

	DescribeTable(
		"Should stamp on arrival when the index field cannot index",
		func(ctx SpecContext, next icd.ICD, reason string) {
			messages, changes := repair(ctx, statusRead, next)
			Expect(messages).To(Equal([]icd.ReadMessage{
				readOf(status, bind(temp, valueKey), bind(ticks, levelKey)),
			}))
			Expect(changes).To(Equal([]icd.Change{lossy(
				"In message Status, the task stamps samples on arrival, because " +
					"index field ts " + reason + ".",
			)}))
		},
		Entry("deleted", statusWithout(ts), "no longer exists"),
		Entry(
			"not a timestamp",
			statusWith(as(ts, icd.FloatJSONField{})),
			"is no longer a timestamp field",
		),
		Entry("defaulted", statusWith(as(ts, defaulted.Variant)), "has a default"),
	)

	It("Should stop reading a message with no bindings left", func(ctx SpecContext) {
		rd := readAt(0, plant.Key, readOf(status, bind(temp, valueKey)))
		messages, changes := repair(ctx, rd, statusWithout(temp))
		Expect(messages).To(BeEmpty())
		Expect(changes).To(Equal([]icd.Change{
			lossy(
				"In message Status, the task stops reading field temp, which no " +
					"longer exists. Channel value stops getting its values.",
			),
			lossy("The task stops reading message Status, which has no bindings left."),
		}))
	})

	It("Should keep a message that had no bindings", func(ctx SpecContext) {
		rd := readAt(0, plant.Key, readOf(status))
		messages, changes := repair(ctx, rd, statusWithout(temp))
		Expect(messages).To(Equal(rd.Messages))
		Expect(changes).To(BeEmpty())
	})

	It("Should clear the channel of a field that changes kind", func(ctx SpecContext) {
		next := statusWith(as(temp, icd.StringJSONField{}))
		messages, changes := repair(ctx, statusRead, next)
		Expect(messages).To(Equal([]icd.ReadMessage{indexed(ts, readOf(
			status,
			bind(ts, timeKey),
			bind(temp, 0),
			bind(ticks, levelKey),
		))}))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Status, field temp no longer reads into channel value of " +
				"data type float64. Configure creates a new channel for it.",
		)}))
	})

	It("Should clear a data type that no longer converts", func(ctx SpecContext) {
		rd := readAt(0, plant.Key, readOf(status, typed(bind(temp, 0), telem.Float64T)))
		next := statusWith(as(temp, icd.StringJSONField{}))
		messages, changes := repair(ctx, rd, next)
		Expect(messages).To(Equal([]icd.ReadMessage{readOf(status, bind(temp, 0))}))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Status, field temp no longer converts to data type float64. " +
				"Configure creates a channel of its default data type.",
		)}))
	})

	It("Should drop an unfit data type when the channel fits", func(ctx SpecContext) {
		counter := jsonField("counter", "/counter", icd.IntegerJSONField{})
		meter := jsonMessage("Meter", counter)
		prev := icd.ICD{Key: uuid.New(), Name: "Bench", Messages: []icd.Message{meter}}
		scaled := jsonRebased(counter, func(b *icd.JSONBaseField) { b.Scale = 2 })
		next := edited(prev, withFields(meter, scaled))
		b := typed(bind(counter, valueKey), telem.Int64T)
		rd := readAt(0, prev.Key, readOf(meter, b))
		messages, changes := repairFrom(ctx, rd, prev, next)
		Expect(messages).To(Equal([]icd.ReadMessage{
			readOf(meter, bind(counter, valueKey)),
		}))
		Expect(changes).To(Equal([]icd.Change{{
			Description: "In message Meter, field counter no longer converts to data " +
				"type int64, so its binding drops that data type.",
		}}))
	})

	It("Should drop a bytes field that loses its pointer", func(ctx SpecContext) {
		blob := jsonField("blob", "/blob", icd.BytesJSONField{})
		sample := jsonMessage("Sample", blob, temp)
		prev := icd.ICD{Key: uuid.New(), Name: "Bench", Messages: []icd.Message{sample}}
		unplaced := jsonRebased(blob, func(b *icd.JSONBaseField) { b.Pointer = "" })
		next := edited(prev, withFields(sample, unplaced, temp))
		rd := readAt(0, prev.Key, readOf(sample, bind(blob, 0), bind(temp, valueKey)))
		messages, changes := repairFrom(ctx, rd, prev, next)
		Expect(messages).To(Equal([]icd.ReadMessage{
			readOf(sample, bind(temp, valueKey)),
		}))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Sample, the task stops reading bytes field blob, which has " +
				"no JSON pointer.",
		)}))
	})

	It("Should stop reading a deleted message", func(ctx SpecContext) {
		messages, changes := repair(ctx, statusRead, without(plant, status))
		Expect(messages).To(BeEmpty())
		Expect(changes).To(Equal([]icd.Change{
			lossy("The task stops reading message Status, which no longer exists."),
		}))
	})

	It("Should stop reading the messages of a deleted ICD", func(ctx SpecContext) {
		rd := readAt(0, plant.Key, indexedRead, readOf(stamped))
		messages, changes := repair(ctx, rd, icd.ICD{Key: plant.Key})
		Expect(messages).To(BeEmpty())
		Expect(changes).To(Equal([]icd.Change{
			lossy("The task stops reading message Status, which no longer exists."),
			lossy("The task stops reading message Stamped, which no longer exists."),
		}))
	})

	It("Should stop reading a message that is no longer JSON", func(ctx SpecContext) {
		next := edited(plant, withBinaryFields(status))
		messages, changes := repair(ctx, statusRead, next)
		Expect(messages).To(BeEmpty())
		Expect(changes).To(Equal([]icd.Change{lossy(
			"The task stops reading message Status, which is no longer a JSON message.",
		)}))
	})

	It("Should stop reading a message that changes format", func(ctx SpecContext) {
		rd := framed(readAt(0, plant.Key, readOf(frame, icd.ReadBinding{
			Field: count.Base().Key, Channel: totalKey,
		})))
		messages, changes := repair(ctx, rd, edited(plant, withFields(frame)))
		Expect(messages).To(BeEmpty())
		Expect(changes).To(Equal([]icd.Change{
			lossy("The task stops reading message Frame, which changed its format."),
		}))
	})

	DescribeTable(
		"Should clear the channel of a binary field that outgrows it",
		func(
			ctx SpecContext,
			f icd.BinaryField,
			key channel.Key,
			edit func(*icd.BinaryBaseField),
		) {
			b := icd.ReadBinding{Field: f.Base().Key, Channel: key}
			rd := framed(readAt(0, plant.Key, readOf(frame, b)))
			fields := slices.Clone(payloadOf[icd.BinaryPayload](frame).Fields)
			i := slices.IndexFunc(fields, func(o icd.BinaryField) bool {
				return o.Base().Key == f.Base().Key
			})
			fields[i] = binaryRebased(f, edit)
			next := edited(plant, withBinaryFields(frame, fields...))
			messages, changes := repair(ctx, rd, next)
			b.Channel = 0
			Expect(messages).To(Equal([]icd.ReadMessage{readOf(frame, b)}))
			Expect(changes).To(HaveExactElements(HaveField("Lossy", true)))
		},
		Entry("a new scale", count, totalKey, func(b *icd.BinaryBaseField) {
			b.Scale = 2
		}),
		Entry("a new offset", count, totalKey, func(b *icd.BinaryBaseField) {
			b.Offset = 1
		}),
		Entry("a new bit length", wide, wordKey, func(b *icd.BinaryBaseField) {
			b.BitLength = 32
		}),
	)

	It("Should drop the binding of a deleted text field", func(ctx SpecContext) {
		rd := framed(readAt(0, plant.Key, readOf(line, icd.ReadBinding{
			Field: volts.Base().Key, Channel: valueKey,
		})))
		p := payloadOf[icd.TextPayload](line)
		p.Fields = nil
		next := line
		next.Payload = icd.Payload{Variant: p}
		messages, changes := repair(ctx, rd, edited(plant, next))
		Expect(messages).To(BeEmpty())
		Expect(changes).To(Equal([]icd.Change{
			lossy(
				"In message Line, the task stops reading field volts, which no " +
					"longer exists. Channel value stops getting its values.",
			),
			lossy("The task stops reading message Line, which has no bindings left."),
		}))
	})

	It("Should name no key for parts that a migration finds gone", func() {
		stored := statusWithout(temp)
		rd := readAt(
			0,
			plant.Key,
			readOf(status, bind(temp, valueKey), bind(ticks, levelKey)),
			readOf(icd.Message{Key: uuid.New()}),
		)
		messages, changes := read.Repair(rd, stored, stored, channels)
		Expect(messages).To(Equal([]icd.ReadMessage{
			readOf(status, bind(ticks, levelKey)),
		}))
		Expect(changes).To(Equal([]icd.Change{
			lossy(
				"In message Status, the task stops reading a field that no longer " +
					"exists. Channel value stops getting its values.",
			),
			lossy("The task stops reading a message that no longer exists."),
		}))
	})
})

var _ = Describe("RepairEntries", func() {
	It("Should remove an entry that reads no messages now", func() {
		volt := icd.ReadBinding{Field: volts.Base().Key, Channel: valueKey}
		counted := icd.ReadBinding{Field: count.Base().Key, Channel: totalKey}
		entries := []icd.ReadEntry{
			{ICD: plant.Key, Messages: []icd.ReadMessage{readOf(line, volt)}},
			{ICD: plant.Key, Messages: []icd.ReadMessage{readOf(frame, counted)}},
			{ICD: plant.Key},
		}
		kept, changes := read.RepairEntries(entries, plant, without(plant, line), held)
		Expect(kept).To(Equal(entries[1:]))
		Expect(changes).To(Equal([]icd.Change{
			lossy("The task stops reading message Line, which no longer exists."),
			lossy(
				"The task removes an entry of ICD Plant, because the entry reads no " +
					"messages now.",
			),
		}))
	})
})
