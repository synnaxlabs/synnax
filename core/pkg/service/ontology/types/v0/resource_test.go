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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ontology/types/v0"
)

var _ = Describe("Resource", func() {
	Describe("Type", func() {
		Describe("String", func() {
			It("Should return the string representation of the type", func() {
				Expect(v0.ResourceTypeChannel.String()).To(Equal("channel"))
			})
		})
	})
	Describe("ID", func() {
		Describe("Validate", func() {
			It("Should return an error if the ID does not have a type", func() {
				id := v0.ID{Key: "foo"}
				Expect(id.Validate()).To(And(
					MatchError(ContainSubstring("type: invalid type")),
				))
			})
			It("Should return nil if the resource ID is valid", func() {
				id := v0.ID{Type: v0.ResourceTypeChannel, Key: "bar"}
				Expect(id.Validate()).To(Succeed())
			})
		})
		Describe("String", func() {
			It("Should return the string representation of the ID", func() {
				Expect(v0.ID{Key: "dog", Type: v0.ResourceTypeChannel}.String()).
					To(Equal("channel:dog"))
			})
		})
		Describe("IsZero", func() {
			It("Should return true if both the type and key are empty", func() {
				Expect(v0.ID{}.IsZero()).To(BeTrue())
			})
			It("Should return false when the type is not empty", func() {
				Expect(v0.ID{Type: v0.ResourceTypeChannel}.IsZero()).To(BeFalse())
			})
			It("Should return false when the key is not empty", func() {
				Expect(v0.ID{Key: "cat"}.IsZero()).To(BeFalse())
			})
		})
		Describe("IsType", func() {
			It("Should return true if the Key is empty", func() {
				Expect(v0.ID{Type: v0.ResourceTypeChannel}.IsType()).To(BeTrue())
			})
			It("Should return false if the Key is not empty", func() {
				Expect(v0.ID{Type: v0.ResourceTypeChannel, Key: "foo"}.IsType()).To(BeFalse())
			})
		})
	})
})
