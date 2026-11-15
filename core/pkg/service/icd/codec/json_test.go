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
	"strings"
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
		dec, err := codec.NewJSON(f.Payload, bindings)
		if f.Error != "" {
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(f.Error)))
			return
		}
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Cases).ToNot(BeEmpty())
		for i, c := range f.Cases {
			By("case " + string(rune('0'+i)))
			doc := []byte(c.Document)
			if c.Text != nil {
				doc = []byte(*c.Text)
			}
			samples := make([][]byte, len(bindings))
			ok, err := dec.Decode(doc, samples)
			switch {
			case c.Error != "":
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(ContainSubstring(c.Error)))
				Expect(samples).To(HaveEach(BeEmpty()))
			case c.Unmatched:
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
				Expect(samples).To(HaveEach(BeEmpty()))
			default:
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
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

	It("Should append to the samples of earlier documents", func() {
		key := uuid.New()
		p := icd.JSONPayload{Fields: []icd.JSONField{{Variant: icd.FloatJSONField{
			JSONBaseField: icd.JSONBaseField{
				BaseField: icd.BaseField{Key: key, Name: "v", Scale: 1},
				Pointer:   "/v",
			},
		}}}}
		dec := MustSucceed(codec.NewJSON(
			p, []codec.Binding{{Field: key, DataType: telem.Float64T}},
		))
		samples := make([][]byte, 1)
		Expect(dec.Decode([]byte(`{"v": 1}`), samples)).To(BeTrue())
		Expect(dec.Decode([]byte(`{"v": 2}`), samples)).To(BeTrue())
		_, err := dec.Decode([]byte(`{"v": "x"}`), samples)
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(telem.Series{DataType: telem.Float64T, Data: samples[0]}).
			To(telem.MatchSeriesDataV[float64](1, 2))
	})
})
