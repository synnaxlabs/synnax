// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/orc"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Window", func() {
	dynamic := telem.Window{Variant: telem.DynamicWindow{TimeSpan: telem.TimeSpan(60)}}
	static := telem.Window{
		Variant: telem.StaticWindow{
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(2),
				End:   telem.TimeStamp(3),
			},
		},
	}

	Describe("MarshalJSON", func() {
		It("Should tag the dynamic variant", func() {
			b := MustSucceed(json.Marshal(dynamic))
			var fields map[string]jsontext.Value
			Expect(json.Unmarshal(b, &fields)).To(Succeed())
			Expect(fields).To(HaveKeyWithValue("variant", jsontext.Value(`"dynamic"`)))
			Expect(fields).To(HaveKey("time_span"))
		})

		It("Should tag the static variant and nest its time range", func() {
			b := MustSucceed(json.Marshal(static))
			var fields map[string]jsontext.Value
			Expect(json.Unmarshal(b, &fields)).To(Succeed())
			Expect(fields).To(HaveKeyWithValue("variant", jsontext.Value(`"static"`)))
			var tr map[string]jsontext.Value
			Expect(json.Unmarshal(fields["time_range"], &tr)).To(Succeed())
			Expect(tr).To(HaveKey("start"))
			Expect(tr).To(HaveKey("end"))
		})

		It("Should encode a nil variant as null", func() {
			Expect(MustSucceed(json.Marshal(telem.Window{}))).To(Equal([]byte("null")))
		})
	})

	Describe("UnmarshalJSON", func() {
		DescribeTable("Should round-trip a variant",
			func(w telem.Window) {
				var decoded telem.Window
				Expect(json.Unmarshal(MustSucceed(json.Marshal(w)), &decoded)).
					To(Succeed())
				Expect(decoded).To(Equal(w))
			},
			Entry("dynamic", dynamic),
			Entry("static", static),
		)

		It("Should decode null as a nil variant", func() {
			decoded := dynamic
			Expect(json.Unmarshal([]byte("null"), &decoded)).To(Succeed())
			Expect(decoded.Variant).To(BeNil())
		})

		It("Should reject an unknown variant", func() {
			var decoded telem.Window
			Expect(json.Unmarshal([]byte(`{"variant":"bogus"}`), &decoded)).
				To(MatchError(ContainSubstring("unknown variant")))
		})

		It("Should reject malformed JSON", func() {
			var decoded telem.Window
			Expect(json.Unmarshal([]byte(`{`), &decoded)).
				To(MatchError(ContainSubstring("unexpected EOF")))
		})

		DescribeTable("Should reject a variant with mistyped fields",
			func(payload, path string) {
				var decoded telem.Window
				Expect(json.Unmarshal([]byte(payload), &decoded)).
					To(MatchError(ContainSubstring(`within "` + path + `"`)))
			},
			Entry(
				"dynamic time span",
				`{"variant":"dynamic","time_span":true}`,
				"/time_span",
			),
			Entry(
				"static time range",
				`{"variant":"static","time_range":true}`,
				"/time_range",
			),
		)
	})

	Describe("EncodeOrc", func() {
		It("Should reject a nil variant", func() {
			w := orc.NewWriter(0)
			Expect(telem.Window{}.EncodeOrc(w)).
				To(MatchError(ContainSubstring("nil or unknown variant")))
		})
	})

	Describe("DecodeOrc", func() {
		It("Should reject an unknown tag", func() {
			w := orc.NewWriter(0)
			w.String("bogus")
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			var decoded telem.Window
			Expect(decoded.DecodeOrc(r)).
				To(MatchError(ContainSubstring("unknown variant")))
		})

		DescribeTable("Should reject truncated input",
			func(win telem.Window) {
				w := orc.NewWriter(0)
				Expect(win.EncodeOrc(w)).To(Succeed())
				full := w.Bytes()
				for i := range len(full) {
					r := orc.NewReader(nil)
					r.ResetBytes(full[:i])
					var decoded telem.Window
					Expect(decoded.DecodeOrc(r)).To(SatisfyAny(
						MatchError(io.EOF),
						MatchError(io.ErrUnexpectedEOF),
					))
				}
			},
			Entry("dynamic", dynamic),
			Entry("static", static),
		)
	})
})
