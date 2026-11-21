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
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// decodeFixture is the JSON messages of one source from testdata/decode and the
// documents they decode. A fixture with an error names the error NewJSON must return,
// and has no cases.
type decodeFixture struct {
	Description string           `json:"description"`
	Messages    []fixtureMessage `json:"messages"`
	Error       string           `json:"error"`
	Cases       []decodeCase     `json:"cases"`
}

type fixtureMessage struct {
	Name     string          `json:"name"`
	Payload  icd.JSONPayload `json:"payload"`
	Bindings []binding       `json:"bindings"`
}

type binding struct {
	Field    icd.FieldKey   `json:"field"`
	DataType telem.DataType `json:"data_type"`
}

// decodeCase is one document. A document that belongs to no message sets unmatched, and
// one that does not decode names the error. Otherwise message names the message it
// belongs to, and samples holds one JSON value for each binding of that message.
type decodeCase struct {
	Document  json.RawMessage   `json:"document"`
	Text      *string           `json:"text"`
	Unmatched bool              `json:"unmatched"`
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	Samples   []json.RawMessage `json:"samples"`
}

func fixtureEntry(path string) TableEntry {
	return Entry(strings.TrimSuffix(filepath.Base(path), ".json"), path)
}

func fixturePaths() []string {
	GinkgoHelper()
	paths := MustSucceed(
		filepath.Glob(filepath.Join("..", "testdata", "decode", "*.json")),
	)
	Expect(paths).ToNot(BeEmpty())
	return paths
}

func decodeEntries() []TableEntry {
	GinkgoHelper()
	return lo.Map(fixturePaths(), func(path string, _ int) TableEntry {
		return fixtureEntry(path)
	})
}

// sampleJSON is the JSON form of one sample of dt, as the fixtures write it.
func sampleJSON(dt telem.DataType, data []byte) any {
	GinkgoHelper()
	if dt.IsVariable() {
		data = data[4:]
	}
	switch dt {
	case telem.Float64T:
		return math.Float64frombits(telem.ByteOrder.Uint64(data))
	case telem.Float32T:
		return float64(math.Float32frombits(telem.ByteOrder.Uint32(data)))
	case telem.Int64T:
		return int64(telem.ByteOrder.Uint64(data))
	case telem.Int32T:
		return int32(telem.ByteOrder.Uint32(data))
	case telem.Int16T:
		return int16(telem.ByteOrder.Uint16(data))
	case telem.Int8T:
		return int8(data[0])
	case telem.Uint64T, telem.TimestampT:
		return telem.ByteOrder.Uint64(data)
	case telem.Uint32T:
		return telem.ByteOrder.Uint32(data)
	case telem.Uint16T:
		return telem.ByteOrder.Uint16(data)
	case telem.Uint8T:
		return data[0]
	case telem.BooleanT:
		return data[0] == 1
	case telem.StringT:
		return string(data)
	case telem.JSONT:
		return json.RawMessage(data)
	case telem.UUIDT:
		return uuid.UUID(data).String()
	case telem.BytesT:
		return base64.StdEncoding.EncodeToString(data)
	case telem.UnknownT:
	}
	Fail("unknown data type " + string(dt))
	return nil
}

func readDecodeFixture(path string) decodeFixture {
	GinkgoHelper()
	var f decodeFixture
	Expect(
		json.Unmarshal(MustSucceed(os.ReadFile(filepath.Clean(path))), &f),
	).To(Succeed())
	Expect(f.Description).ToNot(BeEmpty())
	return f
}

func (f decodeFixture) messages() []codec.JSONMessage {
	return lo.Map(f.Messages, func(m fixtureMessage, _ int) codec.JSONMessage {
		return codec.JSONMessage{
			Name:    m.Name,
			Payload: m.Payload,
			Bindings: lo.Map(m.Bindings, func(b binding, _ int) codec.Binding {
				return codec.Binding{Field: b.Field, DataType: b.DataType}
			}),
		}
	})
}

// newSamples returns empty sample slices for each binding of each message.
func newSamples(messages []codec.JSONMessage) [][][]byte {
	return lo.Map(messages, func(m codec.JSONMessage, _ int) [][]byte {
		return make([][]byte, len(m.Bindings))
	})
}

var _ = Describe("JSON", func() {
	DescribeTable("Should decode the documents of a fixture", func(path string) {
		f := readDecodeFixture(path)
		messages := f.messages()
		if f.Error != "" {
			Expect(codec.NewJSON(messages)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(f.Error),
			))
			return
		}
		dec := MustSucceed(codec.NewJSON(messages))
		Expect(f.Cases).ToNot(BeEmpty())
		for i, c := range f.Cases {
			By("case " + strconv.Itoa(i))
			doc := []byte(c.Document)
			if c.Text != nil {
				doc = []byte(*c.Text)
			}
			samples := newSamples(messages)
			switch {
			case c.Error != "":
				Expect(dec.Decode(doc, samples)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(c.Error),
				))
				Expect(samples).To(HaveEach(HaveEach(BeEmpty())))
			case c.Unmatched:
				_, ok := MustSucceed2(dec.Decode(doc, samples))
				Expect(ok).To(BeFalse())
				Expect(samples).To(HaveEach(HaveEach(BeEmpty())))
			default:
				m, ok := MustSucceed2(dec.Decode(doc, samples))
				Expect(ok).To(BeTrue())
				Expect(messages[m].Name).To(Equal(c.Message))
				bindings := messages[m].Bindings
				Expect(c.Samples).To(HaveLen(len(bindings)))
				for k, b := range bindings {
					actual := MustSucceed(
						json.Marshal(sampleJSON(b.DataType, samples[m][k])),
					)
					Expect(actual).To(MatchJSON(c.Samples[k]), "binding %d", k)
				}
				for other := range samples {
					if other != m {
						Expect(samples[other]).To(HaveEach(BeEmpty()))
					}
				}
			}
		}
	}, decodeEntries())

	It("Should reject a match rule with no type", func() {
		key := uuid.New()
		Expect(codec.NewJSON([]codec.JSONMessage{{
			Name: "status",
			Payload: icd.JSONPayload{
				Match: &icd.JSONMatch{},
				Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
					Key: key, Name: "v", Scale: 1, Pointer: "/v",
				}}},
			},
			Bindings: []codec.Binding{{Field: key, DataType: telem.Float64T}},
		}})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message status: match type is required: validation error"),
		))
	})

	It("Should append to the samples of earlier documents", func() {
		key := uuid.New()
		dec := MustSucceed(codec.NewJSON([]codec.JSONMessage{
			{
				Name: "reading",
				Payload: icd.JSONPayload{
					Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
						Key: key, Name: "v", Scale: 1, Pointer: "/v",
					}}},
				},
				Bindings: []codec.Binding{{Field: key, DataType: telem.Float64T}},
			},
		}))
		samples := [][][]byte{make([][]byte, 1)}
		for _, doc := range []string{`{"v": 1}`, `{"v": 2}`} {
			_, ok := MustSucceed2(dec.Decode([]byte(doc), samples))
			Expect(ok).To(BeTrue())
		}
		Expect(dec.Decode([]byte(`{"v": "x"}`), samples)).
			Error().To(MatchError(validate.ErrValidation))
		Expect(telem.Series{DataType: telem.Float64T, Data: samples[0][0]}).
			To(telem.MatchSeriesDataV[float64](1, 2))
	})

	DescribeTable("Should decode a document without allocating",
		func(first, second icd.JSONMatch, doc []byte) {
			temp, count, state, other := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			messages := []codec.JSONMessage{
				{
					Name: "other",
					Payload: icd.JSONPayload{
						Match: &first,
						Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
							Key: other, Name: "other", Scale: 1, Pointer: "/other",
						}}},
					},
					Bindings: []codec.Binding{{Field: other, DataType: telem.Float64T}},
				},
				{
					Name: "status",
					Payload: icd.JSONPayload{
						Match: &second,
						Fields: []icd.JSONField{
							{Variant: icd.FloatJSONField{
								Key:     temp,
								Name:    "temp",
								Scale:   0.5,
								Offset:  -10,
								Pointer: "/temp",
							}},
							{Variant: icd.IntegerJSONField{
								Key: count, Name: "count", Scale: 1, Pointer: "/count",
							}},
							{Variant: icd.StringJSONField{
								Key: state, Name: "state", Scale: 1, Pointer: "/state",
							}},
						},
					},
					Bindings: []codec.Binding{
						{Field: temp, DataType: telem.Float32T},
						{Field: count, DataType: telem.Float64T},
						{Field: state, DataType: telem.StringT},
					},
				},
			}
			dec := MustSucceed(codec.NewJSON(messages))
			samples := newSamples(messages)
			var (
				message int
				ok      bool
				err     error
			)
			decode := func() {
				for i := range samples[1] {
					samples[1][i] = samples[1][i][:0]
				}
				message, ok, err = dec.Decode(doc, samples)
			}
			Expect(testing.AllocsPerRun(100, decode)).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(message).To(Equal(1))
		},
		Entry("by a string",
			icd.JSONMatch{Variant: icd.StringJSONMatch{
				JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/type"}, Value: "event",
			}},
			icd.JSONMatch{Variant: icd.StringJSONMatch{
				JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/type"}, Value: "status",
			}},
			[]byte(`{"type":"status","temp":40,"count":3,"state":"run"}`),
		),
		Entry("by a number",
			icd.JSONMatch{Variant: icd.FloatJSONMatch{
				JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/id"}, Value: 2.5,
			}},
			icd.JSONMatch{Variant: icd.IntegerJSONMatch{
				JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/id"}, Value: 7,
			}},
			[]byte(`{"id":7,"temp":40,"count":3,"state":"run"}`),
		),
	)
})

var _ = Describe("Converts", func() {
	DescribeTable(
		"Should report whether a field kind converts to a data type",
		func(k payload.Kind, dt telem.DataType, expected bool) {
			Expect(codec.Converts(k, dt)).To(Equal(expected))
		},
		Entry("a kind to its own data type", payload.KindString, telem.StringT, true),
		Entry("an integer to a float", payload.KindInteger, telem.Float32T, true),
		Entry("an unsigned to a float", payload.KindUnsigned, telem.Float64T, true),
		Entry("a boolean to a float", payload.KindBoolean, telem.Float64T, true),
		Entry("an unsigned to an integer", payload.KindUnsigned, telem.Int64T, true),
		Entry("a boolean to an integer", payload.KindBoolean, telem.Int8T, true),
		Entry("a boolean to an unsigned", payload.KindBoolean, telem.Uint8T, true),
		Entry("a float to an integer", payload.KindFloat, telem.Int32T, false),
		Entry("an integer to an unsigned", payload.KindInteger, telem.Uint32T, false),
		Entry("a timestamp to an integer", payload.KindTimestamp, telem.Int64T, false),
		Entry("a string to a float", payload.KindString, telem.Float64T, false),
		Entry(
			"a kind to an unknown data type",
			payload.KindFloat,
			telem.DataType("complex128"),
			false,
		),
	)
})
