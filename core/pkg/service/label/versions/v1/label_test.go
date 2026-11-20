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
	v1 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ = Describe("Label", func() {
	Describe("GorpKey", func() {
		It("Should return the label's key", func() {
			k := uuid.New()
			Expect(v1.Label{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the label resource identifier", func() {
			k := uuid.New()
			Expect(v1.Label{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeLabel, Key: k.String(),
			}))
		})
	})

	Describe("Validate", func() {
		It("Should accept a label with a name", func() {
			Expect(v1.Label{Name: "Critical"}.Validate()).To(Succeed())
		})
		It("Should reject a label without a name", func() {
			Expect(
				v1.Label{}.Validate(),
			).To(MatchError(ContainSubstring("name: required")))
		})
	})
})
