// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
	. "github.com/synnaxlabs/x/testutil"
)

func field(name string, startBit uint16) library.BinaryField {
	return library.BinaryField{
		Key:  uuid.New(),
		Name: name, Scale: 1,
		StartBit:  startBit,
		BitLength: 8,
		ByteOrder: library.ByteOrderLittleEndian,
	}
}

func message(name string, fields ...library.BinaryField) library.MessageEntry {
	return library.MessageEntry{
		Key:     uuid.New(),
		Name:    name,
		Payload: library.Payload{Variant: library.BinaryPayload{Fields: fields}},
	}
}

// identify returns m identified by the raw value 1 of the field with the given key.
func identify(m library.MessageEntry, field library.FieldKey) library.MessageEntry {
	p := binaryPayload(m)
	p.Identifier = &library.Identifier{
		Variant: library.FieldIdentifier{Field: field, Value: 1},
	}
	m.Payload.Variant = p
	return m
}

func enum(name string, values ...library.EnumValue) library.EnumEntry {
	return library.EnumEntry{
		Key:    uuid.New(),
		Name:   name,
		Values: values,
	}
}

func entries(variants ...library.EntryVariant) []library.Entry {
	out := make([]library.Entry, len(variants))
	for i, v := range variants {
		out[i] = library.Entry{Variant: v}
	}
	return out
}

var _ = Describe("Merge", func() {
	DescribeTable("Should keep every key when the same file is imported again",
		func(name string) {
			data, _ := readFixture(name)
			first := library.Library{
				Key:     uuid.New(),
				Name:    "Bus",
				Entries: MustSucceed(icd.Parse(icd.FormatDBC, data)),
			}
			Expect(
				icd.Merge(first, MustSucceed(icd.Parse(icd.FormatDBC, data))),
			).To(Equal(first))
		},
		Entry("enum references", "basic"),
		Entry("multiplexor references", "extended_multiplexed"),
	)

	It("Should drop missing entries and fields and add new ones", func() {
		oldRPM, oldTemp := field("rpm", 0), field("temp", 8)
		oldEngine := message("Engine", oldRPM, oldTemp)
		oldGear := enum("Gear", library.EnumValue{Value: 0, Name: "Park"})
		existing := library.Library{
			Key:     uuid.New(),
			Name:    "Bus",
			Entries: entries(oldEngine, message("Brakes"), oldGear),
		}
		newRPM, newLoad := field("rpm", 16), field("load", 24)
		newEngine := message("Engine", newRPM, newLoad)
		newGear := enum("Gear", library.EnumValue{Value: 1, Name: "Drive"})
		newDoors := message("Doors")
		merged := icd.Merge(existing, entries(newEngine, newGear, newDoors))
		Expect(merged.Key).To(Equal(existing.Key))
		Expect(merged.Name).To(Equal("Bus"))
		Expect(merged.Entries).To(HaveLen(3))
		engine := messageNamed(merged.Entries, "Engine")
		Expect(engine.Key).To(Equal(oldEngine.Key))
		fields := binaryFields(engine)
		Expect(fields).To(HaveLen(2))
		Expect(fields[0].Key).To(Equal(oldRPM.Key))
		Expect(fields[0].StartBit).To(Equal(uint16(16)))
		Expect(fields[1].Key).To(Equal(newLoad.Key))
		gear := enumNamed(merged.Entries, "Gear")
		Expect(gear.Key).To(Equal(oldGear.Key))
		Expect(gear.Values).To(Equal(newGear.Values))
		Expect(messageNamed(merged.Entries, "Doors").Key).To(Equal(newDoors.Key))
	})

	It("Should rewrite references to the kept keys", func() {
		oldMode, oldValue := field("mode", 0), field("value", 8)
		oldStatus := enum("Status")
		oldMessage := identify(message("Frame", oldMode, oldValue), oldMode.Key)
		existing := library.Library{Entries: entries(oldMessage, oldStatus)}
		newMode, newValue := field("mode", 0), field("value", 8)
		newStatus := enum("Status")
		newValue.Enumeration = new(newStatus.Key)
		newValue.Multiplexor = new(newMode.Key)
		newValue.MultiplexValues = []int32{1}
		newMessage := identify(message("Frame", newMode, newValue), newMode.Key)
		merged := icd.Merge(existing, entries(newMessage, newStatus))
		frame := messageNamed(merged.Entries, "Frame")
		value := fieldNamed(frame, "value")
		Expect(*value.Enumeration).To(Equal(oldStatus.Key))
		Expect(*value.Multiplexor).To(Equal(oldMode.Key))
		Expect(binaryPayload(frame).Identifier.Variant).To(Equal(
			library.FieldIdentifier{Field: oldMode.Key, Value: 1},
		))
		Expect(*newValue.Multiplexor).To(Equal(newMode.Key))
		Expect(binaryFields(newMessage)[1].Key).To(Equal(newValue.Key))
	})

	It("Should not match entries of different kinds", func() {
		old := enum("Engine")
		imported := message("Engine")
		merged := icd.Merge(library.Library{Entries: entries(old)}, entries(imported))
		Expect(messageNamed(merged.Entries, "Engine").Key).To(Equal(imported.Key))
	})

	It("Should keep the keys of text fields", func() {
		tagged := func(name string) library.TextField {
			return library.TextField{Variant: library.TaggedTextField{
				BaseField: library.BaseField{Key: uuid.New(), Name: name},
				Tag:       name + "=",
			}}
		}
		line := func(fields ...library.TextField) library.MessageEntry {
			return library.MessageEntry{
				Key:  uuid.New(),
				Name: "Line",
				Payload: library.Payload{Variant: library.TextPayload{
					Delimiter: ",",
					Fields:    fields,
				}},
			}
		}
		old := line(tagged("temp"))
		imported := line(tagged("temp"), tagged("rh"))
		merged := icd.Merge(library.Library{Entries: entries(old)}, entries(imported))
		fields := messageNamed(merged.Entries, "Line").Payload.FieldBases()
		Expect(fields).To(HaveLen(2))
		Expect(fields[0].Key).To(Equal(old.Payload.FieldBases()[0].Key))
		Expect(fields[1].Key).To(Equal(imported.Payload.FieldBases()[1].Key))
	})
})
