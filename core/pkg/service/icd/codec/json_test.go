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
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
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

// fixturePaths returns the fixtures in the dir folder of testdata.
func fixturePaths(dir string) []string {
	GinkgoHelper()
	paths := MustSucceed(
		filepath.Glob(filepath.Join("..", "testdata", dir, "*.json")),
	)
	Expect(paths).ToNot(BeEmpty())
	return paths
}

// fixtureEntries returns an entry for each fixture in the dir folder of testdata.
func fixtureEntries(dir string) []TableEntry {
	GinkgoHelper()
	return lo.Map(fixturePaths(dir), func(path string, _ int) TableEntry {
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
		return floatJSON(math.Float64frombits(telem.ByteOrder.Uint64(data)))
	case telem.Float32T:
		return floatJSON(float64(math.Float32frombits(telem.ByteOrder.Uint32(data))))
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

// floatJSON returns f as the fixtures write it: a number, or a string for NaN,
// Infinity, and -Infinity.
func floatJSON(f float64) any {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return f
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

func (m fixtureMessage) jsonMessage() codec.JSONMessage {
	return codec.JSONMessage{
		Name:    m.Name,
		Payload: m.Payload,
		Bindings: lo.Map(m.Bindings, func(b binding, _ int) codec.Binding {
			return codec.Binding{Field: b.Field, DataType: b.DataType}
		}),
	}
}

func (f decodeFixture) messages() []codec.JSONMessage {
	return lo.Map(f.Messages, func(m fixtureMessage, _ int) codec.JSONMessage {
		return m.jsonMessage()
	})
}

// newSamples returns empty sample slices for each binding of each message.
func newSamples(messages []codec.JSONMessage) [][][]byte {
	return lo.Map(messages, func(m codec.JSONMessage, _ int) [][]byte {
		return make([][]byte, len(m.Bindings))
	})
}

// decode decodes doc into samples, and fails the spec on an error or a failed field. It
// returns the message of doc, and whether doc belongs to one.
func decode(dec *codec.JSON, doc []byte, samples [][][]byte) (int, bool) {
	GinkgoHelper()
	message, ok, failures, err := dec.Decode(doc, samples)
	Expect(err).ToNot(HaveOccurred())
	Expect(failures).To(BeNil())
	return message, ok
}

// decodeError returns the error of decoding doc, or else the error of its first failed
// field.
func decodeError(dec *codec.JSON, doc []byte, samples [][][]byte) error {
	_, _, failures, err := dec.Decode(doc, samples)
	if err == nil && len(failures) > 0 {
		return failures[0].Err
	}
	return err
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
				m, _, failures, err := dec.Decode(doc, samples)
				if err == nil {
					Expect(failures).ToNot(BeEmpty())
					err = failures[0].Err
					for _, failure := range failures {
						Expect(samples[m][failure.Binding]).To(BeEmpty())
					}
				} else {
					Expect(samples).To(HaveEach(HaveEach(BeEmpty())))
				}
				Expect(err).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(c.Error),
				))
			case c.Unmatched:
				_, ok := decode(dec, doc, samples)
				Expect(ok).To(BeFalse())
				Expect(samples).To(HaveEach(HaveEach(BeEmpty())))
			default:
				m, ok := decode(dec, doc, samples)
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
	}, fixtureEntries("decode"))

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

	Describe("Generated defaults", func() {
		decoder := func(f icd.JSONField, dt telem.DataType) *codec.JSON {
			GinkgoHelper()
			return MustSucceed(codec.NewJSON([]codec.JSONMessage{{
				Name:     "reading",
				Payload:  icd.JSONPayload{Fields: []icd.JSONField{f}},
				Bindings: []codec.Binding{{Field: f.Base().Key, DataType: dt}},
			}}))
		}

		It("Should fill a missing timestamp with the time of the read", func() {
			dec := decoder(icd.JSONField{Variant: icd.TimestampJSONField{
				Key: uuid.New(), Name: "ts", Scale: 1, Pointer: "/ts",
				TimeFormat: payload.TimeFormatUnixMs,
				Default:    &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
			}}, telem.TimestampT)
			samples := [][][]byte{make([][]byte, 1)}
			before := telem.Now()
			_, ok := decode(dec, []byte(`{}`), samples)
			after := telem.Now()
			Expect(ok).To(BeTrue())
			read := telem.TimeStamp(telem.ByteOrder.Uint64(samples[0][0]))
			Expect(read).To(BeNumerically(">=", before))
			Expect(read).To(BeNumerically("<=", after))
		})

		It("Should fill a missing UUID with a new random UUID on each read", func() {
			dec := decoder(icd.JSONField{Variant: icd.UuidJSONField{
				Key: uuid.New(), Name: "id", Scale: 1, Pointer: "/id",
				Default: &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			}}, telem.UUIDT)
			samples := [][][]byte{make([][]byte, 1)}
			for range 2 {
				_, ok := decode(dec, []byte(`{}`), samples)
				Expect(ok).To(BeTrue())
			}
			read := samples[0][0]
			Expect(read).To(HaveLen(32))
			first, second := uuid.UUID(read[:16]), uuid.UUID(read[16:])
			Expect(first).ToNot(Equal(second))
			Expect(first[6] >> 4).To(Equal(byte(4)))
			Expect(second[6] >> 4).To(Equal(byte(4)))
		})

		It("Should fill a missing JSON value with every digit of its default", func() {
			dec := decoder(icd.JSONField{Variant: icd.JSONJSONField{
				Key: uuid.New(), Name: "doc", Scale: 1, Pointer: "/doc",
				Default: new(`{"id": 9007199254740993}`),
			}}, telem.JSONT)
			samples := [][][]byte{make([][]byte, 1)}
			_, ok := decode(dec, []byte(`{}`), samples)
			Expect(ok).To(BeTrue())
			Expect(string(samples[0][0][4:])).To(Equal(`{"id":9007199254740993}`))
		})

		It("Should reject a JSON default that is not valid JSON", func() {
			f := icd.JSONField{Variant: icd.JSONJSONField{
				Key: uuid.New(), Name: "doc", Scale: 1, Pointer: "/doc",
				Default: new(`{"id":`),
			}}
			Expect(codec.NewJSON([]codec.JSONMessage{{
				Name:     "reading",
				Payload:  icd.JSONPayload{Fields: []icd.JSONField{f}},
				Bindings: []codec.Binding{{Field: f.Base().Key, DataType: telem.JSONT}},
			}})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					"message reading: field doc: default: not valid JSON: validation error",
				),
			))
		})

		DescribeTable("Should reject a default with no type",
			func(f icd.JSONField, dt telem.DataType) {
				Expect(codec.NewJSON([]codec.JSONMessage{{
					Name:     "reading",
					Payload:  icd.JSONPayload{Fields: []icd.JSONField{f}},
					Bindings: []codec.Binding{{Field: f.Base().Key, DataType: dt}},
				}})).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError("message reading: field v: default: type is required: "+
						"validation error"),
				))
			},
			Entry("for a timestamp", icd.JSONField{Variant: icd.TimestampJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
				TimeFormat: payload.TimeFormatUnixMs,
				Default:    &icd.TimestampDefault{},
			}}, telem.TimestampT),
			Entry("for a UUID", icd.JSONField{Variant: icd.UuidJSONField{
				Key: uuid.New(), Name: "v", Scale: 1, Pointer: "/v",
				Default: &icd.UUIDDefault{},
			}}, telem.UUIDT),
		)
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
			_, ok := decode(dec, []byte(doc), samples)
			Expect(ok).To(BeTrue())
		}
		Expect(decodeError(dec, []byte(`{"v": "x"}`), samples)).
			To(MatchError(validate.ErrValidation))
		Expect(telem.Series{DataType: telem.Float64T, Data: samples[0][0]}).
			To(MatchSeriesDataV[float64](1, 2))
	})

	Describe("Failures", func() {
		// reading returns a decoder of one message with n float fields, at /a, /b, and
		// so on.
		reading := func(n int) (*codec.JSON, [][][]byte) {
			GinkgoHelper()
			m := codec.JSONMessage{Name: "reading"}
			for i := range n {
				name, key := string(rune('a'+i)), uuid.New()
				m.Payload.Fields = append(m.Payload.Fields, icd.JSONField{
					Variant: icd.FloatJSONField{
						Key: key, Name: name, Scale: 1, Pointer: "/" + name,
					},
				})
				m.Bindings = append(m.Bindings, codec.Binding{
					Field: key, DataType: telem.Float64T,
				})
			}
			messages := []codec.JSONMessage{m}
			return MustSucceed(codec.NewJSON(messages)), newSamples(messages)
		}

		series := func(data []byte) telem.Series {
			return telem.Series{DataType: telem.Float64T, Data: data}
		}

		It(
			"Should list each field that does not convert and append the others",
			func() {
				dec, samples := reading(3)
				_, ok, failures, err := dec.Decode(
					[]byte(`{"a": "x", "b": 2, "c": "y"}`), samples,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(failures).To(HaveLen(2))
				Expect(failures[0].Binding).To(Equal(0))
				Expect(failures[0].Missing).To(BeFalse())
				Expect(failures[0].Err).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("field a: ")),
				))
				Expect(failures[1].Binding).To(Equal(2))
				Expect(samples[0][0]).To(BeEmpty())
				Expect(series(samples[0][1])).To(MatchSeriesDataV[float64](2))
				Expect(samples[0][2]).To(BeEmpty())
			},
		)

		It("Should list a field with no value and no default as missing", func() {
			dec, samples := reading(2)
			_, _, failures, err := dec.Decode([]byte(`{"a": 1}`), samples)
			Expect(err).ToNot(HaveOccurred())
			Expect(failures).To(HaveLen(1))
			Expect(failures[0].Binding).To(Equal(1))
			Expect(failures[0].Missing).To(BeTrue())
			Expect(failures[0].Err).
				To(MatchError("field b: no value at /b: validation error"))
		})

		It("Should clear the failures of the last decode", func() {
			dec, samples := reading(1)
			Expect(decodeError(dec, []byte(`{"a": "x"}`), samples)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("field a: ")),
			))
			decode(dec, []byte(`{"a": 1}`), samples)
		})

		It("Should name the message of a failure when the decoder holds more", func() {
			status, other := uuid.New(), uuid.New()
			messages := []codec.JSONMessage{
				{
					Name: "status",
					Payload: icd.JSONPayload{
						Match: &icd.JSONMatch{Variant: icd.StringJSONMatch{
							JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/type"},
							Value:         "status",
						}},
						Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
							Key: status, Name: "a", Scale: 1, Pointer: "/a",
						}}},
					},
					Bindings: []codec.Binding{
						{Field: status, DataType: telem.Float64T},
					},
				},
				{
					Name: "other",
					Payload: icd.JSONPayload{
						Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
							Key: other, Name: "b", Scale: 1, Pointer: "/b",
						}}},
					},
					Bindings: []codec.Binding{{Field: other, DataType: telem.Float64T}},
				},
			}
			dec := MustSucceed(codec.NewJSON(messages))
			doc := []byte(`{"type": "status"}`)
			Expect(decodeError(dec, doc, newSamples(messages))).To(MatchError(
				"message status: field a: no value at /a: validation error",
			))
		})
	})

	DescribeTable("Should accept the present rules that Infer gives the kinds of a key",
		func(kinds ...string) {
			docs := make([]xjson.Document, len(kinds))
			for i, k := range kinds {
				docs[i] = xjson.Document{Source: "pump", Data: []byte(k)}
			}
			shapes := xjson.Infer(docs)
			messages := make([]codec.JSONMessage, len(shapes))
			for i, shape := range shapes {
				key, field := uuid.New(), shape.Fields[0]
				var match *icd.JSONMatch
				if shape.Present >= 0 {
					field = shape.Fields[shape.Present]
					match = &icd.JSONMatch{Variant: icd.PresentJSONMatch{
						JSONBaseMatch: icd.JSONBaseMatch{
							Pointer: field.Pointer.String(),
						},
					}}
				}
				messages[i] = codec.JSONMessage{
					Name: "kind " + strconv.Itoa(i),
					Payload: icd.JSONPayload{
						Match: match,
						Fields: []icd.JSONField{{Variant: icd.StringJSONField{
							Key:     key,
							Name:    "v",
							Scale:   1,
							Pointer: field.Pointer.String(),
						}}},
					},
					Bindings: []codec.Binding{{Field: key, DataType: telem.StringT}},
				}
			}
			Expect(codec.NewJSON(messages)).Error().ToNot(HaveOccurred())
		},
		Entry("an object and a string at one pointer",
			`{"status":{"code":1}}`, `{"status":"ok"}`,
		),
		Entry("a string and an object at one pointer, with a shared field",
			`{"t":1,"status":"ok"}`, `{"t":1,"status":{"code":3}}`,
		),
		Entry("an object and a string at one pointer, with fields that differ",
			`{"status":{"code":1},"fault":2}`, `{"status":"ok","rpm":3}`,
		),
	)

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
				message  int
				ok       bool
				failures []codec.Failure
				err      error
			)
			decode := func() {
				for i := range samples[1] {
					samples[1][i] = samples[1][i][:0]
				}
				message, ok, failures, err = dec.Decode(doc, samples)
			}
			Expect(testing.AllocsPerRun(100, decode)).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
			Expect(failures).To(BeNil())
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
