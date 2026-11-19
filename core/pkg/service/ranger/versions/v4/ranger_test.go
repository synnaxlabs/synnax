// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ = Describe("Range", func() {
	Describe("GorpKey", func() {
		It("Should return the range's key", func() {
			k := uuid.New()
			Expect(v4.Range{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the range resource identifier", func() {
			k := uuid.New()
			Expect(v4.Range{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeRange, Key: k.String(),
			}))
		})
	})

	Describe("Validate", func() {
		It("Should accept a range with no color", func() {
			Expect(v4.Range{Name: "Hot fire"}.Validate()).To(Succeed())
		})

		It("Should reject a color with no type", func() {
			Expect(v4.Range{Name: "Hot fire", Color: &color.Ref{}}.Validate()).
				To(MatchError(ContainSubstring("color.type: required")))
		})

		It("Should reject an empty name", func() {
			Expect(v4.Range{}.Validate()).To(MatchError(ContainSubstring("name")))
		})
	})
})
