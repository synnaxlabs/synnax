// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v3 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v3"
)

var _ = Describe("View", func() {
	var (
		key    uuid.UUID
		ranges v3.View
		status v3.View
	)
	BeforeEach(func() {
		key = uuid.New()
		ranges = v3.View{Variant: v3.RangeView{Base: v3.Base{Key: key, Name: "Ranges"}}}
		status = v3.View{
			Variant: v3.StatusView{Base: v3.Base{Key: key, Name: "Statuses"}},
		}
	})

	Describe("WithKey", func() {
		It("Should set the key of each variant", func() {
			next := uuid.New()
			Expect(ranges.WithKey(next).GorpKey()).To(Equal(next))
			Expect(status.WithKey(next).GorpKey()).To(Equal(next))
			Expect(ranges.GorpKey()).To(Equal(key))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the view resource identifier", func() {
			Expect(ranges.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeView,
				Key:  key.String(),
			}))
		})
	})
})
