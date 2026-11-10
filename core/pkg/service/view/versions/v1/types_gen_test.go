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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v1"
)

var _ = Describe("Validate", func() {
	It("Should accept a named view", func() {
		Expect(v1.RangeView{Name: "Named"}.Validate()).To(Succeed())
	})
	It("Should reject an empty name", func() {
		Expect(v1.StatusView{}.Validate()).
			To(MatchError(ContainSubstring("name: required")))
	})
})

var _ = Describe("Base", func() {
	var key uuid.UUID
	BeforeEach(func() { key = uuid.New() })

	It("Should return the base of each variant", func() {
		ranges := v1.View{
			Variant: v1.RangeView{Base: v1.Base{Key: key, Name: "Ranges"}},
		}
		status := v1.View{
			Variant: v1.StatusView{Base: v1.Base{Key: key, Name: "Statuses"}},
		}
		Expect(ranges.Base()).To(Equal(v1.Base{Key: key, Name: "Ranges"}))
		Expect(status.Base()).To(Equal(v1.Base{Key: key, Name: "Statuses"}))
	})

	It("Should return a zero base for a view with no variant", func() {
		Expect(v1.View{}.Base()).To(Equal(v1.Base{}))
	})
})

var _ = Describe("SetBase", func() {
	It("Should replace the base of each variant and keep the rest", func() {
		query := v1.RangeQuery{HasLabels: []uuid.UUID{uuid.New()}}
		ranges := v1.View{
			Variant: v1.RangeView{Base: v1.Base{Name: "Before"}, Query: query},
		}
		status := v1.View{Variant: v1.StatusView{Base: v1.Base{Name: "Before"}}}
		next := v1.Base{Key: uuid.New(), Name: "After"}
		ranges.SetBase(next)
		status.SetBase(next)
		Expect(ranges.Variant).To(Equal(v1.RangeView{Base: next, Query: query}))
		Expect(status.Variant).To(Equal(v1.StatusView{Base: next}))
	})

	It("Should panic for a view with no variant", func() {
		var v v1.View
		Expect(func() { v.SetBase(v1.Base{Name: "After"}) }).
			To(PanicWith(MatchError("View: cannot set the base of variant <nil>")))
	})
})
