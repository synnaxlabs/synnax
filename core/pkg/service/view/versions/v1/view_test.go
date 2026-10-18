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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	statusv1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("View", func() {
	var (
		key    uuid.UUID
		ranges v1.View
		status v1.View
	)
	BeforeEach(func() {
		key = uuid.New()
		ranges = v1.View{Variant: v1.RangeView{Base: v1.Base{Key: key, Name: "Ranges"}}}
		status = v1.View{
			Variant: v1.StatusView{Base: v1.Base{Key: key, Name: "Statuses"}},
		}
	})

	Describe("Base", func() {
		It("Should return the base of each variant", func() {
			Expect(ranges.Base()).To(Equal(v1.Base{Key: key, Name: "Ranges"}))
			Expect(status.Base()).To(Equal(v1.Base{Key: key, Name: "Statuses"}))
		})
		It("Should return a zero base for a view with no variant", func() {
			Expect(v1.View{}.Base()).To(Equal(v1.Base{}))
		})
	})

	Describe("Type", func() {
		It("Should return the type of each variant", func() {
			Expect(ranges.Type()).To(Equal(v1.RangeViewType))
			Expect(status.Type()).To(Equal(v1.StatusViewType))
		})
		It("Should return an empty type for a view with no variant", func() {
			Expect(v1.View{}.Type()).To(BeEmpty())
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
			func(v v1.View) {
				var decoded v1.View
				Expect(json.Unmarshal(MustSucceed(json.Marshal(v)), &decoded)).
					To(Succeed())
				Expect(decoded).To(Equal(v))
			},
			Entry("range", v1.View{Variant: v1.RangeView{
				Base:  v1.Base{Key: uuid.New(), Name: "Ranges"},
				Query: v1.RangeQuery{HasLabels: []uuid.UUID{uuid.New()}},
			}}),
			Entry("status", v1.View{Variant: v1.StatusView{
				Base: v1.Base{Key: uuid.New(), Name: "Statuses"},
				Query: v1.StatusQuery{
					HasLabels: []uuid.UUID{},
					Variants:  []statusv1.Variant{statusv1.VariantError},
				},
			}}),
		)

		It("Should put the type beside the base and query fields", func() {
			key := uuid.New()
			b := MustSucceed(json.Marshal(v1.View{Variant: v1.RangeView{
				Base:  v1.Base{Key: key, Name: "Ranges"},
				Query: v1.RangeQuery{HasLabels: []uuid.UUID{}},
			}}))
			Expect(b).To(MatchJSON(`{
				"type": "range",
				"key": "` + key.String() + `",
				"name": "Ranges",
				"query": {"has_labels": []}
			}`))
		})

		It("Should encode a view with no variant as null", func() {
			b := MustSucceed(json.Marshal(v1.View{}))
			Expect(string(b)).To(Equal("null"))
			var decoded v1.View
			Expect(json.Unmarshal(b, &decoded)).To(Succeed())
			Expect(decoded.Variant).To(BeNil())
		})

		It("Should reject an unknown type", func() {
			var decoded v1.View
			Expect(json.Unmarshal([]byte(`{"type":"lineplot","name":"x"}`), &decoded)).
				To(MatchError(ContainSubstring(`unknown type "lineplot"`)))
		})
	})
})
