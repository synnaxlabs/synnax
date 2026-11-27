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
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/x/telem"
)

// countKey is an int64 channel.
const countKey channel.Key = 4

var channels = map[channel.Key]icdv0.Channel{
	valueKey: {Name: "value", DataType: telem.Float64T},
	modeKey:  {Name: "mode", DataType: telem.StringT},
	countKey: {Name: "count", DataType: telem.Int64T},
}

// site holds a JSON, a binary, a text, and a multiplexed message.
var site = icd.ICD{
	Key:      uuid.New(),
	Name:     "Site",
	Messages: []icd.Message{setpoint, counts, line, actuator},
}

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

// withPayload returns m with the payload variant p.
func withPayload(m icd.Message, p icd.PayloadVariant) icd.Message {
	m.Payload = icd.Payload{Variant: p}
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

// withBinary returns m, a binary message, with each of fields in place of the field
// with its key.
func withBinary(m icd.Message, fields ...icd.BinaryField) icd.Message {
	p := payloadOf[icd.BinaryPayload](m)
	p.Fields = slices.Clone(p.Fields)
	for _, f := range fields {
		i := slices.IndexFunc(p.Fields, func(o icd.BinaryField) bool {
			return o.Base().Key == f.Base().Key
		})
		p.Fields[i] = f
	}
	return withPayload(m, p)
}

// muxedBy returns f, a field of actuator, selected when the selector sends values.
func muxedBy(f icd.BinaryField, values ...int32) icd.BinaryField {
	b := f.Base()
	b.MultiplexValues = values
	f.SetBase(b)
	return f
}

func unbound(f icd.FieldKey) icd.WriteBinding { return channelOf(f, 0, nil) }

func lossy(description string) icd.Change {
	return icd.Change{Description: description, Lossy: true}
}

func additive(description string) icd.Change {
	return icd.Change{Description: description}
}

var _ = Describe("Repair", func() {
	var resolver write.Resolver
	BeforeEach(func() {
		resolver = write.NewResolver(icdSvc, func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			return channels, nil
		})
	})
	// repair repairs w, which the save check accepts against site, for next, and
	// expects the save check to accept the repaired write against next.
	repair := func(
		ctx context.Context,
		w write.Write,
		next icd.ICD,
	) (*icd.WriteMessage, []icd.Change) {
		GinkgoHelper()
		before := map[icd.Key]icd.ICD{site.Key: site}
		Expect(resolver.Check(ctx, db, []write.Write{w}, before)).To(Succeed())
		wm, changes := write.Repair(w, site, next, channels)
		if wm != nil {
			w.Message = *wm
			after := map[icd.Key]icd.ICD{next.Key: next}
			Expect(resolver.Check(ctx, db, []write.Write{w}, after)).To(Succeed())
		}
		return wm, changes
	}
	manual := icd.StringWriteValue{Value: "manual"}
	setpointWrite := writeAt(
		0,
		site.Key,
		writeOf(setpoint, bind(temp), fixed(mode, manual)),
	)
	// setpointWith returns site with setpoint holding fields.
	setpointWith := func(fields ...icd.JSONField) icd.ICD {
		return edited(site, withFields(setpoint, fields...))
	}

	It("Should keep the message of a write of another ICD", func() {
		w := writeAt(0, uuid.New(), setpointWrite.Message)
		wm, changes := write.Repair(w, site, without(site, setpoint), channels)
		Expect(wm).To(Equal(&w.Message))
		Expect(changes).To(BeEmpty())
	})

	It("Should make no change for a rename or a new pointer", func(ctx SpecContext) {
		renamed := jsonRebased(temp, func(b *icd.JSONBaseField) {
			b.Name, b.Pointer = "target", "/target"
		})
		wm, changes := repair(ctx, setpointWrite, setpointWith(renamed, mode, limit))
		Expect(wm).To(Equal(&setpointWrite.Message))
		Expect(changes).To(BeEmpty())
	})

	It("Should drop the binding of a deleted field", func(ctx SpecContext) {
		wm, changes := repair(ctx, setpointWrite, setpointWith(temp, limit))
		Expect(wm).To(Equal(new(writeOf(setpoint, bind(temp)))))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Setpoint, the task drops the binding of field mode, which no " +
				"longer exists.",
		)}))
	})

	It("Should stop sending a message with no channels left", func(ctx SpecContext) {
		wm, changes := repair(ctx, setpointWrite, setpointWith(mode, limit))
		Expect(wm).To(BeNil())
		Expect(changes).To(Equal([]icd.Change{
			lossy(
				"In message Setpoint, the task drops the binding of field temp, " +
					"which no longer exists.",
			),
			lossy(
				"The task stops sending message Setpoint, which has no channel " +
					"bindings left.",
			),
		}))
	})

	It("Should keep a message that had no channel bindings", func() {
		w := writeAt(0, site.Key, writeOf(setpoint, fixed(mode, manual)))
		wm, changes := write.Repair(w, site, setpointWith(mode, limit), channels)
		Expect(wm).To(Equal(&w.Message))
		Expect(changes).To(BeEmpty())
	})

	It("Should clear the channel of a field that changes kind", func(ctx SpecContext) {
		next := setpointWith(as(temp, icd.StringJSONField{}), mode, limit)
		wm, changes := repair(ctx, setpointWrite, next)
		Expect(wm).To(Equal(new(writeOf(
			setpoint,
			unbound(temp.Base().Key),
			fixed(mode, manual),
		))))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Setpoint, channel value of data type float64 can no longer " +
				"set field temp. Configure creates a new channel for it.",
		)}))
	})

	It("Should clear a data type that no longer sets its field", func(ctx SpecContext) {
		b := channelOf(temp.Base().Key, 0, new(telem.Float64T))
		w := writeAt(0, site.Key, writeOf(setpoint, b))
		next := setpointWith(as(temp, icd.StringJSONField{}), mode, limit)
		wm, changes := repair(ctx, w, next)
		Expect(wm).To(Equal(new(writeOf(setpoint, unbound(temp.Base().Key)))))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Setpoint, data type float64 can no longer set field temp. " +
				"Configure creates a channel of its default data type.",
		)}))
	})

	It("Should drop an unfit data type when the channel fits", func(ctx SpecContext) {
		b := channelOf(temp.Base().Key, valueKey, new(telem.Int64T))
		w := writeAt(0, site.Key, writeOf(setpoint, b))
		scaled := jsonRebased(temp, func(b *icd.JSONBaseField) { b.Scale = 2 })
		wm, changes := repair(ctx, w, setpointWith(scaled, mode, limit))
		Expect(wm).To(Equal(new(writeOf(setpoint, bind(temp)))))
		Expect(changes).To(Equal([]icd.Change{additive(
			"In message Setpoint, data type int64 can no longer set field temp, so " +
				"its binding drops that data type.",
		)}))
	})

	It("Should bind a field that loses its value to a channel", func(ctx SpecContext) {
		next := setpointWith(temp, as(mode, icd.FloatJSONField{}), limit)
		wm, changes := repair(ctx, setpointWrite, next)
		Expect(wm).To(Equal(new(writeOf(
			setpoint,
			bind(temp),
			unbound(mode.Base().Key),
		))))
		Expect(changes).To(Equal([]icd.Change{
			lossy(
				"In message Setpoint, field mode can no longer take its value, so " +
					"the task drops the value.",
			),
			additive(
				"In message Setpoint, field mode has no default, so Configure " +
					"creates a channel for it.",
			),
		}))
	})

	It("Should drop a value that its field can no longer hold", func(ctx SpecContext) {
		fifty := icd.FloatWriteValue{Value: 50}
		w := writeAt(0, site.Key, writeOf(setpoint, bind(temp), fixed(limit, fifty)))
		wm, changes := repair(ctx, w, setpointWith(temp, mode, withMax(limit, 10)))
		Expect(wm).To(Equal(new(writeOf(setpoint, bind(temp)))))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Setpoint, field limit can no longer take its value, so the " +
				"task drops the value.",
		)}))
	})

	added := jsonField("added", icd.FloatJSONField{})
	DescribeTable(
		"Should bind a field with no default to a new channel",
		func(ctx SpecContext, next icd.ICD, f icd.JSONField) {
			w := writeAt(0, site.Key, writeOf(setpoint, bind(temp)))
			wm, changes := repair(ctx, w, next)
			Expect(wm).To(Equal(new(writeOf(
				setpoint,
				bind(temp),
				unbound(f.Base().Key),
			))))
			Expect(changes).To(Equal([]icd.Change{additive(
				"In message Setpoint, field " + f.Base().Name + " has no default, so " +
					"Configure creates a channel for it.",
			)}))
		},
		Entry("an added field", setpointWith(temp, mode, limit, added), added),
		Entry(
			"a removed default",
			setpointWith(temp, mode, as(limit, icd.FloatJSONField{})),
			limit,
		),
	)

	It("Should stop sending a deleted message", func(ctx SpecContext) {
		wm, changes := repair(ctx, setpointWrite, without(site, setpoint))
		Expect(wm).To(BeNil())
		Expect(changes).To(Equal([]icd.Change{
			lossy("The task stops sending message Setpoint, which no longer exists."),
		}))
	})

	It("Should stop sending a message that is no longer JSON", func(ctx SpecContext) {
		next := edited(site, withPayload(setpoint, icd.BinaryPayload{}))
		wm, changes := repair(ctx, setpointWrite, next)
		Expect(wm).To(BeNil())
		Expect(changes).To(Equal([]icd.Change{lossy(
			"The task stops sending message Setpoint, which is no longer a JSON " +
				"message.",
		)}))
	})

	It("Should drop the bindings of a message that changes format", func(
		ctx SpecContext,
	) {
		b := channelOf(volts.Base().Key, valueKey, nil)
		w := framed(writeAt(0, site.Key, writeOf(line, b)))
		level := jsonField("level", icd.FloatJSONField{})
		wm, changes := repair(ctx, w, edited(site, withFields(line, level)))
		Expect(wm).To(Equal(new(writeOf(line, unbound(level.Base().Key)))))
		Expect(changes).To(Equal([]icd.Change{
			lossy("Message Line changed its format, so the task drops its bindings."),
			additive(
				"In message Line, field level has no default, so Configure creates a " +
					"channel for it.",
			),
		}))
	})

	It("Should clear the channel of a text field that gains a scale", func(
		ctx SpecContext,
	) {
		b := channelOf(volts.Base().Key, countKey, nil)
		w := framed(writeAt(0, site.Key, writeOf(line, b)))
		gained := volts
		base := volts.Base()
		base.Scale = 2
		gained.SetBase(base)
		p := payloadOf[icd.TextPayload](line)
		p.Fields = []icd.TextField{gained, trim}
		wm, changes := repair(ctx, w, edited(site, withPayload(line, p)))
		Expect(wm).To(Equal(new(writeOf(line, unbound(volts.Base().Key)))))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Line, channel count of data type int64 can no longer set " +
				"field volts. Configure creates a new channel for it.",
		)}))
	})

	It("Should drop the binding of a field that holds the identifier now", func(
		ctx SpecContext,
	) {
		gained := channelOf(gain.Base().Key, valueKey, nil)
		w := framed(writeAt(0, site.Key, writeOf(
			counts,
			gained,
			channelOf(unit.Base().Key, countKey, nil),
		)))
		p := payloadOf[icd.BinaryPayload](counts)
		p.Identifier = &icd.Identifier{Variant: icd.FieldIdentifier{
			Field: unit.Base().Key, Value: 1,
		}}
		wm, changes := repair(ctx, w, edited(site, withPayload(counts, p)))
		Expect(wm).To(Equal(new(writeOf(counts, gained))))
		Expect(changes).To(Equal([]icd.Change{lossy(
			"In message Counts, field unit now holds the identifier, so the task " +
				"drops its binding.",
		)}))
	})

	It("Should move a channel to the field that its multiplexor selects now", func(
		ctx SpecContext,
	) {
		zero := valueOf(selector.Base().Key, icd.UnsignedWriteValue{Value: 0})
		w := framed(writeAt(0, site.Key, writeOf(
			actuator,
			zero,
			channelOf(opening.Base().Key, valueKey, nil),
		)))
		next := edited(
			site,
			withBinary(actuator, muxedBy(opening, 1), muxedBy(rate, 0)),
		)
		wm, changes := repair(ctx, w, next)
		Expect(wm).To(Equal(new(writeOf(actuator, zero, unbound(rate.Base().Key)))))
		Expect(changes).To(Equal([]icd.Change{
			lossy(
				"In message Actuator, multiplexor selector never selects field " +
					"opening now, so the task drops its channel binding.",
			),
			additive(
				"In message Actuator, field rate has no default, so Configure " +
					"creates a channel for it.",
			),
		}))
	})

	It("Should name no key for parts that a migration finds gone", func() {
		stored := setpointWith(mode, limit)
		_, changes := write.Repair(setpointWrite, stored, stored, channels)
		Expect(changes).To(Equal([]icd.Change{
			lossy(
				"In message Setpoint, the task drops the binding of a field that no " +
					"longer exists.",
			),
			lossy(
				"The task stops sending message Setpoint, which has no channel " +
					"bindings left.",
			),
		}))
		gone := writeAt(0, site.Key, icd.WriteMessage{Message: uuid.New()})
		_, changes = write.Repair(gone, stored, stored, channels)
		Expect(changes).To(Equal([]icd.Change{
			lossy("The task stops sending a message that no longer exists."),
		}))
	})
})

var _ = Describe("RepairTargets", func() {
	It("Should remove a target that can no longer send its message", func() {
		period := telem.Second
		volt := channelOf(volts.Base().Key, valueKey, nil)
		targets := []icd.WriteTarget{
			{ICD: site.Key, Message: writeOf(line, volt)},
			{ICD: site.Key, Message: writeOf(setpoint, bind(temp)), Period: &period},
		}
		kept, changes := write.RepairTargets(
			targets,
			site,
			without(site, line),
			channels,
		)
		Expect(kept).To(Equal(targets[1:]))
		Expect(changes).To(Equal([]icd.Change{
			lossy("The task stops sending message Line, which no longer exists."),
		}))
	})

	It("Should keep each target of another ICD", func() {
		targets := []icd.WriteTarget{
			{ICD: uuid.New(), Message: writeOf(setpoint, bind(temp))},
		}
		kept, changes := write.RepairTargets(
			targets,
			site,
			icd.ICD{Key: site.Key},
			channels,
		)
		Expect(kept).To(Equal(targets))
		Expect(changes).To(BeEmpty())
	})
})
