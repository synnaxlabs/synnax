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
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	. "github.com/synnaxlabs/x/testutil"
)

func field(name string, startBit uint16) icd.BinaryField {
	return icd.BinaryField{
		Key:  uuid.New(),
		Name: name, Scale: 1,
		StartBit:  startBit,
		BitLength: 8,
		ByteOrder: icd.ByteOrderLittleEndian,
	}
}

func message(name string, fields ...icd.BinaryField) icd.Message {
	return icd.Message{
		Key:     uuid.New(),
		Name:    name,
		Payload: icd.Payload{Variant: icd.BinaryPayload{Fields: fields}},
	}
}

// identify returns m identified by the raw value 1 of the field with the given key.
func identify(m icd.Message, field icd.FieldKey) icd.Message {
	p := binaryPayload(m)
	p.Identifier = &icd.Identifier{
		Variant: icd.FieldIdentifier{Field: field, Value: 1},
	}
	m.Payload.Variant = p
	return m
}

var _ = Describe("Merge", func() {
	DescribeTable("Should keep every key when the same file is imported again",
		func(name string) {
			data, _ := readFixture(name)
			first := icd.ICD{
				Key:      uuid.New(),
				Name:     "Bus",
				Messages: MustSucceed(icd.Parse(icd.FormatDBC, data)),
			}
			Expect(
				icd.Merge(first, MustSucceed(icd.Parse(icd.FormatDBC, data))),
			).To(Equal(first))
		},
		Entry("value tables", "basic"),
		Entry("multiplexor references", "extended_multiplexed"),
	)

	It("Should drop missing messages and fields and add new ones", func() {
		oldRPM, oldTemp := field("rpm", 0), field("temp", 8)
		oldEngine := message("Engine", oldRPM, oldTemp)
		existing := icd.ICD{
			Key:      uuid.New(),
			Name:     "Bus",
			Messages: []icd.Message{oldEngine, message("Brakes")},
		}
		newRPM, newLoad := field("rpm", 16), field("load", 24)
		newEngine := message("Engine", newRPM, newLoad)
		newDoors := message("Doors")
		merged := icd.Merge(existing, []icd.Message{newEngine, newDoors})
		Expect(merged.Key).To(Equal(existing.Key))
		Expect(merged.Name).To(Equal("Bus"))
		Expect(merged.Messages).To(HaveLen(2))
		engine := messageNamed(merged.Messages, "Engine")
		Expect(engine.Key).To(Equal(oldEngine.Key))
		fields := binaryFields(engine)
		Expect(fields).To(HaveLen(2))
		Expect(fields[0].Key).To(Equal(oldRPM.Key))
		Expect(fields[0].StartBit).To(Equal(uint16(16)))
		Expect(fields[1].Key).To(Equal(newLoad.Key))
		Expect(messageNamed(merged.Messages, "Doors").Key).To(Equal(newDoors.Key))
	})

	It("Should rewrite references to the kept keys", func() {
		oldMode, oldValue := field("mode", 0), field("value", 8)
		oldMessage := identify(message("Frame", oldMode, oldValue), oldMode.Key)
		existing := icd.ICD{Messages: []icd.Message{oldMessage}}
		newMode, newValue := field("mode", 0), field("value", 8)
		newValue.Multiplexor = new(newMode.Key)
		newValue.MultiplexValues = []int32{1}
		newMessage := identify(message("Frame", newMode, newValue), newMode.Key)
		merged := icd.Merge(existing, []icd.Message{newMessage})
		frame := messageNamed(merged.Messages, "Frame")
		value := fieldNamed(frame, "value")
		Expect(*value.Multiplexor).To(Equal(oldMode.Key))
		Expect(binaryPayload(frame).Identifier.Variant).To(Equal(
			icd.FieldIdentifier{Field: oldMode.Key, Value: 1},
		))
		Expect(*newValue.Multiplexor).To(Equal(newMode.Key))
		Expect(binaryFields(newMessage)[1].Key).To(Equal(newValue.Key))
	})

	It("Should keep the keys of text fields", func() {
		tagged := func(name string) icd.TextField {
			return icd.TextField{Variant: icd.TaggedTextField{
				BaseField: icd.BaseField{Key: uuid.New(), Name: name},
				Tag:       name + "=",
			}}
		}
		line := func(fields ...icd.TextField) icd.Message {
			return icd.Message{
				Key:  uuid.New(),
				Name: "Line",
				Payload: icd.Payload{Variant: icd.TextPayload{
					Delimiter: ",",
					Fields:    fields,
				}},
			}
		}
		old := line(tagged("temp"))
		imported := line(tagged("temp"), tagged("rh"))
		merged := icd.Merge(
			icd.ICD{Messages: []icd.Message{old}},
			[]icd.Message{imported},
		)
		fields := messageNamed(merged.Messages, "Line").Payload.FieldBases()
		Expect(fields).To(HaveLen(2))
		Expect(fields[0].Key).To(Equal(old.Payload.FieldBases()[0].Key))
		Expect(fields[1].Key).To(Equal(imported.Payload.FieldBases()[1].Key))
	})
})
