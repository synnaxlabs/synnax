// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"encoding/json"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	"github.com/synnaxlabs/x/encoding/orc"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

// truncatedErr matches the reader errors a truncated Orc buffer produces.
var truncatedErr = SatisfyAny(MatchError(io.EOF), MatchError(io.ErrUnexpectedEOF))

var _ = Describe("CustomRange", func() {
	dynamic := v6.CustomRange{Variant: v6.DynamicCustomRange{Span: telem.TimeSpan(60)}}
	static := v6.CustomRange{
		Variant: v6.StaticCustomRange{
			Start: telem.TimeStamp(2),
			End:   telem.TimeStamp(3),
		},
	}

	Describe("MarshalJSON", func() {
		It("Should tag the dynamic variant", func() {
			b := MustSucceed(json.Marshal(dynamic))
			var fields map[string]json.RawMessage
			Expect(json.Unmarshal(b, &fields)).To(Succeed())
			Expect(fields).To(HaveKeyWithValue("variant", json.RawMessage(`"dynamic"`)))
			Expect(fields).To(HaveKey("span"))
		})

		It("Should tag the static variant", func() {
			b := MustSucceed(json.Marshal(static))
			var fields map[string]json.RawMessage
			Expect(json.Unmarshal(b, &fields)).To(Succeed())
			Expect(fields).To(HaveKeyWithValue("variant", json.RawMessage(`"static"`)))
			Expect(fields).To(HaveKey("start"))
			Expect(fields).To(HaveKey("end"))
		})

		It("Should encode a nil variant as null", func() {
			Expect(MustSucceed(json.Marshal(v6.CustomRange{}))).
				To(Equal([]byte("null")))
		})
	})

	Describe("UnmarshalJSON", func() {
		It("Should round-trip the dynamic variant", func() {
			var decoded v6.CustomRange
			Expect(json.Unmarshal(MustSucceed(json.Marshal(dynamic)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(dynamic))
		})

		It("Should round-trip the static variant", func() {
			var decoded v6.CustomRange
			Expect(json.Unmarshal(MustSucceed(json.Marshal(static)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(static))
		})

		It("Should decode null as a nil variant", func() {
			decoded := dynamic
			Expect(json.Unmarshal([]byte("null"), &decoded)).To(Succeed())
			Expect(decoded.Variant).To(BeNil())
		})

		It("Should reject an unknown variant", func() {
			var decoded v6.CustomRange
			Expect(json.Unmarshal([]byte(`{"variant":"bogus"}`), &decoded)).
				To(MatchError(ContainSubstring("unknown variant")))
		})

		It("Should reject malformed JSON", func() {
			var decoded v6.CustomRange
			Expect(json.Unmarshal([]byte(`{`), &decoded)).
				To(MatchError(ContainSubstring("unexpected end of JSON input")))
		})

		DescribeTable("Should reject a variant with mistyped fields",
			func(payload string) {
				var decoded v6.CustomRange
				Expect(json.Unmarshal([]byte(payload), &decoded)).
					To(MatchError(ContainSubstring("cannot unmarshal")))
			},
			Entry("dynamic span", `{"variant":"dynamic","span":true}`),
			Entry("static start", `{"variant":"static","start":true}`),
		)
	})

	Describe("EncodeOrc", func() {
		It("Should reject a nil variant", func() {
			w := orc.NewWriter(0)
			Expect(v6.CustomRange{}.EncodeOrc(w)).
				To(MatchError(ContainSubstring("nil or unknown variant")))
		})
	})

	Describe("DecodeOrc", func() {
		It("Should reject an unknown tag", func() {
			w := orc.NewWriter(0)
			w.String("bogus")
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			var decoded v6.CustomRange
			Expect(decoded.DecodeOrc(r)).
				To(MatchError(ContainSubstring("unknown variant")))
		})

		DescribeTable("Should reject truncated input",
			func(cr v6.CustomRange) {
				w := orc.NewWriter(0)
				Expect(cr.EncodeOrc(w)).To(Succeed())
				full := w.Bytes()
				for i := range len(full) {
					r := orc.NewReader(nil)
					r.ResetBytes(full[:i])
					var decoded v6.CustomRange
					Expect(decoded.DecodeOrc(r)).To(truncatedErr)
				}
			},
			Entry("dynamic", dynamic),
			Entry("static", static),
		)
	})
})

var _ = Describe("Ranges custom window codec", func() {
	dynamic := v6.CustomRange{Variant: v6.DynamicCustomRange{Span: telem.TimeSpan(60)}}

	It("Should propagate a custom range encoding failure", func() {
		w := orc.NewWriter(0)
		rv := v6.Ranges{Custom: &v6.CustomRange{}}
		Expect(rv.EncodeOrc(w)).
			To(MatchError(ContainSubstring("nil or unknown variant")))
	})

	It("Should reject truncated input with a custom range present", func() {
		w := orc.NewWriter(0)
		rv := v6.Ranges{X1: []string{"custom"}, Custom: &dynamic}
		Expect(rv.EncodeOrc(w)).To(Succeed())
		full := w.Bytes()
		for i := range len(full) {
			r := orc.NewReader(nil)
			r.ResetBytes(full[:i])
			var decoded v6.Ranges
			Expect(decoded.DecodeOrc(r)).To(truncatedErr)
		}
	})
})
