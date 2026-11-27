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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// unprefixed returns sample without the length prefix of a variable-density sample.
func unprefixed(dt telem.DataType, sample []byte) []byte {
	if !dt.IsVariable() {
		return sample
	}
	data, _ := telem.UnmarshalVariableSample(sample)
	return data
}

// finite reports whether c writes each float sample of m as a number. The fixtures
// write NaN and infinity as strings, as JSON holds neither.
func finite(m codec.JSONMessage, c decodeCase) bool {
	for k, b := range m.Bindings {
		float := b.DataType == telem.Float64T || b.DataType == telem.Float32T
		if float && c.Samples[k][0] == '"' {
			return false
		}
	}
	return true
}

// documentEntries returns an entry for each decode fixture that holds documents.
func documentEntries() []TableEntry {
	GinkgoHelper()
	var entries []TableEntry
	for _, path := range fixturePaths("decode") {
		if readDecodeFixture(path).Error == "" {
			entries = append(entries, fixtureEntry(path))
		}
	}
	return entries
}

// encodeFixture is one JSON message from testdata/encode and the documents it encodes.
// A fixture with an error names the error NewJSONEncoder must return, and has no cases.
type encodeFixture struct {
	Description string         `json:"description"`
	Message     fixtureMessage `json:"message"`
	Error       string         `json:"error"`
	Cases       []encodeCase   `json:"cases"`
}

// encodeCase holds one JSON value for each binding, or null for a missing sample. A
// case that does not encode names the error, which Check also returns for the one
// sample that its field cannot hold. Otherwise document is the document of the samples,
// or text is its exact text.
type encodeCase struct {
	Samples  []jsontext.Value `json:"samples"`
	Document jsontext.Value   `json:"document"`
	Text     *string          `json:"text"`
	Error    string           `json:"error"`
}

func readEncodeFixture(path string) encodeFixture {
	GinkgoHelper()
	var f encodeFixture
	Expect(
		json.Unmarshal(MustSucceed(os.ReadFile(filepath.Clean(path))), &f),
	).To(Succeed())
	Expect(f.Description).ToNot(BeEmpty())
	return f
}

// sampleData is the sample of dt that a fixture writes as raw, or nil for null.
func sampleData(dt telem.DataType, raw jsontext.Value) []byte {
	GinkgoHelper()
	if raw.Kind() == 'n' {
		return nil
	}
	if dt == telem.BytesT {
		var text string
		Expect(json.Unmarshal(raw, &text)).To(Succeed())
		return MustSucceed(base64.StdEncoding.DecodeString(text))
	}
	var v any
	// An integer reads without a float64, which holds only 53 bits.
	switch kind, _ := payload.KindOf(dt); {
	case raw.Kind() == '0' && kind == payload.KindUnsigned:
		v = MustSucceed(strconv.ParseUint(string(raw), 10, 64))
	case raw.Kind() == '0' &&
		(kind == payload.KindInteger || kind == payload.KindTimestamp):
		v = MustSucceed(strconv.ParseInt(string(raw), 10, 64))
	default:
		Expect(json.Unmarshal(raw, &v)).To(Succeed())
	}
	return unprefixed(dt, telem.NewSeriesFromAny(v, dt).Data)
}

func float64Sample(v float64) []byte {
	return telem.NewSeriesV(v).Data
}

func floatField(name string, key icd.FieldKey, scale, offset float64) icd.JSONField {
	return icd.JSONField{Variant: icd.FloatJSONField{
		Key: key, Name: name, Scale: scale, Offset: offset, Pointer: "/" + name,
	}}
}

// nanField returns a float field at /name whose default is NaN.
func nanField(name string, key icd.FieldKey) icd.JSONField {
	return icd.JSONField{Variant: icd.FloatJSONField{
		Key:     key,
		Name:    name,
		Scale:   1,
		Pointer: "/" + name,
		Default: new(payload.Float(math.NaN())),
	}}
}

func message(fields []icd.JSONField, bindings ...codec.Binding) codec.JSONMessage {
	return codec.JSONMessage{
		Name:     "command",
		Payload:  icd.JSONPayload{Fields: fields},
		Bindings: bindings,
	}
}

var _ = Describe("JSONEncoder", func() {
	DescribeTable(
		"Should encode the samples of a fixture into documents that decode to them",
		func(path string) {
			f := readDecodeFixture(path)
			messages := f.messages()
			dec := MustSucceed(codec.NewJSON(messages))
			for i, m := range messages {
				By("message " + m.Name)
				enc := MustSucceed(codec.NewJSONEncoder(m))
				for k, c := range f.Cases {
					if c.Error != "" || c.Unmatched || c.Message != m.Name {
						continue
					}
					By("case " + strconv.Itoa(k))
					doc := []byte(c.Document)
					if c.Text != nil {
						doc = []byte(*c.Text)
					}
					decoded := newSamples(messages)
					decode(dec, doc, decoded)
					samples := make([][]byte, len(m.Bindings))
					for b, binding := range m.Bindings {
						samples[b] = unprefixed(binding.DataType, decoded[i][b])
					}
					if !finite(m, c) {
						Expect(enc.Encode(nil, samples)).Error().To(SatisfyAll(
							MatchError(validate.ErrValidation),
							MatchError(ContainSubstring("is not a JSON number")),
						))
						continue
					}
					encoded := MustSucceed(enc.Encode(nil, samples))
					again := newSamples(messages)
					routed, ok := decode(dec, encoded, again)
					Expect(ok).To(BeTrue(), string(encoded))
					Expect(routed).To(Equal(i), string(encoded))
					Expect(again[i]).To(Equal(decoded[i]), string(encoded))
				}
			}
		},
		documentEntries(),
	)

	DescribeTable("Should encode the cases of a fixture", func(path string) {
		f := readEncodeFixture(path)
		m := f.Message.jsonMessage()
		if f.Error != "" {
			Expect(codec.NewJSONEncoder(m)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(f.Error),
			))
			return
		}
		enc := MustSucceed(codec.NewJSONEncoder(m))
		Expect(f.Cases).ToNot(BeEmpty())
		for i, c := range f.Cases {
			By("case " + strconv.Itoa(i))
			Expect(c.Samples).To(HaveLen(len(m.Bindings)))
			samples := make([][]byte, len(c.Samples))
			var rejected []error
			for k, raw := range c.Samples {
				samples[k] = sampleData(m.Bindings[k].DataType, raw)
				if samples[k] == nil {
					continue
				}
				if err := enc.Check(k, samples[k]); err != nil {
					rejected = append(rejected, err)
				}
			}
			if c.Error != "" {
				Expect(enc.Encode(nil, samples)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(c.Error),
				))
				Expect(rejected).To(HaveExactElements(MatchError(c.Error)))
				continue
			}
			Expect(rejected).To(BeEmpty())
			doc := MustSucceed(enc.Encode(nil, samples))
			if c.Text != nil {
				Expect(string(doc)).To(Equal(*c.Text))
			} else {
				Expect(doc).To(MatchJSON([]byte(c.Document)))
			}
		}
	}, fixtureEntries("encode"))

	It("Should write the time of each encode for a now default", func() {
		enc := MustSucceed(codec.NewJSONEncoder(message([]icd.JSONField{
			{Variant: icd.TimestampJSONField{
				Key:        uuid.New(),
				Name:       "v",
				Scale:      1,
				Pointer:    "/v",
				TimeFormat: payload.TimeFormatUnixNs,
				Default:    &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
			}},
		})))
		timeOf := func() telem.TimeStamp {
			doc := MustSucceed(enc.Encode(nil, nil))
			var v struct {
				V int64 `json:"v"`
			}
			Expect(json.Unmarshal(doc, &v)).To(Succeed())
			return telem.TimeStamp(v.V)
		}
		before := telem.Now()
		first := timeOf()
		second := timeOf()
		Expect(first).To(BeNumerically(">=", before))
		Expect(second).To(BeNumerically(">=", first))
		Expect(second).To(BeNumerically("<=", telem.Now()))
	})

	It("Should write a new UUID at each encode for a random default", func() {
		enc := MustSucceed(codec.NewJSONEncoder(message([]icd.JSONField{
			{Variant: icd.UuidJSONField{
				Key:     uuid.New(),
				Name:    "v",
				Scale:   1,
				Pointer: "/v",
				Default: &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			}},
		})))
		idOf := func() uuid.UUID {
			GinkgoHelper()
			doc := MustSucceed(enc.Encode(nil, nil))
			var v struct {
				V string `json:"v"`
			}
			Expect(json.Unmarshal(doc, &v)).To(Succeed())
			return MustSucceed(uuid.Parse(v.V))
		}
		first, second := idOf(), idOf()
		Expect(first).ToNot(Equal(uuid.Nil()))
		Expect(second).ToNot(Equal(first))
	})

	It("Should report which bindings have a default", func() {
		mode, temp := uuid.New(), uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(message(
			[]icd.JSONField{
				{Variant: icd.StringJSONField{
					Key:     mode,
					Name:    "mode",
					Scale:   1,
					Pointer: "/mode",
					Default: new("auto"),
				}},
				floatField("temp", temp, 1, 0),
			},
			codec.Binding{Field: mode, DataType: telem.StringT},
			codec.Binding{Field: temp, DataType: telem.Float64T},
		)))
		Expect(enc.HasDefault(0)).To(BeTrue())
		Expect(enc.HasDefault(1)).To(BeFalse())
	})

	It("Should write a new UUID at each encode for a bound field with a nil sample",
		func() {
			key := uuid.New()
			enc := MustSucceed(codec.NewJSONEncoder(message(
				[]icd.JSONField{{Variant: icd.UuidJSONField{
					Key:     key,
					Name:    "v",
					Scale:   1,
					Pointer: "/v",
					Default: &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
				}}},
				codec.Binding{Field: key, DataType: telem.UUIDT},
			)))
			idOf := func() uuid.UUID {
				GinkgoHelper()
				doc := MustSucceed(enc.Encode(nil, [][]byte{nil}))
				var v struct {
					V string `json:"v"`
				}
				Expect(json.Unmarshal(doc, &v)).To(Succeed())
				return MustSucceed(uuid.Parse(v.V))
			}
			Expect(idOf()).ToNot(Equal(idOf()))
		},
	)

	It("Should fail to encode a nil sample whose field has no default", func() {
		key := uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(message(
			[]icd.JSONField{floatField("temp", key, 1, 0)},
			codec.Binding{Field: key, DataType: telem.Float64T},
		)))
		Expect(enc.Encode(nil, [][]byte{nil})).Error().
			To(MatchError("message command: field temp: no sample and no default"))
	})

	It("Should refuse a NaN default for a field with no binding", func() {
		f := nanField("v", uuid.New())
		Expect(codec.NewJSONEncoder(message([]icd.JSONField{f}))).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message command: field v: default: NaN is not a JSON number: "+
				"validation error"),
		))
	})

	It("Should refuse a bound field with a zero scale", func() {
		key := uuid.New()
		Expect(codec.NewJSONEncoder(message(
			[]icd.JSONField{floatField("temp", key, 0, 0)},
			codec.Binding{Field: key, DataType: telem.Float64T},
		))).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(
				"message command: field temp: scale must be non-zero: validation error",
			),
		))
	})

	It("Should write the default of an unbound field with a zero scale", func() {
		f := icd.JSONField{Variant: icd.FloatJSONField{
			Key: uuid.New(), Name: "v", Pointer: "/v", Default: new(payload.Float(3)),
		}}
		enc := MustSucceed(codec.NewJSONEncoder(message([]icd.JSONField{f})))
		Expect(enc.Encode(nil, nil)).To(MatchJSON(`{"v":3}`))
	})

	It("Should fail only the encode of a nil sample for a bound NaN default", func() {
		key := uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(message(
			[]icd.JSONField{nanField("v", key)},
			codec.Binding{Field: key, DataType: telem.Float64T},
		)))
		Expect(enc.HasDefault(0)).To(BeTrue())
		Expect(enc.Encode(nil, [][]byte{float64Sample(2.5)})).To(MatchJSON(`{"v":2.5}`))
		Expect(enc.Encode(nil, [][]byte{nil})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message command: field v: default: NaN is not a JSON number: "+
				"validation error"),
		))
	})

	DescribeTable("Should write a generated default beside a match rule",
		func(match icd.JSONMatch, expectedType string) {
			enc := MustSucceed(codec.NewJSONEncoder(codec.JSONMessage{
				Name: "status",
				Payload: icd.JSONPayload{
					Match: &match,
					Fields: []icd.JSONField{{Variant: icd.UuidJSONField{
						Key:     uuid.New(),
						Name:    "id",
						Scale:   1,
						Pointer: "/id",
						Default: &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
					}}},
				},
			}))
			var doc struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			}
			Expect(
				json.Unmarshal(MustSucceed(enc.Encode(nil, nil)), &doc),
			).To(Succeed())
			Expect(doc.Type).To(Equal(expectedType))
			Expect(MustSucceed(uuid.Parse(doc.ID))).ToNot(Equal(uuid.Nil()))
		},
		Entry("a rule that writes its value",
			icd.JSONMatch{Variant: icd.StringJSONMatch{
				JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/type"},
				Value:         "status",
			}},
			"status",
		),
		Entry("a rule on a present field",
			icd.JSONMatch{Variant: icd.PresentJSONMatch{
				JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/id"},
			}},
			"",
		),
	)

	It("Should reject a NaN sample for a float field", func() {
		key := uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(message(
			[]icd.JSONField{floatField("v", key, 2, 0)},
			codec.Binding{Field: key, DataType: telem.Float64T},
		)))
		sample := float64Sample(math.NaN())
		expected := "message command: field v: NaN is not a JSON number: " +
			"validation error"
		Expect(enc.Encode(nil, [][]byte{sample})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(expected),
		))
		Expect(enc.Check(0, sample)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(expected),
		))
	})

	Describe("Check", func() {
		var (
			key = uuid.New()
			enc *codec.JSONEncoder
		)
		BeforeEach(func() {
			enc = MustSucceed(codec.NewJSONEncoder(message(
				[]icd.JSONField{floatField("v", key, 1, 0)},
				codec.Binding{Field: key, DataType: telem.Float64T},
			)))
		})

		It("Should encode a sample that differs from the checked sample", func() {
			Expect(enc.Check(0, float64Sample(1.5))).To(Succeed())
			Expect(enc.Encode(nil, [][]byte{float64Sample(2.5)})).
				To(MatchJSON(`{"v":2.5}`))
		})

		It("Should encode a sample whose buffer changed after its check", func() {
			sample := float64Sample(1.5)
			Expect(enc.Check(0, sample)).To(Succeed())
			copy(sample, float64Sample(2.5))
			Expect(enc.Encode(nil, [][]byte{sample})).To(MatchJSON(`{"v":2.5}`))
		})

		It("Should keep the text of the last good sample after a failed check", func() {
			expected := "message command: field v: NaN is not a JSON number: " +
				"validation error"
			Expect(enc.Check(0, float64Sample(1.5))).To(Succeed())
			Expect(enc.Check(0, float64Sample(math.NaN()))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
			Expect(enc.Encode(nil, [][]byte{float64Sample(1.5)})).
				To(MatchJSON(`{"v":1.5}`))
			Expect(enc.Encode(nil, [][]byte{float64Sample(math.NaN())})).Error().
				To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(expected)))
		})

		It("Should encode an empty string after the check of a nil sample", func() {
			enc := MustSucceed(codec.NewJSONEncoder(message(
				[]icd.JSONField{{Variant: icd.StringJSONField{
					Key:     key,
					Name:    "s",
					Scale:   1,
					Pointer: "/s",
					Default: new("synnax"),
				}}},
				codec.Binding{Field: key, DataType: telem.StringT},
			)))
			Expect(enc.Check(0, nil)).To(Succeed())
			Expect(enc.Encode(nil, [][]byte{{}})).To(MatchJSON(`{"s":""}`))
			Expect(enc.Encode(nil, [][]byte{nil})).To(MatchJSON(`{"s":"synnax"}`))
		})
	})

	Describe("JSON samples", func() {
		var enc *codec.JSONEncoder
		BeforeEach(func() {
			key := uuid.New()
			enc = MustSucceed(codec.NewJSONEncoder(message(
				[]icd.JSONField{{Variant: icd.JSONJSONField{
					Key: key, Name: "config", Scale: 1, Pointer: "/config",
				}}},
				codec.Binding{Field: key, DataType: telem.JSONT},
			)))
		})

		It("Should write a JSON sample as it is", func() {
			Expect(enc.Encode(nil, [][]byte{[]byte(`{"rate":10}`)})).
				To(MatchJSON(`{"config":{"rate":10}}`))
		})

		DescribeTable("Should refuse a JSON sample that is not valid",
			func(sample string) {
				Expect(enc.Encode(nil, [][]byte{[]byte(sample)})).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						"message command: field config: invalid JSON: validation error",
					),
				))
			},
			Entry("truncated text", `{"rate"`),
			Entry("a repeated name", `{"rate":1,"rate":2}`),
		)
	})

	DescribeTable("Should reject a message whose rule or default has no type",
		func(m codec.JSONMessage, expected string) {
			Expect(codec.NewJSONEncoder(m)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry("a timestamp default with no type",
			message([]icd.JSONField{{Variant: icd.TimestampJSONField{
				Key:        uuid.New(),
				Name:       "at",
				Scale:      1,
				Pointer:    "/at",
				TimeFormat: payload.TimeFormatUnixNs,
				Default:    &icd.TimestampDefault{},
			}}}),
			"message command: field at: default: type is required: validation error",
		),
		Entry("a UUID default with no type",
			message([]icd.JSONField{{Variant: icd.UuidJSONField{
				Key:     uuid.New(),
				Name:    "id",
				Scale:   1,
				Pointer: "/id",
				Default: &icd.UUIDDefault{},
			}}}),
			"message command: field id: default: type is required: validation error",
		),
		Entry("a bound field whose default has no type",
			message(
				[]icd.JSONField{{Variant: icd.UuidJSONField{
					Key:     fixedKey,
					Name:    "id",
					Scale:   1,
					Pointer: "/id",
					Default: &icd.UUIDDefault{},
				}}},
				codec.Binding{Field: fixedKey, DataType: telem.UUIDT},
			),
			"message command: field id: default: type is required: validation error",
		),
		Entry("a match rule with no type",
			codec.JSONMessage{
				Name:    "command",
				Payload: icd.JSONPayload{Match: &icd.JSONMatch{}},
			},
			"message command: match: type is required: validation error",
		),
	)

	It("Should encode a document without allocating", func() {
		temp, count, state := uuid.New(), uuid.New(), uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(codec.JSONMessage{
			Name: "status",
			Payload: icd.JSONPayload{
				Match: &icd.JSONMatch{Variant: icd.IntegerJSONMatch{
					JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/id"}, Value: 7,
				}},
				Fields: []icd.JSONField{
					floatField("temp", temp, 0.5, -10),
					{Variant: icd.IntegerJSONField{
						Key: count, Name: "count", Scale: 1, Pointer: "/count",
					}},
					{Variant: icd.StringJSONField{
						Key: state, Name: "state", Scale: 1, Pointer: "/state",
					}},
				},
			},
			Bindings: []codec.Binding{
				{Field: temp, DataType: telem.Float64T},
				{Field: count, DataType: telem.Float64T},
				{Field: state, DataType: telem.StringT},
			},
		}))
		samples := [][]byte{float64Sample(10), float64Sample(3), []byte("run")}
		dst := make([]byte, 0, 256)
		var err error
		encode := func() { dst, err = enc.Encode(dst[:0], samples) }
		Expect(testing.AllocsPerRun(100, encode)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
		Expect(dst).To(MatchJSON(`{"id":7,"temp":40,"count":3,"state":"run"}`))
	})

	It("Should write a now default without allocating", func() {
		enc := MustSucceed(codec.NewJSONEncoder(message([]icd.JSONField{
			{Variant: icd.TimestampJSONField{
				Key:        uuid.New(),
				Name:       "at",
				Scale:      1,
				Pointer:    "/at",
				TimeFormat: payload.TimeFormatISO8601,
				Default:    &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
			}},
		})))
		dst := make([]byte, 0, 256)
		var err error
		encode := func() { dst, err = enc.Encode(dst[:0], nil) }
		Expect(testing.AllocsPerRun(100, encode)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should write a random default without allocating", func() {
		if raceEnabled {
			Skip("uuid.NewV4 allocates under the race detector")
		}
		enc := MustSucceed(codec.NewJSONEncoder(message([]icd.JSONField{
			{Variant: icd.UuidJSONField{
				Key:     uuid.New(),
				Name:    "id",
				Scale:   1,
				Pointer: "/id",
				Default: &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			}},
		})))
		dst := make([]byte, 0, 256)
		var err error
		encode := func() { dst, err = enc.Encode(dst[:0], nil) }
		Expect(testing.AllocsPerRun(100, encode)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
	})
})

var fixedKey = uuid.MustParse("00000000-0000-4000-8000-0000000000aa")

var _ = Describe("CheckDefault", func() {
	DescribeTable("Should accept a default that its field can send",
		func(v icd.JSONFieldVariant) {
			Expect(codec.CheckDefault(icd.JSONField{Variant: v})).To(Succeed())
		},
		Entry("no default", icd.JSONJSONField{}),
		Entry("JSON text", icd.JSONJSONField{Default: new(`{"max":10}`)}),
		Entry("base64 text", icd.BytesJSONField{Default: new("AQI=")}),
		Entry("a finite float", icd.FloatJSONField{Default: new(payload.Float(2.5))}),
	)

	DescribeTable("Should refuse a default without naming it",
		func(v icd.JSONFieldVariant, expected string) {
			Expect(codec.CheckDefault(icd.JSONField{Variant: v})).
				To(MatchError(expected))
		},
		Entry("a default that icd.CheckDefault refuses",
			icd.FloatJSONField{
				Scale: 1, Max: new(100.0), Default: new(payload.Float(120)),
			},
			"120 is above the max 100: validation error",
		),
		Entry("a NaN float, which JSON cannot hold",
			icd.FloatJSONField{Scale: 1, Default: new(payload.Float(math.NaN()))},
			"NaN is not a JSON number: validation error",
		),
		Entry("a timestamp default with no type",
			icd.TimestampJSONField{
				TimeFormat: payload.TimeFormatUnixNs,
				Default:    &icd.TimestampDefault{},
			},
			"type is required: validation error",
		),
		Entry("a UUID default with no type",
			icd.UuidJSONField{Default: &icd.UUIDDefault{}},
			"type is required: validation error",
		),
	)
})

var _ = Describe("CheckWrite", func() {
	DescribeTable(
		"Should accept samples that can write a field",
		func(k payload.Kind, scaled bool, dt telem.DataType) {
			Expect(codec.CheckWrite(k, scaled, dt)).To(Succeed())
		},
		Entry("a channel of the kind of the field",
			payload.KindString, false, telem.StringT,
		),
		Entry("a number channel to a field of another number kind",
			payload.KindUnsigned, false, telem.Float64T,
		),
		Entry("a boolean channel to a number field",
			payload.KindFloat, false, telem.BooleanT,
		),
		Entry(
			"a float channel to a scaled field",
			payload.KindInteger,
			true,
			telem.Float32T,
		),
	)

	DescribeTable("Should refuse samples that cannot write a field",
		func(k payload.Kind, scaled bool, dt telem.DataType, expected string) {
			Expect(codec.CheckWrite(k, scaled, dt)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry("a channel whose kind does not write the field",
			payload.KindFloat, false, telem.StringT,
			"cannot write string channels to float fields: validation error",
		),
		Entry("a number channel to a string field",
			payload.KindString, false, telem.Int32T,
			"cannot write int32 channels to string fields: validation error",
		),
		Entry("an integer channel to a scaled field",
			payload.KindInteger, true, telem.Int64T,
			"a scaled field requires a float channel, not int64: validation error",
		),
	)
})
