// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"encoding/json/v2"
	"io"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	rangerv1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	statusv1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v2"
	"github.com/synnaxlabs/x/encoding/orc"
	filter "github.com/synnaxlabs/x/filter/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("View", func() {
	var (
		key    uuid.UUID
		ranges v2.View
		status v2.View
	)
	BeforeEach(func() {
		key = uuid.New()
		ranges = v2.View{Variant: v2.RangeView{Base: v2.Base{Key: key, Name: "Ranges"}}}
		status = v2.View{
			Variant: v2.StatusView{Base: v2.Base{Key: key, Name: "Statuses"}},
		}
	})

	Describe("Base", func() {
		It("Should return the base of each variant", func() {
			Expect(ranges.Base()).To(Equal(v2.Base{Key: key, Name: "Ranges"}))
			Expect(status.Base()).To(Equal(v2.Base{Key: key, Name: "Statuses"}))
		})
		It("Should return a zero base for a view with no variant", func() {
			Expect(v2.View{}.Base()).To(Equal(v2.Base{}))
		})
	})

	Describe("Type", func() {
		It("Should return the type of each variant", func() {
			Expect(ranges.Type()).To(Equal(v2.TypeRange))
			Expect(status.Type()).To(Equal(v2.TypeStatus))
		})
		It("Should return an empty type for a view with no variant", func() {
			Expect(v2.View{}.Type()).To(BeEmpty())
		})
	})

	Describe("WithKey", func() {
		It("Should set the key of each variant", func() {
			next := uuid.New()
			Expect(ranges.WithKey(next).GorpKey()).To(Equal(next))
			Expect(status.WithKey(next).GorpKey()).To(Equal(next))
			Expect(ranges.GorpKey()).To(Equal(key))
		})
	})

	Describe("WithName", func() {
		It("Should set the name of each variant", func() {
			Expect(ranges.WithName("Renamed").Base().Name).To(Equal("Renamed"))
			Expect(status.WithName("Renamed").Base().Name).To(Equal("Renamed"))
			Expect(ranges.Base().Name).ToNot(Equal("Renamed"))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the view's key", func() {
			Expect(ranges.GorpKey()).To(Equal(key))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(ranges.SetOptions()).To(BeNil())
		})
	})

	Describe("OntologyID", func() {
		It("Should return the view ontology identifier", func() {
			Expect(ranges.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeView, Key: key.String(),
			}))
		})
	})

	Describe("JSON", func() {
		DescribeTable("should round-trip each variant",
			func(v v2.View) {
				var decoded v2.View
				Expect(json.Unmarshal(MustSucceed(json.Marshal(v)), &decoded)).
					To(Succeed())
				Expect(decoded).To(Equal(v))
			},
			Entry("range", v2.View{Variant: v2.RangeView{
				Base: v2.Base{Key: uuid.New(), Name: "Ranges"},
				Where: &rangerv1.Where{Or: []rangerv1.Where{
					{Labels: []uuid.UUID{uuid.New()}},
					{Not: &rangerv1.Where{Name: &filter.String{Eq: new("Run")}}},
				}},
			}}),
			Entry("status", v2.View{Variant: v2.StatusView{
				Base: v2.Base{Key: uuid.New(), Name: "Statuses"},
				Where: &statusv1.Where{Variant: &filter.Set[statusv1.Variant]{
					In: []statusv1.Variant{statusv1.VariantError},
				}},
			}}),
		)

		It("Should put the type beside the base and where fields", func() {
			key, label := uuid.New(), uuid.New()
			b := MustSucceed(json.Marshal(v2.View{Variant: v2.RangeView{
				Base:  v2.Base{Key: key, Name: "Ranges"},
				Where: &rangerv1.Where{Labels: []uuid.UUID{label}},
			}}))
			Expect(b).To(MatchJSON(`{
				"type": "range",
				"key": "` + key.String() + `",
				"name": "Ranges",
				"where": {"labels": ["` + label.String() + `"]}
			}`))
		})

		It("Should encode a view with no variant as null", func() {
			b := MustSucceed(json.Marshal(v2.View{}))
			Expect(string(b)).To(Equal("null"))
			var decoded v2.View
			Expect(json.Unmarshal(b, &decoded)).To(Succeed())
			Expect(decoded.Variant).To(BeNil())
		})

		It("Should reject an unknown type", func() {
			var decoded v2.View
			Expect(json.Unmarshal([]byte(`{"type":"lineplot","name":"x"}`), &decoded)).
				To(MatchError(ContainSubstring(`unknown type "lineplot"`)))
		})
	})

	Describe("Orc", func() {
		It("Should not encode a view with no variant", func() {
			Expect(v2.View{}.EncodeOrc(orc.NewWriter(0))).
				To(MatchError(ContainSubstring("nil or unknown variant")))
		})

		It("Should reject an unknown variant tag", func() {
			w := orc.NewWriter(0)
			w.String("lineplot")
			var v v2.View
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(
				v.DecodeOrc(r),
			).To(MatchError(ContainSubstring(`unknown variant "lineplot"`)))
		})

		DescribeTable("should fail to decode every truncation of a view",
			func(view v2.View) {
				w := orc.NewWriter(0)
				Expect(view.EncodeOrc(w)).To(Succeed())
				b := w.Bytes()
				for n := range len(b) {
					var decoded v2.View
					r := orc.NewReader(nil)
					r.ResetBytes(b[:n])
					Expect(decoded.DecodeOrc(r)).To(SatisfyAny(
						MatchError(io.EOF),
						MatchError(io.ErrUnexpectedEOF),
					), "prefix length %d", n)
				}
			},
			Entry("range", v2.View{Variant: v2.RangeView{
				Base: v2.Base{Key: uuid.New(), Name: "Ranges"},
				Where: &rangerv1.Where{Or: []rangerv1.Where{
					{Labels: []uuid.UUID{uuid.New()}},
					{Not: &rangerv1.Where{Name: &filter.String{Eq: new("Run")}}},
				}},
			}}),
			Entry("status", v2.View{Variant: v2.StatusView{
				Base: v2.Base{Key: uuid.New(), Name: "Statuses"},
				Where: &statusv1.Where{
					Labels: []uuid.UUID{uuid.New()},
					Variant: &filter.Set[statusv1.Variant]{
						Eq: new(statusv1.VariantError),
					},
				},
			}}),
		)
	})
})
