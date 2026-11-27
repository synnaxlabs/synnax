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

func (m fixtureMessage) textMessage() codec.TextMessage {
	GinkgoHelper()
	p, ok := m.Payload.Variant.(icd.TextPayload)
	Expect(ok).To(BeTrue(), "message %s is not text", m.Name)
	return codec.TextMessage{Name: m.Name, Payload: p, Bindings: m.bindings()}
}

func (f decodeFixture) textMessages() []codec.TextMessage {
	GinkgoHelper()
	return lo.Map(f.Messages, func(m fixtureMessage, _ int) codec.TextMessage {
		return m.textMessage()
	})
}

// textDecode adapts the Decode of dec, which has no failed fields, to a decodeFunc.
func textDecode(dec *codec.Text) decodeFunc {
	return func(
		line []byte,
		samples [][][]byte,
	) (int, bool, []codec.Failure, error) {
		message, ok, err := dec.Decode(line, samples)
		return message, ok, nil, err
	}
}

// delimited returns a delimited text field named name at position.
func delimited(name string, position uint32) icd.TextField {
	return icd.TextField{Variant: icd.DelimitedTextField{
		TextBaseField: icd.TextBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: name, Scale: 1},
		},
		Position: position,
	}}
}

// tagged returns a tagged text field named name with tag.
func tagged(name, tag string) icd.TextField {
	return icd.TextField{Variant: icd.TaggedTextField{
		TextBaseField: icd.TextBaseField{
			BaseField: icd.BaseField{Key: uuid.New(), Name: name, Scale: 1},
		},
		Tag: tag,
	}}
}

// textMessage returns a message named name with prefix whose fields each bind to a
// float64 channel.
func textMessage(name, prefix string, fields ...icd.TextField) codec.TextMessage {
	return codec.TextMessage{
		Name:    name,
		Payload: icd.TextPayload{Delimiter: ",", Prefix: prefix, Fields: fields},
		Bindings: lo.Map(fields, func(f icd.TextField, _ int) codec.Binding {
			return codec.Binding{Field: f.Base().Key, DataType: telem.Float64T}
		}),
	}
}

// newTextSamples returns empty sample slices for each binding of each message.
func newTextSamples(messages []codec.TextMessage) [][][]byte {
	return lo.Map(messages, func(m codec.TextMessage, _ int) [][]byte {
		return make([][]byte, len(m.Bindings))
	})
}

var _ = Describe("Text", func() {
	DescribeTable("Should decode the lines of a fixture", func(path string) {
		f := readDecodeFixture(path)
		messages := f.textMessages()
		if f.Error != "" {
			Expect(codec.NewText(messages)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(f.Error),
			))
			return
		}
		f.checkCases(textDecode(MustSucceed(codec.NewText(messages))))
	}, fixtureEntries("decode", payload.FormatText))

	It("Should route and decode lines without allocating", func() {
		messages := []codec.TextMessage{
			textMessage("gga", "$GPGGA", delimited("a", 1), tagged("t", "T=")),
			textMessage("any", "", delimited("a", 0), delimited("b", 3)),
		}
		dec := MustSucceed(codec.NewText(messages))
		samples := newTextSamples(messages)
		gga := []byte("$GPGGA,-122.41941550000,T=1e3")
		bare := []byte("1,2,3,0.000123456789012")
		var (
			first, second int
			err           error
		)
		decode := func() {
			for _, s := range samples {
				for i := range s {
					s[i] = s[i][:0]
				}
			}
			if first, _, err = dec.Decode(gga, samples); err != nil {
				return
			}
			second, _, err = dec.Decode(bare, samples)
		}
		Expect(testing.AllocsPerRun(100, decode)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
		Expect(first).To(Equal(0))
		Expect(second).To(Equal(1))
		Expect(samples[0][0]).To(Equal(telem.NewSeriesV(-122.4194155).Data))
		Expect(samples[1][1]).To(Equal(telem.NewSeriesV(0.000123456789012).Data))
	})
})
