// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	"github.com/synnaxlabs/x/encoding/orc"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ref", func() {
	key := v1.Ref{
		Variant: v1.KeyRef{Key: uuid.MustParse("4b6b2fd6-3c93-4b7f-8f43-7d1f8a0b9c2e")},
	}
	window := v1.Ref{
		Variant: v1.WindowRef{
			Window: telem.Window{
				Variant: telem.DynamicWindow{TimeSpan: telem.TimeSpan(60)},
			},
		},
	}

	Describe("MarshalJSON", func() {
		It("Should tag the key variant", func() {
			b := MustSucceed(json.Marshal(key))
			var fields map[string]jsontext.Value
			Expect(json.Unmarshal(b, &fields)).To(Succeed())
			Expect(fields).To(HaveKeyWithValue("variant", jsontext.Value(`"key"`)))
			Expect(fields).To(HaveKey("key"))
		})

		It("Should tag the window variant and nest its window", func() {
			b := MustSucceed(json.Marshal(window))
			var fields map[string]jsontext.Value
			Expect(json.Unmarshal(b, &fields)).To(Succeed())
			Expect(fields).To(HaveKeyWithValue("variant", jsontext.Value(`"window"`)))
			Expect(fields).NotTo(HaveKey("key"))
			var w map[string]jsontext.Value
			Expect(json.Unmarshal(fields["window"], &w)).To(Succeed())
			Expect(w).To(HaveKeyWithValue("variant", jsontext.Value(`"dynamic"`)))
		})

		It("Should encode a nil variant as null", func() {
			Expect(MustSucceed(json.Marshal(v1.Ref{}))).To(Equal([]byte("null")))
		})
	})

	Describe("UnmarshalJSON", func() {
		DescribeTable("Should round-trip a variant",
			func(r v1.Ref) {
				var decoded v1.Ref
				Expect(json.Unmarshal(MustSucceed(json.Marshal(r)), &decoded)).
					To(Succeed())
				Expect(decoded).To(Equal(r))
			},
			Entry("key", key),
			Entry("window", window),
		)

		It("Should decode null as a nil variant", func() {
			decoded := key
			Expect(json.Unmarshal([]byte("null"), &decoded)).To(Succeed())
			Expect(decoded.Variant).To(BeNil())
		})

		It("Should reject an unknown variant", func() {
			var decoded v1.Ref
			Expect(json.Unmarshal([]byte(`{"variant":"bogus"}`), &decoded)).
				To(MatchError(ContainSubstring("unknown variant")))
		})

		It("Should reject malformed JSON", func() {
			var decoded v1.Ref
			Expect(json.Unmarshal([]byte(`{`), &decoded)).
				To(MatchError(ContainSubstring("unexpected EOF")))
		})

		DescribeTable("Should reject a variant with mistyped fields",
			func(payload, path string) {
				var decoded v1.Ref
				Expect(json.Unmarshal([]byte(payload), &decoded)).
					To(MatchError(ContainSubstring(`within "` + path + `"`)))
			},
			Entry("key key", `{"variant":"key","key":true}`, "/key"),
			Entry("window window", `{"variant":"window","window":true}`, "/window"),
		)
	})

	Describe("EncodeOrc", func() {
		It("Should reject a nil variant", func() {
			w := orc.NewWriter(0)
			Expect(v1.Ref{}.EncodeOrc(w)).
				To(MatchError(ContainSubstring("nil or unknown variant")))
		})

		It("Should propagate a nil window variant", func() {
			w := orc.NewWriter(0)
			r := v1.Ref{Variant: v1.WindowRef{}}
			Expect(r.EncodeOrc(w)).
				To(MatchError(ContainSubstring("Window: nil or unknown variant")))
		})
	})

	Describe("DecodeOrc", func() {
		It("Should reject an unknown tag", func() {
			w := orc.NewWriter(0)
			w.String("bogus")
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			var decoded v1.Ref
			Expect(decoded.DecodeOrc(r)).
				To(MatchError(ContainSubstring("unknown variant")))
		})

		DescribeTable("Should reject truncated input",
			func(ref v1.Ref) {
				w := orc.NewWriter(0)
				Expect(ref.EncodeOrc(w)).To(Succeed())
				full := w.Bytes()
				for i := range len(full) {
					r := orc.NewReader(nil)
					r.ResetBytes(full[:i])
					var decoded v1.Ref
					Expect(decoded.DecodeOrc(r)).To(SatisfyAny(
						MatchError(io.EOF),
						MatchError(io.ErrUnexpectedEOF),
					))
				}
			},
			Entry("key", key),
			Entry("window", window),
		)
	})
})
