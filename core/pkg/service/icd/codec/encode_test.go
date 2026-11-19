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
	"encoding/json/v2"
	"math"
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

// documentEntries returns an entry for each decode fixture that holds documents.
func documentEntries() []TableEntry {
	GinkgoHelper()
	var entries []TableEntry
	for _, path := range fixturePaths() {
		if readDecodeFixture(path).Error == "" {
			entries = append(entries, fixtureEntry(path))
		}
	}
	return entries
}

func float64Sample(v float64) []byte {
	return telem.NewSeriesV(v).Data
}

func int64Sample(v int64) []byte {
	return telem.NewSeriesV(v).Data
}

func floatField(name string, key icd.FieldKey, scale, offset float64) icd.JSONField {
	return icd.JSONField{Variant: icd.FloatJSONField{
		Key: key, Name: name, Scale: scale, Offset: offset, Pointer: "/" + name,
	}}
}

func booleanField() icd.JSONField {
	return icd.JSONField{Variant: icd.BooleanJSONField{
		Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
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
					MustSucceed2(dec.Decode(doc, decoded))
					samples := make([][]byte, len(m.Bindings))
					for b, binding := range m.Bindings {
						samples[b] = unprefixed(binding.DataType, decoded[i][b])
					}
					encoded := MustSucceed(enc.Encode(nil, samples))
					again := newSamples(messages)
					routed, ok := MustSucceed2(dec.Decode(encoded, again))
					Expect(ok).To(BeTrue(), string(encoded))
					Expect(routed).To(Equal(i), string(encoded))
					Expect(again[i]).To(Equal(decoded[i]), string(encoded))
				}
			}
		},
		documentEntries(),
	)

	It("Should write the match value, each bound field, and each default", func() {
		temp, mode, count := uuid.New(), uuid.New(), uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(codec.JSONMessage{
			Name: "command",
			Payload: icd.JSONPayload{
				Match: &icd.JSONMatch{Variant: icd.StringJSONMatch{
					JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/type"},
					Value:         "set",
				}},
				Fields: []icd.JSONField{
					{Variant: icd.FloatJSONField{
						Key: temp, Name: "temp", Scale: 1, Pointer: "/target/temp",
					}},
					{Variant: icd.StringJSONField{
						Key:     mode,
						Name:    "mode",
						Scale:   1,
						Pointer: "/target/mode",
						Default: new("auto"),
					}},
					{Variant: icd.IntegerJSONField{
						Key: count, Name: "count", Scale: 1, Pointer: "/count",
					}},
				},
			},
			Bindings: []codec.Binding{
				{Field: count, DataType: telem.Int64T},
				{Field: temp, DataType: telem.Float64T},
			},
		}))
		Expect(enc.Encode(nil, [][]byte{int64Sample(3), float64Sample(21.5)})).To(
			MatchJSON(`{"type":"set","target":{"temp":21.5,"mode":"auto"},"count":3}`),
		)
	})

	DescribeTable("Should write the default of a field with no channel",
		func(field icd.JSONField, expected string) {
			enc := MustSucceed(codec.NewJSONEncoder(message([]icd.JSONField{field})))
			Expect(enc.Encode(nil, nil)).To(MatchJSON(expected))
		},
		Entry("a float, without its scale",
			icd.JSONField{Variant: icd.FloatJSONField{
				Key:     uuid.New(),
				Name:    "v",
				Scale:   2,
				Pointer: "/v",
				Default: new(payload.Float(1.5)),
			}},
			`{"v":1.5}`,
		),
		Entry("an integer",
			icd.JSONField{Variant: icd.IntegerJSONField{
				Key:     uuid.New(),
				Name:    "v",
				Scale:   1,
				Pointer: "/v",
				Default: new(payload.Integer(-3)),
			}},
			`{"v":-3}`,
		),
		Entry("an unsigned integer",
			icd.JSONField{Variant: icd.UnsignedJSONField{
				Key:     uuid.New(),
				Name:    "v",
				Scale:   1,
				Pointer: "/v",
				Default: new(payload.Unsigned(7)),
			}},
			`{"v":7}`,
		),
		Entry("a boolean",
			icd.JSONField{Variant: icd.BooleanJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v", Default: new(true),
			}},
			`{"v":true}`,
		),
		Entry("a timestamp in its time format",
			icd.JSONField{Variant: icd.TimestampJSONField{
				Key:        uuid.New(),
				Name:       "v",
				Scale:      1,
				Pointer:    "/v",
				TimeFormat: payload.TimeFormatUnixNs,
				Default: &icd.TimestampDefault{Variant: icd.ConstantTimestampDefault{
					Value: 1700000000000000000,
				}},
			}},
			`{"v":1700000000000000000}`,
		),
		Entry("any JSON value",
			icd.JSONField{Variant: icd.JSONJSONField{
				Key:     uuid.New(),
				Name:    "v",
				Scale:   1,
				Pointer: "/v",
				Default: new(any(map[string]any{"set_point": 1.5, "mode": "auto"})),
			}},
			`{"v":{"set_point":1.5,"mode":"auto"}}`,
		),
		Entry("a UUID",
			icd.JSONField{Variant: icd.UuidJSONField{
				Key:     uuid.New(),
				Name:    "v",
				Scale:   1,
				Pointer: "/v",
				Default: &icd.UUIDDefault{Variant: icd.ConstantUUIDDefault{
					Value: uuid.MustParse("00000000-0000-4000-8000-0000000000aa"),
				}},
			}},
			`{"v":"00000000-0000-4000-8000-0000000000aa"}`,
		),
		Entry("bytes as base64",
			icd.JSONField{Variant: icd.BytesJSONField{
				Key:     uuid.New(),
				Name:    "v",
				Scale:   1,
				Pointer: "/v",
				Default: new("AQI="),
			}},
			`{"v":"AQI="}`,
		),
	)

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

	It("Should write a float before its scale and offset", func() {
		key := uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(message(
			[]icd.JSONField{floatField("temp", key, 0.5, -10)},
			codec.Binding{Field: key, DataType: telem.Float64T},
		)))
		Expect(enc.Encode(nil, [][]byte{float64Sample(10)})).
			To(MatchJSON(`{"temp":40}`))
	})

	DescribeTable("Should write a number channel to a field of another number kind",
		func(field icd.JSONField, dt telem.DataType, sample []byte, expected string) {
			enc := MustSucceed(codec.NewJSONEncoder(message(
				[]icd.JSONField{field},
				codec.Binding{Field: field.Base().Key, DataType: dt},
			)))
			Expect(enc.Encode(nil, [][]byte{sample})).To(MatchJSON(expected))
		},
		Entry("an integer to a float field",
			floatField("v", uuid.New(), 1, 0),
			telem.Int64T, int64Sample(3), `{"v":3}`,
		),
		Entry("an unsigned integer to an integer field",
			icd.JSONField{Variant: icd.IntegerJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
			}},
			telem.Uint8T, telem.NewSeriesV[uint8](7).Data, `{"v":7}`,
		),
		Entry("a boolean to an unsigned field",
			icd.JSONField{Variant: icd.UnsignedJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
			}},
			telem.BooleanT, telem.NewSeriesV(true).Data, `{"v":1}`,
		),
		Entry("a 1 to a boolean field",
			booleanField(),
			telem.Uint8T, telem.NewSeriesV[uint8](1).Data, `{"v":true}`,
		),
		Entry("a 0 to a boolean field",
			booleanField(),
			telem.Float64T, float64Sample(0), `{"v":false}`,
		),
	)

	DescribeTable("Should round the value of an integer field",
		func(field icd.JSONField, sample float64, expected string) {
			enc := MustSucceed(codec.NewJSONEncoder(message(
				[]icd.JSONField{field},
				codec.Binding{Field: field.Base().Key, DataType: telem.Float64T},
			)))
			Expect(enc.Encode(nil, [][]byte{float64Sample(sample)})).
				To(MatchJSON(expected))
		},
		Entry("an integer",
			icd.JSONField{Variant: icd.IntegerJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
			}},
			-2.5, `{"v":-3}`,
		),
		Entry("a scaled unsigned integer",
			icd.JSONField{Variant: icd.UnsignedJSONField{
				Key: uuid.New(), Name: "v", Scale: 0.1, Pointer: "/v",
			}},
			1.26, `{"v":13}`,
		),
	)

	It("Should write a value that an enum labels as the label", func() {
		key := uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(message(
			[]icd.JSONField{{Variant: icd.IntegerJSONField{
				Key:     key,
				Name:    "state",
				Scale:   2,
				Pointer: "/state",
				EnumValues: []payload.IntegerEntry{
					{Label: "idle", Value: 0}, {Label: "run", Value: 1},
				},
			}}},
			codec.Binding{Field: key, DataType: telem.Float64T},
		)))
		Expect(enc.Encode(nil, [][]byte{float64Sample(1)})).
			To(MatchJSON(`{"state":"run"}`))
		Expect(enc.Encode(nil, [][]byte{float64Sample(8)})).
			To(MatchJSON(`{"state":4}`))
	})

	It("Should write the sample of a JSON field as it is", func() {
		key := uuid.New()
		enc := MustSucceed(codec.NewJSONEncoder(message(
			[]icd.JSONField{{Variant: icd.JSONJSONField{
				Key: key, Name: "config", Scale: 1, Pointer: "/config",
			}}},
			codec.Binding{Field: key, DataType: telem.JSONT},
		)))
		Expect(enc.Encode(nil, [][]byte{[]byte(`{"rate":10}`)})).
			To(MatchJSON(`{"config":{"rate":10}}`))
		Expect(enc.Encode(nil, [][]byte{[]byte(`{"rate"`)})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("message command: field config: invalid JSON: validation error"),
		))
	})

	DescribeTable("Should reject a sample that its field cannot hold",
		func(field icd.JSONField, dt telem.DataType, sample []byte, expected string) {
			enc := MustSucceed(codec.NewJSONEncoder(message(
				[]icd.JSONField{field},
				codec.Binding{Field: field.Base().Key, DataType: dt},
			)))
			Expect(enc.Encode(nil, [][]byte{sample})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry("an integer past an int64",
			icd.JSONField{Variant: icd.IntegerJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
			}},
			telem.Float64T, float64Sample(1e19),
			"message command: field v: 1e+19 is out of range for integer values: "+
				"validation error",
		),
		Entry("a negative unsigned integer",
			icd.JSONField{Variant: icd.UnsignedJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
			}},
			telem.Int64T, int64Sample(-1),
			"message command: field v: -1 is out of range for unsigned values: "+
				"validation error",
		),
		Entry("a NaN float",
			floatField("v", uuid.New(), 2, 0),
			telem.Float64T, float64Sample(math.NaN()),
			"message command: field v: NaN is not a JSON number: validation error",
		),
		Entry("a number other than 0 or 1 for a boolean field",
			booleanField(),
			telem.Float64T, float64Sample(2.5),
			"message command: field v: 2.5 is not 0 or 1: validation error",
		),
	)

	DescribeTable("Should reject a message it cannot write",
		func(m codec.JSONMessage, expected string) {
			Expect(codec.NewJSONEncoder(m)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry("a field with no binding and no default",
			message([]icd.JSONField{floatField("temp", uuid.New(), 1, 0)}),
			"message command: field temp: no channel and no default: validation error",
		),
		Entry(
			"a bytes default that is not base64",
			message([]icd.JSONField{{Variant: icd.BytesJSONField{
				Key:     uuid.New(),
				Name:    "blob",
				Scale:   1,
				Pointer: "/blob",
				Default: new("AQI"),
			}}}),
			"message command: field blob: default: illegal base64 data at input byte 0: "+
				"validation error",
		),
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
		Entry("a binding to a field the message does not have",
			message(
				[]icd.JSONField{},
				codec.Binding{
					Field:    uuid.MustParse("00000000-0000-4000-8000-000000000001"),
					DataType: telem.Float64T,
				},
			),
			"message command: field 00000000-0000-4000-8000-000000000001 does not "+
				"exist: validation error",
		),
		Entry("a field with two bindings",
			message(
				[]icd.JSONField{floatField("temp", fixedKey, 1, 0)},
				codec.Binding{Field: fixedKey, DataType: telem.Float64T},
				codec.Binding{Field: fixedKey, DataType: telem.Float32T},
			),
			"message command: field temp has more than one binding: validation error",
		),
		Entry("a channel that does not write to its field",
			message(
				[]icd.JSONField{floatField("temp", fixedKey, 1, 0)},
				codec.Binding{Field: fixedKey, DataType: telem.StringT},
			),
			"message command: field temp: cannot write string channels to float "+
				"fields: validation error",
		),
		Entry("a field pointer with an array index",
			message(
				[]icd.JSONField{{Variant: icd.FloatJSONField{
					Key: fixedKey, Name: "v", Scale: 1, Pointer: "/values/0",
				}}},
				codec.Binding{Field: fixedKey, DataType: telem.Float64T},
			),
			`message command: field v: pointer "/values/0" holds an array index, `+
				"which a document cannot place: validation error",
		),
		Entry("a match rule with no type",
			codec.JSONMessage{
				Name:    "command",
				Payload: icd.JSONPayload{Match: &icd.JSONMatch{}},
			},
			"message command: match: type is required: validation error",
		),
		Entry("a match pointer with an array index",
			codec.JSONMessage{
				Name: "command",
				Payload: icd.JSONPayload{Match: &icd.JSONMatch{
					Variant: icd.IntegerJSONMatch{
						JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/10"}, Value: 1,
					},
				}},
			},
			`message command: match: pointer "/10" holds an array index, which a `+
				"document cannot place: validation error",
		),
		Entry("a scaled field with an integer channel",
			message(
				[]icd.JSONField{{Variant: icd.IntegerJSONField{
					Key: fixedKey, Name: "count", Scale: 2, Pointer: "/count",
				}}},
				codec.Binding{Field: fixedKey, DataType: telem.Int64T},
			),
			"message command: field count: a scaled field requires a float channel, "+
				"not int64: validation error",
		),
		Entry("two fields at one pointer",
			message(
				[]icd.JSONField{
					floatField("temp", fixedKey, 1, 0),
					{Variant: icd.FloatJSONField{
						Key:     uuid.New(),
						Name:    "inner",
						Scale:   1,
						Pointer: "/temp/inner",
						Default: new(payload.Float(1)),
					}},
				},
				codec.Binding{Field: fixedKey, DataType: telem.Float64T},
			),
			`message command: field temp "/temp" and field inner "/temp/inner" `+
				"overlap: validation error",
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

	It("Should write a now and a random default without allocating", func() {
		enc := MustSucceed(codec.NewJSONEncoder(message([]icd.JSONField{
			{Variant: icd.TimestampJSONField{
				Key:        uuid.New(),
				Name:       "at",
				Scale:      1,
				Pointer:    "/at",
				TimeFormat: payload.TimeFormatISO8601,
				Default:    &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
			}},
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
