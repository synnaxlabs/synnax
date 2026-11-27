// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func (m fixtureMessage) binaryMessage() codec.BinaryMessage {
	GinkgoHelper()
	p, ok := m.Payload.Variant.(icd.BinaryPayload)
	Expect(ok).To(BeTrue(), "message %s is not binary", m.Name)
	return codec.BinaryMessage{Name: m.Name, Payload: p, Bindings: m.bindings()}
}

func (f decodeFixture) binaryMessages() []codec.BinaryMessage {
	GinkgoHelper()
	return lo.Map(f.Messages, func(m fixtureMessage, _ int) codec.BinaryMessage {
		return m.binaryMessage()
	})
}

// can reports whether a message of f has a CAN identifier, which Go does not read.
func (f decodeFixture) can() bool {
	return lo.SomeBy(f.Messages, func(m fixtureMessage) bool {
		p, ok := m.Payload.Variant.(icd.BinaryPayload)
		if !ok || p.Identifier == nil {
			return false
		}
		_, can := p.Identifier.Variant.(icd.CanIdentifier)
		return can
	})
}

// binaryFixtureEntries returns an entry for each binary decode fixture with no CAN
// identifier.
func binaryFixtureEntries() []TableEntry {
	GinkgoHelper()
	paths := lo.Filter(
		fixturePaths("decode", payload.FormatBinary),
		func(path string, _ int) bool { return !readDecodeFixture(path).can() },
	)
	return lo.Map(paths, func(path string, _ int) TableEntry {
		return fixtureEntry(path)
	})
}

// byteField returns an unsigned binary field named name in the byte at offset.
func byteField(name string, offset uint16) icd.BinaryField {
	return icd.BinaryField{Variant: icd.UnsignedBinaryField{
		BinaryBaseField: icd.BinaryBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: name, Scale: 1},
			StartBit:  offset * 8,
			BitLength: 8,
			ByteOrder: icd.ByteOrderLittleEndian,
		},
	}}
}

var _ = Describe("Binary", func() {
	DescribeTable("Should decode the payloads of a fixture", func(path string) {
		f := readDecodeFixture(path)
		messages := f.binaryMessages()
		if f.Error != "" {
			Expect(codec.NewBinary(messages)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(f.Error),
			))
			return
		}
		f.checkCases(MustSucceed(codec.NewBinary(messages)).Decode)
	}, binaryFixtureEntries())

	It("Should reject a message with a CAN identifier", func() {
		f := readDecodeFixture("../testdata/decode/binary_can_exact.json")
		Expect(codec.NewBinary(f.binaryMessages())).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message a: identifier: only the Driver reads CAN frames: "+
				"validation error"),
		))
	})

	It("Should name the message of a payload too short for its multiplexors when "+
		"the decoder holds more than one", func() {
		// identified returns a message named name that kind with value identifies.
		identified := func(
			name string,
			value int32,
			fields ...icd.BinaryField,
		) codec.BinaryMessage {
			kind := byteField("kind", 0)
			return codec.BinaryMessage{
				Name: name,
				Payload: icd.BinaryPayload{
					Identifier: &icd.Identifier{Variant: icd.FieldIdentifier{
						Field: kind.Base().Key,
						Value: value,
					}},
					Fields: append([]icd.BinaryField{kind}, fields...),
				},
				Bindings: []codec.Binding{
					{Field: fields[len(fields)-1].Base().Key, DataType: telem.Uint8T},
				},
			}
		}
		mode := byteField("mode", 2)
		selected := byteField("a", 1)
		b := selected.Variant.(icd.UnsignedBinaryField)
		b.Multiplexor, b.MultiplexValues = new(mode.Base().Key), []int32{1}
		selected.Variant = b
		dec := MustSucceed(codec.NewBinary([]codec.BinaryMessage{
			identified("muxed", 1, mode, selected),
			identified("plain", 2, byteField("v", 1)),
		}))
		samples := [][][]byte{make([][]byte, 1), make([][]byte, 1)}
		message, ok, failures, err := dec.Decode([]byte{1, 0}, samples)
		Expect(err).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message muxed: field mode: the payload has 2 bytes and the "+
				"field needs 3: validation error"),
		))
		Expect(message).To(BeZero())
		Expect(ok).To(BeTrue())
		Expect(failures).To(BeNil())
	})

	It("Should route and decode payloads without allocating", func() {
		kind, v := byteField("kind", 0), byteField("v", 1)
		m := codec.BinaryMessage{
			Name: "frame",
			Payload: icd.BinaryPayload{
				Identifier: &icd.Identifier{Variant: icd.FieldIdentifier{
					Field: kind.Base().Key,
					Value: 1,
				}},
				Fields: []icd.BinaryField{kind, v},
			},
			Bindings: []codec.Binding{
				{Field: v.Base().Key, DataType: telem.Float64T},
				{Field: kind.Base().Key, DataType: telem.Uint8T},
			},
		}
		dec := MustSucceed(codec.NewBinary([]codec.BinaryMessage{m}))
		samples := [][][]byte{make([][]byte, 2)}
		data := []byte{1, 42}
		var (
			ok       bool
			failures []codec.Failure
			err      error
		)
		decode := func() {
			for i := range samples[0] {
				samples[0][i] = samples[0][i][:0]
			}
			_, ok, failures, err = dec.Decode(data, samples)
		}
		Expect(testing.AllocsPerRun(100, decode)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(failures).To(BeNil())
		Expect(samples[0][0]).To(Equal(telem.NewSeriesV(42.0).Data))
		Expect(samples[0][1]).To(Equal([]byte{1}))
	})
})
