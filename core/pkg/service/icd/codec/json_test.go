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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// decodeFixture is one JSON message from testdata/decode and the documents it decodes.
// A fixture with an error names the error NewJSON must return, and has no cases.
type decodeFixture struct {
	Description string          `json:"description"`
	Payload     icd.JSONPayload `json:"payload"`
	Bindings    []binding       `json:"bindings"`
	Error       string          `json:"error"`
	Cases       []decodeCase    `json:"cases"`
}

type binding struct {
	Field    icd.FieldKey   `json:"field"`
	DataType telem.DataType `json:"data_type"`
}

// decodeCase is one document. A document that does not match sets unmatched, and one
// that does not decode names the error. Otherwise samples holds one JSON value for each
// binding.
type decodeCase struct {
	Document  json.RawMessage   `json:"document"`
	Text      *string           `json:"text"`
	Unmatched bool              `json:"unmatched"`
	Error     string            `json:"error"`
	Samples   []json.RawMessage `json:"samples"`
}

func decodeEntries() []TableEntry {
	GinkgoHelper()
	paths := MustSucceed(
		filepath.Glob(filepath.Join("..", "testdata", "decode", "*.json")),
	)
	Expect(paths).ToNot(BeEmpty())
	return lo.Map(paths, func(path string, _ int) TableEntry {
		return Entry(strings.TrimSuffix(filepath.Base(path), ".json"), path)
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
	}
	Fail("unknown data type " + string(dt))
	return nil
}

var _ = Describe("JSON", func() {
	DescribeTable("Should decode the documents of a fixture", func(path string) {
		var f decodeFixture
		Expect(json.Unmarshal(MustSucceed(os.ReadFile(path)), &f)).To(Succeed())
		Expect(f.Description).ToNot(BeEmpty())
		bindings := lo.Map(f.Bindings, func(b binding, _ int) codec.Binding {
			return codec.Binding{Field: b.Field, DataType: b.DataType}
		})
		if f.Error != "" {
			Expect(codec.NewJSON(f.Payload, bindings)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(f.Error)),
			))
			return
		}
		dec := MustSucceed(codec.NewJSON(f.Payload, bindings))
		Expect(f.Cases).ToNot(BeEmpty())
		for i, c := range f.Cases {
			By("case " + strconv.Itoa(i))
			doc := []byte(c.Document)
			if c.Text != nil {
				doc = []byte(*c.Text)
			}
			samples := make([][]byte, len(bindings))
			switch {
			case c.Error != "":
				Expect(dec.Decode(doc, samples)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(c.Error)),
				))
				Expect(samples).To(HaveEach(BeEmpty()))
			case c.Unmatched:
				Expect(dec.Decode(doc, samples)).To(BeFalse())
				Expect(samples).To(HaveEach(BeEmpty()))
			default:
				Expect(dec.Decode(doc, samples)).To(BeTrue())
				Expect(c.Samples).To(HaveLen(len(bindings)))
				for k, b := range bindings {
					actual := MustSucceed(
						json.Marshal(sampleJSON(b.DataType, samples[k])),
					)
					Expect(actual).To(MatchJSON(c.Samples[k]), "binding %d", k)
				}
			}
		}
	}, decodeEntries())

	DescribeTable("Should match a number of any Go numeric type", func(value any) {
		key := uuid.New()
		p := icd.JSONPayload{
			Match: &icd.JSONMatch{Pointer: "/type", Value: value},
			Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
				Key: key, Name: "v", Scale: 1, Pointer: "/v",
			}}},
		}
		dec := MustSucceed(codec.NewJSON(
			p, []codec.Binding{{Field: key, DataType: telem.Float64T}},
		))
		samples := make([][]byte, 1)
		Expect(dec.Decode([]byte(`{"type": 3, "v": 1}`), samples)).To(BeTrue())
		Expect(dec.Decode([]byte(`{"type": 4, "v": 1}`), samples)).To(BeFalse())
	},
		Entry("float64", float64(3)),
		Entry("float32", float32(3)),
		Entry("int", 3),
		Entry("int8", int8(3)),
		Entry("int16", int16(3)),
		Entry("int32", int32(3)),
		Entry("int64", int64(3)),
		Entry("uint", uint(3)),
		Entry("uint8", uint8(3)),
		Entry("uint16", uint16(3)),
		Entry("uint32", uint32(3)),
		Entry("uint64", uint64(3)),
	)

	It("Should append to the samples of earlier documents", func() {
		key := uuid.New()
		p := icd.JSONPayload{Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
			Key: key, Name: "v", Scale: 1, Pointer: "/v",
		}}}}
		dec := MustSucceed(codec.NewJSON(
			p, []codec.Binding{{Field: key, DataType: telem.Float64T}},
		))
		samples := make([][]byte, 1)
		Expect(dec.Decode([]byte(`{"v": 1}`), samples)).To(BeTrue())
		Expect(dec.Decode([]byte(`{"v": 2}`), samples)).To(BeTrue())
		Expect(dec.Decode([]byte(`{"v": "x"}`), samples)).
			Error().To(MatchError(validate.ErrValidation))
		Expect(telem.Series{DataType: telem.Float64T, Data: samples[0]}).
			To(telem.MatchSeriesDataV[float64](1, 2))
	})

	It("Should decode a document without allocating", func() {
		temp, count, state := uuid.New(), uuid.New(), uuid.New()
		p := icd.JSONPayload{
			Match: &icd.JSONMatch{Pointer: "/type", Value: "status"},
			Fields: []icd.JSONField{
				{Variant: icd.FloatJSONField{
					Key: temp, Name: "temp", Scale: 0.5, Offset: -10, Pointer: "/temp",
				}},
				{Variant: icd.IntegerJSONField{
					Key: count, Name: "count", Scale: 1, Pointer: "/count",
				}},
				{Variant: icd.StringJSONField{
					Key: state, Name: "state", Scale: 1, Pointer: "/state",
				}},
			},
		}
		dec := MustSucceed(codec.NewJSON(p, []codec.Binding{
			{Field: temp, DataType: telem.Float32T},
			{Field: count, DataType: telem.Float64T},
			{Field: state, DataType: telem.StringT},
		}))
		doc := []byte(`{"type":"status","temp":40,"count":3,"state":"run"}`)
		samples := make([][]byte, 3)
		var (
			ok  bool
			err error
		)
		decode := func() {
			for i := range samples {
				samples[i] = samples[i][:0]
			}
			ok, err = dec.Decode(doc, samples)
		}
		Expect(testing.AllocsPerRun(100, decode)).To(BeZero())
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	})
})
